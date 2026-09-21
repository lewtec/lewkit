package onnx

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/ndarray"
)

func bool01[T ndarray.Number](c *ndarray.Tensor[int32]) *ndarray.Tensor[T] {
	return c.Cast[T]()
}

func applyWhere[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	if len(node.Inputs) < 3 {
		return nil, ErrOp
	}
	cond, x, err := twoInputs(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	y, ok := values[node.Inputs[2]]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrGraph, node.Inputs[2])
	}
	x, y, err = broadcast(ctx, evaluator, x, y)
	if err != nil {
		return nil, err
	}
	cond, x, err = broadcast(ctx, evaluator, cond, x)
	if err != nil {
		return nil, err
	}
	cond, y, err = broadcast(ctx, evaluator, cond, y)
	if err != nil {
		return nil, err
	}
	return cond.CmpNe(ndarray.Const(T(0))).Where(x, y), nil
}

func applyBitShift[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, values map[string]*ndarray.Tensor[T], node Node) (*ndarray.Tensor[T], error) {
	left, right, err := twoInputs(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	left, right, err = broadcast(ctx, evaluator, left, right)
	if err != nil {
		return nil, err
	}
	switch node.attributeString("direction") {
	case "", "LEFT":
		return left.Shl(right), nil
	case "RIGHT":
		return left.Shr(right), nil
	default:
		return nil, fmt.Errorf("%w: bitshift %s", ErrOp, node.attributeString("direction"))
	}
}
