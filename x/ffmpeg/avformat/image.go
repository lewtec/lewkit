package avformat

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

func openImage2(ctx context.Context, url string, opt Options) (Demuxer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !strings.Contains(url, "%d") {
		b, err := os.ReadFile(url)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", avutil.ErrInvalidData, err)
		}
		return imagePackets(ImageCodec(url), [][]byte{b})
	}
	var blobs [][]byte
	for n := 1; n < 100000; n++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		path := strings.Replace(url, "%d", strconv.Itoa(n), 1)
		b, err := os.ReadFile(path)
		if err != nil {
			break
		}
		blobs = append(blobs, b)
	}
	if len(blobs) == 0 {
		return nil, fmt.Errorf("%w: %s", avutil.ErrInvalidData, url)
	}
	return imagePackets(ImageCodec(url), blobs)
}

func openImagePipe(data []byte) (Demuxer, error) {
	id := sniffImage(data)
	return imagePackets(id, [][]byte{data})
}

func sniffImage(b []byte) avcodec.ID {
	if len(b) >= 8 && string(b[:8]) == "\x89PNG\r\n\x1a\n" {
		return avcodec.IDPNG
	}
	if len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 {
		return avcodec.IDMJPEG
	}
	if len(b) >= 3 && string(b[:3]) == "GIF" {
		return avcodec.IDGIF
	}
	if len(b) >= 2 && string(b[:2]) == "BM" {
		return avcodec.IDBMP
	}
	if len(b) >= 2 && b[0] == 'P' && (b[1] == '6' || b[1] == '3') {
		return avcodec.IDPPM
	}
	return avcodec.IDPNG
}

func imagePackets(id avcodec.ID, blobs [][]byte) (Demuxer, error) {
	tb := avutil.Q(1, 25)
	var pkts []*avutil.Packet
	for i, b := range blobs {
		pkts = append(pkts, &avutil.Packet{
			Data: b, PTS: int64(i), DTS: int64(i), Duration: 1, TimeBase: tb, Key: true,
		})
	}
	return &memDemux{streams: []Stream{{
		Par:      avcodec.Parameters{Type: avutil.MediaVideo, ID: id, TimeBase: tb},
		TimeBase: tb,
	}}, pkts: pkts}, nil
}

type imageMux struct {
	ctx    context.Context
	url    string
	format string
	opt    Options
	n      int
}

func (m *imageMux) AddStream(avcodec.Parameters) (int, error) { return 0, nil }
func (m *imageMux) WriteHeader() error                        { return nil }

func (m *imageMux) WritePacket(pkt *avutil.Packet) error {
	if err := m.ctx.Err(); err != nil {
		return err
	}
	if pkt == nil {
		return nil
	}
	m.n++
	if m.url == "-" || m.format == "image2pipe" {
		w := m.opt.Stdout
		if w == nil {
			return fmt.Errorf("%w: stdout", avutil.ErrInvalidData)
		}
		_, err := w.Write(pkt.Data)
		if err != nil {
			return fmt.Errorf("%w: %s", avutil.ErrInvalidData, err)
		}
		return nil
	}
	path := m.url
	if strings.Contains(path, "%d") {
		path = strings.Replace(path, "%d", strconv.Itoa(m.n), 1)
	}
	if err := os.WriteFile(path, pkt.Data, 0o644); err != nil {
		return fmt.Errorf("%w: %s", avutil.ErrInvalidData, err)
	}
	return nil
}

func (m *imageMux) Close() error { return nil }
