package onnx

// implementedOperators is the ONNX op_type names Apply can lower.
func implementedOperators() []string {
	return []string{
		"Abs", "Acos", "Acosh", "Add", "And", "ArgMax", "ArgMin", "Asin", "Asinh", "Atan", "Atanh",
		"AveragePool", "BatchNormalization", "BitwiseAnd", "BitwiseNot", "BitwiseOr",
		"BitwiseXor", "BlackmanWindow", "Ceil", "Celu", "CenterCropPad",
		"Clip", "Compress", "Concat", "Constant", "ConstantOfShape",
		"Conv", "Cos", "Cosh", "CumSum", "DepthToSpace", "Div", "Dropout", "Elu", "Equal",
		"Erf", "Exp", "Expand", "EyeLike", "Flatten", "Floor", "Gather", "GatherElements", "GatherND",
		"Gelu", "Gemm", "GlobalAveragePool", "GlobalMaxPool", "Greater", "GreaterOrEqual",
		"GroupNormalization", "HammingWindow", "HannWindow",
		"HardSigmoid", "HardSwish", "Hardmax", "Identity", "InstanceNormalization",
		"IsInf", "IsNaN", "LayerNormalization", "LeakyRelu", "Less",
		"LessOrEqual", "Log", "LogSoftmax", "LpNormalization", "MatMul", "Max", "MaxPool", "Mean",
		"MeanVarianceNormalization", "Min", "Mish", "Mod", "Mul", "Neg", "NonZero", "Not", "OneHot", "Or", "Pad", "PRelu",
		"RMSNormalization", "Range", "Reciprocal", "ReduceL1", "ReduceL2", "ReduceLogSum",
		"ReduceMax", "ReduceMean", "ReduceMin", "ReduceProd", "ReduceSum", "ReduceSumSquare",
		"Relu", "Reshape", "ReverseSequence", "Round", "Scatter", "ScatterElements", "ScatterND",
		"Selu", "Shape", "Shrink", "Sigmoid", "Sign", "Sin", "Sinh", "Size", "Slice", "Softmax",
		"Softplus", "Softsign", "SpaceToDepth", "Split", "Sqrt", "Squeeze", "Sub", "Sum",
		"Swish", "Tanh", "ThresholdedRelu", "Tile", "Transpose", "Trilu",
		"Unsqueeze", "Where", "Xor",
	}
}

func implementedOperator(name string) bool {
	for _, op := range implementedOperators() {
		if op == name {
			return true
		}
	}
	return false
}

// onnxOperators is the ai.onnx catalog (unique op_type, latest docs).
// Not every opset version: one name per operator.
var onnxOperators = []string{
	"Abs", "Acos", "Acosh", "Add", "AffineGrid", "And", "ArgMax", "ArgMin",
	"Asin", "Asinh", "Atan", "Atanh", "Attention", "AveragePool",
	"BatchNormalization", "Bernoulli", "BitCast", "BitShift", "BitwiseAnd",
	"BitwiseNot", "BitwiseOr", "BitwiseXor", "BlackmanWindow",
	"Cast", "CastLike", "CausalConvWithState", "Ceil", "Celu", "CenterCropPad",
	"Clip", "Col2Im", "Compress", "Concat", "ConcatFromSequence", "Constant",
	"ConstantOfShape", "Conv", "ConvInteger", "ConvTranspose", "Cos", "Cosh",
	"CumProd", "CumSum",
	"DFT", "DeformConv", "DepthToSpace", "DequantizeLinear", "Det", "Div",
	"Dropout", "DynamicQuantizeLinear",
	"Einsum", "Elu", "Equal", "Erf", "Exp", "Expand", "EyeLike",
	"Flatten", "Floor",
	"GRU", "Gather", "GatherElements", "GatherND", "Gelu", "Gemm",
	"GlobalAveragePool", "GlobalLpPool", "GlobalMaxPool", "Greater",
	"GreaterOrEqual", "GridSample", "GroupNormalization",
	"HammingWindow", "HannWindow", "HardSigmoid", "HardSwish", "Hardmax",
	"Identity", "If", "ImageDecoder", "InstanceNormalization", "IsInf", "IsNaN",
	"LRN", "LSTM", "LayerNormalization", "LeakyRelu", "Less", "LessOrEqual",
	"LinearAttention", "Log", "LogSoftmax", "Loop", "LpNormalization", "LpPool",
	"MatMul", "MatMulInteger", "Max", "MaxPool", "MaxRoiPool", "MaxUnpool",
	"Mean", "MeanVarianceNormalization", "MelWeightMatrix", "Min", "Mish",
	"Mod", "Mul", "Multinomial",
	"Neg", "NonMaxSuppression", "NonZero", "Not",
	"OneHot", "Optional", "OptionalGetElement", "OptionalHasElement", "Or",
	"PRelu", "Pad", "Pow",
	"QLinearConv", "QLinearMatMul", "QuantizeLinear",
	"RNN", "RMSNormalization", "RandomNormal", "RandomNormalLike",
	"RandomUniform", "RandomUniformLike", "Range", "Reciprocal",
	"ReduceL1", "ReduceL2", "ReduceLogSum", "ReduceLogSumExp", "ReduceMax",
	"ReduceMean", "ReduceMin", "ReduceProd", "ReduceSum", "ReduceSumSquare",
	"Relu", "Reshape", "Resize", "ReverseSequence", "RoiAlign", "RotaryEmbedding",
	"Round",
	"STFT", "Scan", "Scatter", "ScatterElements", "ScatterND", "Selu",
	"SequenceAt", "SequenceConstruct", "SequenceEmpty", "SequenceErase",
	"SequenceInsert", "SequenceLength", "Shape", "Shrink", "Sigmoid", "Sign",
	"Sin", "Sinh", "Size", "Slice", "Softmax", "SoftmaxCrossEntropyLoss",
	"Softplus", "Softsign", "SpaceToDepth", "Split", "SplitToSequence", "Sqrt",
	"Squeeze", "StringNormalizer", "Sub", "Sum", "SwiGLU", "Swish",
	"Tan", "Tanh", "TensorScatter", "TfIdfVectorizer", "ThresholdedRelu",
	"Tile", "TopK", "Transpose", "Trilu",
	"Unique", "Unsqueeze", "Upsample",
	"Where",
	"Xor",
}
