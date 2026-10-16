package onnx

import (
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

func decodeModel(b []byte) (*File, error) {
	fields, err := split(b)
	if err != nil {
		return nil, err
	}
	file := &File{}
	for _, item := range fields {
		switch item.num {
		case 1:
			file.IRVersion, err = varint64(item)
		case 2:
			file.ProducerName, err = text(item)
		case 3:
			file.ProducerVersion, err = text(item)
		case 4:
			file.Domain, err = text(item)
		case 5:
			file.ModelVersion, err = varint64(item)
		case 6:
			file.DocString, err = text(item)
		case 7:
			file.Graph, err = decodeGraphField(item, 1)
		case 8:
			var op Opset
			op, err = decodeOpset(item)
			if err == nil {
				file.Opsets = append(file.Opsets, op)
			}
		}
		if err != nil {
			return nil, err
		}
	}
	return file, nil
}

func decodeOpset(item field) (Opset, error) {
	raw, err := messageBytes(item)
	if err != nil {
		return Opset{}, err
	}
	fields, err := split(raw)
	if err != nil {
		return Opset{}, err
	}
	var op Opset
	for _, item := range fields {
		switch item.num {
		case 1:
			op.Domain, err = text(item)
		case 2:
			op.Version, err = varint64(item)
		}
		if err != nil {
			return Opset{}, err
		}
	}
	return op, nil
}

func decodeGraphField(item field, depth int) (*Graph, error) {
	raw, err := messageBytes(item)
	if err != nil {
		return nil, err
	}
	return decodeGraph(raw, depth)
}

func decodeGraph(b []byte, depth int) (*Graph, error) {
	if depth > maxDepth {
		return nil, ErrFormat
	}
	fields, err := split(b)
	if err != nil {
		return nil, err
	}
	graph := &Graph{}
	for _, item := range fields {
		switch item.num {
		case 1:
			var node *Node
			node, err = decodeNodeField(item, depth+1)
			if err == nil {
				graph.Nodes = append(graph.Nodes, node)
			}
		case 2:
			graph.Name, err = text(item)
		case 5:
			var tensor *Tensor
			tensor, err = decodeTensorField(item, depth+1)
			if err == nil {
				graph.Initializers = append(graph.Initializers, tensor)
			}
		case 10:
			graph.DocString, err = text(item)
		case 11:
			var value *Value
			value, err = decodeValueField(item, depth+1)
			if err == nil {
				graph.Inputs = append(graph.Inputs, value)
			}
		case 12:
			var value *Value
			value, err = decodeValueField(item, depth+1)
			if err == nil {
				graph.Outputs = append(graph.Outputs, value)
			}
		case 13:
			var value *Value
			value, err = decodeValueField(item, depth+1)
			if err == nil {
				graph.ValueInfos = append(graph.ValueInfos, value)
			}
		case 15:
			var sparse *Sparse
			sparse, err = decodeSparseField(item, depth+1)
			if err == nil {
				graph.SparseInitializers = append(graph.SparseInitializers, sparse)
			}
		}
		if err != nil {
			return nil, err
		}
	}
	return graph, nil
}

func decodeValueField(item field, depth int) (*Value, error) {
	raw, err := messageBytes(item)
	if err != nil {
		return nil, err
	}
	fields, err := split(raw)
	if err != nil {
		return nil, err
	}
	value := &Value{}
	for _, item := range fields {
		switch item.num {
		case 1:
			value.Name, err = text(item)
		case 2:
			value.Type, err = decodeTypeField(item, depth+1)
		case 3:
			value.DocString, err = text(item)
		}
		if err != nil {
			return nil, err
		}
	}
	return value, nil
}

func decodeNodeField(item field, depth int) (*Node, error) {
	raw, err := messageBytes(item)
	if err != nil {
		return nil, err
	}
	fields, err := split(raw)
	if err != nil {
		return nil, err
	}
	node := &Node{}
	for _, item := range fields {
		switch item.num {
		case 1:
			var name string
			name, err = text(item)
			if err == nil {
				node.Inputs = append(node.Inputs, name)
			}
		case 2:
			var name string
			name, err = text(item)
			if err == nil {
				node.Outputs = append(node.Outputs, name)
			}
		case 3:
			node.Name, err = text(item)
		case 4:
			node.OpType, err = text(item)
		case 5:
			var attr *Attribute
			attr, err = decodeAttributeField(item, depth+1)
			if err == nil {
				node.Attributes = append(node.Attributes, attr)
			}
		case 6:
			node.DocString, err = text(item)
		case 7:
			node.Domain, err = text(item)
		case 8:
			node.Overload, err = text(item)
		}
		if err != nil {
			return nil, err
		}
	}
	return node, nil
}

func decodeAttributeField(item field, depth int) (*Attribute, error) {
	if depth > maxDepth {
		return nil, ErrFormat
	}
	raw, err := messageBytes(item)
	if err != nil {
		return nil, err
	}
	fields, err := split(raw)
	if err != nil {
		return nil, err
	}
	attr := &Attribute{}
	var sawType, sawFloat, sawInt, sawText bool
	for _, item := range fields {
		switch item.num {
		case 1:
			attr.Name, err = text(item)
		case 2:
			var bits uint32
			bits, err = fixed32(item)
			attr.Float = math.Float32frombits(bits)
			sawFloat = err == nil
		case 3:
			attr.Int, err = varint64(item)
			sawInt = err == nil
		case 4:
			attr.Text, err = text(item)
			sawText = err == nil
		case 5:
			attr.Tensor, err = decodeTensorField(item, depth+1)
		case 6:
			attr.Graph, err = decodeGraphField(item, depth+1)
		case 7:
			attr.Floats, err = takeFloat32s(attr.Floats, item)
		case 8:
			attr.Ints, err = takeInt64s(attr.Ints, item)
		case 9:
			if item.typ != protowire.BytesType {
				err = ErrFormat
				break
			}
			attr.Strings = append(attr.Strings, cloneBytes(item.b))
		case 10:
			var tensor *Tensor
			tensor, err = decodeTensorField(item, depth+1)
			if err == nil {
				attr.Tensors = append(attr.Tensors, tensor)
			}
		case 11:
			var graph *Graph
			graph, err = decodeGraphField(item, depth+1)
			if err == nil {
				attr.Graphs = append(attr.Graphs, graph)
			}
		case 13:
			attr.DocString, err = text(item)
		case 14:
			attr.TypeProto, err = decodeTypeField(item, depth+1)
		case 15:
			var typ *Type
			typ, err = decodeTypeField(item, depth+1)
			if err == nil {
				attr.TypeProtos = append(attr.TypeProtos, typ)
			}
		case 20:
			var kind int64
			kind, err = varint64(item)
			attr.Type = AttributeType(kind)
			sawType = err == nil
		case 21:
			attr.RefAttrName, err = text(item)
		case 22:
			attr.Sparse, err = decodeSparseField(item, depth+1)
		case 23:
			var sparse *Sparse
			sparse, err = decodeSparseField(item, depth+1)
			if err == nil {
				attr.Sparses = append(attr.Sparses, sparse)
			}
		}
		if err != nil {
			return nil, err
		}
	}
	if !sawType {
		attr.Type = inferAttribute(attr, sawFloat, sawInt, sawText)
	}
	return attr, nil
}

func inferAttribute(attr *Attribute, sawFloat, sawInt, sawText bool) AttributeType {
	switch {
	case sawFloat:
		return AttributeTypeFloat
	case sawInt:
		return AttributeTypeInt
	case sawText:
		return AttributeTypeString
	case attr.Tensor != nil:
		return AttributeTypeTensor
	case attr.Graph != nil:
		return AttributeTypeGraph
	case attr.Sparse != nil:
		return AttributeTypeSparseTensor
	case attr.TypeProto != nil:
		return AttributeTypeType
	case len(attr.Floats) > 0:
		return AttributeTypeFloats
	case len(attr.Ints) > 0:
		return AttributeTypeInts
	case len(attr.Strings) > 0:
		return AttributeTypeStrings
	case len(attr.Tensors) > 0:
		return AttributeTypeTensors
	case len(attr.Graphs) > 0:
		return AttributeTypeGraphs
	case len(attr.Sparses) > 0:
		return AttributeTypeSparseTensors
	case len(attr.TypeProtos) > 0:
		return AttributeTypeTypes
	default:
		return 0
	}
}

func decodeTensorField(item field, depth int) (*Tensor, error) {
	raw, err := messageBytes(item)
	if err != nil {
		return nil, err
	}
	return decodeTensor(raw, depth)
}

func decodeTensor(b []byte, depth int) (*Tensor, error) {
	if depth > maxDepth {
		return nil, ErrFormat
	}
	fields, err := split(b)
	if err != nil {
		return nil, err
	}
	tensor := &Tensor{}
	for _, item := range fields {
		switch item.num {
		case 1:
			tensor.Dims, err = takeInt64s(tensor.Dims, item)
		case 2:
			var kind int64
			kind, err = varint64(item)
			tensor.Element = DataType(kind)
		case 3:
			tensor.Segment, err = decodeSegment(item)
		case 4:
			tensor.FloatData, err = takeFloat32s(tensor.FloatData, item)
		case 5:
			tensor.Int32Data, err = takeInt32s(tensor.Int32Data, item)
		case 6:
			if item.typ != protowire.BytesType {
				err = ErrFormat
				break
			}
			tensor.StringData = append(tensor.StringData, cloneBytes(item.b))
		case 7:
			tensor.Int64Data, err = takeInt64s(tensor.Int64Data, item)
		case 8:
			tensor.Name, err = text(item)
		case 9:
			if item.typ != protowire.BytesType {
				err = ErrFormat
				break
			}
			tensor.Raw = cloneBytes(item.b)
		case 10:
			tensor.DoubleData, err = takeFloat64s(tensor.DoubleData, item)
		case 11:
			tensor.Uint64Data, err = takeUint64s(tensor.Uint64Data, item)
		case 12:
			tensor.DocString, err = text(item)
		case 13:
			var pair Pair
			pair, err = decodePair(item)
			if err == nil {
				tensor.External = append(tensor.External, pair)
			}
		case 14:
			var loc int64
			loc, err = varint64(item)
			tensor.Location = DataLocation(loc)
		}
		if err != nil {
			return nil, err
		}
	}
	return tensor, nil
}

func decodeSegment(item field) (*Segment, error) {
	raw, err := messageBytes(item)
	if err != nil {
		return nil, err
	}
	fields, err := split(raw)
	if err != nil {
		return nil, err
	}
	segment := &Segment{}
	for _, item := range fields {
		switch item.num {
		case 1:
			segment.Begin, err = varint64(item)
		case 2:
			segment.End, err = varint64(item)
		}
		if err != nil {
			return nil, err
		}
	}
	return segment, nil
}

func decodePair(item field) (Pair, error) {
	raw, err := messageBytes(item)
	if err != nil {
		return Pair{}, err
	}
	fields, err := split(raw)
	if err != nil {
		return Pair{}, err
	}
	var pair Pair
	for _, item := range fields {
		switch item.num {
		case 1:
			pair.Key, err = text(item)
		case 2:
			pair.Value, err = text(item)
		}
		if err != nil {
			return Pair{}, err
		}
	}
	return pair, nil
}

func decodeSparseField(item field, depth int) (*Sparse, error) {
	if depth > maxDepth {
		return nil, ErrFormat
	}
	raw, err := messageBytes(item)
	if err != nil {
		return nil, err
	}
	fields, err := split(raw)
	if err != nil {
		return nil, err
	}
	sparse := &Sparse{}
	for _, item := range fields {
		switch item.num {
		case 1:
			sparse.Values, err = decodeTensorField(item, depth+1)
		case 2:
			sparse.Indices, err = decodeTensorField(item, depth+1)
		case 3:
			sparse.Dims, err = takeInt64s(sparse.Dims, item)
		}
		if err != nil {
			return nil, err
		}
	}
	return sparse, nil
}

func decodeTypeField(item field, depth int) (*Type, error) {
	raw, err := messageBytes(item)
	if err != nil {
		return nil, err
	}
	return decodeType(raw, depth)
}

func decodeType(b []byte, depth int) (*Type, error) {
	if depth > maxDepth {
		return nil, ErrFormat
	}
	fields, err := split(b)
	if err != nil {
		return nil, err
	}
	typ := &Type{}
	for _, item := range fields {
		switch item.num {
		case 1:
			err = decodeTensorKind(typ, item, depth+1)
			typ.Kind = TypeKindTensor
		case 4:
			typ.Nested, err = decodeWrappedType(item, depth+1)
			typ.Kind = TypeKindSequence
		case 5:
			err = decodeMapType(typ, item, depth+1)
			typ.Kind = TypeKindMap
		case 6:
			typ.Denotation, err = text(item)
		case 8:
			err = decodeTensorKind(typ, item, depth+1)
			typ.Kind = TypeKindSparse
		case 9:
			typ.Nested, err = decodeWrappedType(item, depth+1)
			typ.Kind = TypeKindOptional
		}
		if err != nil {
			return nil, err
		}
	}
	return typ, nil
}

func decodeTensorKind(typ *Type, item field, depth int) error {
	raw, err := messageBytes(item)
	if err != nil {
		return err
	}
	fields, err := split(raw)
	if err != nil {
		return err
	}
	for _, item := range fields {
		switch item.num {
		case 1:
			var kind int64
			kind, err = varint64(item)
			typ.Element = DataType(kind)
		case 2:
			typ.Shape, err = decodeShape(item)
			typ.HasShape = err == nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func decodeShape(item field) ([]Dim, error) {
	raw, err := messageBytes(item)
	if err != nil {
		return nil, err
	}
	fields, err := split(raw)
	if err != nil {
		return nil, err
	}
	var dims []Dim
	for _, item := range fields {
		if item.num != 1 {
			continue
		}
		dim, err := decodeDim(item)
		if err != nil {
			return nil, err
		}
		dims = append(dims, dim)
	}
	return dims, nil
}

func decodeDim(item field) (Dim, error) {
	raw, err := messageBytes(item)
	if err != nil {
		return Dim{}, err
	}
	fields, err := split(raw)
	if err != nil {
		return Dim{}, err
	}
	var dim Dim
	for _, item := range fields {
		switch item.num {
		case 1:
			dim.Value, err = varint64(item)
			dim.Known = err == nil
		case 2:
			dim.Param, err = text(item)
		case 3:
			dim.Denotation, err = text(item)
		}
		if err != nil {
			return Dim{}, err
		}
	}
	return dim, nil
}

func decodeMapType(typ *Type, item field, depth int) error {
	raw, err := messageBytes(item)
	if err != nil {
		return err
	}
	fields, err := split(raw)
	if err != nil {
		return err
	}
	for _, item := range fields {
		switch item.num {
		case 1:
			var key int64
			key, err = varint64(item)
			typ.Key = DataType(key)
		case 2:
			typ.Nested, err = decodeTypeField(item, depth+1)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func decodeWrappedType(item field, depth int) (*Type, error) {
	raw, err := messageBytes(item)
	if err != nil {
		return nil, err
	}
	fields, err := split(raw)
	if err != nil {
		return nil, err
	}
	var nested *Type
	for _, item := range fields {
		if item.num != 1 {
			continue
		}
		nested, err = decodeTypeField(item, depth+1)
		if err != nil {
			return nil, err
		}
	}
	return nested, nil
}
