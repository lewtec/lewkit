package avutil

import "math/bits"

// Rational is a numerator and a denominator.
// The fields are 32-bit so overflow matches libavutil.
type Rational struct {
	Num int32
	Den int32
}

// Q builds a rational and does not reduce it.
func Q(num, den int32) Rational { return Rational{Num: num, Den: den} }

// GCD is Stein's binary GCD, as av_gcd.
func GCD(a, b int64) int64 {
	if a == 0 {
		return b
	}
	if b == 0 {
		return a
	}
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	za := bits.TrailingZeros64(uint64(a))
	zb := bits.TrailingZeros64(uint64(b))
	k := za
	if zb < k {
		k = zb
	}
	u := a >> za
	v := b >> zb
	for u != v {
		if u > v {
			u, v = v, u
		}
		v -= u
		v >>= bits.TrailingZeros64(uint64(v))
	}
	return u << k
}

// Reduce simplifies num/den so both magnitudes are at most max.
// The bool is true when the result is exact.
func Reduce(num, den, max int64) (Rational, bool) {
	a0n, a0d := int64(0), int64(1)
	a1n, a1d := int64(1), int64(0)
	sign := (num < 0) != (den < 0)
	if num < 0 {
		num = -num
	}
	if den < 0 {
		den = -den
	}
	gcd := GCD(num, den)
	if gcd != 0 {
		num /= gcd
		den /= gcd
	}
	exact := true
	if num <= max && den <= max {
		a1n, a1d = num, den
		den = 0
	}
	for den != 0 {
		x := num / den
		nextDen := num - den*x
		a2n := x*a1n + a0n
		a2d := x*a1d + a0d
		if a2n > max || a2d > max {
			if a1n != 0 {
				x = (max - a0n) / a1n
			}
			if a1d != 0 {
				limit := (max - a0d) / a1d
				if limit < x {
					x = limit
				}
			}
			if den*(2*x*a1d+a0d) > num*a1d {
				a1n = x*a1n + a0n
				a1d = x*a1d + a0d
			}
			exact = false
			break
		}
		a0n, a0d = a1n, a1d
		a1n, a1d = a2n, a2d
		num = den
		den = nextDen
	}
	if sign {
		a1n = -a1n
	}
	return Rational{Num: int32(a1n), Den: int32(a1d)}, exact
}

// Mul multiplies two rationals and reduces the product.
func (b Rational) Mul(c Rational) Rational {
	r, _ := Reduce(int64(b.Num)*int64(c.Num), int64(b.Den)*int64(c.Den), int64(int32(^uint32(0)>>1)))
	return r
}

// Div divides b by c.
func (b Rational) Div(c Rational) Rational {
	return b.Mul(Rational{Num: c.Den, Den: c.Num})
}

// Add adds two rationals.
func (b Rational) Add(c Rational) Rational {
	r, _ := Reduce(
		int64(b.Num)*int64(c.Den)+int64(c.Num)*int64(b.Den),
		int64(b.Den)*int64(c.Den),
		int64(int32(^uint32(0)>>1)),
	)
	return r
}

// Sub subtracts c from b.
func (b Rational) Sub(c Rational) Rational {
	return b.Add(Rational{Num: -c.Num, Den: c.Den})
}

// Inv swaps the numerator and the denominator.
func (b Rational) Inv() Rational { return Rational{Num: b.Den, Den: b.Num} }

// Float64 is the double value. A zero denominator is zero.
func (b Rational) Float64() float64 {
	if b.Den == 0 {
		return 0
	}
	return float64(b.Num) / float64(b.Den)
}

// Cmp compares two rationals.
// The result is negative, zero, or positive. A missing denominator is the minimum int32.
func (a Rational) Cmp(b Rational) int32 {
	tmp := int64(a.Num)*int64(b.Den) - int64(b.Num)*int64(a.Den)
	if tmp != 0 {
		if tmp < 0 {
			return -1
		}
		return 1
	}
	if b.Den != 0 && a.Den != 0 {
		return 0
	}
	if a.Num != 0 && b.Num != 0 {
		return (a.Num >> 31) - (b.Num >> 31)
	}
	return int32(^uint32(0) >> 1) * -1
}
