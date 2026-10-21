package avcodec

import (
	"fmt"
	"io"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

// Parameters describe one elementary stream.
type Parameters struct {
	Type         avutil.MediaType
	ID           ID
	Width        int
	Height       int
	Format       avutil.PixelFormat
	SampleFormat avutil.SampleFormat
	SampleRate   int
	Layout       avutil.ChannelLayout
	BitRate      int
	QScale       int
	Extra        []byte
	TimeBase     avutil.Rational
	SAR          avutil.Rational
}

// Codec is one implemented encoder or decoder.
type Codec struct {
	ID       ID
	Name     string
	Long     string
	Type     avutil.MediaType
	Decode   bool
	Encode   bool
	Intra    bool
	Lossy    bool
	Lossless bool
}

// Decoder turns packets into frames.
type Decoder interface {
	Decode(pkt *avutil.Packet) ([]*avutil.Frame, error)
	io.Closer
}

// Encoder turns frames into packets.
type Encoder interface {
	Encode(frame *avutil.Frame) ([]*avutil.Packet, error)
	Flush() ([]*avutil.Packet, error)
	io.Closer
}

// Codecs lists what this build can run. The order is the -codecs order.
func Codecs() []Codec {
	return []Codec{
		{IDRawVideo, "rawvideo", "raw video", avutil.MediaVideo, true, true, true, false, true},
		{IDPNG, "png", "PNG image", avutil.MediaVideo, true, true, true, false, true},
		{IDMJPEG, "mjpeg", "Motion JPEG", avutil.MediaVideo, true, true, true, true, false},
		{IDGIF, "gif", "GIF image", avutil.MediaVideo, true, true, true, true, false},
		{IDBMP, "bmp", "BMP image", avutil.MediaVideo, true, true, true, false, true},
		{IDPPM, "ppm", "PPM image", avutil.MediaVideo, true, true, true, false, true},
		{IDPCMU8, "pcm_u8", "PCM unsigned 8-bit", avutil.MediaAudio, true, true, true, false, true},
		{IDPCMS16LE, "pcm_s16le", "PCM signed 16-bit little-endian", avutil.MediaAudio, true, true, true, false, true},
		{IDPCMS16BE, "pcm_s16be", "PCM signed 16-bit big-endian", avutil.MediaAudio, true, true, true, false, true},
		{IDPCMS24LE, "pcm_s24le", "PCM signed 24-bit little-endian", avutil.MediaAudio, true, true, true, false, true},
		{IDPCMS32LE, "pcm_s32le", "PCM signed 32-bit little-endian", avutil.MediaAudio, true, true, true, false, true},
		{IDPCMF32LE, "pcm_f32le", "PCM 32-bit float little-endian", avutil.MediaAudio, true, true, true, false, true},
		{IDPCMALAW, "pcm_alaw", "PCM A-law", avutil.MediaAudio, true, true, true, true, false},
		{IDPCMMULAW, "pcm_mulaw", "PCM mu-law", avutil.MediaAudio, true, true, true, true, false},
		{IDFLAC, "flac", "FLAC", avutil.MediaAudio, true, true, false, false, true},
		{IDMP3, "mp3", "MP3", avutil.MediaAudio, true, false, false, true, false},
		{IDVorbis, "vorbis", "Vorbis", avutil.MediaAudio, true, false, false, true, false},
		{IDGSM, "gsm", "GSM 06.10", avutil.MediaAudio, true, true, false, true, false},
		{IDGSMMS, "gsm_ms", "GSM 06.10 Microsoft", avutil.MediaAudio, false, true, false, true, false},
		{IDMP2, "mp2fixed", "MP2 fixed-point", avutil.MediaAudio, false, true, false, true, false},
		{IDADPCMIMAWAV, "adpcm_ima_wav", "ADPCM IMA WAV", avutil.MediaAudio, true, true, false, true, false},
		{IDADPCMMS, "adpcm_ms", "ADPCM Microsoft", avutil.MediaAudio, true, true, false, true, false},
		{IDADPCMYAMAHA, "adpcm_yamaha", "ADPCM Yamaha", avutil.MediaAudio, true, true, false, true, false},
		{IDADPCMIMAQT, "adpcm_ima_qt", "ADPCM IMA QuickTime", avutil.MediaAudio, true, true, false, true, false},
		{IDG722, "g722", "G.722 ADPCM", avutil.MediaAudio, true, true, false, true, false},
		{IDLIBSHINE, "libshine", "shine MP3", avutil.MediaAudio, false, true, false, true, false},
		{IDG726, "g726", "G.726 ADPCM", avutil.MediaAudio, true, true, false, true, false},
		{IDG726LE, "g726le", "G.726 ADPCM little-endian", avutil.MediaAudio, true, true, false, true, false},
		{IDADPCMSWF, "adpcm_swf", "ADPCM Shockwave Flash", avutil.MediaAudio, false, true, false, true, false},
		{IDH264, "h264", "H.264 constrained baseline", avutil.MediaVideo, true, true, true, false, true},
		{IDHEVC, "hevc", "HEVC main PCM", avutil.MediaVideo, true, true, true, false, true},
	}
}

// FindByName resolves a codec name such as pcm_s16le or png.
// libx264, libx265, libgsm, and libgsm_ms are the in-tree stand-ins for those external encoders.
func FindByName(name string) (Codec, bool) {
	switch name {
	case "libx264", "libx264rgb":
		name = "h264"
	case "libx265":
		name = "hevc"
	case "libgsm":
		name = "gsm"
	case "libgsm_ms":
		name = "gsm_ms"
	}
	if id, err := ParseID(name); err == nil && id != IDNone {
		for _, item := range Codecs() {
			if item.ID == id {
				return item, true
			}
		}
	}
	for _, item := range Codecs() {
		if item.Name == name {
			return item, true
		}
	}
	return Codec{}, false
}

// OpenDecoder opens an implemented decoder.
func OpenDecoder(par Parameters) (Decoder, error) {
	switch par.ID {
	case IDPCMU8, IDPCMS16LE, IDPCMS16BE, IDPCMS24LE, IDPCMS32LE, IDPCMF32LE:
		return &pcmCodec{par: par}, nil
	case IDPCMALAW, IDPCMMULAW:
		return &xlawCodec{par: par}, nil
	case IDRawVideo:
		return &rawCodec{par: par}, nil
	case IDPNG, IDMJPEG, IDGIF, IDBMP, IDPPM:
		return &imageCodec{par: par, decode: true}, nil
	case IDFLAC:
		return &flacCodec{par: par}, nil
	case IDMP3:
		return &mp3Codec{}, nil
	case IDVorbis:
		return &vorbisCodec{}, nil
	case IDGSM:
		return newGSMCodec(par), nil
	case IDG722:
		return newG722Codec(par), nil
	case IDG726:
		return newG726(par), nil
	case IDG726LE:
		return newG726LE(par), nil
	case IDADPCMIMAWAV:
		return newIMAWAV(par), nil
	case IDADPCMMS:
		return newMSADPCM(par), nil
	case IDADPCMYAMAHA:
		return newYamaha(par), nil
	case IDADPCMIMAQT:
		return newIMAQT(par), nil
	case IDH264:
		return newH264(par), nil
	case IDHEVC:
		return newHEVC(par), nil
	default:
		return nil, fmt.Errorf("%w: decoder %s", avutil.ErrUnsupported, par.ID)
	}
}

// OpenEncoder opens an implemented encoder.
func OpenEncoder(par Parameters) (Encoder, error) {
	switch par.ID {
	case IDPCMU8, IDPCMS16LE, IDPCMS16BE, IDPCMS24LE, IDPCMS32LE, IDPCMF32LE:
		return &pcmCodec{par: par}, nil
	case IDPCMALAW, IDPCMMULAW:
		return &xlawCodec{par: par}, nil
	case IDRawVideo:
		return &rawCodec{par: par}, nil
	case IDPNG, IDMJPEG, IDGIF, IDBMP, IDPPM:
		return &imageCodec{par: par}, nil
	case IDFLAC:
		return &flacCodec{par: par}, nil
	case IDGSM:
		return newGSMCodec(par), nil
	case IDGSMMS:
		return newGSMMS(par), nil
	case IDMP2:
		return newMP2(par), nil
	case IDADPCMIMAWAV:
		return newIMAWAV(par), nil
	case IDADPCMMS:
		return newMSADPCM(par), nil
	case IDADPCMYAMAHA:
		return newYamaha(par), nil
	case IDADPCMIMAQT:
		return newIMAQT(par), nil
	case IDG722:
		return newG722Codec(par), nil
	case IDLIBSHINE:
		return newShineCodec(par)
	case IDG726:
		return newG726(par), nil
	case IDG726LE:
		return newG726LE(par), nil
	case IDADPCMSWF:
		return newSWF(par)
	case IDH264:
		return newH264(par), nil
	case IDHEVC:
		return newHEVC(par), nil
	default:
		return nil, fmt.Errorf("%w: encoder %s", avutil.ErrUnsupported, par.ID)
	}
}

func (pcmCodec) Close() error    { return nil }
func (xlawCodec) Close() error   { return nil }
func (rawCodec) Close() error    { return nil }
func (imageCodec) Close() error  { return nil }
func (flacCodec) Close() error   { return nil }
func (mp3Codec) Close() error    { return nil }
func (vorbisCodec) Close() error { return nil }
func (gsmCodec) Close() error    { return nil }
func (mp2Codec) Close() error    { return nil }
func (imaCodec) Close() error    { return nil }
func (msCodec) Close() error     { return nil }
func (yamahaCodec) Close() error { return nil }
func (qtCodec) Close() error     { return nil }
func (g722Codec) Close() error   { return nil }
func (shineCodec) Close() error  { return nil }
func (g726Codec) Close() error   { return nil }
func (swfCodec) Close() error    { return nil }
func (h264Codec) Close() error   { return nil }
func (hevcCodec) Close() error   { return nil }

func (pcmCodec) Flush() ([]*avutil.Packet, error)   { return nil, nil }
func (xlawCodec) Flush() ([]*avutil.Packet, error)  { return nil, nil }
func (rawCodec) Flush() ([]*avutil.Packet, error)   { return nil, nil }
func (imageCodec) Flush() ([]*avutil.Packet, error) { return nil, nil }
