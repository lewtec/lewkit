package avformat

import (
	"fmt"
	"io"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

type bufMux struct {
	name    string
	w       io.WriteCloser
	opt     Options
	url     string
	streams []Stream
	pkts    []*avutil.Packet
}

func (m *bufMux) AddStream(par avcodec.Parameters) (int, error) {
	if par.TimeBase.Den == 0 {
		if par.Type == avutil.MediaAudio && par.SampleRate > 0 {
			par.TimeBase = avutil.Q(1, int32(par.SampleRate))
		} else {
			par.TimeBase = rateOr(m.opt.FrameRate).Inv()
		}
	}
	m.streams = append(m.streams, Stream{Index: len(m.streams), Par: par, TimeBase: par.TimeBase, SAR: par.SAR})
	return len(m.streams) - 1, nil
}

func (m *bufMux) WriteHeader() error { return nil }

func (m *bufMux) WritePacket(pkt *avutil.Packet) error {
	if pkt == nil {
		return nil
	}
	cp := pkt.Clone()
	m.pkts = append(m.pkts, cp)
	return nil
}

func (m *bufMux) Close() error {
	var body []byte
	var err error
	switch m.name {
	case "wav":
		body, err = writeWAV(m.streams, m.pkts)
	case "aiff":
		body, err = writeAIFF(m.streams, m.pkts)
	case "wsaud":
		body, err = writeAUD(m.streams, m.pkts)
	case "kvag":
		body, err = writeKVAG(m.streams, m.pkts)
	case "alp":
		body, err = writeALP(m.url, m.streams, m.pkts)
	case "apm":
		body, err = writeAPM(m.streams, m.pkts)
	case "argo_asf":
		body, err = writeArgoASF(m.url, m.streams, m.pkts)
	case "codec2":
		body, err = writeCodec2(m.streams, m.pkts)
	case "avi":
		body, err = writeAVI(m.streams, m.pkts)
	case "mp4", "mov":
		body, err = writeMOV(m.streams, m.pkts)
	case "flac":
		body = writeBlob(m.pkts)
	default:
		if _, _, ok := pcmCodec(m.name); ok || m.name == "rawvideo" || m.name == "h264" || m.name == "hevc" || m.name == "gsm" || m.name == "g722" || m.name == "g726" || m.name == "g726le" || m.name == "mp2" || m.name == "mp3" || m.name == "codec2raw" {
			body = writeBlob(m.pkts)
		} else {
			err = fmt.Errorf("%w: muxer %s", avutil.ErrUnsupported, m.name)
		}
	}
	if err != nil {
		_ = m.w.Close()
		return err
	}
	if _, err = m.w.Write(body); err != nil {
		_ = m.w.Close()
		return fmt.Errorf("%w: %s", avutil.ErrInvalidData, err)
	}
	return m.w.Close()
}
