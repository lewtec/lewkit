package onnx

// DataType is the element type of a tensor.
// The numeric values are the ONNX TensorProto.DataType numbers.
//
//lewkit:enum
type DataType int32

const (
	DataTypeUndefined      DataType = 0
	DataTypeFloat          DataType = 1
	DataTypeUint8          DataType = 2
	DataTypeInt8           DataType = 3
	DataTypeUint16         DataType = 4
	DataTypeInt16          DataType = 5
	DataTypeInt32          DataType = 6
	DataTypeInt64          DataType = 7
	DataTypeString         DataType = 8
	DataTypeBool           DataType = 9
	DataTypeFloat16        DataType = 10
	DataTypeDouble         DataType = 11
	DataTypeUint32         DataType = 12
	DataTypeUint64         DataType = 13
	DataTypeComplex64      DataType = 14
	DataTypeComplex128     DataType = 15
	DataTypeBfloat16       DataType = 16
	DataTypeFloat8E4M3FN   DataType = 17
	DataTypeFloat8E4M3FNUZ DataType = 18
	DataTypeFloat8E5M2     DataType = 19
	DataTypeFloat8E5M2FNUZ DataType = 20
	DataTypeUint4          DataType = 21
	DataTypeInt4           DataType = 22
	DataTypeFloat4E2M1     DataType = 23
	DataTypeFloat8E8M0     DataType = 24
	DataTypeUint2          DataType = 25
	DataTypeInt2           DataType = 26
)

// AttributeType selects which value of an attribute is in use.
// The numeric values are the ONNX AttributeProto.AttributeType numbers.
//
//lewkit:enum
type AttributeType int32

const (
	AttributeTypeUndefined     AttributeType = 0
	AttributeTypeFloat         AttributeType = 1
	AttributeTypeInt           AttributeType = 2
	AttributeTypeString        AttributeType = 3
	AttributeTypeTensor        AttributeType = 4
	AttributeTypeGraph         AttributeType = 5
	AttributeTypeFloats        AttributeType = 6
	AttributeTypeInts          AttributeType = 7
	AttributeTypeStrings       AttributeType = 8
	AttributeTypeTensors       AttributeType = 9
	AttributeTypeGraphs        AttributeType = 10
	AttributeTypeSparseTensor  AttributeType = 11
	AttributeTypeSparseTensors AttributeType = 12
	AttributeTypeType          AttributeType = 13
	AttributeTypeTypes         AttributeType = 14
)

// DataLocation says where a tensor's elements live.
// Zero is the protobuf default: inside the model.
//
//lewkit:enum
type DataLocation int32

const (
	DataLocationDefault  DataLocation = 0
	DataLocationExternal DataLocation = 1
)

// TypeKind selects the TypeProto alternative.
// Zero means the type was absent.
//
//lewkit:enum invalid=unset
type TypeKind uint8

const (
	TypeKindTensor TypeKind = iota + 1
	TypeKindSequence
	TypeKindMap
	TypeKindSparse
	TypeKindOptional
)

// Dim is one tensor dimension.
// Known reports that Value is a present integer, including zero.
// Param is a symbolic name. Denotation is the optional standard label.
type Dim struct {
	Known      bool
	Value      int64
	Param      string
	Denotation string
}

// FixedDim is a dimension with an integer size.
func FixedDim(n int64) Dim {
	return Dim{Known: true, Value: n}
}

// ParamDim is a symbolic dimension.
func ParamDim(name string) Dim {
	return Dim{Param: name}
}
