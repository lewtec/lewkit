package onnx

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/ndarray"
	ndonnx "github.com/lewtec/lewkit/x/ndarray/onnx"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestComposedRoundTrip(t *testing.T) {
	t.Parallel()
	file := New().
		SetIR(8).
		Opset("", 18).
		Opset("ai.onnx.ml", 3).
		SetProducer("lewkit", "0").
		SetDomain("ai.lewkit").
		SetVersion(1).
		SetDoc("composed").
		SetGraph("main")
	file.Graph.SetDoc("graph")
	file.Input("x").SetDoc("image").SetTensor(DataTypeFloat, ParamDim("N"), FixedDim(3))
	file.Output("y").SetTensor(DataTypeFloat, ParamDim("N"), FixedDim(3))
	file.ValueInfo("mid").SetTensor(DataTypeInt64, FixedDim(0))
	file.Graph.ValueInfos[0].Type = &Type{
		Kind: TypeKindSequence,
		Nested: &Type{
			Kind:     TypeKindTensor,
			Element:  DataTypeFloat,
			HasShape: true,
			Shape:    []Dim{FixedDim(2)},
		},
	}
	file.Initializer("w").
		SetElement(DataTypeFloat).
		Shape(2).
		SetFloatData(1, -2)
	file.Initializer("shape").
		SetElement(DataTypeInt64).
		Shape(2).
		SetInt64Data(1, 3)
	file.Initializer("raw").
		SetElement(DataTypeUint8).
		Shape(3).
		SetRaw([]byte{9, 8, 7}).
		ExternalData("location", "raw.bin").
		SetLocation(DataLocationExternal)
	file.Initializer("words").
		SetElement(DataTypeString).
		Shape(2).
		SetStringData("a", "")
	sparse := file.SparseInitializer()
	sparse.SetValues("nz").SetElement(DataTypeFloat).Shape(1).SetFloatData(4)
	sparse.SetIndices("ix").SetElement(DataTypeInt64).Shape(1).SetInt64Data(0)
	sparse.SetDims(4)

	then := file.Node("cond").
		Op("If").
		SetDomain("ai.onnx").
		SetDoc("branch").
		Input("c", "", "x").
		Output("y").
		Attr("then_branch").
		SetDoc("then").
		SetGraph("then")
	then.Node("keep").Op("Identity").Input("x").Output("y")
	add := file.Node("add").Op("Add").Input("x", "w").Output("mid")
	add.Attr("axis").SetInt(0)
	add.Attr("pads").SetInts(1, 1)
	add.Attr("alpha").SetFloat(0)
	add.Attr("auto_pad").SetText("")
	add.Attr("bias").SetFloats(0.5)
	add.Attr("labels").SetStrings("yes", "no")
	add.Attr("scale").SetTensor("s").SetElement(DataTypeFloat).Shape(1).SetFloatData(1)

	mapInfo := file.ValueInfo("table")
	mapInfo.Type = &Type{
		Kind: TypeKindMap,
		Key:  DataTypeInt64,
		Nested: &Type{
			Kind:    TypeKindTensor,
			Element: DataTypeString,
		},
		Denotation: "map",
	}

	first := takeMade(t, file)
	require.Equal(t, file.IRVersion, first.IRVersion)
	require.Equal(t, file.Opsets, first.Opsets)
	require.Equal(t, "x", first.Graph.Inputs[0].Name)
	require.Equal(t, ParamDim("N"), first.Graph.Inputs[0].Type.Shape[0])
	require.Equal(t, []float32{1, -2}, first.Graph.Initializers[0].FloatData)
	require.Equal(t, []string{"c", "", "x"}, first.Graph.Nodes[0].Inputs)
	require.Equal(t, "Identity", first.Graph.Nodes[0].Attributes[0].Graph.Nodes[0].OpType)
	require.Equal(t, int64(0), first.Graph.Nodes[1].Attributes[0].Int)
	require.Equal(t, AttributeTypeFloat, first.Graph.Nodes[1].Attributes[2].Type)
	require.Equal(t, float32(0), first.Graph.Nodes[1].Attributes[2].Float)
	require.Equal(t, file, first)

	second := takeMade(t, first)
	require.Equal(t, first, second)
}

func TestRealMNIST(t *testing.T) {
	t.Parallel()
	dir := filepath.Join("..", "..", "ndarray", "onnx", "testdata", "mnist-12")
	raw := mustRead(t, filepath.Join(dir, "mnist-12.onnx"))
	first, err := Take(bytes.NewReader(raw))
	require.NoError(t, err)
	require.Equal(t, int64(7), first.IRVersion)
	require.Equal(t, "CNTK", first.ProducerName)
	require.Equal(t, "2.5.1", first.ProducerVersion)
	require.Equal(t, "ai.cntk", first.Domain)
	require.Equal(t, int64(1), first.ModelVersion)
	require.Equal(t, []Opset{{Version: 12}}, first.Opsets)
	require.Equal(t, "CNTKGraph", first.Graph.Name)
	require.Len(t, first.Graph.Nodes, 12)
	require.Len(t, first.Graph.Initializers, 8)
	require.Len(t, first.Graph.ValueInfos, 11)
	require.Equal(t, "Input3", first.Graph.Inputs[0].Name)
	require.Equal(t, DataTypeFloat, first.Graph.Inputs[0].Type.Element)
	require.True(t, first.Graph.Inputs[0].Type.HasShape)
	require.Equal(t, []Dim{FixedDim(1), FixedDim(1), FixedDim(28), FixedDim(28)}, first.Graph.Inputs[0].Type.Shape)
	require.Equal(t, "Plus214_Output_0", first.Graph.Outputs[0].Name)
	require.Equal(t, "Conv", first.Graph.Nodes[0].OpType)
	require.Equal(t, []int64{5, 5}, attrInts(t, first.Graph.Nodes[0], "kernel_shape"))
	var floats, ints int
	for _, tensor := range first.Graph.Initializers {
		require.NotEmpty(t, tensor.Name)
		require.Empty(t, tensor.Raw)
		switch tensor.Element {
		case DataTypeFloat:
			require.Equal(t, denseCount(tensor.Dims), len(tensor.FloatData))
			floats++
		case DataTypeInt64:
			require.Equal(t, denseCount(tensor.Dims), len(tensor.Int64Data))
			ints++
		default:
			t.Fatalf("initializer %s element %s", tensor.Name, tensor.Element)
		}
	}
	require.Positive(t, floats)
	require.Positive(t, ints)

	var buf bytes.Buffer
	require.NoError(t, first.Make(&buf))
	made := append([]byte(nil), buf.Bytes()...)
	second, err := Take(bytes.NewReader(made))
	require.NoError(t, err)
	require.Equal(t, first, second)

	loaded, err := ndonnx.Load(bytes.NewReader(made))
	require.NoError(t, err)
	require.Equal(t, "CNTK", loaded.ProducerName)
	require.Equal(t, int64(7), loaded.IRVersion)
	require.Equal(t, []ndonnx.Opset{{Version: 12}}, loaded.Opsets)
	require.Len(t, loaded.Graph.Nodes, 12)
	require.Equal(t, "Conv", loaded.Graph.Nodes[0].OperatorType)
	require.Equal(t, []int64{5, 5}, loadedInts(t, loaded.Graph.Nodes[0], "kernel_shape"))
	require.Len(t, loaded.Graph.Initializers, 8)

	original, err := ndonnx.Load(bytes.NewReader(raw))
	require.NoError(t, err)
	input := floatsFromTensor(t, filepath.Join(dir, "input_0.pb"))
	require.Len(t, input, 28*28)
	official := floatsFromTensor(t, filepath.Join(dir, "output_0.pb"))
	require.Len(t, official, 10)
	require.Equal(t, 3, argmax(official))

	gotOriginal := applyMNIST(t, original, input)
	gotMade := applyMNIST(t, loaded, input)
	require.Equal(t, ndarray.Shape{1, 10}, gotOriginal.Shape())
	require.Equal(t, gotOriginal.Shape(), gotMade.Shape())
}

func TestRejects(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, ErrNilReader)
	require.ErrorIs(t, err, errs.ErrInvalid)

	require.ErrorIs(t, New().Make(nil), ErrNilWriter)
	require.ErrorIs(t, (*File)(nil).Make(io.Discard), ErrNilFile)

	_, err = Take(bytes.NewReader([]byte{0xff, 0x00}))
	require.ErrorIs(t, err, ErrFormat)
	require.ErrorIs(t, err, errs.ErrSyntax)

	_, err = Take(bytes.NewReader([]byte{0x0a, 0x05}))
	require.ErrorIs(t, err, ErrFormat)

	_, err = Take(test.ErrorReader{Err: io.ErrUnexpectedEOF})
	require.ErrorIs(t, err, ErrFormat)

	empty, err := Take(bytes.NewReader(nil))
	require.NoError(t, err)
	require.Equal(t, New(), empty)
	require.Nil(t, empty.Graph)
}

func takeMade(t *testing.T, file *File) *File {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	got, err := Take(&buf)
	require.NoError(t, err)
	return got
}

func attrInts(t *testing.T, node *Node, name string) []int64 {
	t.Helper()
	for _, attr := range node.Attributes {
		if attr.Name == name {
			return attr.Ints
		}
	}
	t.Fatalf("missing attribute %s", name)
	return nil
}

func loadedInts(t *testing.T, node ndonnx.Node, name string) []int64 {
	t.Helper()
	for _, attr := range node.Attributes {
		if attr.Name == name {
			return attr.Integers
		}
	}
	t.Fatalf("missing attribute %s", name)
	return nil
}

func denseCount(dims []int64) int {
	n := 1
	for _, dim := range dims {
		n *= int(dim)
	}
	return n
}

func floatsFromTensor(t *testing.T, path string) []float32 {
	t.Helper()
	tensor, err := decodeTensor(mustRead(t, path), 0)
	require.NoError(t, err)
	if len(tensor.FloatData) > 0 {
		return tensor.FloatData
	}
	raw := tensor.Raw
	out := make([]float32, len(raw)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
	}
	return out
}

func applyMNIST(t *testing.T, model *ndonnx.Model, input []float32) *ndarray.Tensor[float32] {
	t.Helper()
	function, err := ndonnx.FunctionOf[float32](model)
	require.NoError(t, err)
	copied := append([]float32(nil), input...)
	x, err := ndarray.New(copied, ndarray.Shape{1, 1, 28, 28})
	require.NoError(t, err)
	y, err := function.Apply(t.Context(), x)
	require.NoError(t, err)
	return y
}

func argmax(values []float32) int {
	best := 0
	for i, value := range values {
		if value > values[best] {
			best = i
		}
	}
	return best
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}
