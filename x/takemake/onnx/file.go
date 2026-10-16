package onnx

import (
	"fmt"
	"io"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrNilReader = errs.Wrap(errs.ErrInvalid, "onnx: nil reader")
	ErrNilWriter = errs.Wrap(errs.ErrInvalid, "onnx: nil writer")
	ErrNilFile   = errs.Wrap(errs.ErrInvalid, "onnx: nil file")
	ErrFormat    = errs.Wrap(errs.ErrSyntax, "onnx: invalid model")
)

// Opset is one imported operator set.
// An empty Domain is the default ONNX domain.
type Opset struct {
	Domain  string
	Version int64
}

// Pair is one string key and value.
// External tensor data and nothing else uses it in this model.
type Pair struct {
	Key   string
	Value string
}

// Segment is the slice of a tensor stored in this message.
type Segment struct {
	Begin int64
	End   int64
}

// Type is a TypeProto.
// HasShape distinguishes a scalar, or a known rank, from an absent shape.
// Nested is the sequence element, the optional element, or the map value.
// Key is the map key type.
type Type struct {
	Kind       TypeKind
	Element    DataType
	Shape      []Dim
	HasShape   bool
	Denotation string
	Key        DataType
	Nested     *Type
}

// Value is a named graph input, output, or value info.
type Value struct {
	Name      string
	DocString string
	Type      *Type
}

// Tensor is a dense tensor.
// Elements live in one of the typed slices, in Raw, or outside the model.
type Tensor struct {
	Name       string
	DocString  string
	Element    DataType
	Dims       []int64
	Segment    *Segment
	FloatData  []float32
	Int32Data  []int32
	Int64Data  []int64
	DoubleData []float64
	Uint64Data []uint64
	StringData [][]byte
	Raw        []byte
	External   []Pair
	Location   DataLocation
}

// Sparse is a sparse tensor.
// Values and Indices follow the ONNX SparseTensorProto layout.
type Sparse struct {
	Values  *Tensor
	Indices *Tensor
	Dims    []int64
}

// Attribute is one named node attribute.
// Type selects the payload. The other fields keep what Take read,
// so a message that sets more than one payload still round-trips.
type Attribute struct {
	Name        string
	DocString   string
	Type        AttributeType
	RefAttrName string
	Float       float32
	Int         int64
	Text        string
	Tensor      *Tensor
	Graph       *Graph
	Floats      []float32
	Ints        []int64
	Strings     [][]byte
	Tensors     []*Tensor
	Graphs      []*Graph
	Sparse      *Sparse
	Sparses     []*Sparse
	TypeProto   *Type
	TypeProtos  []*Type
}

// Node is one operator call.
// An empty string in Inputs or Outputs is an omitted optional value.
type Node struct {
	Name       string
	OpType     string
	Domain     string
	Overload   string
	DocString  string
	Inputs     []string
	Outputs    []string
	Attributes []*Attribute
}

// Graph is the model's computation graph, or a graph attribute.
type Graph struct {
	Name               string
	DocString          string
	Inputs             []*Value
	Outputs            []*Value
	ValueInfos         []*Value
	Initializers       []*Tensor
	SparseInitializers []*Sparse
	Nodes              []*Node
}

// File is one ONNX model.
type File struct {
	IRVersion       int64
	Opsets          []Opset
	ProducerName    string
	ProducerVersion string
	Domain          string
	ModelVersion    int64
	DocString       string
	Graph           *Graph
}

// New returns an empty file.
func New() *File {
	return &File{}
}

// SetIR sets the IR version and returns the file.
func (f *File) SetIR(version int64) *File {
	if f == nil {
		return nil
	}
	f.IRVersion = version
	return f
}

// Opset appends an operator-set import and returns the file.
func (f *File) Opset(domain string, version int64) *File {
	if f == nil {
		return nil
	}
	f.Opsets = append(f.Opsets, Opset{Domain: domain, Version: version})
	return f
}

// SetProducer sets the tool that wrote the model and returns the file.
func (f *File) SetProducer(name, version string) *File {
	if f == nil {
		return nil
	}
	f.ProducerName = name
	f.ProducerVersion = version
	return f
}

// SetDomain sets the model domain and returns the file.
func (f *File) SetDomain(domain string) *File {
	if f == nil {
		return nil
	}
	f.Domain = domain
	return f
}

// SetVersion sets the model version and returns the file.
func (f *File) SetVersion(version int64) *File {
	if f == nil {
		return nil
	}
	f.ModelVersion = version
	return f
}

// SetDoc sets the model documentation and returns the file.
func (f *File) SetDoc(text string) *File {
	if f == nil {
		return nil
	}
	f.DocString = text
	return f
}

// SetGraph sets the graph name and returns the file.
func (f *File) SetGraph(name string) *File {
	if f == nil {
		return nil
	}
	f.ensure().Name = name
	return f
}

// Input appends a graph input and returns it.
func (f *File) Input(name string) *Value {
	if f == nil {
		return nil
	}
	return f.ensure().Input(name)
}

// Output appends a graph output and returns it.
func (f *File) Output(name string) *Value {
	if f == nil {
		return nil
	}
	return f.ensure().Output(name)
}

// ValueInfo appends an intermediate value and returns it.
func (f *File) ValueInfo(name string) *Value {
	if f == nil {
		return nil
	}
	return f.ensure().ValueInfo(name)
}

// Initializer appends a named tensor and returns it.
func (f *File) Initializer(name string) *Tensor {
	if f == nil {
		return nil
	}
	return f.ensure().Initializer(name)
}

// SparseInitializer appends a sparse tensor and returns it.
func (f *File) SparseInitializer() *Sparse {
	if f == nil {
		return nil
	}
	return f.ensure().SparseInitializer()
}

// Node appends a node and returns it.
func (f *File) Node(name string) *Node {
	if f == nil {
		return nil
	}
	return f.ensure().Node(name)
}

func (f *File) ensure() *Graph {
	if f.Graph == nil {
		f.Graph = &Graph{}
	}
	return f.Graph
}

// SetDoc sets the graph documentation and returns the graph.
func (g *Graph) SetDoc(text string) *Graph {
	if g == nil {
		return nil
	}
	g.DocString = text
	return g
}

// Input appends a graph input and returns it.
func (g *Graph) Input(name string) *Value {
	if g == nil {
		return nil
	}
	value := &Value{Name: name}
	g.Inputs = append(g.Inputs, value)
	return value
}

// Output appends a graph output and returns it.
func (g *Graph) Output(name string) *Value {
	if g == nil {
		return nil
	}
	value := &Value{Name: name}
	g.Outputs = append(g.Outputs, value)
	return value
}

// ValueInfo appends an intermediate value and returns it.
func (g *Graph) ValueInfo(name string) *Value {
	if g == nil {
		return nil
	}
	value := &Value{Name: name}
	g.ValueInfos = append(g.ValueInfos, value)
	return value
}

// Initializer appends a named tensor and returns it.
func (g *Graph) Initializer(name string) *Tensor {
	if g == nil {
		return nil
	}
	tensor := &Tensor{Name: name}
	g.Initializers = append(g.Initializers, tensor)
	return tensor
}

// SparseInitializer appends a sparse tensor and returns it.
func (g *Graph) SparseInitializer() *Sparse {
	if g == nil {
		return nil
	}
	sparse := &Sparse{}
	g.SparseInitializers = append(g.SparseInitializers, sparse)
	return sparse
}

// Node appends a node and returns it.
func (g *Graph) Node(name string) *Node {
	if g == nil {
		return nil
	}
	node := &Node{Name: name}
	g.Nodes = append(g.Nodes, node)
	return node
}

// SetDoc sets the value documentation and returns the value.
func (v *Value) SetDoc(text string) *Value {
	if v == nil {
		return nil
	}
	v.DocString = text
	return v
}

// SetTensor sets a tensor type and returns the value.
// The shape is known, including a scalar when dims is empty.
func (v *Value) SetTensor(element DataType, dims ...Dim) *Value {
	if v == nil {
		return nil
	}
	v.Type = &Type{
		Kind:     TypeKindTensor,
		Element:  element,
		Shape:    append([]Dim(nil), dims...),
		HasShape: true,
	}
	return v
}

// SetElement sets the tensor data type and returns the tensor.
func (t *Tensor) SetElement(element DataType) *Tensor {
	if t == nil {
		return nil
	}
	t.Element = element
	return t
}

// Shape sets the tensor dimensions and returns the tensor.
func (t *Tensor) Shape(dims ...int64) *Tensor {
	if t == nil {
		return nil
	}
	t.Dims = append([]int64(nil), dims...)
	return t
}

// SetDoc sets the tensor documentation and returns the tensor.
func (t *Tensor) SetDoc(text string) *Tensor {
	if t == nil {
		return nil
	}
	t.DocString = text
	return t
}

// SetSegment records the stored slice and returns the tensor.
func (t *Tensor) SetSegment(begin, end int64) *Tensor {
	if t == nil {
		return nil
	}
	t.Segment = &Segment{Begin: begin, End: end}
	return t
}

// SetFloatData sets packed float elements and returns the tensor.
func (t *Tensor) SetFloatData(values ...float32) *Tensor {
	if t == nil {
		return nil
	}
	t.FloatData = append([]float32(nil), values...)
	return t
}

// SetInt32Data sets packed int32 elements and returns the tensor.
func (t *Tensor) SetInt32Data(values ...int32) *Tensor {
	if t == nil {
		return nil
	}
	t.Int32Data = append([]int32(nil), values...)
	return t
}

// SetInt64Data sets packed int64 elements and returns the tensor.
func (t *Tensor) SetInt64Data(values ...int64) *Tensor {
	if t == nil {
		return nil
	}
	t.Int64Data = append([]int64(nil), values...)
	return t
}

// SetDoubleData sets packed double elements and returns the tensor.
func (t *Tensor) SetDoubleData(values ...float64) *Tensor {
	if t == nil {
		return nil
	}
	t.DoubleData = append([]float64(nil), values...)
	return t
}

// SetUint64Data sets packed uint64 elements and returns the tensor.
func (t *Tensor) SetUint64Data(values ...uint64) *Tensor {
	if t == nil {
		return nil
	}
	t.Uint64Data = append([]uint64(nil), values...)
	return t
}

// SetStringData sets string elements and returns the tensor.
func (t *Tensor) SetStringData(values ...string) *Tensor {
	if t == nil {
		return nil
	}
	t.StringData = cloneStrings(values)
	return t
}

// SetRaw sets little-endian raw elements and returns the tensor.
func (t *Tensor) SetRaw(data []byte) *Tensor {
	if t == nil {
		return nil
	}
	t.Raw = cloneBytes(data)
	return t
}

// ExternalData appends one external-data entry and returns the tensor.
func (t *Tensor) ExternalData(key, value string) *Tensor {
	if t == nil {
		return nil
	}
	t.External = append(t.External, Pair{Key: key, Value: value})
	return t
}

// SetLocation sets where the elements live and returns the tensor.
func (t *Tensor) SetLocation(location DataLocation) *Tensor {
	if t == nil {
		return nil
	}
	t.Location = location
	return t
}

// SetValues sets the sparse values tensor and returns it.
func (s *Sparse) SetValues(name string) *Tensor {
	if s == nil {
		return nil
	}
	s.Values = &Tensor{Name: name}
	return s.Values
}

// SetIndices sets the sparse indices tensor and returns it.
func (s *Sparse) SetIndices(name string) *Tensor {
	if s == nil {
		return nil
	}
	s.Indices = &Tensor{Name: name}
	return s.Indices
}

// SetDims sets the dense shape and returns the sparse tensor.
func (s *Sparse) SetDims(dims ...int64) *Sparse {
	if s == nil {
		return nil
	}
	s.Dims = append([]int64(nil), dims...)
	return s
}

// Op sets the operator type and returns the node.
func (n *Node) Op(opType string) *Node {
	if n == nil {
		return nil
	}
	n.OpType = opType
	return n
}

// SetDomain sets the operator-set domain and returns the node.
func (n *Node) SetDomain(domain string) *Node {
	if n == nil {
		return nil
	}
	n.Domain = domain
	return n
}

// SetOverload sets the function overload and returns the node.
func (n *Node) SetOverload(name string) *Node {
	if n == nil {
		return nil
	}
	n.Overload = name
	return n
}

// SetDoc sets the node documentation and returns the node.
func (n *Node) SetDoc(text string) *Node {
	if n == nil {
		return nil
	}
	n.DocString = text
	return n
}

// Input appends input names and returns the node.
func (n *Node) Input(names ...string) *Node {
	if n == nil {
		return nil
	}
	n.Inputs = append(n.Inputs, names...)
	return n
}

// Output appends output names and returns the node.
func (n *Node) Output(names ...string) *Node {
	if n == nil {
		return nil
	}
	n.Outputs = append(n.Outputs, names...)
	return n
}

// Attr appends an attribute and returns it.
func (n *Node) Attr(name string) *Attribute {
	if n == nil {
		return nil
	}
	attr := &Attribute{Name: name}
	n.Attributes = append(n.Attributes, attr)
	return attr
}

// SetDoc sets the attribute documentation and returns the attribute.
func (a *Attribute) SetDoc(text string) *Attribute {
	if a == nil {
		return nil
	}
	a.DocString = text
	return a
}

// SetRef points the attribute at a parent function attribute.
func (a *Attribute) SetRef(name string) *Attribute {
	if a == nil {
		return nil
	}
	a.RefAttrName = name
	return a
}

// SetFloat stores a float and returns the attribute.
func (a *Attribute) SetFloat(value float32) *Attribute {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeFloat
	a.Float = value
	return a
}

// SetInt stores an int and returns the attribute.
func (a *Attribute) SetInt(value int64) *Attribute {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeInt
	a.Int = value
	return a
}

// SetText stores a string and returns the attribute.
func (a *Attribute) SetText(text string) *Attribute {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeString
	a.Text = text
	return a
}

// SetFloats stores a float list and returns the attribute.
func (a *Attribute) SetFloats(values ...float32) *Attribute {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeFloats
	a.Floats = append([]float32(nil), values...)
	return a
}

// SetInts stores an int list and returns the attribute.
func (a *Attribute) SetInts(values ...int64) *Attribute {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeInts
	a.Ints = append([]int64(nil), values...)
	return a
}

// SetStrings stores a string list and returns the attribute.
func (a *Attribute) SetStrings(values ...string) *Attribute {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeStrings
	a.Strings = cloneStrings(values)
	return a
}

// SetTensor stores one tensor and returns it.
func (a *Attribute) SetTensor(name string) *Tensor {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeTensor
	a.Tensor = &Tensor{Name: name}
	return a.Tensor
}

// AddTensor appends a tensor and returns it.
func (a *Attribute) AddTensor(name string) *Tensor {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeTensors
	tensor := &Tensor{Name: name}
	a.Tensors = append(a.Tensors, tensor)
	return tensor
}

// SetGraph stores one graph and returns it.
func (a *Attribute) SetGraph(name string) *Graph {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeGraph
	a.Graph = &Graph{Name: name}
	return a.Graph
}

// AddGraph appends a graph and returns it.
func (a *Attribute) AddGraph(name string) *Graph {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeGraphs
	graph := &Graph{Name: name}
	a.Graphs = append(a.Graphs, graph)
	return graph
}

// SetSparse stores one sparse tensor and returns it.
func (a *Attribute) SetSparse() *Sparse {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeSparseTensor
	a.Sparse = &Sparse{}
	return a.Sparse
}

// AddSparse appends a sparse tensor and returns it.
func (a *Attribute) AddSparse() *Sparse {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeSparseTensors
	sparse := &Sparse{}
	a.Sparses = append(a.Sparses, sparse)
	return sparse
}

// SetType stores one type and returns the attribute.
func (a *Attribute) SetType(typ *Type) *Attribute {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeType
	a.TypeProto = typ
	return a
}

// AddType appends a type and returns the attribute.
func (a *Attribute) AddType(typ *Type) *Attribute {
	if a == nil {
		return nil
	}
	a.Type = AttributeTypeTypes
	a.TypeProtos = append(a.TypeProtos, typ)
	return a
}

// Take reads a ModelProto into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, ErrNilReader
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFormat, err)
	}
	return decodeModel(raw)
}

// Make writes the model as a ModelProto.
func (f *File) Make(w io.Writer) error {
	if f == nil {
		return ErrNilFile
	}
	if w == nil {
		return ErrNilWriter
	}
	_, err := w.Write(encodeModel(f))
	if err != nil {
		return err
	}
	return nil
}

func cloneBytes(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	return append([]byte(nil), data...)
}

func cloneStrings(values []string) [][]byte {
	if len(values) == 0 {
		return nil
	}
	out := make([][]byte, len(values))
	for i, value := range values {
		if value != "" {
			out[i] = []byte(value)
		}
	}
	return out
}
