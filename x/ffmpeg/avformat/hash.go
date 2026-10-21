package avformat

import (
	"crypto/md5"
	"fmt"
	"io"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

type nullMux struct{}

func (nullMux) AddStream(avcodec.Parameters) (int, error) { return 0, nil }
func (nullMux) WriteHeader() error                        { return nil }
func (nullMux) WritePacket(*avutil.Packet) error          { return nil }
func (nullMux) Close() error                              { return nil }

type hashMux struct {
	name    string
	w       io.WriteCloser
	opt     Options
	streams []avcodec.Parameters
	body    []byte
	header  bool
}

func (m *hashMux) AddStream(par avcodec.Parameters) (int, error) {
	m.streams = append(m.streams, par)
	return len(m.streams) - 1, nil
}

func (m *hashMux) WriteHeader() error {
	if m.name != "framecrc" && m.name != "framemd5" {
		return nil
	}
	if !m.opt.Bitexact {
		if _, err := fmt.Fprintf(m.w, "#software: n9.0.2-lewkit\n"); err != nil {
			return err
		}
	}
	for i, par := range m.streams {
		tb := par.TimeBase
		if tb.Den == 0 {
			tb = avutil.Q(1, 25)
		}
		kind := "video"
		if par.Type == avutil.MediaAudio {
			kind = "audio"
		}
		if _, err := fmt.Fprintf(m.w, "#tb %d: %d/%d\n", i, tb.Num, tb.Den); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(m.w, "#media_type %d: %s\n", i, kind); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(m.w, "#codec_id %d: %s\n", i, par.ID.String()); err != nil {
			return err
		}
		if par.Type == avutil.MediaAudio {
			if _, err := fmt.Fprintf(m.w, "#sample_rate %d: %d\n", i, par.SampleRate); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(m.w, "#channel_layout_name %d: %s\n", i, avutil.LayoutString(par.Layout)); err != nil {
				return err
			}
		} else {
			sar := par.SAR
			if sar.Den == 0 {
				sar = avutil.Q(1, 1)
			}
			if _, err := fmt.Fprintf(m.w, "#dimensions %d: %dx%d\n", i, par.Width, par.Height); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(m.w, "#sar %d: %d/%d\n", i, sar.Num, sar.Den); err != nil {
				return err
			}
		}
	}
	m.header = true
	return nil
}

func (m *hashMux) WritePacket(pkt *avutil.Packet) error {
	if pkt == nil {
		return nil
	}
	switch m.name {
	case "crc", "md5":
		m.body = append(m.body, pkt.Data...)
		return nil
	case "framecrc":
		_, err := fmt.Fprintf(m.w, "%d, %10d, %10d, %8d, %8d, 0x%08x\n",
			pkt.Stream, pkt.DTS, pkt.PTS, pkt.Duration, len(pkt.Data), avutil.Adler32(pkt.Data))
		return err
	case "framemd5":
		sum := md5.Sum(pkt.Data)
		_, err := fmt.Fprintf(m.w, "%d, %10d, %10d, %8d, %8d, %x\n",
			pkt.Stream, pkt.DTS, pkt.PTS, pkt.Duration, len(pkt.Data), sum)
		return err
	default:
		return fmt.Errorf("%w: %s", avutil.ErrUnsupported, m.name)
	}
}

func (m *hashMux) Close() error {
	var err error
	switch m.name {
	case "crc":
		_, err = fmt.Fprintf(m.w, "CRC=0x%08x\n", avutil.Adler32(m.body))
	case "md5":
		_, err = fmt.Fprintf(m.w, "MD5=%x\n", md5.Sum(m.body))
	}
	if closeErr := m.w.Close(); err == nil {
		err = closeErr
	}
	return err
}
