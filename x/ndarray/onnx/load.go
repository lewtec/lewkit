package onnx

import (
	"fmt"
	"io"

	"github.com/lewtec/lewkit/x/ndarray/onnx/internal/proto"
	goproto "google.golang.org/protobuf/proto"
)

// Model is a loaded ONNX graph. Initializers stay as TensorProto until FunctionOf.
type Model struct {
	IRVersion    int64
	ProducerName string
	Opsets       []Opset
	Graph        Graph
}

// Opset is a (domain, version) pair from ModelProto.opset_import.
type Opset struct {
	Domain  string
	Version int64
}

// Graph is the model's computation graph.
type Graph struct {
	Name         string
	Inputs       []Value
	Outputs      []Value
	Nodes        []Node
	Initializers []*proto.TensorProto
}

// Value is a named graph input or output.
type Value struct {
	Name string
}

// Node is one operator call.
type Node struct {
	Name         string
	OperatorType string
	Domain       string
	Inputs       []string
	Outputs      []string
	Attributes   []Attribute
}

// Attribute is a node attribute.
type Attribute struct {
	Name     string
	Integer  int64
	Integers []int64
	Float    float32
	Floats   []float32
	String   string
	Tensor   *proto.TensorProto
}

func (n Node) attributeIntegers(name string) []int64 {
	for _, a := range n.Attributes {
		if a.Name == name {
			if len(a.Integers) > 0 {
				return a.Integers
			}
			if a.Integer != 0 {
				return []int64{a.Integer}
			}
		}
	}
	return nil
}

func (n Node) attributeInteger(name string, fallback int64) int64 {
	for _, a := range n.Attributes {
		if a.Name != name {
			continue
		}
		if len(a.Integers) == 1 {
			return a.Integers[0]
		}
		return a.Integer
	}
	return fallback
}

func (n Node) attributeString(name string) string {
	for _, a := range n.Attributes {
		if a.Name == name {
			return a.String
		}
	}
	return ""
}

func (n Node) attributeFloat(name string, fallback float32) float32 {
	for _, a := range n.Attributes {
		if a.Name == name {
			return a.Float
		}
	}
	return fallback
}

func (n Node) attributeFloats(name string) []float32 {
	for _, a := range n.Attributes {
		if a.Name == name {
			if len(a.Floats) > 0 {
				return a.Floats
			}
			if a.Float != 0 {
				return []float32{a.Float}
			}
		}
	}
	return nil
}

func (n Node) attributeTensor(name string) *proto.TensorProto {
	for _, a := range n.Attributes {
		if a.Name == name {
			return a.Tensor
		}
	}
	return nil
}

// Load unmarshals an ONNX ModelProto from r.
func Load(r io.Reader) (*Model, error) {
	if r == nil {
		return nil, ErrEmpty
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrModel, err)
	}
	return LoadBytes(b)
}

// LoadBytes unmarshals an ONNX ModelProto from b.
func LoadBytes(b []byte) (*Model, error) {
	if len(b) == 0 {
		return nil, ErrEmpty
	}
	msg := new(proto.ModelProto)
	if err := goproto.Unmarshal(b, msg); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrModel, err)
	}
	g := msg.GetGraph()
	if g == nil {
		return nil, ErrGraph
	}
	m := &Model{
		IRVersion:    msg.GetIrVersion(),
		ProducerName: msg.GetProducerName(),
		Graph: Graph{
			Name: g.GetName(),
		},
	}
	for _, o := range msg.GetOpsetImport() {
		m.Opsets = append(m.Opsets, Opset{Domain: o.GetDomain(), Version: o.GetVersion()})
	}
	for _, v := range g.GetInput() {
		m.Graph.Inputs = append(m.Graph.Inputs, Value{Name: v.GetName()})
	}
	for _, v := range g.GetOutput() {
		m.Graph.Outputs = append(m.Graph.Outputs, Value{Name: v.GetName()})
	}
	for _, n := range g.GetNode() {
		node := Node{
			Name:         n.GetName(),
			OperatorType: n.GetOpType(),
			Domain:       n.GetDomain(),
			Inputs:       append([]string(nil), n.GetInput()...),
			Outputs:      append([]string(nil), n.GetOutput()...),
		}
		for _, a := range n.GetAttribute() {
			node.Attributes = append(node.Attributes, Attribute{
				Name:     a.GetName(),
				Integer:  a.GetI(),
				Integers: append([]int64(nil), a.GetInts()...),
				Float:    a.GetF(),
				Floats:   append([]float32(nil), a.GetFloats()...),
				String:   string(a.GetS()),
				Tensor:   a.GetT(),
			})
		}
		m.Graph.Nodes = append(m.Graph.Nodes, node)
	}
	m.Graph.Initializers = append(m.Graph.Initializers, g.GetInitializer()...)
	return m, nil
}
