package onnx

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	onnxpb "github.com/lewtec/lewkit/x/ndarray/onnx/internal/proto"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
	goproto "google.golang.org/protobuf/proto"
)

func TestLoadRoundTrip(t *testing.T) {
	raw := []byte{0, 1, 2, 3}
	src := &onnxpb.ModelProto{
		IrVersion:    goproto.Int64(10),
		ProducerName: goproto.String("lewkit-test"),
		OpsetImport: []*onnxpb.OperatorSetIdProto{{
			Domain:  goproto.String(""),
			Version: goproto.Int64(18),
		}},
		Graph: &onnxpb.GraphProto{
			Name: goproto.String("add"),
			Input: []*onnxpb.ValueInfoProto{
				{Name: goproto.String("a")},
				{Name: goproto.String("b")},
			},
			Output: []*onnxpb.ValueInfoProto{
				{Name: goproto.String("c")},
			},
			Node: []*onnxpb.NodeProto{{
				Name:   goproto.String("add0"),
				OpType: goproto.String("Add"),
				Input:  []string{"a", "b"},
				Output: []string{"c"},
			}},
			Initializer: []*onnxpb.TensorProto{{
				Name:     goproto.String("b"),
				Dims:     []int64{2},
				DataType: goproto.Int32(1),
				RawData:  raw,
			}},
		},
	}
	b, err := goproto.Marshal(src)
	require.NoError(t, err)

	m, err := Load(bytes.NewReader(b))
	require.NoError(t, err)
	require.Equal(t, int64(10), m.IRVersion)
	require.Equal(t, "lewkit-test", m.ProducerName)
	require.Equal(t, []Opset{{Domain: "", Version: 18}}, m.Opsets)
	require.Equal(t, "add", m.Graph.Name)
	require.Equal(t, []Value{{Name: "a"}, {Name: "b"}}, m.Graph.Inputs)
	require.Equal(t, []Value{{Name: "c"}}, m.Graph.Outputs)
	require.Equal(t, []Node{{
		Name:         "add0",
		OperatorType: "Add",
		Inputs:       []string{"a", "b"},
		Outputs:      []string{"c"},
	}}, m.Graph.Nodes)
	require.Len(t, m.Graph.Initializers, 1)
	got := m.Graph.Initializers[0]
	require.Equal(t, "b", got.GetName())
	require.Equal(t, []int64{2}, got.GetDims())
	require.Equal(t, int32(1), got.GetDataType())
	require.Equal(t, raw, got.GetRawData())
}

func TestLoadEmpty(t *testing.T) {
	_, err := LoadBytes(nil)
	require.ErrorIs(t, err, ErrEmpty)
	_, err = Load(bytes.NewReader(nil))
	require.ErrorIs(t, err, ErrEmpty)
	_, err = Load(nil)
	require.ErrorIs(t, err, ErrEmpty)
}

func TestLoadGarbage(t *testing.T) {
	_, err := LoadBytes([]byte("not onnx"))
	require.ErrorIs(t, err, ErrModel)
}

func TestLoadNoGraph(t *testing.T) {
	b, err := goproto.Marshal(&onnxpb.ModelProto{IrVersion: goproto.Int64(10)})
	require.NoError(t, err)
	_, err = LoadBytes(b)
	require.ErrorIs(t, err, ErrGraph)
}

func TestLoadReaderError(t *testing.T) {
	_, err := Load(test.ErrorReader{Err: io.ErrUnexpectedEOF})
	require.ErrorIs(t, err, ErrModel)
}

// MNIST-12 from onnx/models, with official test_data_set_0.
// Input is a preprocessed 1x1x28x28 float32 tensor; logits argmax is 3.
func TestLoadMNIST(t *testing.T) {
	dir := filepath.Join("testdata", "mnist-12")
	b, err := os.ReadFile(filepath.Join(dir, "mnist-12.onnx"))
	require.NoError(t, err)
	m, err := LoadBytes(b)
	require.NoError(t, err)
	require.Equal(t, "CNTK", m.ProducerName)
	require.Equal(t, []Opset{{Version: 12}}, m.Opsets)
	require.Equal(t, 12, len(m.Graph.Nodes))
	require.Equal(t, 8, len(m.Graph.Initializers))
	w, err := tensorFromProto[float32](m.Graph.Initializers[0])
	require.NoError(t, err)
	require.Greater(t, w.Size(), 0)

	in := loadTensorProto(t, filepath.Join(dir, "input_0.pb"))
	require.Equal(t, "Input3", in.GetName())
	require.Equal(t, []int64{1, 1, 28, 28}, in.GetDims())
	require.Len(t, tensorFloats(in), 28*28)

	out := loadTensorProto(t, filepath.Join(dir, "output_0.pb"))
	require.Equal(t, "Plus214_Output_0", out.GetName())
	logits := tensorFloats(out)
	require.Len(t, logits, 10)
	best := 0
	for i, v := range logits {
		if v > logits[best] {
			best = i
		}
	}
	require.Equal(t, 3, best)
}

func TestApplyMNIST(t *testing.T) {
	dir := filepath.Join("testdata", "mnist-12")
	b, err := os.ReadFile(filepath.Join(dir, "mnist-12.onnx"))
	require.NoError(t, err)
	m, err := LoadBytes(b)
	require.NoError(t, err)
	function, err := FunctionOf[float32](m)
	require.NoError(t, err)
	in := loadTensorProto(t, filepath.Join(dir, "input_0.pb"))
	x, err := ndarray.New(tensorFloats(in), ndarray.Shape{1, 1, 28, 28})
	require.NoError(t, err)
	y, err := function.Apply(t.Context(), x)
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{1, 10}, y.Shape())
}

func loadTensorProto(t *testing.T, path string) *onnxpb.TensorProto {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	msg := new(onnxpb.TensorProto)
	require.NoError(t, goproto.Unmarshal(b, msg))
	return msg
}

func tensorFloats(t *onnxpb.TensorProto) []float32 {
	if n := t.GetFloatData(); len(n) > 0 {
		return n
	}
	raw := t.GetRawData()
	out := make([]float32, len(raw)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
	}
	return out
}
