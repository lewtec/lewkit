//go:build !amd64

package avutil

func bswap16Asm([]byte) bool { return false }

func bswap32Asm([]byte) bool { return false }
