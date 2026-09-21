package onnx

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"maps"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ndarray/onnx/internal/proto"
	"github.com/lewtec/lewkit/x/path/pick"
	"github.com/stretchr/testify/require"
	goproto "google.golang.org/protobuf/proto"
)

//go:embed all:testdata/node
var officialNodeTests embed.FS

type nodeCase struct {
	name      string
	model     []byte
	inputs    map[string]*proto.TensorProto
	outputs   map[string]*proto.TensorProto
	operators []string
}

func TestNodeHarness(t *testing.T) {
	cases := generatedNodeCases(t)
	root, err := iofs.Sub(officialNodeTests, "testdata/node")
	require.NoError(t, err)
	official, err := discoverNodeCases(t.Context(), root)
	require.NoError(t, err)
	if len(official) == 0 {
		t.Run("official", func(t *testing.T) {
			t.Skip("onnx node tests not downloaded")
		})
	}
	cases = append(cases, official...)
	var results []CaseResult
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := runNodeCase(t, c)
			results = append(results, result)
			switch result.Status {
			case StatusSkipped:
				t.Log(result.Detail)
			case StatusFailed:
				t.Logf("FAIL: %s", result.Detail)
				if strings.HasPrefix(c.name, "generated_") {
					t.Fatal(result.Detail)
				}
			}
		})
	}
	report := CoverageFromResults(results)
	t.Log("\n" + report.String())
}

func TestCoverageEmpty(t *testing.T) {
	report := CoverageFromResults(nil)
	require.Equal(t, len(onnxOperators), report.Catalog)
	require.Equal(t, len(implementedOperators()), report.Implemented)
	require.Equal(t, 0, report.Passed)
	require.Greater(t, report.Catalog, 100)
	t.Log("\n" + report.String())
}

func TestDiscoverNodeCasesEmpty(t *testing.T) {
	cases, err := discoverNodeCases(t.Context(), fstest.MapFS{})
	require.NoError(t, err)
	require.Empty(t, cases)
}

func runNodeCase(t *testing.T, c nodeCase) CaseResult {
	t.Helper()
	result := CaseResult{Name: c.name, Operators: c.operators, Status: StatusFailed}
	switch caseDtype(c) {
	case proto.TensorProto_INT32:
		return runNodeCaseT[int32](t, c, result)
	case proto.TensorProto_UINT8:
		return runNodeCaseT[uint8](t, c, result)
	default:
		return runNodeCaseT[float32](t, c, result)
	}
}

func caseDtype(c nodeCase) proto.TensorProto_DataType {
	var sawInt32, sawUint8, sawBool bool
	for _, tensors := range []map[string]*proto.TensorProto{c.inputs, c.outputs} {
		for _, tensor := range tensors {
			dt := proto.TensorProto_DataType(tensor.GetDataType())
			switch dt {
			case proto.TensorProto_FLOAT, proto.TensorProto_DOUBLE:
				return proto.TensorProto_FLOAT
			case proto.TensorProto_INT32:
				sawInt32 = true
			case proto.TensorProto_UINT8:
				sawUint8 = true
			case proto.TensorProto_BOOL:
				sawBool = true
			}
		}
	}
	switch {
	case sawInt32:
		return proto.TensorProto_INT32
	case sawBool:
		return proto.TensorProto_INT32
	case sawUint8:
		return proto.TensorProto_UINT8
	default:
		return proto.TensorProto_FLOAT
	}
}

func runNodeCaseT[T ndarray.Number](t *testing.T, c nodeCase, result CaseResult) CaseResult {
	t.Helper()
	model, err := LoadBytes(c.model)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	function, err := FunctionOf[T](model)
	if err != nil {
		result.Detail = err.Error()
		return result
	}
	if result.Operators == nil {
		result.Operators = function.operators()
	}
	if err := function.supported(); err != nil {
		result.Status = StatusSkipped
		result.Detail = err.Error()
		return result
	}
	caseInputs := remapCaseTensors(c.inputs, function.inputs, "input_")
	caseOutputs := remapCaseTensors(c.outputs, function.outputs, "output_")
	inputs := map[string]*ndarray.Tensor[T]{}
	for name, p := range caseInputs {
		if dt := proto.TensorProto_DataType(p.GetDataType()); dt == proto.TensorProto_INT64 || dt == proto.TensorProto_INT32 {
			ints, err := int64sFromNumeric(p)
			if err != nil {
				result.Detail = err.Error()
				return result
			}
			if function.integerShapes == nil {
				function.integerShapes = map[string][]int64{}
			}
			function.integerShapes[name] = ints
		}
		value, err := tensorFromProto[T](p)
		if err != nil {
			result.Status = StatusSkipped
			result.Detail = err.Error()
			return result
		}
		inputs[name] = value
	}
	outputs, err := function.ApplyInputs(t.Context(), inputs)
	if err != nil {
		if errors.Is(err, ErrOp) || errors.Is(err, ndarray.ErrType) {
			result.Status = StatusSkipped
		}
		result.Detail = err.Error()
		return result
	}
	var zero T
	_, isFloat := any(zero).(float32)
	for name, want := range caseOutputs {
		got, ok := outputs[name]
		if !ok {
			result.Detail = "missing output " + name
			return result
		}
		data := make([]T, got.Size())
		if err := got.Eval(t.Context(), ndarray.CPU, data); err != nil {
			if errors.Is(err, ndarray.ErrType) || errors.Is(err, ErrOp) {
				result.Status = StatusSkipped
			}
			result.Detail = err.Error()
			return result
		}
		wantData, err := valuesFromProto[T](want)
		if err != nil {
			result.Status = StatusSkipped
			result.Detail = err.Error()
			return result
		}
		if len(data) != len(wantData) {
			result.Detail = fmt.Sprintf("output size got %d want %d shape %v", len(data), len(wantData), got.Shape())
			return result
		}
		if isFloat {
			const delta = float32(1e-4)
			fd := any(data).([]float32)
			fw := any(wantData).([]float32)
			for i := range fd {
				d := fd[i] - fw[i]
				if d < -delta || d > delta {
					result.Detail = "output mismatch"
					return result
				}
			}
			continue
		}
		for i := range data {
			if data[i] != wantData[i] {
				result.Detail = "output mismatch"
				return result
			}
		}
	}
	result.Status = StatusPassed
	return result
}

func generatedNodeCases(t *testing.T) []nodeCase {
	t.Helper()
	return []nodeCase{
		makeAbsCase(t),
		makeAddCase(t),
		makeReluCase(t),
	}
}

func makeAbsCase(t *testing.T) nodeCase {
	t.Helper()
	model := mustModel(t, "Abs", []string{"X"}, []string{"Y"})
	return nodeCase{
		name:      "generated_abs",
		operators: []string{"Abs"},
		model:     model,
		inputs: map[string]*proto.TensorProto{
			"X": floatProto("X", []int64{3}, []float32{-2, 0, 3}),
		},
		outputs: map[string]*proto.TensorProto{
			"Y": floatProto("Y", []int64{3}, []float32{2, 0, 3}),
		},
	}
}

func makeAddCase(t *testing.T) nodeCase {
	t.Helper()
	model := mustModel(t, "Add", []string{"A", "B"}, []string{"C"})
	return nodeCase{
		name:      "generated_add",
		model:     model,
		operators: []string{"Add"},
		inputs: map[string]*proto.TensorProto{
			"A": floatProto("A", []int64{2}, []float32{1, 2}),
			"B": floatProto("B", []int64{2}, []float32{3, 4}),
		},
		outputs: map[string]*proto.TensorProto{
			"C": floatProto("C", []int64{2}, []float32{4, 6}),
		},
	}
}

func makeReluCase(t *testing.T) nodeCase {
	t.Helper()
	model := mustModel(t, "Relu", []string{"X"}, []string{"Y"})
	return nodeCase{
		name:      "generated_relu",
		model:     model,
		operators: []string{"Relu"},
		inputs: map[string]*proto.TensorProto{
			"X": floatProto("X", []int64{3}, []float32{-1, 0, 2}),
		},
		outputs: map[string]*proto.TensorProto{
			"Y": floatProto("Y", []int64{3}, []float32{0, 0, 2}),
		},
	}
}

func mustModel(t *testing.T, op string, inputs, outputs []string) []byte {
	t.Helper()
	g := &proto.GraphProto{Name: goproto.String(op)}
	for _, name := range inputs {
		g.Input = append(g.Input, &proto.ValueInfoProto{Name: goproto.String(name)})
	}
	for _, name := range outputs {
		g.Output = append(g.Output, &proto.ValueInfoProto{Name: goproto.String(name)})
	}
	g.Node = []*proto.NodeProto{{
		OpType: goproto.String(op),
		Input:  inputs,
		Output: outputs,
	}}
	msg := &proto.ModelProto{
		IrVersion:    goproto.Int64(8),
		ProducerName: goproto.String("lewkit-test"),
		OpsetImport:  []*proto.OperatorSetIdProto{{Version: goproto.Int64(13)}},
		Graph:        g,
	}
	b, err := goproto.Marshal(msg)
	require.NoError(t, err)
	return b
}

func floatProto(name string, dims []int64, data []float32) *proto.TensorProto {
	return &proto.TensorProto{
		Name:      goproto.String(name),
		Dims:      dims,
		DataType:  goproto.Int32(int32(proto.TensorProto_FLOAT)),
		FloatData: data,
	}
}

func discoverNodeCases(ctx context.Context, fsys iofs.FS) ([]nodeCase, error) {
	byName := map[string]*nodeCase{}
	var order []string
	pred := pick.Or(
		pick.Glob("**/model.onnx"),
		pick.Match("**/input_*.pb"),
		pick.Match("**/output_*.pb"),
	)
	for file, err := range lewfs.Walk(ctx, fsys, pred) {
		if err != nil {
			return nil, err
		}
		if skipExtraDataSet(file.Name.Parts()) {
			continue
		}
		body, err := io.ReadAll(file.Reader)
		if err != nil {
			return nil, err
		}
		parts := file.Name.Parts()
		if len(parts) == 0 {
			continue
		}
		caseName := parts[0]
		c, ok := byName[caseName]
		if !ok {
			c = &nodeCase{name: caseName, inputs: map[string]*proto.TensorProto{}, outputs: map[string]*proto.TensorProto{}}
			byName[caseName] = c
			order = append(order, caseName)
		}
		base := file.Name.Name()
		switch {
		case base == "model.onnx":
			c.model = body
		case strings.HasPrefix(base, "input_"), strings.HasPrefix(base, "output_"):
			tensor, err := unmarshalTensorProto(body)
			if err != nil {
				return nil, err
			}
			name := tensor.GetName()
			if name == "" {
				name = file.Name.Stem()
			}
			if strings.HasPrefix(base, "input_") {
				c.inputs[name] = tensor
			} else {
				c.outputs[name] = tensor
			}
		}
	}
	var cases []nodeCase
	for _, name := range order {
		c := byName[name]
		if len(c.model) == 0 {
			continue
		}
		cases = append(cases, *c)
	}
	return cases, nil
}

func skipExtraDataSet(parts []string) bool {
	for _, part := range parts {
		if after, ok := strings.CutPrefix(part, "test_data_set_"); ok && after != "0" {
			return true
		}
	}
	return false
}

func remapCaseTensors(named map[string]*proto.TensorProto, graphNames []string, prefix string) map[string]*proto.TensorProto {
	out := maps.Clone(named)
	if out == nil {
		out = map[string]*proto.TensorProto{}
	}
	for i, name := range graphNames {
		if _, ok := out[name]; ok {
			continue
		}
		if tensor, ok := named[prefix+strconv.Itoa(i)]; ok {
			out[name] = tensor
		}
	}
	return out
}
