package disasm

// Section is one allocated slice of an object image.
// Address is the link address. A PE address includes the image base.
// A section with no file bytes keeps Size and a nil Bytes slice.
type Section struct {
	Name    string
	Address uint64
	Size    uint64
	Bytes   []byte
	Exec    bool
	Write   bool
	Read    bool
}
