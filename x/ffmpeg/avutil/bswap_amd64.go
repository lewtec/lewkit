//go:build amd64

package avutil

//go:noescape
func bswap16Pairs(p *byte, n int)

//go:noescape
func bswap32Groups(p *byte, n int)

func bswap16Asm(b []byte) bool {
	if len(b) < 2 {
		return true
	}
	n := len(b) / 2
	bswap16Pairs(&b[0], n)
	return true
}

func bswap32Asm(b []byte) bool {
	if len(b) < 4 {
		return true
	}
	n := len(b) / 4
	bswap32Groups(&b[0], n)
	return true
}
