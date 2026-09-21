package onnx

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ndarray/onnx/internal/proto"
	goproto "google.golang.org/protobuf/proto"
)

// LoadTensor unmarshals a TensorProto into an ndarray of T.
func LoadTensor[T ndarray.Number](b []byte) (*ndarray.Tensor[T], error) {
	p, err := unmarshalTensorProto(b)
	if err != nil {
		return nil, err
	}
	return tensorFromProto[T](p)
}

func unmarshalTensorProto(b []byte) (*proto.TensorProto, error) {
	if len(b) == 0 {
		return nil, ErrEmpty
	}
	msg := new(proto.TensorProto)
	if err := goproto.Unmarshal(b, msg); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrModel, err)
	}
	return msg, nil
}

func tensorFromProto[T ndarray.Number](p *proto.TensorProto) (*ndarray.Tensor[T], error) {
	if p == nil {
		return nil, ErrEmpty
	}
	data, err := valuesFromProto[T](p)
	if err != nil {
		return nil, err
	}
	return ndarray.New(data, shapeFromDimensions(p.GetDims()))
}

func int64sFromNumeric(p *proto.TensorProto) ([]int64, error) {
	if p == nil {
		return nil, ErrEmpty
	}
	dt := proto.TensorProto_DataType(p.GetDataType())
	need := protoNeed(p.GetDims())
	switch dt {
	case proto.TensorProto_INT64:
		return int64sFromProto(p)
	case proto.TensorProto_INT32:
		var data []int32
		if i := p.GetInt32Data(); len(i) > 0 {
			data = i
		} else if raw := p.GetRawData(); len(raw) == need*4 {
			data = int32FromRaw(raw)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("%w: not int32", ErrOp)
		}
		out := make([]int64, len(data))
		for i, v := range data {
			out[i] = int64(v)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%w: not int", ErrOp)
	}
}

func int64sFromProto(p *proto.TensorProto) ([]int64, error) {
	if p == nil {
		return nil, ErrEmpty
	}
	if proto.TensorProto_DataType(p.GetDataType()) != proto.TensorProto_INT64 {
		return nil, fmt.Errorf("%w: not int64", ErrOp)
	}
	if i := p.GetInt64Data(); len(i) > 0 {
		return append([]int64(nil), i...), nil
	}
	need := protoNeed(p.GetDims())
	if raw := p.GetRawData(); len(raw) == need*8 {
		return int64FromRaw(raw), nil
	}
	return nil, fmt.Errorf("%w: not int64", ErrOp)
}

func valuesFromProto[T ndarray.Number](p *proto.TensorProto) ([]T, error) {
	need := protoNeed(p.GetDims())
	dt := proto.TensorProto_DataType(p.GetDataType())
	if dt == proto.TensorProto_INT64 {
		ints, err := int64sFromProto(p)
		if err != nil {
			return nil, err
		}
		out := make([]T, len(ints))
		for i, v := range ints {
			out[i] = T(v)
		}
		return out, nil
	}
	if dt == proto.TensorProto_BOOL {
		bits, err := boolBits(p, need)
		if err != nil {
			return nil, err
		}
		out := make([]T, len(bits))
		for i, b := range bits {
			if b != 0 {
				out[i] = T(1)
			}
		}
		return out, nil
	}
	var z T
	switch any(z).(type) {
	case float32:
		data, err := float32Values(p, dt, need)
		if err != nil {
			return nil, err
		}
		return any(data).([]T), nil
	case int32:
		if dt != proto.TensorProto_INT32 {
			return nil, fmt.Errorf("%w: not int32", ErrOp)
		}
		var data []int32
		if i := p.GetInt32Data(); len(i) > 0 {
			data = i
		} else if raw := p.GetRawData(); len(raw) == need*4 {
			data = int32FromRaw(raw)
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("%w: not int32", ErrOp)
		}
		return any(data).([]T), nil
	case uint8:
		if dt != proto.TensorProto_UINT8 {
			return nil, fmt.Errorf("%w: not uint8", ErrOp)
		}
		var data []uint8
		if raw := p.GetRawData(); len(raw) == need {
			data = raw
		} else if i := p.GetInt32Data(); len(i) == need {
			data = make([]uint8, need)
			for j, v := range i {
				data[j] = uint8(v)
			}
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("%w: not uint8", ErrOp)
		}
		return any(data).([]T), nil
	default:
		return nil, ErrOp
	}
}

func boolBits(p *proto.TensorProto, need int) ([]byte, error) {
	if raw := p.GetRawData(); len(raw) == need {
		return raw, nil
	}
	if i := p.GetInt32Data(); len(i) == need {
		out := make([]byte, need)
		for j, v := range i {
			if v != 0 {
				out[j] = 1
			}
		}
		return out, nil
	}
	return nil, fmt.Errorf("%w: not bool", ErrOp)
}

func float32Values(p *proto.TensorProto, dt proto.TensorProto_DataType, need int) ([]float32, error) {
	switch dt {
	case proto.TensorProto_FLOAT:
		if f := p.GetFloatData(); len(f) > 0 {
			return f, nil
		}
		if raw := p.GetRawData(); len(raw) == need*4 {
			return float32FromRaw(raw), nil
		}
	case proto.TensorProto_DOUBLE:
		if d := p.GetDoubleData(); len(d) > 0 {
			out := make([]float32, len(d))
			for i, v := range d {
				out[i] = float32(v)
			}
			return out, nil
		}
		if raw := p.GetRawData(); len(raw) == need*8 {
			return float32FromFloat64Raw(raw), nil
		}
	case proto.TensorProto_INT32:
		if i := p.GetInt32Data(); len(i) > 0 {
			out := make([]float32, len(i))
			for j, v := range i {
				out[j] = float32(v)
			}
			return out, nil
		}
		if raw := p.GetRawData(); len(raw) == need*4 {
			ints := int32FromRaw(raw)
			out := make([]float32, len(ints))
			for j, v := range ints {
				out[j] = float32(v)
			}
			return out, nil
		}
	}
	return nil, fmt.Errorf("%w: not float32", ErrOp)
}

func float32FromFloat64Raw(b []byte) []float32 {
	out := make([]float32, len(b)/8)
	for i := range out {
		out[i] = float32(math.Float64frombits(binary.LittleEndian.Uint64(b[i*8:])))
	}
	return out
}

func protoNeed(dims []int64) int {
	need := 1
	for _, d := range dims {
		need *= int(d)
	}
	return need
}

func float32FromRaw(b []byte) []float32 {
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func int32FromRaw(b []byte) []int32 {
	out := make([]int32, len(b)/4)
	for i := range out {
		out[i] = int32(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func int64FromRaw(b []byte) []int64 {
	out := make([]int64, len(b)/8)
	for i := range out {
		out[i] = int64(binary.LittleEndian.Uint64(b[i*8:]))
	}
	return out
}
