package avutil

// Packet is one compressed or raw access unit.
type Packet struct {
	Data     []byte
	Stream   int
	PTS      int64
	DTS      int64
	Duration int64
	TimeBase Rational
	Key      bool
	Pos      int64
}

// Clone copies the packet bytes.
func (pkt *Packet) Clone() *Packet {
	if pkt == nil {
		return nil
	}
	out := *pkt
	out.Data = append([]byte(nil), pkt.Data...)
	return &out
}
