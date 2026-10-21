package avformat

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// track is one ISOBMFF sample table built while packets are grouped into mdat.
type track struct {
	st      Stream
	sizes   []int
	deltas  []int
	offset  int
	bytes   int
}

func writeMOV(streams []Stream, pkts []*avutil.Packet) ([]byte, error) {
	if len(streams) == 0 {
		return nil, fmt.Errorf("%w: mov", avutil.ErrInvalidData)
	}
	ftyp := box("ftyp", append(append([]byte("isom"), be32(0x200)...), []byte("isomiso2mp41")...))
	var mdat []byte
	tracks := make([]track, len(streams))
	for i, st := range streams {
		tracks[i].st = st
		tracks[i].offset = len(mdat)
		for _, pkt := range pkts {
			if pkt.Stream != i {
				continue
			}
			delta := int(pkt.Duration)
			if delta <= 0 {
				delta = 1
			}
			tracks[i].sizes = append(tracks[i].sizes, len(pkt.Data))
			tracks[i].deltas = append(tracks[i].deltas, delta)
			tracks[i].bytes += len(pkt.Data)
			mdat = append(mdat, pkt.Data...)
		}
	}
	mdatBox := box("mdat", mdat)
	dataOff := len(ftyp) + 8
	var kids []byte
	kids = append(kids, mvhd(tracks)...)
	for i := range tracks {
		if tracks[i].offset < 0 {
			tracks[i].offset = 0
		}
		kids = append(kids, trakBox(i+1, tracks[i], dataOff)...)
	}
	moov := box("moov", kids)
	return append(append(ftyp, mdatBox...), moov...), nil
}

func mvhd(tracks []track) []byte {
	b := make([]byte, 100)
	binary.BigEndian.PutUint32(b[12:], 1000)
	var dur int64
	for _, tr := range tracks {
		if ms := trackMillis(tr); ms > dur {
			dur = ms
		}
	}
	binary.BigEndian.PutUint32(b[16:], uint32(dur))
	binary.BigEndian.PutUint32(b[20:], 0x00010000)
	binary.BigEndian.PutUint16(b[24:], 0x0100)
	unity(b[36:])
	binary.BigEndian.PutUint32(b[96:], uint32(len(tracks)+1))
	return box("mvhd", b)
}

func trakBox(id int, tr track, dataOff int) []byte {
	video := tr.st.Par.Type != avutil.MediaAudio
	tkhd := make([]byte, 84)
	tkhd[3] = 7
	binary.BigEndian.PutUint32(tkhd[12:], uint32(id))
	binary.BigEndian.PutUint32(tkhd[20:], uint32(trackMillis(tr)))
	unity(tkhd[40:])
	if video {
		binary.BigEndian.PutUint32(tkhd[76:], uint32(tr.st.Par.Width)<<16)
		binary.BigEndian.PutUint32(tkhd[80:], uint32(tr.st.Par.Height)<<16)
	} else {
		binary.BigEndian.PutUint16(tkhd[36:], 0x0100)
	}
	mdhd := make([]byte, 24)
	scale := timescale(tr.st)
	binary.BigEndian.PutUint32(mdhd[12:], uint32(scale))
	binary.BigEndian.PutUint32(mdhd[16:], uint32(trackMediaTicks(tr, scale)))
	binary.BigEndian.PutUint16(mdhd[20:], 0x55c4)
	hdlr := make([]byte, 4+4+4+12+1)
	if video {
		copy(hdlr[8:], "vide")
	} else {
		copy(hdlr[8:], "soun")
	}
	minf := vmhd()
	if !video {
		minf = box("smhd", make([]byte, 8))
	}
	dref := box("dref", append(be32(0), append(be32(1), box("url ", []byte{0, 0, 0, 1})...)...))
	stbl := append(stsd(tr.st), stts(tr.deltas)...)
	stbl = append(stbl, stsc(len(tr.sizes))...)
	stbl = append(stbl, stsz(tr.sizes)...)
	stbl = append(stbl, stco(dataOff+tr.offset)...)
	mdia := append(box("mdhd", mdhd), box("hdlr", hdlr)...)
	mdia = append(mdia, box("minf", append(append(minf, box("dinf", dref)...), box("stbl", stbl)...))...)
	return box("trak", append(box("tkhd", tkhd), box("mdia", mdia)...))
}

func vmhd() []byte {
	b := make([]byte, 12)
	b[3] = 1
	return box("vmhd", b)
}

func stsd(st Stream) []byte {
	var entry []byte
	if st.Par.Type == avutil.MediaAudio {
		entry = soundEntry(st.Par)
	} else {
		entry = visualEntry(st.Par)
	}
	body := append(be32(0), be32(1)...)
	body = append(body, entry...)
	return box("stsd", body)
}

func visualEntry(par avcodec.Parameters) []byte {
	b := make([]byte, 86)
	kind := "jpeg"
	if par.ID == avcodec.IDRawVideo {
		kind = "raw "
	}
	copy(b[4:], kind)
	binary.BigEndian.PutUint16(b[14:], 1)
	binary.BigEndian.PutUint16(b[32:], uint16(par.Width))
	binary.BigEndian.PutUint16(b[34:], uint16(par.Height))
	binary.BigEndian.PutUint32(b[36:], 0x00480000)
	binary.BigEndian.PutUint32(b[40:], 0x00480000)
	binary.BigEndian.PutUint16(b[48:], 1)
	binary.BigEndian.PutUint16(b[82:], 0x0018)
	binary.BigEndian.PutUint16(b[84:], 0xffff)
	binary.BigEndian.PutUint32(b[0:], 86)
	return b
}

func soundEntry(par avcodec.Parameters) []byte {
	b := make([]byte, 36)
	kind := "sowt"
	if par.ID == avcodec.IDPCMS16BE {
		kind = "twos"
	}
	copy(b[4:], kind)
	binary.BigEndian.PutUint16(b[14:], 1)
	ch := par.Layout.Channels()
	if ch <= 0 {
		ch = 1
	}
	binary.BigEndian.PutUint16(b[24:], uint16(ch))
	binary.BigEndian.PutUint16(b[26:], wavBits(par.ID))
	rate := par.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	binary.BigEndian.PutUint32(b[32:], uint32(rate)<<16)
	binary.BigEndian.PutUint32(b[0:], 36)
	return b
}

func stts(deltas []int) []byte {
	body := append(be32(0), be32(uint32(len(deltas)))...)
	for _, d := range deltas {
		body = append(body, be32(1)...)
		body = append(body, be32(uint32(d))...)
	}
	return box("stts", body)
}

func stsc(n int) []byte {
	if n < 1 {
		n = 1
	}
	body := append(be32(0), be32(1)...)
	body = append(body, be32(1)...)
	body = append(body, be32(uint32(n))...)
	body = append(body, be32(1)...)
	return box("stsc", body)
}

func stsz(sizes []int) []byte {
	body := append(be32(0), be32(0)...)
	body = append(body, be32(uint32(len(sizes)))...)
	for _, n := range sizes {
		body = append(body, be32(uint32(n))...)
	}
	return box("stsz", body)
}

func stco(offset int) []byte {
	body := append(be32(0), be32(1)...)
	body = append(body, be32(uint32(offset))...)
	return box("stco", body)
}

func trackMillis(tr track) int64 {
	ticks := 0
	for _, d := range tr.deltas {
		ticks += d
	}
	tb := tr.st.TimeBase
	if tb.Den == 0 || tb.Num == 0 {
		return int64(ticks) * 1000
	}
	return (int64(ticks)*int64(tb.Num)*1000 + int64(tb.Den)/2) / int64(tb.Den)
}

func trackMediaTicks(tr track, scale int) int {
	ticks := 0
	for _, d := range tr.deltas {
		ticks += d
	}
	tb := tr.st.TimeBase
	if tb.Den == 0 || tb.Num == 0 || scale == int(tb.Den) && tb.Num == 1 {
		return ticks
	}
	return int(int64(ticks) * int64(tb.Num) * int64(scale) / int64(tb.Den))
}

func timescale(st Stream) int {
	if st.Par.Type == avutil.MediaAudio && st.Par.SampleRate > 0 {
		return st.Par.SampleRate
	}
	if st.TimeBase.Num == 1 && st.TimeBase.Den > 0 {
		return int(st.TimeBase.Den)
	}
	if st.TimeBase.Den > 0 {
		return int(st.TimeBase.Den)
	}
	return 25
}

func unity(b []byte) {
	binary.BigEndian.PutUint32(b[0:], 0x00010000)
	binary.BigEndian.PutUint32(b[16:], 0x00010000)
	binary.BigEndian.PutUint32(b[32:], 0x40000000)
}

func box(typ string, payload []byte) []byte {
	out := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(out[0:], uint32(len(out)))
	copy(out[4:], typ)
	copy(out[8:], payload)
	return out
}

func be32(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

type atom struct {
	typ     string
	payload []byte
	kids    []atom
}

func parseAtoms(b []byte) []atom {
	var out []atom
	for len(b) >= 8 {
		n := int(binary.BigEndian.Uint32(b[:4]))
		if n < 8 || n > len(b) {
			break
		}
		typ := string(b[4:8])
		payload := b[8:n]
		item := atom{typ: typ, payload: payload}
		if typ == "moov" || typ == "trak" || typ == "mdia" || typ == "minf" || typ == "stbl" || typ == "dinf" {
			item.kids = parseAtoms(payload)
		}
		out = append(out, item)
		b = b[n:]
	}
	return out
}

func openMOV(b []byte) (Demuxer, error) {
	atoms := parseAtoms(b)
	var mdat []byte
	var mdatOff int
	off := 0
	for _, a := range atoms {
		if a.typ == "mdat" {
			mdat = a.payload
			mdatOff = off + 8
		}
		off += 8 + len(a.payload)
	}
	var moov atom
	for _, a := range atoms {
		if a.typ == "moov" {
			moov = a
		}
	}
	if mdat == nil || moov.typ == "" {
		return nil, fmt.Errorf("%w: mov", avutil.ErrInvalidData)
	}
	var streams []Stream
	var pkts []*avutil.Packet
	for _, trak := range moov.kids {
		if trak.typ != "trak" {
			continue
		}
		st, samples := movTrack(trak, mdat, mdatOff)
		index := len(streams)
		st.Index = index
		streams = append(streams, st)
		pts := int64(0)
		for _, sample := range samples {
			pkts = append(pkts, &avutil.Packet{
				Data: sample.data, Stream: index, PTS: pts, DTS: pts,
				Duration: int64(sample.delta), TimeBase: st.TimeBase, Key: true,
			})
			pts += int64(sample.delta)
		}
	}
	if len(streams) == 0 {
		return nil, fmt.Errorf("%w: mov tracks", avutil.ErrInvalidData)
	}
	return &memDemux{streams: streams, pkts: pkts}, nil
}

type movSample struct {
	data  []byte
	delta int
}

func movTrack(trak atom, mdat []byte, mdatOff int) (Stream, []movSample) {
	st := Stream{TimeBase: avutil.Q(1, 25), SAR: avutil.Q(1, 1)}
	var mdia atom
	for _, k := range trak.kids {
		if k.typ == "mdia" {
			mdia = k
		}
		if k.typ == "tkhd" && len(k.payload) >= 84 {
			st.Par.Width = int(binary.BigEndian.Uint32(k.payload[76:80]) >> 16)
			st.Par.Height = int(binary.BigEndian.Uint32(k.payload[80:84]) >> 16)
		}
	}
	var hdlr, mdhd, stbl atom
	for _, k := range mdia.kids {
		switch k.typ {
		case "hdlr":
			hdlr = k
		case "mdhd":
			mdhd = k
		case "minf":
			for _, m := range k.kids {
				if m.typ == "stbl" {
					stbl = m
				}
			}
		}
	}
	scale := 25
	if len(mdhd.payload) >= 20 && mdhd.payload[0] == 0 {
		scale = int(binary.BigEndian.Uint32(mdhd.payload[12:16]))
	}
	if scale <= 0 {
		scale = 25
	}
	video := true
	if len(hdlr.payload) >= 12 && string(hdlr.payload[8:12]) == "soun" {
		video = false
	}
	var entry string
	var sizes []int
	var deltas []int
	var chunk int
	for _, k := range stbl.kids {
		switch k.typ {
		case "stsd":
			entry = stsdType(k.payload)
		case "stsz":
			sizes = stszSizes(k.payload)
		case "stts":
			deltas = sttsDeltas(k.payload, len(sizes))
		case "stco":
			if len(k.payload) >= 12 {
				chunk = int(binary.BigEndian.Uint32(k.payload[8:12]))
			}
		}
	}
	if video {
		st.Par.Type = avutil.MediaVideo
		st.Par.ID = avcodec.IDMJPEG
		st.Par.Format = avutil.PixYUVJ420P
		if entry == "raw " {
			st.Par.ID = avcodec.IDRawVideo
			st.Par.Format = avutil.PixRGB24
		}
		st.TimeBase = avutil.Q(1, int32(scale))
	} else {
		st.Par.Type = avutil.MediaAudio
		st.Par.ID = avcodec.IDPCMS16LE
		st.Par.SampleFormat = avutil.SampleS16
		st.Par.SampleRate = scale
		st.Par.Layout = avutil.Stereo()
		if entry == "twos" {
			st.Par.ID = avcodec.IDPCMS16BE
		}
		st.TimeBase = avutil.Q(1, int32(scale))
		if ch, ok := stsdChannels(stbl); ok {
			st.Par.Layout = layoutOr(avutil.ChannelLayout{}, ch)
		}
	}
	st.Par.TimeBase = st.TimeBase
	rel := chunk - mdatOff
	var samples []movSample
	for i, n := range sizes {
		if rel < 0 || rel+n > len(mdat) {
			break
		}
		delta := 1
		if i < len(deltas) {
			delta = deltas[i]
		}
		samples = append(samples, movSample{data: append([]byte(nil), mdat[rel:rel+n]...), delta: delta})
		rel += n
	}
	return st, samples
}

func stsdType(b []byte) string {
	if len(b) < 16 {
		return ""
	}
	return string(b[12:16])
}

func stsdChannels(stbl atom) (int, bool) {
	for _, k := range stbl.kids {
		if k.typ == "stsd" && len(k.payload) >= 36 {
			return int(binary.BigEndian.Uint16(k.payload[32:34])), true
		}
	}
	return 0, false
}

func stszSizes(b []byte) []int {
	if len(b) < 12 {
		return nil
	}
	fixed := binary.BigEndian.Uint32(b[4:8])
	n := int(binary.BigEndian.Uint32(b[8:12]))
	if fixed != 0 {
		out := make([]int, n)
		for i := range out {
			out[i] = int(fixed)
		}
		return out
	}
	out := make([]int, 0, n)
	for i := 0; i < n && 12+4*i+4 <= len(b); i++ {
		out = append(out, int(binary.BigEndian.Uint32(b[12+4*i:])))
	}
	return out
}

func sttsDeltas(b []byte, n int) []int {
	if len(b) < 8 {
		return nil
	}
	entries := int(binary.BigEndian.Uint32(b[4:8]))
	var out []int
	pos := 8
	for i := 0; i < entries && pos+8 <= len(b); i++ {
		count := int(binary.BigEndian.Uint32(b[pos:]))
		delta := int(binary.BigEndian.Uint32(b[pos+4:]))
		for j := 0; j < count; j++ {
			out = append(out, delta)
		}
		pos += 8
	}
	return out
}
