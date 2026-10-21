package avformat

import (
	"encoding/binary"
	"fmt"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func writeAVI(streams []Stream, pkts []*avutil.Packet) ([]byte, error) {
	if len(streams) == 0 {
		return nil, fmt.Errorf("%w: avi", avutil.ErrInvalidData)
	}
	var hdr []byte
	var movi []byte
	type idx struct {
		id     string
		offset int
		size   int
	}
	var index []idx
	avih := make([]byte, 56)
	var videoFrames int
	var maxChunk int
	for _, pkt := range pkts {
		if pkt.Stream >= 0 && pkt.Stream < len(streams) && streams[pkt.Stream].Par.Type == avutil.MediaVideo {
			videoFrames++
		}
		if len(pkt.Data) > maxChunk {
			maxChunk = len(pkt.Data)
		}
	}
	width, height := 0, 0
	tb := avutil.Q(1, 25)
	for _, st := range streams {
		if st.Par.Type == avutil.MediaVideo {
			width, height = st.Par.Width, st.Par.Height
			if st.TimeBase.Den != 0 {
				tb = st.TimeBase
			}
		}
	}
	usec := 0
	if tb.Den != 0 {
		usec = int(int64(tb.Num) * 1000000 / int64(tb.Den))
	}
	binary.LittleEndian.PutUint32(avih[0:], uint32(usec))
	binary.LittleEndian.PutUint32(avih[12:], 0x110)
	binary.LittleEndian.PutUint32(avih[16:], uint32(videoFrames))
	binary.LittleEndian.PutUint32(avih[24:], uint32(len(streams)))
	binary.LittleEndian.PutUint32(avih[28:], uint32(maxChunk))
	binary.LittleEndian.PutUint32(avih[32:], uint32(width))
	binary.LittleEndian.PutUint32(avih[36:], uint32(height))
	hdr = append(hdr, chunk("avih", avih)...)
	for i, st := range streams {
		hdr = append(hdr, list("strl", aviStream(i, st, pkts))...)
	}
	for _, pkt := range pkts {
		if pkt.Stream < 0 || pkt.Stream >= len(streams) {
			continue
		}
		id := aviChunkID(pkt.Stream, streams[pkt.Stream].Par)
		index = append(index, idx{id: id, offset: len(movi), size: len(pkt.Data)})
		movi = append(movi, chunk(id, pkt.Data)...)
	}
	hdrList := list("hdrl", hdr)
	moviList := list("movi", movi)
	idxBody := make([]byte, 16*len(index))
	moviFour := 12 + len(hdrList) + 8
	for i, item := range index {
		off := 16 * i
		copy(idxBody[off:], item.id)
		binary.LittleEndian.PutUint32(idxBody[off+4:], 0x10)
		binary.LittleEndian.PutUint32(idxBody[off+8:], uint32(moviFour+4+item.offset-(12+len(hdrList)+8)))
		binary.LittleEndian.PutUint32(idxBody[off+12:], uint32(item.size))
	}
	body := append(hdrList, moviList...)
	body = append(body, chunk("idx1", idxBody)...)
	out := make([]byte, 12+len(body))
	copy(out, "RIFF")
	binary.LittleEndian.PutUint32(out[4:], uint32(4+len(body)))
	copy(out[8:], "AVI ")
	copy(out[12:], body)
	_ = moviFour
	return out, nil
}

func aviStream(index int, st Stream, pkts []*avutil.Packet) []byte {
	strh := make([]byte, 56)
	var n int
	var bytes int
	for _, pkt := range pkts {
		if pkt.Stream == index {
			n++
			bytes += len(pkt.Data)
		}
	}
	if st.Par.Type == avutil.MediaAudio {
		copy(strh[0:], "auds")
		binary.LittleEndian.PutUint32(strh[20:], 1)
		rate := uint32(st.Par.SampleRate)
		if rate == 0 {
			rate = 44100
		}
		binary.LittleEndian.PutUint32(strh[24:], rate)
		samples := 0
		for _, pkt := range pkts {
			if pkt.Stream == index {
				samples += int(pkt.Duration)
			}
		}
		if samples == 0 {
			width := encodedWidth(st.Par)
			if width > 0 {
				samples = bytes / width
			}
		}
		binary.LittleEndian.PutUint32(strh[32:], uint32(samples))
		binary.LittleEndian.PutUint32(strh[40:], 0xffffffff)
		binary.LittleEndian.PutUint32(strh[44:], uint32(encodedWidth(st.Par)))
		strf := make([]byte, 16)
		binary.LittleEndian.PutUint16(strf[0:], wavTag(st.Par.ID))
		ch := uint16(st.Par.Layout.Channels())
		if ch == 0 {
			ch = 1
		}
		binary.LittleEndian.PutUint16(strf[2:], ch)
		binary.LittleEndian.PutUint32(strf[4:], rate)
		block := uint16(encodedWidth(st.Par))
		binary.LittleEndian.PutUint32(strf[8:], rate*uint32(block))
		binary.LittleEndian.PutUint16(strf[12:], block)
		binary.LittleEndian.PutUint16(strf[14:], wavBits(st.Par.ID))
		return append(chunk("strh", strh), chunk("strf", strf)...)
	}
	copy(strh[0:], "vids")
	tag := aviHandler(st.Par)
	copy(strh[4:], tag)
	tb := st.TimeBase
	if tb.Den == 0 {
		tb = avutil.Q(1, 25)
	}
	binary.LittleEndian.PutUint32(strh[20:], uint32(tb.Num))
	binary.LittleEndian.PutUint32(strh[24:], uint32(tb.Den))
	binary.LittleEndian.PutUint32(strh[32:], uint32(n))
	binary.LittleEndian.PutUint32(strh[40:], 0xffffffff)
	binary.LittleEndian.PutUint16(strh[52:], uint16(st.Par.Width))
	binary.LittleEndian.PutUint16(strh[54:], uint16(st.Par.Height))
	strf := make([]byte, 40)
	binary.LittleEndian.PutUint32(strf[0:], 40)
	binary.LittleEndian.PutUint32(strf[4:], uint32(st.Par.Width))
	binary.LittleEndian.PutUint32(strf[8:], uint32(st.Par.Height))
	binary.LittleEndian.PutUint16(strf[12:], 1)
	binary.LittleEndian.PutUint16(strf[14:], 24)
	binary.LittleEndian.PutUint32(strf[16:], fourcc(tag))
	return append(chunk("strh", strh), chunk("strf", strf)...)
}

func aviHandler(par avcodec.Parameters) string {
	switch par.ID {
	case avcodec.IDMJPEG:
		return "MJPG"
	case avcodec.IDPNG:
		return "PNG "
	default:
		if par.Format == avutil.PixYUV420P || par.Format == 0 {
			return "I420"
		}
		return "RGB "
	}
}

func aviChunkID(index int, par avcodec.Parameters) string {
	kind := "wb"
	if par.Type == avutil.MediaVideo {
		if par.ID == avcodec.IDRawVideo {
			kind = "db"
		} else {
			kind = "dc"
		}
	}
	return string([]byte{'0' + byte(index/10), '0' + byte(index%10)}) + kind
}

func wavTag(id avcodec.ID) uint16 {
	switch id {
	case avcodec.IDPCMU8, avcodec.IDPCMS16LE, avcodec.IDPCMS24LE, avcodec.IDPCMS32LE:
		return 1
	case avcodec.IDPCMF32LE:
		return 3
	case avcodec.IDPCMALAW:
		return 6
	case avcodec.IDPCMMULAW:
		return 7
	default:
		return 1
	}
}

func wavBits(id avcodec.ID) uint16 {
	switch id {
	case avcodec.IDPCMU8, avcodec.IDPCMALAW, avcodec.IDPCMMULAW:
		return 8
	case avcodec.IDPCMS24LE:
		return 24
	case avcodec.IDPCMS32LE, avcodec.IDPCMF32LE:
		return 32
	default:
		return 16
	}
}

func encodedWidth(par avcodec.Parameters) int {
	ch := par.Layout.Channels()
	if ch <= 0 {
		ch = 1
	}
	switch par.ID {
	case avcodec.IDPCMU8, avcodec.IDPCMALAW, avcodec.IDPCMMULAW:
		return ch
	case avcodec.IDPCMS24LE:
		return 3 * ch
	case avcodec.IDPCMS32LE, avcodec.IDPCMF32LE:
		return 4 * ch
	default:
		return 2 * ch
	}
}

func chunk(id string, data []byte) []byte {
	n := len(data)
	out := make([]byte, 8+n+n%2)
	copy(out, id)
	binary.LittleEndian.PutUint32(out[4:], uint32(n))
	copy(out[8:], data)
	return out
}

func list(typ string, data []byte) []byte {
	out := make([]byte, 12+len(data))
	copy(out, "LIST")
	binary.LittleEndian.PutUint32(out[4:], uint32(4+len(data)))
	copy(out[8:], typ)
	copy(out[12:], data)
	return out
}

func fourcc(s string) uint32 {
	for len(s) < 4 {
		s += " "
	}
	return uint32(s[0]) | uint32(s[1])<<8 | uint32(s[2])<<16 | uint32(s[3])<<24
}

func openAVI(b []byte) (Demuxer, error) {
	if len(b) < 12 || string(b[0:4]) != "RIFF" {
		return nil, fmt.Errorf("%w: avi", avutil.ErrInvalidData)
	}
	streams, err := aviHeaders(b)
	if err != nil {
		return nil, err
	}
	var pkts []*avutil.Packet
	pos := 12
	for pos+8 <= len(b) {
		id := string(b[pos : pos+4])
		sz := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
		if sz < 0 || pos+8+sz > len(b) {
			break
		}
		payload := b[pos+8 : pos+8+sz]
		if id == "LIST" && len(payload) >= 4 && string(payload[:4]) == "movi" {
			pkts = aviPackets(payload[4:], streams)
			break
		}
		pos += 8 + sz
		if sz%2 == 1 {
			pos++
		}
	}
	if len(streams) == 0 {
		return nil, fmt.Errorf("%w: avi streams", avutil.ErrInvalidData)
	}
	return &memDemux{streams: streams, pkts: pkts}, nil
}

func aviHeaders(b []byte) ([]Stream, error) {
	var streams []Stream
	pos := 12
	for pos+8 <= len(b) {
		id := string(b[pos : pos+4])
		sz := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
		if sz < 0 || pos+8+sz > len(b) {
			break
		}
		if id == "LIST" && sz >= 4 && string(b[pos+8:pos+12]) == "hdrl" {
			streams = aviStrl(b[pos+12 : pos+8+sz])
			break
		}
		pos += 8 + sz
		if sz%2 == 1 {
			pos++
		}
	}
	return streams, nil
}

func aviStrl(b []byte) []Stream {
	var streams []Stream
	for pos := 0; pos+8 <= len(b); {
		id := string(b[pos : pos+4])
		sz := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
		if sz < 0 || pos+8+sz > len(b) {
			break
		}
		if id == "LIST" && sz >= 4 && string(b[pos+8:pos+12]) == "strl" {
			streams = append(streams, aviOne(b[pos+12:pos+8+sz]))
		}
		pos += 8 + sz
		if sz%2 == 1 && pos < len(b) {
			pos++
		}
	}
	return streams
}

func aviOne(b []byte) Stream {
	st := Stream{TimeBase: avutil.Q(1, 25)}
	var strh, strf []byte
	for pos := 0; pos+8 <= len(b); {
		id := string(b[pos : pos+4])
		sz := int(binary.LittleEndian.Uint32(b[pos+4 : pos+8]))
		if sz < 0 || pos+8+sz > len(b) {
			break
		}
		switch id {
		case "strh":
			strh = b[pos+8 : pos+8+sz]
		case "strf":
			strf = b[pos+8 : pos+8+sz]
		}
		pos += 8 + sz
		if sz%2 == 1 {
			pos++
		}
	}
	if len(strh) >= 4 && string(strh[:4]) == "auds" {
		st.Par.Type = avutil.MediaAudio
		st.Par.ID = avcodec.IDPCMS16LE
		st.Par.SampleFormat = avutil.SampleS16
		st.Par.Layout = avutil.Mono()
		if len(strf) >= 16 {
			_, id := wavCodec(binary.LittleEndian.Uint16(strf[0:2]), int(binary.LittleEndian.Uint16(strf[14:16])))
			if id != 0 {
				st.Par.ID = id
				st.Par.SampleFormat = sampleFormat(id)
			}
			ch := int(binary.LittleEndian.Uint16(strf[2:4]))
			st.Par.Layout = layoutOr(avutil.ChannelLayout{}, ch)
			st.Par.SampleRate = int(binary.LittleEndian.Uint32(strf[4:8]))
		}
		if st.Par.SampleRate <= 0 {
			st.Par.SampleRate = 44100
		}
		st.TimeBase = avutil.Q(1, int32(st.Par.SampleRate))
		st.Par.TimeBase = st.TimeBase
		return st
	}
	st.Par.Type = avutil.MediaVideo
	st.Par.ID = avcodec.IDMJPEG
	st.Par.Format = avutil.PixYUVJ420P
	if len(strh) >= 8 {
		switch string(strh[4:8]) {
		case "MJPG":
			st.Par.ID = avcodec.IDMJPEG
		case "I420":
			st.Par.ID = avcodec.IDRawVideo
			st.Par.Format = avutil.PixYUV420P
		case "RGB ":
			st.Par.ID = avcodec.IDRawVideo
			st.Par.Format = avutil.PixRGB24
		}
	}
	if len(strh) >= 28 {
		scale := binary.LittleEndian.Uint32(strh[20:24])
		rate := binary.LittleEndian.Uint32(strh[24:28])
		if scale > 0 && rate > 0 {
			st.TimeBase = avutil.Q(int32(scale), int32(rate))
		}
	}
	if len(strf) >= 20 {
		st.Par.Width = int(int32(binary.LittleEndian.Uint32(strf[4:8])))
		h := int32(binary.LittleEndian.Uint32(strf[8:12]))
		if h < 0 {
			h = -h
		}
		st.Par.Height = int(h)
		comp := binary.LittleEndian.Uint32(strf[16:20])
		if comp == fourcc("MJPG") {
			st.Par.ID = avcodec.IDMJPEG
		}
	}
	st.Par.TimeBase = st.TimeBase
	st.SAR = avutil.Q(1, 1)
	return st
}

func aviPackets(movi []byte, streams []Stream) []*avutil.Packet {
	var pkts []*avutil.Packet
	counts := map[int]int64{}
	for pos := 0; pos+8 <= len(movi); {
		id := string(movi[pos : pos+4])
		sz := int(binary.LittleEndian.Uint32(movi[pos+4 : pos+8]))
		if sz < 0 || pos+8+sz > len(movi) {
			break
		}
		if len(id) == 4 && id[0] >= '0' && id[0] <= '9' {
			index := int(id[0]-'0')*10 + int(id[1]-'0')
			if index >= 0 && index < len(streams) {
				pkt := &avutil.Packet{
					Data: append([]byte(nil), movi[pos+8:pos+8+sz]...), Stream: index,
					PTS: counts[index], DTS: counts[index], Duration: 1,
					TimeBase: streams[index].TimeBase, Key: true,
				}
				if streams[index].Par.Type == avutil.MediaAudio {
					width := encodedWidth(streams[index].Par)
					if width > 0 {
						pkt.Duration = int64(sz / width)
					}
					pkt.PTS = counts[index]
					counts[index] += pkt.Duration
				} else {
					counts[index]++
				}
				pkts = append(pkts, pkt)
			}
		}
		pos += 8 + sz
		if sz%2 == 1 {
			pos++
		}
	}
	return pkts
}
