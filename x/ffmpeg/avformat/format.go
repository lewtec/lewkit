package avformat

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// Stream is one elementary stream in a container.
type Stream struct {
	Index    int
	Par      avcodec.Parameters
	TimeBase avutil.Rational
	SAR      avutil.Rational
}

// Demuxer reads packets until io.EOF.
type Demuxer interface {
	Streams() []Stream
	ReadPacket() (*avutil.Packet, error)
	Close() error
}

// Muxer writes an output container.
type Muxer interface {
	AddStream(par avcodec.Parameters) (int, error)
	WriteHeader() error
	WritePacket(pkt *avutil.Packet) error
	Close() error
}

// Options carry the values -s, -pix_fmt, -r, -ar, and -ac set before the file.
type Options struct {
	Width, Height int
	Format        avutil.PixelFormat
	FrameRate     avutil.Rational
	SampleRate    int
	Layout        avutil.ChannelLayout
	SampleFormat  avutil.SampleFormat
	Bitexact      bool
	Stdin         io.Reader
	Stdout        io.Writer
}

// Info is one container.
type Info struct {
	Name  string
	Long  string
	Demux bool
	Mux   bool
}

// Formats lists containers this build can open or write.
func Formats() []Info {
	return []Info{
		{Name: "wav", Long: "WAV / WAVE", Demux: true, Mux: true},
		{Name: "flac", Long: "raw FLAC", Demux: true, Mux: true},
		{Name: "mp3", Long: "MP3", Demux: true, Mux: true},
		{Name: "ogg", Long: "Ogg Vorbis", Demux: true, Mux: false},
		{Name: "image2", Long: "image2 sequence", Demux: true, Mux: true},
		{Name: "image2pipe", Long: "piped image2 sequence", Demux: true, Mux: true},
		{Name: "rawvideo", Long: "raw video", Demux: true, Mux: true},
		{Name: "u8", Long: "PCM unsigned 8-bit", Demux: true, Mux: true},
		{Name: "s16le", Long: "PCM signed 16-bit little-endian", Demux: true, Mux: true},
		{Name: "s16be", Long: "PCM signed 16-bit big-endian", Demux: true, Mux: true},
		{Name: "s24le", Long: "PCM signed 24-bit little-endian", Demux: true, Mux: true},
		{Name: "s32le", Long: "PCM signed 32-bit little-endian", Demux: true, Mux: true},
		{Name: "f32le", Long: "PCM 32-bit float little-endian", Demux: true, Mux: true},
		{Name: "alaw", Long: "PCM A-law", Demux: true, Mux: true},
		{Name: "mulaw", Long: "PCM mu-law", Demux: true, Mux: true},
		{Name: "avi", Long: "AVI (MJPEG and PCM)", Demux: true, Mux: true},
		{Name: "mp4", Long: "MP4 (MJPEG and PCM)", Demux: true, Mux: true},
		{Name: "mov", Long: "QuickTime (MJPEG and PCM)", Demux: true, Mux: true},
		{Name: "lavfi", Long: "Libavfilter virtual input", Demux: true, Mux: false},
		{Name: "null", Long: "raw null", Demux: false, Mux: true},
		{Name: "crc", Long: "CRC testing", Demux: false, Mux: true},
		{Name: "framecrc", Long: "framecrc testing", Demux: false, Mux: true},
		{Name: "md5", Long: "MD5 testing", Demux: false, Mux: true},
		{Name: "framemd5", Long: "frame MD5 testing", Demux: false, Mux: true},
		{Name: "h264", Long: "raw H.264 annex B", Demux: true, Mux: true},
		{Name: "hevc", Long: "raw HEVC annex B", Demux: true, Mux: true},
		{Name: "gsm", Long: "raw GSM 06.10", Demux: true, Mux: true},
		{Name: "g722", Long: "raw G.722", Demux: true, Mux: true},
		{Name: "g726", Long: "raw G.726", Demux: true, Mux: true},
		{Name: "g726le", Long: "raw little-endian G.726", Demux: true, Mux: true},
		{Name: "mp2", Long: "MPEG audio layer 2", Demux: false, Mux: true},
		{Name: "aiff", Long: "Audio IFF", Demux: true, Mux: true},
	}
}

// Guess picks a format name from a path extension.
func Guess(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".wav":
		return "wav"
	case ".flac":
		return "flac"
	case ".mp3":
		return "mp3"
	case ".ogg", ".oga":
		return "ogg"
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ppm":
		return "image2"
	case ".avi":
		return "avi"
	case ".mp4", ".m4v", ".mov", ".m4a":
		return "mp4"
	case ".yuv", ".rgb":
		return "rawvideo"
	case ".h264", ".264":
		return "h264"
	case ".hevc", ".h265", ".265":
		return "hevc"
	case ".gsm":
		return "gsm"
	case ".g722":
		return "g722"
	case ".g726":
		return "g726"
	case ".g726le":
		return "g726le"
	case ".mp2":
		return "mp2"
	case ".aiff", ".aif":
		return "aiff"
	case ".pcm":
		return "s16le"
	default:
		return ""
	}
}

// ImageCodec is the codec an image path extension selects.
func ImageCodec(path string) avcodec.ID {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return avcodec.IDMJPEG
	case ".gif":
		return avcodec.IDGIF
	case ".bmp":
		return avcodec.IDBMP
	case ".ppm":
		return avcodec.IDPPM
	default:
		return avcodec.IDPNG
	}
}

// OpenInput opens a demuxer. format overrides the extension.
func OpenInput(ctx context.Context, url, format string, opt Options) (Demuxer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if format == "" {
		format = Guess(url)
	}
	if format == "" {
		return nil, fmt.Errorf("%w: format for %s", avutil.ErrOption, url)
	}
	switch format {
	case "lavfi":
		return openLavfi(ctx, url, opt)
	case "image2":
		return openImage2(ctx, url, opt)
	case "wav", "flac", "mp3", "ogg", "avi", "mp4", "mov", "image2pipe", "rawvideo", "h264", "hevc", "gsm", "g722", "g726", "g726le", "aiff", "u8", "s16le", "s16be", "s24le", "s32le", "f32le", "alaw", "mulaw":
		rc, err := openRead(ctx, url, opt)
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		if closeErr := rc.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return nil, err
		}
		return openBytes(format, data, opt)
	default:
		return nil, fmt.Errorf("%w: demuxer %s", avutil.ErrUnsupported, format)
	}
}

// Create opens a muxer. An existing file is replaced by the caller.
func Create(ctx context.Context, url, format string, opt Options) (Muxer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if format == "" {
		format = Guess(url)
	}
	if format == "" {
		return nil, fmt.Errorf("%w: format for %s", avutil.ErrOption, url)
	}
	switch format {
	case "null":
		return &nullMux{}, nil
	case "crc", "framecrc", "md5", "framemd5":
		w, err := openWrite(ctx, url, opt)
		if err != nil {
			return nil, err
		}
		return &hashMux{name: format, w: w, opt: opt}, nil
	case "image2", "image2pipe":
		return &imageMux{ctx: ctx, url: url, format: format, opt: opt}, nil
	case "rawvideo", "h264", "hevc", "gsm", "g722", "g726", "g726le", "mp2", "mp3", "aiff", "u8", "s16le", "s16be", "s24le", "s32le", "f32le", "alaw", "mulaw", "flac", "wav", "avi", "mp4", "mov":
		w, err := openWrite(ctx, url, opt)
		if err != nil {
			return nil, err
		}
		return &bufMux{name: format, w: w, opt: opt}, nil
	default:
		return nil, fmt.Errorf("%w: muxer %s", avutil.ErrUnsupported, format)
	}
}

func openRead(ctx context.Context, url string, opt Options) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if url == "-" {
		if opt.Stdin == nil {
			return nil, fmt.Errorf("%w: stdin", avutil.ErrInvalidData)
		}
		return io.NopCloser(opt.Stdin), nil
	}
	f, err := os.Open(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", avutil.ErrInvalidData, err)
	}
	return f, nil
}

func openWrite(ctx context.Context, url string, opt Options) (io.WriteCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if url == "-" {
		if opt.Stdout == nil {
			return nil, fmt.Errorf("%w: stdout", avutil.ErrInvalidData)
		}
		return nopWrite{opt.Stdout}, nil
	}
	f, err := os.Create(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", avutil.ErrInvalidData, err)
	}
	return f, nil
}

type nopWrite struct{ io.Writer }

func (nopWrite) Close() error { return nil }

func openBytes(format string, data []byte, opt Options) (Demuxer, error) {
	switch format {
	case "wav":
		return openWAV(data)
	case "flac":
		return openBlob(avcodec.IDFLAC, data)
	case "mp3":
		return openBlob(avcodec.IDMP3, data)
	case "ogg":
		return openBlob(avcodec.IDVorbis, data)
	case "avi":
		return openAVI(data)
	case "mp4", "mov":
		return openMOV(data)
	case "image2pipe":
		return openImagePipe(data)
	case "rawvideo":
		return openRawVideo(data, opt)
	case "h264":
		return openH264(data)
	case "hevc":
		return openHEVC(data)
	case "gsm":
		return openGSM(data)
	case "g722":
		return openG722(data)
	case "g726":
		return openG726(data)
	case "g726le":
		return openG726LE(data)
	case "aiff":
		return openAIFF(data)
	default:
		return openRawAudio(format, data, opt)
	}
}

func rateOr(r avutil.Rational) avutil.Rational {
	if r.Num <= 0 || r.Den <= 0 {
		return avutil.Q(25, 1)
	}
	return r
}

func layoutOr(layout avutil.ChannelLayout, channels int) avutil.ChannelLayout {
	if layout.Channels() > 0 {
		return layout
	}
	if channels <= 1 {
		return avutil.Mono()
	}
	if channels == 2 {
		return avutil.Stereo()
	}
	var mask uint64
	for i := 0; i < channels; i++ {
		mask |= 1 << i
	}
	return avutil.ChannelLayout{Mask: mask, Count: channels}
}
