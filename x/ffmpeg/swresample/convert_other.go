//go:build !amd64

package swresample

func s16ToF32Asm([]float32, []int16) bool { return false }

func f32ToS16Asm([]int16, []float32) bool { return false }
