//go:build !amd64

package avcodec

func gsmMult40Asm([]int16, []int16, int16) bool { return false }
