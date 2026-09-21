package onnx

import "github.com/lewtec/lewkit/x/ndarray"

func erf[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	ax := absT(x)
	p := ndarray.Const(float32(0.3275911)).Cast[T]()
	t := ndarray.Const(T(1)).Mul(ndarray.Const(T(1)).Add(p.Mul(ax)).Reciprocal())
	a1 := ndarray.Const(float32(0.254829592)).Cast[T]()
	a2 := ndarray.Const(float32(-0.284496736)).Cast[T]()
	a3 := ndarray.Const(float32(1.421413741)).Cast[T]()
	a4 := ndarray.Const(float32(-1.453152027)).Cast[T]()
	a5 := ndarray.Const(float32(1.061405429)).Cast[T]()
	poly := a5.Mul(t).Add(a4)
	poly = poly.Mul(t).Add(a3)
	poly = poly.Mul(t).Add(a2)
	poly = poly.Mul(t).Add(a1)
	tau := poly.Mul(t).Mul(exp(ax.Mul(ax).Neg()))
	pos := ndarray.Const(T(1)).Add(tau.Neg())
	return x.CmpLt(ndarray.Const(T(0))).Where(pos.Neg(), pos)
}

func atan01[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	x2 := x.Mul(x)
	p := ndarray.Const(float32(0.0208351)).Cast[T]()
	p = p.Mul(x2).Add(ndarray.Const(float32(-0.0851330)).Cast[T]())
	p = p.Mul(x2).Add(ndarray.Const(float32(0.1801410)).Cast[T]())
	p = p.Mul(x2).Add(ndarray.Const(float32(-0.3302995)).Cast[T]())
	p = p.Mul(x2).Add(ndarray.Const(float32(0.9998660)).Cast[T]())
	return x.Mul(p)
}

func atan[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	ax := absT(x)
	one := ndarray.Const(T(1))
	hi := one.CmpLt(ax)
	z := hi.Where(ax.Reciprocal(), ax)
	p := atan01(z)
	p = hi.Where(ndarray.Const(piHalf).Cast[T]().Add(p.Neg()), p)
	return x.CmpLt(ndarray.Const(T(0))).Where(p.Neg(), p)
}

func acos[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	one := ndarray.Const(T(1))
	s := one.Add(x.Neg()).Mul(one.Add(x).Reciprocal()).Sqrt()
	return atan(s).Mul(ndarray.Const(T(2)))
}

func asin[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	return ndarray.Const(piHalf).Cast[T]().Add(acos(x).Neg())
}

func geluErf[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	s := ndarray.Const(float32(0.7071067811865476)).Cast[T]()
	return x.Mul(ndarray.Const(float32(0.5)).Cast[T]()).Mul(ndarray.Const(T(1)).Add(erf(x.Mul(s))))
}

func geluTanh[T ndarray.Number](x *ndarray.Tensor[T]) *ndarray.Tensor[T] {
	c := ndarray.Const(float32(0.7978845608028654)).Cast[T]()
	inner := x.Add(x.Mul(x).Mul(x).Mul(ndarray.Const(float32(0.044715)).Cast[T]()))
	return x.Mul(ndarray.Const(float32(0.5)).Cast[T]()).Mul(ndarray.Const(T(1)).Add(tanh(c.Mul(inner))))
}
