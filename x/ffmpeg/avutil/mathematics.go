package avutil

import "math"

// Rounding is av_rescale_rnd's rounding mode.
//
//lewkit:enum fold
type Rounding int

const (
	// RoundZero truncates toward zero.
	RoundZero Rounding = 0 // zero
	// RoundInf rounds away from zero.
	RoundInf Rounding = 1 // inf
	// RoundDown rounds toward negative infinity.
	RoundDown Rounding = 2 // down
	// RoundUp rounds toward positive infinity.
	RoundUp Rounding = 3 // up
	// RoundNearInf rounds to nearest, ties away from zero.
	RoundNearInf Rounding = 5 // near_inf
	// RoundPassMinMax returns int64 min and max unchanged.
	RoundPassMinMax Rounding = 8192 // pass_minmax
)

const (
	int32Max int64 = 2147483647
	int64Max int64 = 9223372036854775807
	int64Min int64 = -9223372036854775808
)

// RescaleRnd computes a*b/c with the requested rounding.
// A bad denominator or multiplier returns the minimum int64.
func RescaleRnd(a, b, c int64, rnd Rounding) int64 {
	mode := rnd &^ RoundPassMinMax
	if c <= 0 || b < 0 || mode > 5 || mode == 4 {
		return int64Min
	}
	if rnd&RoundPassMinMax != 0 {
		if a == int64Min || a == int64Max {
			return a
		}
		rnd &^= RoundPassMinMax
	}
	if a < 0 {
		flipped := a
		if flipped < -int64Max {
			flipped = -int64Max
		}
		return -int64(uint64(RescaleRnd(-flipped, b, c, rnd^((rnd >> 1) & 1))))
	}
	var r int64
	if rnd == RoundNearInf {
		r = c / 2
	} else if rnd&1 != 0 {
		r = c - 1
	}
	if b <= int32Max && c <= int32Max {
		if a <= int32Max {
			return (a*b + r) / c
		}
		ad := a / c
		a2 := (a%c*b + r) / c
		if ad >= int32Max && b != 0 && ad > (int64Max-a2)/b {
			return int64Min
		}
		return ad*b + a2
	}
	a0 := uint64(a) & 0xFFFFFFFF
	a1 := uint64(a) >> 32
	b0 := uint64(b) & 0xFFFFFFFF
	b1 := uint64(b) >> 32
	t1 := a0*b1 + a1*b0
	t1a := t1 << 32
	a0 = a0*b0 + t1a
	a1 = a1*b1 + (t1 >> 32)
	if a0 < t1a {
		a1++
	}
	a0 += uint64(r)
	if a0 < uint64(r) {
		a1++
	}
	var quot uint64
	for i := 63; i >= 0; i-- {
		a1 += a1 + ((a0 >> i) & 1)
		quot += quot
		if uint64(c) <= a1 {
			a1 -= uint64(c)
			quot++
		}
	}
	if quot > uint64(int64Max) {
		return int64Min
	}
	return int64(quot)
}

// Rescale is RescaleRnd with near-infinity rounding.
func Rescale(a, b, c int64) int64 { return RescaleRnd(a, b, c, RoundNearInf) }

// RescaleQ converts a timestamp from one time base to another.
func RescaleQ(a int64, from, to Rational) int64 {
	return RescaleQRnd(a, from, to, RoundNearInf)
}

// RescaleQRnd converts a timestamp with an explicit rounding mode.
func RescaleQRnd(a int64, from, to Rational, rnd Rounding) int64 {
	return RescaleRnd(a, int64(from.Num)*int64(to.Den), int64(to.Num)*int64(from.Den), rnd)
}

// ClipInt clamps v to [min, max].
func ClipInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ClipUint8 clamps v to a byte.
func ClipUint8(v int) byte {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return byte(v)
}

// ClipInt16 clamps v to a signed 16-bit sample.
func ClipInt16(v int) int16 {
	if v < math.MinInt16 {
		return math.MinInt16
	}
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	return int16(v)
}
