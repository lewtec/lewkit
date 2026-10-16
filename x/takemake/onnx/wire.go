package onnx

import (
	"fmt"
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

const maxDepth = 32

type field struct {
	num protowire.Number
	typ protowire.Type
	u   uint64
	b   []byte
}

func split(b []byte) ([]field, error) {
	var out []field
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return nil, wireErr(n)
		}
		b = b[n:]
		item := field{num: num, typ: typ}
		switch typ {
		case protowire.VarintType:
			item.u, n = protowire.ConsumeVarint(b)
		case protowire.Fixed32Type:
			var v uint32
			v, n = protowire.ConsumeFixed32(b)
			item.u = uint64(v)
		case protowire.Fixed64Type:
			item.u, n = protowire.ConsumeFixed64(b)
		case protowire.BytesType:
			item.b, n = protowire.ConsumeBytes(b)
		default:
			return nil, ErrFormat
		}
		if n < 0 {
			return nil, wireErr(n)
		}
		b = b[n:]
		out = append(out, item)
	}
	return out, nil
}

func wireErr(n int) error {
	return fmt.Errorf("%w: %w", ErrFormat, protowire.ParseError(n))
}

func varint64(f field) (int64, error) {
	if f.typ != protowire.VarintType {
		return 0, ErrFormat
	}
	return int64(f.u), nil
}

func text(f field) (string, error) {
	if f.typ != protowire.BytesType {
		return "", ErrFormat
	}
	return string(f.b), nil
}

func fixed32(f field) (uint32, error) {
	if f.typ != protowire.Fixed32Type {
		return 0, ErrFormat
	}
	return uint32(f.u), nil
}

func messageBytes(f field) ([]byte, error) {
	if f.typ != protowire.BytesType {
		return nil, ErrFormat
	}
	return f.b, nil
}

func takeInt64s(dst []int64, f field) ([]int64, error) {
	switch f.typ {
	case protowire.VarintType:
		return append(dst, int64(f.u)), nil
	case protowire.BytesType:
		vals, err := packedVarints(f.b)
		if err != nil {
			return nil, err
		}
		for _, v := range vals {
			dst = append(dst, int64(v))
		}
		return dst, nil
	default:
		return nil, ErrFormat
	}
}

func takeInt32s(dst []int32, f field) ([]int32, error) {
	switch f.typ {
	case protowire.VarintType:
		return append(dst, int32(f.u)), nil
	case protowire.BytesType:
		vals, err := packedVarints(f.b)
		if err != nil {
			return nil, err
		}
		for _, v := range vals {
			dst = append(dst, int32(v))
		}
		return dst, nil
	default:
		return nil, ErrFormat
	}
}

func takeUint64s(dst []uint64, f field) ([]uint64, error) {
	switch f.typ {
	case protowire.VarintType:
		return append(dst, f.u), nil
	case protowire.BytesType:
		vals, err := packedVarints(f.b)
		if err != nil {
			return nil, err
		}
		return append(dst, vals...), nil
	default:
		return nil, ErrFormat
	}
}

func takeFloat32s(dst []float32, f field) ([]float32, error) {
	switch f.typ {
	case protowire.Fixed32Type:
		return append(dst, math.Float32frombits(uint32(f.u))), nil
	case protowire.BytesType:
		vals, err := packedFixed32(f.b)
		if err != nil {
			return nil, err
		}
		for _, v := range vals {
			dst = append(dst, math.Float32frombits(v))
		}
		return dst, nil
	default:
		return nil, ErrFormat
	}
}

func takeFloat64s(dst []float64, f field) ([]float64, error) {
	switch f.typ {
	case protowire.Fixed64Type:
		return append(dst, math.Float64frombits(f.u)), nil
	case protowire.BytesType:
		vals, err := packedFixed64(f.b)
		if err != nil {
			return nil, err
		}
		for _, v := range vals {
			dst = append(dst, math.Float64frombits(v))
		}
		return dst, nil
	default:
		return nil, ErrFormat
	}
}

func packedVarints(b []byte) ([]uint64, error) {
	var out []uint64
	for len(b) > 0 {
		v, n := protowire.ConsumeVarint(b)
		if n < 0 {
			return nil, wireErr(n)
		}
		b = b[n:]
		out = append(out, v)
	}
	return out, nil
}

func packedFixed32(b []byte) ([]uint32, error) {
	if len(b)%4 != 0 {
		return nil, ErrFormat
	}
	out := make([]uint32, 0, len(b)/4)
	for len(b) > 0 {
		v, n := protowire.ConsumeFixed32(b)
		if n < 0 {
			return nil, wireErr(n)
		}
		b = b[n:]
		out = append(out, v)
	}
	return out, nil
}

func packedFixed64(b []byte) ([]uint64, error) {
	if len(b)%8 != 0 {
		return nil, ErrFormat
	}
	out := make([]uint64, 0, len(b)/8)
	for len(b) > 0 {
		v, n := protowire.ConsumeFixed64(b)
		if n < 0 {
			return nil, wireErr(n)
		}
		b = b[n:]
		out = append(out, v)
	}
	return out, nil
}

func appendVarint(dst []byte, num protowire.Number, v uint64) []byte {
	dst = protowire.AppendTag(dst, num, protowire.VarintType)
	return protowire.AppendVarint(dst, v)
}

func appendOptVarint(dst []byte, num protowire.Number, v uint64) []byte {
	if v == 0 {
		return dst
	}
	return appendVarint(dst, num, v)
}

func appendFixed32Field(dst []byte, num protowire.Number, v uint32) []byte {
	dst = protowire.AppendTag(dst, num, protowire.Fixed32Type)
	return protowire.AppendFixed32(dst, v)
}

func appendFixed64Field(dst []byte, num protowire.Number, v uint64) []byte {
	dst = protowire.AppendTag(dst, num, protowire.Fixed64Type)
	return protowire.AppendFixed64(dst, v)
}

func appendBytes(dst []byte, num protowire.Number, p []byte) []byte {
	dst = protowire.AppendTag(dst, num, protowire.BytesType)
	return protowire.AppendBytes(dst, p)
}

func appendString(dst []byte, num protowire.Number, s string) []byte {
	dst = protowire.AppendTag(dst, num, protowire.BytesType)
	return protowire.AppendString(dst, s)
}

func appendOptString(dst []byte, num protowire.Number, s string) []byte {
	if s == "" {
		return dst
	}
	return appendString(dst, num, s)
}

func appendStrings(dst []byte, num protowire.Number, values []string) []byte {
	for _, value := range values {
		dst = appendString(dst, num, value)
	}
	return dst
}

func appendMessage(dst []byte, num protowire.Number, msg []byte) []byte {
	return appendBytes(dst, num, msg)
}

func appendInt64s(dst []byte, num protowire.Number, values []int64) []byte {
	for _, value := range values {
		dst = appendVarint(dst, num, uint64(value))
	}
	return dst
}

func appendPackedFloat32(dst []byte, num protowire.Number, values []float32) []byte {
	if len(values) == 0 {
		return dst
	}
	inner := make([]byte, 0, 4*len(values))
	for _, value := range values {
		inner = protowire.AppendFixed32(inner, math.Float32bits(value))
	}
	return appendBytes(dst, num, inner)
}

func appendPackedInt32(dst []byte, num protowire.Number, values []int32) []byte {
	if len(values) == 0 {
		return dst
	}
	var inner []byte
	for _, value := range values {
		inner = protowire.AppendVarint(inner, uint64(int64(value)))
	}
	return appendBytes(dst, num, inner)
}

func appendPackedInt64(dst []byte, num protowire.Number, values []int64) []byte {
	if len(values) == 0 {
		return dst
	}
	var inner []byte
	for _, value := range values {
		inner = protowire.AppendVarint(inner, uint64(value))
	}
	return appendBytes(dst, num, inner)
}

func appendPackedUint64(dst []byte, num protowire.Number, values []uint64) []byte {
	if len(values) == 0 {
		return dst
	}
	var inner []byte
	for _, value := range values {
		inner = protowire.AppendVarint(inner, value)
	}
	return appendBytes(dst, num, inner)
}

func appendPackedFloat64(dst []byte, num protowire.Number, values []float64) []byte {
	if len(values) == 0 {
		return dst
	}
	inner := make([]byte, 0, 8*len(values))
	for _, value := range values {
		inner = protowire.AppendFixed64(inner, math.Float64bits(value))
	}
	return appendBytes(dst, num, inner)
}

func appendByteList(dst []byte, num protowire.Number, values [][]byte) []byte {
	for _, value := range values {
		dst = appendBytes(dst, num, value)
	}
	return dst
}
