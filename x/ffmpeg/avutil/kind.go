package avutil

// MediaType is a elementary-stream kind.
//
//lewkit:enum fold
type MediaType uint8

const (
	// MediaVideo is a picture stream.
	MediaVideo MediaType = iota + 1 // video
	// MediaAudio is a sample stream.
	MediaAudio // audio
	// MediaData is a timed data stream.
	MediaData // data
	// MediaSubtitle is a subtitle stream.
	MediaSubtitle // subtitle
	// MediaAttachment is an attached font or image.
	MediaAttachment // attachment
)

// Level is an av_log level. The numbers are FFmpeg's.
//
//lewkit:enum fold
type Level int

const (
	// LevelQuiet prints nothing.
	LevelQuiet Level = -8 // quiet
	// LevelPanic is a crash.
	LevelPanic Level = 0 // panic
	// LevelFatal is a process-ending failure.
	LevelFatal Level = 8 // fatal
	// LevelError is a recoverable failure.
	LevelError Level = 16 // error
	// LevelWarning is a suspicious condition.
	LevelWarning Level = 24 // warning
	// LevelInfo is the default.
	LevelInfo Level = 32 // info
	// LevelVerbose is extra progress.
	LevelVerbose Level = 40 // verbose
	// LevelDebug is for developers.
	LevelDebug Level = 48 // debug
	// LevelTrace is every internal step.
	LevelTrace Level = 56 // trace
)

// PixelFormat is one picture layout.
//
//lewkit:enum fold
type PixelFormat uint16

const (
	PixYUV420P PixelFormat = iota + 1 // yuv420p
	PixYUV422P                        // yuv422p
	PixYUV444P                        // yuv444p
	PixYUV410P                        // yuv410p
	PixYUV411P                        // yuv411p
	PixYUVJ420P                       // yuvj420p
	PixYUVJ422P                       // yuvj422p
	PixYUVJ444P                       // yuvj444p
	PixYUYV422                        // yuyv422
	PixUYVY422                        // uyvy422
	PixNV12                           // nv12
	PixNV21                           // nv21
	PixRGB24                          // rgb24
	PixBGR24                          // bgr24
	PixRGBA                           // rgba
	PixBGRA                           // bgra
	PixARGB                           // argb
	PixABGR                           // abgr
	PixRGB0                           // rgb0
	PixBGR0                           // bgr0
	Pix0RGB                           // 0rgb
	Pix0BGR                           // 0bgr
	PixGray                           // gray
	PixGBRP                           // gbrp
	PixRGB565LE                       // rgb565le
	PixRGB555LE                       // rgb555le
	PixYA8                            // ya8
	PixYUVA420P                       // yuva420p
	PixPAL8                           // pal8
	PixMonoW                          // monow
	PixMonoB                          // monob
)

// SampleFormat is one PCM encoding.
//
//lewkit:enum fold
type SampleFormat uint8

const (
	SampleU8 SampleFormat = iota + 1 // u8
	SampleS16                        // s16
	SampleS32                        // s32
	SampleS64                        // s64
	SampleFLT                        // flt
	SampleDBL                        // dbl
	SampleU8P                        // u8p
	SampleS16P                       // s16p
	SampleS32P                       // s32p
	SampleS64P                       // s64p
	SampleFLTP                       // fltp
	SampleDBLP                       // dblp
)

// LayoutName is a channel preset.
//
//lewkit:enum fold
type LayoutName uint8

const (
	LayoutMono LayoutName = iota + 1 // mono
	LayoutStereo                     // stereo
	LayoutTwoOne                     // 2.1
	LayoutThree                      // 3.0
	LayoutThreeOne                   // 3.1
	LayoutQuad                       // quad, 4.0
	LayoutQuadSide                   // quad(side)
	LayoutFive                       // 5.0
	LayoutFiveSide                   // 5.0(side)
	LayoutFiveOne                    // 5.1
	LayoutFiveOneSide                // 5.1(side)
	LayoutSixOne                     // 6.1
	LayoutSevenOne                   // 7.1
	LayoutSevenOneWide               // 7.1(wide)
)
