package avcodec

// ID is one codec.
//
//lewkit:enum fold
type ID int

const (
	IDNone        ID = iota // none
	IDPCMU8                 // pcm_u8
	IDPCMS16LE              // pcm_s16le
	IDPCMS16BE              // pcm_s16be
	IDPCMS24LE              // pcm_s24le
	IDPCMS32LE              // pcm_s32le
	IDPCMF32LE              // pcm_f32le
	IDPCMALAW               // pcm_alaw
	IDPCMMULAW              // pcm_mulaw
	IDRawVideo              // rawvideo
	IDPNG                   // png
	IDMJPEG                 // mjpeg
	IDGIF                   // gif
	IDBMP                   // bmp
	IDPPM                   // ppm
	IDFLAC                  // flac
	IDMP3                   // mp3
	IDVorbis                // vorbis
	IDGSM                   // gsm
	IDH264                  // h264
	IDHEVC                  // hevc
	IDMP2                   // mp2fixed
	IDGSMMS                 // gsm_ms
	IDADPCMIMAWAV           // adpcm_ima_wav
	IDADPCMMS               // adpcm_ms
	IDADPCMYAMAHA           // adpcm_yamaha
	IDADPCMIMAQT            // adpcm_ima_qt
	IDG722                  // g722
	IDLIBSHINE              // libshine
	IDG726                  // g726
	IDG726LE                // g726le
	IDADPCMSWF              // adpcm_swf
	IDLIBTWOLAME            // libtwolame
	IDADPCMIMAWS            // adpcm_ima_ws
	IDLIBSPEEX              // libspeex
	IDADPCMIMASSI           // adpcm_ima_ssi
	IDADPCMIMAALP           // adpcm_ima_alp
	IDADPCMIMAAPM           // adpcm_ima_apm
	IDADPCMARGO             // adpcm_argo
)
