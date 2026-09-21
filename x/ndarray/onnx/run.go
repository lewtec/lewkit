package onnx

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ndarray/nn"
	"github.com/lewtec/lewkit/x/ndarray/onnx/internal/proto"
)

// Function lowers an ONNX graph onto ndarray.
type Function[T ndarray.Number] struct {
	inputs        []string
	outputs       []string
	nodes         []Node
	weights       map[string]*ndarray.Tensor[T]
	integerShapes map[string][]int64
}

// FunctionOf builds an ndarray graph from the model for element type T.
func FunctionOf[T ndarray.Number](m *Model) (*Function[T], error) {
	if m == nil {
		return nil, ErrGraph
	}
	initializers := make(map[string]*proto.TensorProto, len(m.Graph.Initializers))
	for _, tensor := range m.Graph.Initializers {
		initializers[tensor.GetName()] = tensor
	}
	var inputs []string
	for _, value := range m.Graph.Inputs {
		if _, ok := initializers[value.Name]; !ok {
			inputs = append(inputs, value.Name)
		}
	}
	var outputs []string
	for _, value := range m.Graph.Outputs {
		outputs = append(outputs, value.Name)
	}
	function := &Function[T]{
		inputs:        inputs,
		outputs:       outputs,
		nodes:         m.Graph.Nodes,
		weights:       make(map[string]*ndarray.Tensor[T]),
		integerShapes: make(map[string][]int64),
	}
	for _, p := range m.Graph.Initializers {
		name := p.GetName()
		if proto.TensorProto_DataType(p.GetDataType()) == proto.TensorProto_INT64 {
			ints, err := int64sFromProto(p)
			if err != nil {
				return nil, err
			}
			function.integerShapes[name] = ints
			continue
		}
		weight, err := tensorFromProto[T](p)
		if err != nil {
			continue
		}
		function.weights[name] = weight
	}
	return function, nil
}

func (function *Function[T]) operators() []string {
	seen := map[string]struct{}{}
	var names []string
	for _, node := range function.nodes {
		if _, ok := seen[node.OperatorType]; ok {
			continue
		}
		seen[node.OperatorType] = struct{}{}
		names = append(names, node.OperatorType)
	}
	return names
}

func (function *Function[T]) supported() error {
	for _, node := range function.nodes {
		if !implementedOperator(node.OperatorType) {
			return fmt.Errorf("%w: %s", ErrOp, node.OperatorType)
		}
		if len(node.Outputs) != 1 && node.OperatorType != "LayerNormalization" && node.OperatorType != "Split" {
			return fmt.Errorf("%w: %s outputs", ErrOp, node.OperatorType)
		}
		if len(node.Outputs) == 0 {
			return fmt.Errorf("%w: %s outputs", ErrOp, node.OperatorType)
		}
	}
	return nil
}

// Apply runs a one-input one-output graph.
func (function *Function[T]) Apply(ctx context.Context, evaluator ndarray.Evaluator, input *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	if function == nil || input == nil {
		return nil, ErrOp
	}
	if len(function.inputs) != 1 || len(function.outputs) != 1 {
		return nil, ErrArity
	}
	outputs, err := function.ApplyInputs(ctx, evaluator, map[string]*ndarray.Tensor[T]{
		function.inputs[0]: input,
	})
	if err != nil {
		return nil, err
	}
	return outputs[function.outputs[0]], nil
}

// ApplyInputs runs the graph with named tensors.
func (function *Function[T]) ApplyInputs(ctx context.Context, evaluator ndarray.Evaluator, inputs map[string]*ndarray.Tensor[T]) (map[string]*ndarray.Tensor[T], error) {
	if function == nil {
		return nil, ErrOp
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	integerShapes := maps.Clone(function.integerShapes)
	if integerShapes == nil {
		integerShapes = map[string][]int64{}
	}
	values := make(map[string]*ndarray.Tensor[T], len(function.nodes)+len(function.inputs)+len(function.weights))
	for name, weight := range function.weights {
		values[name] = weight
	}
	for name, tensor := range inputs {
		if tensor == nil {
			return nil, ErrOp
		}
		values[name] = tensor
	}
	for _, name := range function.inputs {
		if _, ok := values[name]; ok {
			continue
		}
		if _, ok := integerShapes[name]; ok {
			continue
		}
		return nil, fmt.Errorf("%w: missing %s", ErrGraph, name)
	}
	for _, node := range function.nodes {
		result, err := function.applyNode(ctx, evaluator, node, values, integerShapes)
		if err != nil {
			return nil, fmt.Errorf("%w: %s %s: %w", ErrOp, node.OperatorType, node.Name, err)
		}
		values[node.Outputs[0]] = result
	}
	outputs := make(map[string]*ndarray.Tensor[T], len(function.outputs))
	for _, name := range function.outputs {
		tensor, ok := values[name]
		if !ok {
			return nil, fmt.Errorf("%w: missing %s", ErrGraph, name)
		}
		outputs[name] = tensor
	}
	return outputs, nil
}

func (function *Function[T]) applyNode(ctx context.Context, evaluator ndarray.Evaluator, node Node, values map[string]*ndarray.Tensor[T], integerShapes map[string][]int64) (*ndarray.Tensor[T], error) {
	switch node.OperatorType {
	case "Abs":
		return unary(values, node, absT[T])
	case "Acos":
		return unary(values, node, acos[T])
	case "Acosh":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return log(x.Add(x.Mul(x).Add(ndarray.Const(float32(-1)).Cast[T]()).Sqrt()))
		})
	case "Add":
		return binaryBroadcast(ctx, evaluator, values, node, (*ndarray.Tensor[T]).Add)
	case "And":
		return binaryBroadcast(ctx, evaluator, values, node, (*ndarray.Tensor[T]).And)
	case "ArgMax":
		return applyArgMinMax(ctx, evaluator, values, node, false)
	case "ArgMin":
		return applyArgMinMax(ctx, evaluator, values, node, true)
	case "Asin":
		return unary(values, node, asin[T])
	case "Asinh":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return log(x.Add(x.Mul(x).Add(ndarray.Const(T(1))).Sqrt()))
		})
	case "Atan":
		return unary(values, node, atan[T])
	case "Atanh":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			one := ndarray.Const(T(1))
			return log(one.Add(x).Mul(one.Add(x.Neg()).Reciprocal())).Mul(ndarray.Const(float32(0.5)).Cast[T]())
		})
	case "AveragePool":
		return applyAveragePool(ctx, evaluator, values, node)
	case "BatchNormalization":
		return applyBatchNorm(ctx, evaluator, values, node)
	case "BitShift":
		return applyBitShift(ctx, evaluator, values, node)
	case "BlackmanWindow":
		return applyWindow(values, node, integerShapes, "blackman")
	case "BitwiseAnd":
		return binaryBroadcast(ctx, evaluator, values, node, (*ndarray.Tensor[T]).And)
	case "BitwiseNot":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return x.Xor(ndarray.Const(allBits[T]()))
		})
	case "BitwiseOr":
		return binaryBroadcast(ctx, evaluator, values, node, (*ndarray.Tensor[T]).Or)
	case "BitwiseXor":
		return binaryBroadcast(ctx, evaluator, values, node, (*ndarray.Tensor[T]).Xor)
	case "Cast":
		return applyCast(values, node)
	case "CastLike":
		return applyCastLike(values, node)
	case "Ceil":
		return unary(values, node, ceil[T])
	case "Celu":
		alpha := node.attributeFloat("alpha", 1)
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			pos := x.Max(ndarray.Const(T(0)))
			curved := exp(x.Mul(ndarray.Const(1 / alpha).Cast[T]())).Add(ndarray.Const(float32(-1)).Cast[T]()).Mul(ndarray.Const(T(alpha)))
			return pos.Add(minimum(ndarray.Const(T(0)), curved))
		})
	case "Compress":
		return applyCompress(ctx, evaluator, values, node)
	case "CenterCropPad":
		return applyCenterCropPad(ctx, evaluator, values, node, integerShapes)
	case "Clip":
		return applyClip(ctx, evaluator, values, node)
	case "Concat":
		return applyConcat(ctx, evaluator, values, node)
	case "Constant":
		return applyConstant[T](node)
	case "ConstantOfShape":
		return applyConstantOfShape[T](node, integerShapes)
	case "Conv":
		return function.convolution(ctx, evaluator, node, values)
	case "Cos":
		return unary(values, node, cos[T])
	case "Cosh":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return exp(x).Add(exp(x.Neg())).Mul(ndarray.Const(float32(0.5)).Cast[T]())
		})
	case "CumProd":
		return applyCumProd(ctx, evaluator, values, node, integerShapes)
	case "CumSum":
		return applyCumSum(ctx, evaluator, values, node, integerShapes)
	case "DepthToSpace":
		return applyDepthToSpace(ctx, evaluator, values, node)
	case "Div":
		return binaryBroadcast(ctx, evaluator, values, node, divide[T])
	case "Dropout":
		if len(node.Inputs) != 1 {
			return nil, fmt.Errorf("%w: dropout", ErrOp)
		}
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] { return x })
	case "Equal":
		return binaryBroadcast(ctx, evaluator, values, node, func(a, b *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return bool01[T](a.Equal(b))
		})
	case "Erf":
		return unary(values, node, erf[T])
	case "Elu":
		alpha := node.attributeFloat("alpha", 1)
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return x.GreaterEqual(ndarray.Const(T(0))).Where(x, exp(x).Add(ndarray.Const(float32(-1)).Cast[T]()).Mul(ndarray.Const(T(alpha))))
		})
	case "Exp":
		return unary(values, node, exp[T])
	case "Expand":
		return applyExpand(ctx, evaluator, values, node, integerShapes)
	case "EyeLike":
		return applyEyeLike(values, node)
	case "Floor":
		return unary(values, node, floor[T])
	case "Flatten":
		return applyFlatten(ctx, evaluator, values, node)
	case "GatherND":
		return applyGatherND(ctx, evaluator, values, node, integerShapes)
	case "Gather":
		return applyGather(ctx, evaluator, values, node, integerShapes)
	case "GatherElements":
		return applyGatherElements(ctx, evaluator, values, node)
	case "Gelu":
		approx := node.attributeString("approximate")
		if approx == "tanh" {
			return unary(values, node, geluTanh[T])
		}
		return unary(values, node, geluErf[T])
	case "Gemm":
		return applyGemm(ctx, evaluator, values, node)
	case "GlobalLpPool":
		return applyGlobalLpPool(ctx, evaluator, values, node)
	case "GlobalAveragePool":
		return applyGlobalPool(ctx, evaluator, values, node, true)
	case "GlobalMaxPool":
		return applyGlobalPool(ctx, evaluator, values, node, false)
	case "HammingWindow":
		return applyWindow(values, node, integerShapes, "hamming")
	case "HannWindow":
		return applyWindow(values, node, integerShapes, "hann")
	case "HardSigmoid":
		alpha := node.attributeFloat("alpha", 0.2)
		beta := node.attributeFloat("beta", 0.5)
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return hardSigmoid(x, alpha, beta)
		})
	case "GroupNormalization":
		return applyGroupNorm(ctx, evaluator, values, node)
	case "Greater":
		return binaryBroadcast(ctx, evaluator, values, node, func(a, b *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return bool01[T](b.CmpLt(a))
		})
	case "GreaterOrEqual":
		return binaryBroadcast(ctx, evaluator, values, node, func(a, b *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return bool01[T](a.GreaterEqual(b))
		})
	case "HardSwish":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return x.Mul(hardSigmoid(x, 1.0/6.0, 0.5))
		})
	case "Hardmax":
		return applyHardmax(ctx, evaluator, values, node)
	case "Identity":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] { return x })
	case "InstanceNormalization":
		return applyInstanceNorm(ctx, evaluator, values, node)
	case "IsInf":
		return applyIsInf(values, node)
	case "IsNaN":
		return applyIsNaN(values, node)
	case "LayerNormalization":
		return applyLayerNorm(ctx, evaluator, values, node)
	case "LeakyRelu":
		alpha := node.attributeFloat("alpha", 0.01)
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return leaky(x, alpha)
		})
	case "Less":
		return binaryBroadcast(ctx, evaluator, values, node, func(a, b *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return bool01[T](a.CmpLt(b))
		})
	case "LessOrEqual":
		return binaryBroadcast(ctx, evaluator, values, node, func(a, b *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return bool01[T](b.GreaterEqual(a))
		})
	case "Log":
		return unary(values, node, log[T])
	case "LogSoftmax":
		return applyLogSoftmax(ctx, evaluator, values, node)
	case "LpNormalization":
		return applyLpNorm(ctx, evaluator, values, node)
	case "LpPool":
		return applyLpPool(ctx, evaluator, values, node)
	case "MatMul":
		left, right, err := twoInputs(values, node.Inputs)
		if err != nil {
			return nil, err
		}
		left, err = leaf(ctx, evaluator, left)
		if err != nil {
			return nil, err
		}
		right, err = leaf(ctx, evaluator, right)
		if err != nil {
			return nil, err
		}
		return nn.MatrixMultiply(left, right)
	case "MeanVarianceNormalization":
		return applyMVN(ctx, evaluator, values, node)
	case "Max":
		return nary(ctx, evaluator, values, node, (*ndarray.Tensor[T]).Max)
	case "MaxPool":
		return function.maximumPool(ctx, evaluator, node, values)
	case "Mean":
		n := len(node.Inputs)
		sum, err := nary(ctx, evaluator, values, node, (*ndarray.Tensor[T]).Add)
		if err != nil {
			return nil, err
		}
		return divide(sum, ndarray.Const(T(n))), nil
	case "Min":
		return nary(ctx, evaluator, values, node, minimum[T])
	case "Mish":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return x.Mul(tanh(log(ndarray.Const(T(1)).Add(exp(x)))))
		})
	case "Mod":
		return applyMod(ctx, evaluator, values, node)
	case "Mul":
		return binaryBroadcast(ctx, evaluator, values, node, (*ndarray.Tensor[T]).Mul)
	case "NonZero":
		return applyNonZero(ctx, evaluator, values, node)
	case "Neg":
		return unary(values, node, (*ndarray.Tensor[T]).Neg)
	case "Not":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return x.Xor(ndarray.Const(T(1)))
		})
	case "Or":
		return binaryBroadcast(ctx, evaluator, values, node, (*ndarray.Tensor[T]).Or)
	case "OneHot":
		return applyOneHot(ctx, evaluator, values, node, integerShapes)
	case "Pad":
		return applyPad(ctx, evaluator, values, node, integerShapes)
	case "Pow":
		x, y, err := twoInputs(values, node.Inputs)
		if err != nil {
			return nil, err
		}
		x, y, err = broadcast(ctx, evaluator, x, y)
		if err != nil {
			return nil, err
		}
		return x.Log2().Mul(y).Exp2(), nil
	case "PRelu":
		return binaryBroadcast(ctx, evaluator, values, node, func(x, slope *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return x.GreaterEqual(ndarray.Const(T(0))).Where(x, x.Mul(slope))
		})
	case "Range":
		return applyRange(ctx, evaluator, values, node, integerShapes)
	case "Reciprocal":
		return unary(values, node, (*ndarray.Tensor[T]).Reciprocal)
	case "ReduceL1":
		x, err := oneInput(values, node.Inputs)
		if err != nil {
			return nil, err
		}
		return applyReduceOn(ctx, evaluator, absT(x), node, integerShapes, (*ndarray.Tensor[T]).Add, false)
	case "ReduceL2":
		x, err := oneInput(values, node.Inputs)
		if err != nil {
			return nil, err
		}
		r, err := applyReduceOn(ctx, evaluator, x.Mul(x), node, integerShapes, (*ndarray.Tensor[T]).Add, false)
		if err != nil {
			return nil, err
		}
		return r.Sqrt(), nil
	case "ReduceLogSum":
		r, err := applyReduce(ctx, evaluator, values, node, integerShapes, (*ndarray.Tensor[T]).Add, false)
		if err != nil {
			return nil, err
		}
		return log(r), nil
	case "ReduceLogSumExp":
		x, err := oneInput(values, node.Inputs)
		if err != nil {
			return nil, err
		}
		m, err := applyReduceOn(ctx, evaluator, x, node, integerShapes, (*ndarray.Tensor[T]).Max, false)
		if err != nil {
			return nil, err
		}
		m, err = expandLike(ctx, evaluator, m, x)
		if err != nil {
			return nil, err
		}
		s, err := applyReduceOn(ctx, evaluator, exp(x.Add(m.Neg())), node, integerShapes, (*ndarray.Tensor[T]).Add, false)
		if err != nil {
			return nil, err
		}
		return log(s).Add(m), nil
	case "ReduceMax":
		return applyReduce(ctx, evaluator, values, node, integerShapes, (*ndarray.Tensor[T]).Max, false)
	case "ReduceMin":
		return applyReduce(ctx, evaluator, values, node, integerShapes, minimum[T], false)
	case "ReduceMean":
		return applyReduce(ctx, evaluator, values, node, integerShapes, (*ndarray.Tensor[T]).Add, true)
	case "ReduceProd":
		return applyReduce(ctx, evaluator, values, node, integerShapes, (*ndarray.Tensor[T]).Mul, false)
	case "ReduceSum":
		return applyReduce(ctx, evaluator, values, node, integerShapes, (*ndarray.Tensor[T]).Add, false)
	case "ReduceSumSquare":
		x, err := oneInput(values, node.Inputs)
		if err != nil {
			return nil, err
		}
		return applyReduceOn(ctx, evaluator, x.Mul(x), node, integerShapes, (*ndarray.Tensor[T]).Add, false)
	case "RMSNormalization":
		return applyRMSNorm(ctx, evaluator, values, node)
	case "Resize":
		return applyResize(ctx, evaluator, values, node, integerShapes)
	case "ReverseSequence":
		return applyReverseSequence(ctx, evaluator, values, node, integerShapes)
	case "Relu":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return x.Max(ndarray.Const(T(0)))
		})
	case "Round":
		return unary(values, node, roundEven[T])
	case "Scatter":
		return applyScatterElements(ctx, evaluator, values, node)
	case "ScatterElements":
		return applyScatterElements(ctx, evaluator, values, node)
	case "ScatterND":
		return applyScatterND(ctx, evaluator, values, node, integerShapes)
	case "Selu":
		alpha := node.attributeFloat("alpha", 1.6732632423543772848170429916717)
		scale := node.attributeFloat("gamma", 1.0507009873554804934193349852946)
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			pos := x.Max(ndarray.Const(T(0)))
			neg := exp(x).Add(ndarray.Const(float32(-1)).Cast[T]()).Mul(ndarray.Const(T(alpha)))
			neg = minimum(neg, ndarray.Const(T(0)))
			return pos.Add(neg).Mul(ndarray.Const(T(scale)))
		})
	case "Shrink":
		lambd := node.attributeFloat("lambd", 0.5)
		bias := node.attributeFloat("bias", 0)
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			zero := ndarray.Const(T(0))
			hi := ndarray.Const(T(lambd))
			lo := ndarray.Const(T(-lambd))
			if bias == 0 {
				return x.GreaterEqual(hi).Where(x, x.CmpLt(lo).Where(x, zero))
			}
			return x.GreaterEqual(hi).Where(x.Add(ndarray.Const(T(-bias))), x.CmpLt(lo).Where(x.Add(ndarray.Const(T(bias))), zero))
		})
	case "Reshape":
		value, err := oneInput(values, node.Inputs[:1])
		if err != nil {
			return nil, err
		}
		if len(node.Inputs) < 2 {
			return nil, ndarray.ErrShape
		}
		shape, ok := integerShapes[node.Inputs[1]]
		if !ok {
			return nil, fmt.Errorf("%w: reshape shape %s", ErrOp, node.Inputs[1])
		}
		dims, err := reshapeDimensions(value.Shape(), shape, node.attributeInteger("allowzero", 0) != 0)
		if err != nil {
			return nil, err
		}
		return withView(ctx, evaluator, value, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
			return t.Reshape(dims)
		})
	case "Shape":
		return applyShape(values, node)
	case "Sigmoid":
		return unary(values, node, sigmoid[T])
	case "Sign":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			zero := ndarray.Const(T(0))
			pos := zero.CmpLt(x).Cast[T]()
			neg := x.CmpLt(zero).Cast[T]()
			return pos.Add(neg.Neg())
		})
	case "Split":
		return applySplit(ctx, evaluator, values, node, integerShapes)
	case "Slice":
		return applySlice(ctx, evaluator, values, node, integerShapes)
	case "Softmax":
		return applySoftmax(ctx, evaluator, values, node)
	case "Size":
		return applySize(values, node)
	case "Sin":
		return unary(values, node, (*ndarray.Tensor[T]).Sin)
	case "Sinh":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return exp(x).Add(exp(x.Neg()).Neg()).Mul(ndarray.Const(float32(0.5)).Cast[T]())
		})
	case "Softplus":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return log(ndarray.Const(T(1)).Add(exp(x)))
		})
	case "Softsign":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return x.Mul(ndarray.Const(T(1)).Add(absT(x)).Reciprocal())
		})
	case "SpaceToDepth":
		return applySpaceToDepth(ctx, evaluator, values, node)
	case "Sqrt":
		return unary(values, node, (*ndarray.Tensor[T]).Sqrt)
	case "Squeeze":
		return applySqueeze(ctx, evaluator, values, node, integerShapes)
	case "Sub":
		return binaryBroadcast(ctx, evaluator, values, node, func(a, b *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return a.Add(b.Neg())
		})
	case "Sum":
		return nary(ctx, evaluator, values, node, (*ndarray.Tensor[T]).Add)
	case "Swish":
		alpha := node.attributeFloat("alpha", 1)
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return x.Mul(sigmoid(x.Mul(ndarray.Const(T(alpha)))))
		})
	case "Tan":
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			return x.Sin().Mul(x.Add(ndarray.Const(piHalf).Cast[T]()).Sin().Reciprocal())
		})
	case "Tanh":
		return unary(values, node, tanh[T])
	case "Tile":
		return applyTile(ctx, evaluator, values, node, integerShapes)
	case "Trilu":
		return applyTrilu(ctx, evaluator, values, node, integerShapes)
	case "ThresholdedRelu":
		alpha := node.attributeFloat("alpha", 1)
		return unary(values, node, func(x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
			zero := ndarray.Const(T(0))
			return ndarray.Const(T(alpha)).CmpLt(x).Where(x, zero)
		})
	case "Transpose":
		return applyTranspose(ctx, evaluator, values, node)
	case "Upsample":
		return applyResize(ctx, evaluator, values, node, integerShapes)
	case "Unsqueeze":
		return applyUnsqueeze(ctx, evaluator, values, node, integerShapes)
	case "Where":
		return applyWhere(ctx, evaluator, values, node)
	case "Xor":
		return binaryBroadcast(ctx, evaluator, values, node, (*ndarray.Tensor[T]).Xor)
	default:
		return nil, fmt.Errorf("%w: %s", ErrOp, node.OperatorType)
	}
}

func (function *Function[T]) convolution(ctx context.Context, evaluator ndarray.Evaluator, node Node, values map[string]*ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	if node.attributeInteger("group", 1) != 1 {
		return nil, fmt.Errorf("%w: group", ErrOp)
	}
	for _, dilation := range node.attributeIntegers("dilations") {
		if dilation != 1 {
			return nil, fmt.Errorf("%w: dilations", ErrOp)
		}
	}
	strideHeight, strideWidth, err := spatialStrides(node.attributeIntegers("strides"))
	if err != nil {
		return nil, err
	}
	input, weight, err := twoInputs(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	input, err = leaf(ctx, evaluator, input)
	if err != nil {
		return nil, err
	}
	weight, err = leaf(ctx, evaluator, weight)
	if err != nil {
		return nil, err
	}
	inputShape, weightShape := input.Shape(), weight.Shape()
	if len(inputShape) != 4 || len(weightShape) != 4 {
		return nil, ndarray.ErrShape
	}
	kernelHeight, kernelWidth := weightShape[2], weightShape[3]
	if kernel := node.attributeIntegers("kernel_shape"); len(kernel) == 2 {
		kernelHeight, kernelWidth = int(kernel[0]), int(kernel[1])
	}
	pads, err := spatialPads(node.attributeString("auto_pad"), node.attributeIntegers("pads"), inputShape[2], inputShape[3], kernelHeight, kernelWidth, strideHeight, strideWidth)
	if err != nil {
		return nil, err
	}
	return nn.Convolution2D(input, weight, pads, strideHeight, strideWidth)
}

func (function *Function[T]) maximumPool(ctx context.Context, evaluator ndarray.Evaluator, node Node, values map[string]*ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	if node.attributeInteger("ceil_mode", 0) != 0 {
		return nil, fmt.Errorf("%w: ceil_mode", ErrOp)
	}
	for _, dilation := range node.attributeIntegers("dilations") {
		if dilation != 1 {
			return nil, fmt.Errorf("%w: dilations", ErrOp)
		}
	}
	kernelShape := node.attributeIntegers("kernel_shape")
	if len(kernelShape) != 2 {
		return nil, fmt.Errorf("%w: kernel_shape", ErrOp)
	}
	strideHeight, strideWidth, err := spatialStrides(node.attributeIntegers("strides"))
	if err != nil {
		return nil, err
	}
	input, err := oneInput(values, node.Inputs)
	if err != nil {
		return nil, err
	}
	input, err = leaf(ctx, evaluator, input)
	if err != nil {
		return nil, err
	}
	inputShape := input.Shape()
	if len(inputShape) != 4 {
		return nil, ndarray.ErrShape
	}
	kernelHeight, kernelWidth := int(kernelShape[0]), int(kernelShape[1])
	pads, err := spatialPads(node.attributeString("auto_pad"), node.attributeIntegers("pads"), inputShape[2], inputShape[3], kernelHeight, kernelWidth, strideHeight, strideWidth)
	if err != nil {
		return nil, err
	}
	return nn.MaximumPool2D(input, kernelHeight, kernelWidth, strideHeight, strideWidth, pads)
}

func spatialStrides(strides []int64) (int, int, error) {
	if len(strides) == 0 {
		return 1, 1, nil
	}
	if len(strides) != 2 || strides[0] < 1 || strides[1] < 1 {
		return 0, 0, fmt.Errorf("%w: strides", ErrOp)
	}
	return int(strides[0]), int(strides[1]), nil
}

func spatialPads(autoPad string, pads []int64, height, width, kernelHeight, kernelWidth, strideHeight, strideWidth int) ([]int, error) {
	switch autoPad {
	case "", "NOTSET":
		out := []int{0, 0, 0, 0}
		if len(pads) == 4 {
			out = []int{int(pads[0]), int(pads[1]), int(pads[2]), int(pads[3])}
		}
		return out, nil
	case "SAME_UPPER":
		heightBefore, heightAfter := samePadding(height, kernelHeight, strideHeight, true)
		widthBefore, widthAfter := samePadding(width, kernelWidth, strideWidth, true)
		return []int{heightBefore, widthBefore, heightAfter, widthAfter}, nil
	case "SAME_LOWER":
		heightBefore, heightAfter := samePadding(height, kernelHeight, strideHeight, false)
		widthBefore, widthAfter := samePadding(width, kernelWidth, strideWidth, false)
		return []int{heightBefore, widthBefore, heightAfter, widthAfter}, nil
	default:
		return nil, fmt.Errorf("%w: auto_pad %s", ErrOp, autoPad)
	}
}

func samePadding(in, kernel, stride int, upper bool) (before, after int) {
	if stride <= 0 {
		stride = 1
	}
	out := (in + stride - 1) / stride
	pad := max(0, (out-1)*stride+kernel-in)
	if upper {
		return pad / 2, pad - pad/2
	}
	return pad - pad/2, pad / 2
}

func realize[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, tensor *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	if tensor == nil {
		return nil, ndarray.ErrOp
	}
	out := make([]T, tensor.Size())
	if err := tensor.Eval(ctx, evaluator, out); err != nil {
		return nil, err
	}
	return ndarray.New(out, tensor.Shape())
}

func leaf[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, tensor *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
	if tensor == nil {
		return nil, ndarray.ErrOp
	}
	if _, err := tensor.Data(); err == nil {
		return tensor, nil
	}
	return realize(ctx, evaluator, tensor)
}

func withView[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, tensor *ndarray.Tensor[T], view func(*ndarray.Tensor[T]) (*ndarray.Tensor[T], error)) (*ndarray.Tensor[T], error) {
	out, err := view(tensor)
	if err == nil {
		return out, nil
	}
	if !errors.Is(err, ndarray.ErrOp) {
		return nil, err
	}
	tensor, err = realize(ctx, evaluator, tensor)
	if err != nil {
		return nil, err
	}
	return view(tensor)
}

func reshapeDimensions(input ndarray.Shape, dims []int64, allowZero bool) (ndarray.Shape, error) {
	out := make(ndarray.Shape, len(dims))
	for i, dim := range dims {
		switch {
		case dim == 0 && !allowZero:
			if i >= len(input) {
				return nil, ndarray.ErrShape
			}
			out[i] = input[i]
		default:
			out[i] = int(dim)
		}
	}
	return out, nil
}

func oneInput[T ndarray.Number](values map[string]*ndarray.Tensor[T], names []string) (*ndarray.Tensor[T], error) {
	if len(names) < 1 {
		return nil, ErrOp
	}
	value, ok := values[names[0]]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrGraph, names[0])
	}
	return value, nil
}

func twoInputs[T ndarray.Number](values map[string]*ndarray.Tensor[T], names []string) (*ndarray.Tensor[T], *ndarray.Tensor[T], error) {
	if len(names) < 2 {
		return nil, nil, ErrOp
	}
	left, ok := values[names[0]]
	if !ok {
		return nil, nil, fmt.Errorf("%w: %s", ErrGraph, names[0])
	}
	right, ok := values[names[1]]
	if !ok {
		return nil, nil, fmt.Errorf("%w: %s", ErrGraph, names[1])
	}
	return left, right, nil
}

func broadcast[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, left, right *ndarray.Tensor[T]) (*ndarray.Tensor[T], *ndarray.Tensor[T], error) {
	leftShape, rightShape := left.Shape(), right.Shape()
	rank := max(len(leftShape), len(rightShape))
	var err error
	left, err = padToRank(ctx, evaluator, left, rank)
	if err != nil {
		return nil, nil, err
	}
	right, err = padToRank(ctx, evaluator, right, rank)
	if err != nil {
		return nil, nil, err
	}
	leftShape, rightShape = left.Shape(), right.Shape()
	out := make(ndarray.Shape, rank)
	for i := range rank {
		dimLeft, dimRight := leftShape[i], rightShape[i]
		switch {
		case dimLeft == dimRight:
			out[i] = dimLeft
		case dimLeft == 1:
			out[i] = dimRight
		case dimRight == 1:
			out[i] = dimLeft
		default:
			return nil, nil, fmt.Errorf("%w: %v vs %v", ndarray.ErrShape, leftShape, rightShape)
		}
	}
	left, err = withView(ctx, evaluator, left, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(out)
	})
	if err != nil {
		return nil, nil, err
	}
	right, err = withView(ctx, evaluator, right, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Expand(out)
	})
	if err != nil {
		return nil, nil, err
	}
	return left, right, nil
}

func padToRank[T ndarray.Number](ctx context.Context, evaluator ndarray.Evaluator, tensor *ndarray.Tensor[T], rank int) (*ndarray.Tensor[T], error) {
	shape := tensor.Shape()
	if len(shape) == rank {
		return tensor, nil
	}
	if len(shape) > rank {
		return nil, ndarray.ErrShape
	}
	padded := make(ndarray.Shape, rank)
	copy(padded[rank-len(shape):], shape)
	for i := 0; i < rank-len(shape); i++ {
		padded[i] = 1
	}
	return withView(ctx, evaluator, tensor, func(t *ndarray.Tensor[T]) (*ndarray.Tensor[T], error) {
		return t.Reshape(padded)
	})
}

func allBits[T ndarray.Number]() T {
	var z T
	switch any(z).(type) {
	case uint8:
		return any(uint8(255)).(T)
	case int32:
		return any(int32(-1)).(T)
	default:
		return z
	}
}

func divide[T ndarray.Number](a, b *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	switch any(T(0)).(type) {
	case float32:
		return a.Div(b)
	default:
		return a.IDiv(b)
	}
}

func shapeFromDimensions(dims []int64) ndarray.Shape {
	shape := make(ndarray.Shape, len(dims))
	for i, dim := range dims {
		shape[i] = int(dim)
	}
	return shape
}
