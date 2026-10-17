package onnx

import "math"

func encodeModel(file *File) []byte {
	if file == nil {
		return nil
	}
	var dst []byte
	dst = appendOptVarint(dst, 1, uint64(file.IRVersion))
	dst = appendOptString(dst, 2, file.ProducerName)
	dst = appendOptString(dst, 3, file.ProducerVersion)
	dst = appendOptString(dst, 4, file.Domain)
	dst = appendOptVarint(dst, 5, uint64(file.ModelVersion))
	dst = appendOptString(dst, 6, file.DocString)
	if file.Graph != nil {
		dst = appendMessage(dst, 7, encodeGraph(file.Graph))
	}
	for _, op := range file.Opsets {
		dst = appendMessage(dst, 8, encodeOpset(op))
	}
	return dst
}

func encodeOpset(op Opset) []byte {
	var dst []byte
	dst = appendOptString(dst, 1, op.Domain)
	dst = appendOptVarint(dst, 2, uint64(op.Version))
	return dst
}

func encodeGraph(graph *Graph) []byte {
	if graph == nil {
		return nil
	}
	var dst []byte
	for _, node := range graph.Nodes {
		dst = appendMessage(dst, 1, encodeNode(node))
	}
	dst = appendOptString(dst, 2, graph.Name)
	for _, tensor := range graph.Initializers {
		dst = appendMessage(dst, 5, encodeTensor(tensor))
	}
	dst = appendOptString(dst, 10, graph.DocString)
	for _, value := range graph.Inputs {
		dst = appendMessage(dst, 11, encodeValue(value))
	}
	for _, value := range graph.Outputs {
		dst = appendMessage(dst, 12, encodeValue(value))
	}
	for _, value := range graph.ValueInfos {
		dst = appendMessage(dst, 13, encodeValue(value))
	}
	for _, sparse := range graph.SparseInitializers {
		dst = appendMessage(dst, 15, encodeSparse(sparse))
	}
	return dst
}

func encodeValue(value *Value) []byte {
	if value == nil {
		return nil
	}
	var dst []byte
	dst = appendOptString(dst, 1, value.Name)
	if value.Type != nil {
		dst = appendMessage(dst, 2, encodeType(value.Type))
	}
	dst = appendOptString(dst, 3, value.DocString)
	return dst
}

func encodeNode(node *Node) []byte {
	if node == nil {
		return nil
	}
	var dst []byte
	dst = appendStrings(dst, 1, node.Inputs)
	dst = appendStrings(dst, 2, node.Outputs)
	dst = appendOptString(dst, 3, node.Name)
	dst = appendOptString(dst, 4, node.OpType)
	for _, attr := range node.Attributes {
		dst = appendMessage(dst, 5, encodeAttribute(attr))
	}
	dst = appendOptString(dst, 6, node.DocString)
	dst = appendOptString(dst, 7, node.Domain)
	dst = appendOptString(dst, 8, node.Overload)
	return dst
}

func encodeAttribute(attr *Attribute) []byte {
	if attr == nil {
		return nil
	}
	var dst []byte
	dst = appendOptString(dst, 1, attr.Name)
	if attr.Type == AttributeTypeFloat || attr.Float != 0 {
		dst = appendFixed32Field(dst, 2, math.Float32bits(attr.Float))
	}
	if attr.Type == AttributeTypeInt || attr.Int != 0 {
		dst = appendVarint(dst, 3, uint64(attr.Int))
	}
	if attr.Type == AttributeTypeString || attr.Text != "" {
		dst = appendString(dst, 4, attr.Text)
	}
	if attr.Tensor != nil {
		dst = appendMessage(dst, 5, encodeTensor(attr.Tensor))
	}
	if attr.Graph != nil {
		dst = appendMessage(dst, 6, encodeGraph(attr.Graph))
	}
	for _, value := range attr.Floats {
		dst = appendFixed32Field(dst, 7, math.Float32bits(value))
	}
	dst = appendInt64s(dst, 8, attr.Ints)
	dst = appendByteList(dst, 9, attr.Strings)
	for _, tensor := range attr.Tensors {
		dst = appendMessage(dst, 10, encodeTensor(tensor))
	}
	for _, graph := range attr.Graphs {
		dst = appendMessage(dst, 11, encodeGraph(graph))
	}
	dst = appendOptString(dst, 13, attr.DocString)
	if attr.TypeProto != nil {
		dst = appendMessage(dst, 14, encodeType(attr.TypeProto))
	}
	for _, typ := range attr.TypeProtos {
		dst = appendMessage(dst, 15, encodeType(typ))
	}
	dst = appendOptVarint(dst, 20, uint64(attr.Type))
	dst = appendOptString(dst, 21, attr.RefAttrName)
	if attr.Sparse != nil {
		dst = appendMessage(dst, 22, encodeSparse(attr.Sparse))
	}
	for _, sparse := range attr.Sparses {
		dst = appendMessage(dst, 23, encodeSparse(sparse))
	}
	return dst
}

func encodeTensor(tensor *Tensor) []byte {
	if tensor == nil {
		return nil
	}
	var dst []byte
	dst = appendInt64s(dst, 1, tensor.Dims)
	dst = appendOptVarint(dst, 2, uint64(tensor.Element))
	if tensor.Segment != nil {
		dst = appendMessage(dst, 3, encodeSegment(tensor.Segment))
	}
	dst = appendPackedFloat32(dst, 4, tensor.FloatData)
	dst = appendPackedInt32(dst, 5, tensor.Int32Data)
	dst = appendByteList(dst, 6, tensor.StringData)
	dst = appendPackedInt64(dst, 7, tensor.Int64Data)
	dst = appendOptString(dst, 8, tensor.Name)
	if len(tensor.Raw) > 0 {
		dst = appendBytes(dst, 9, tensor.Raw)
	}
	dst = appendPackedFloat64(dst, 10, tensor.DoubleData)
	dst = appendPackedUint64(dst, 11, tensor.Uint64Data)
	dst = appendOptString(dst, 12, tensor.DocString)
	for _, pair := range tensor.External {
		dst = appendMessage(dst, 13, encodePair(pair))
	}
	dst = appendOptVarint(dst, 14, uint64(tensor.Location))
	return dst
}

func encodeSegment(segment *Segment) []byte {
	if segment == nil {
		return nil
	}
	var dst []byte
	dst = appendOptVarint(dst, 1, uint64(segment.Begin))
	dst = appendOptVarint(dst, 2, uint64(segment.End))
	return dst
}

func encodePair(pair Pair) []byte {
	var dst []byte
	dst = appendOptString(dst, 1, pair.Key)
	dst = appendOptString(dst, 2, pair.Value)
	return dst
}

func encodeSparse(sparse *Sparse) []byte {
	if sparse == nil {
		return nil
	}
	var dst []byte
	if sparse.Values != nil {
		dst = appendMessage(dst, 1, encodeTensor(sparse.Values))
	}
	if sparse.Indices != nil {
		dst = appendMessage(dst, 2, encodeTensor(sparse.Indices))
	}
	dst = appendInt64s(dst, 3, sparse.Dims)
	return dst
}

func encodeType(typ *Type) []byte {
	if typ == nil {
		return nil
	}
	var dst []byte
	switch typ.Kind {
	case TypeKindTensor:
		dst = appendMessage(dst, 1, encodeTensorType(typ))
	case TypeKindSequence:
		dst = appendMessage(dst, 4, encodeWrappedType(typ.Nested))
	case TypeKindMap:
		dst = appendMessage(dst, 5, encodeMapType(typ))
	case TypeKindSparse:
		dst = appendMessage(dst, 8, encodeTensorType(typ))
	case TypeKindOptional:
		dst = appendMessage(dst, 9, encodeWrappedType(typ.Nested))
	}
	dst = appendOptString(dst, 6, typ.Denotation)
	return dst
}

func encodeTensorType(typ *Type) []byte {
	var dst []byte
	dst = appendOptVarint(dst, 1, uint64(typ.Element))
	if typ.HasShape {
		dst = appendMessage(dst, 2, encodeShape(typ.Shape))
	}
	return dst
}

func encodeWrappedType(typ *Type) []byte {
	if typ == nil {
		return nil
	}
	return appendMessage(nil, 1, encodeType(typ))
}

func encodeMapType(typ *Type) []byte {
	var dst []byte
	dst = appendOptVarint(dst, 1, uint64(typ.Key))
	if typ.Nested != nil {
		dst = appendMessage(dst, 2, encodeType(typ.Nested))
	}
	return dst
}

func encodeShape(dims []Dim) []byte {
	var dst []byte
	for _, dim := range dims {
		dst = appendMessage(dst, 1, encodeDim(dim))
	}
	return dst
}

func encodeDim(dim Dim) []byte {
	var dst []byte
	if dim.Known {
		dst = appendVarint(dst, 1, uint64(dim.Value))
	}
	dst = appendOptString(dst, 2, dim.Param)
	dst = appendOptString(dst, 3, dim.Denotation)
	return dst
}
