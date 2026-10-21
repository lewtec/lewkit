package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avfilter"
	"github.com/lewtec/lewkit/x/ffmpeg/avformat"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/ffmpeg/swscale"
)

// Version is the FFmpeg release pinned in modot.lock.json.
const Version = "n9.0.2-lewkit"

// Run runs the ffmpeg command against the supplied streams.
// A nil context is a caller bug.
func Run(ctx context.Context, args []string, in io.Reader, out, errw io.Writer) int {
	if err := ctx.Err(); err != nil {
		fmt.Fprintln(errw, err)
		return 1
	}
	if len(args) == 0 {
		writeBanner(errw, true, avutil.LevelInfo)
		fmt.Fprint(errw, usageText)
		return 1
	}
	job, err := parseArgs(args)
	if err != nil {
		if job == nil || job.showBanner() {
			level := avutil.LevelInfo
			if job != nil {
				level = job.level
			}
			writeBanner(errw, false, level)
		}
		return writeErr(errw, err)
	}
	if job.help != "" {
		if job.showBanner() {
			writeBanner(errw, false, job.level)
		}
		fmt.Fprint(out, helpText(job.help))
		return 0
	}
	if job.version {
		fmt.Fprint(out, versionText())
		return 0
	}
	if job.license || job.buildconf || job.list != "" {
		if job.showBanner() {
			writeBanner(errw, false, job.level)
		}
		fmt.Fprint(out, infoText(job))
		return 0
	}
	if job.showBanner() {
		writeBanner(errw, false, job.level)
	}
	if err := transcode(ctx, job, in, out, errw); err != nil {
		return writeErr(errw, err)
	}
	return 0
}

type cliError struct {
	code int
	text string
}

func (e *cliError) Error() string { return e.text }

func fail(code int, text string) *cliError { return &cliError{code: code, text: text} }

func writeErr(errw io.Writer, err error) int {
	if ce, ok := err.(*cliError); ok {
		fmt.Fprintln(errw, ce.text)
		if ce.code == 0 {
			return 1
		}
		return ce.code
	}
	fmt.Fprintln(errw, err)
	return 1
}

type fileOpt struct {
	format                 string
	codec                  map[string]string
	vcodec, acodec         string
	maps                   []string
	ss, dur, to            int64
	ssSet, durSet, toSet   bool
	framesV, framesA       int
	framesVSet, framesASet bool
	rate                   avutil.Rational
	rateSet                bool
	width, height          int
	pix                    avutil.PixelFormat
	pixSet                 bool
	ar                     int
	layout                 avutil.ChannelLayout
	layoutSet              bool
	sample                 avutil.SampleFormat
	sampleSet              bool
	vf, af                 string
	sws                    swscale.Flags
	swsSet                 bool
	qscale                 int
	vn, an, sn, shortest   bool
	metadata               [][2]string
	bitrate                string
}

type inputFile struct {
	url string
	opt fileOpt
}

type outputFile struct {
	url string
	opt fileOpt
}

type job struct {
	in        []inputFile
	out       []outputFile
	overwrite int
	nostdin   bool
	hide      bool
	stats     bool
	bitexact  bool
	level     avutil.Level
	levelSet  bool
	progress  string
	filter    string
	cpu       string
	help      string
	list      string
	license   bool
	buildconf bool
	version   bool
}

func (j *job) showBanner() bool {
	if j == nil || j.hide {
		return false
	}
	level := j.level
	if !j.levelSet {
		level = avutil.LevelInfo
	}
	return level > avutil.LevelError
}

func blankOpt() fileOpt { return fileOpt{codec: map[string]string{}} }

func parseArgs(args []string) (*job, error) {
	j := &job{level: avutil.LevelInfo}
	cur := blankOpt()
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-" || !strings.HasPrefix(a, "-") || a == "--" {
			if a == "--" {
				return j, fail(1, "Unrecognized option '-'.\nError splitting the argument list: Option not found")
			}
			j.out = append(j.out, outputFile{url: a, opt: cur})
			cur = blankOpt()
			continue
		}
		name, spec, inline, hasInline := splitOpt(a)
		take := func() (string, error) {
			if hasInline {
				return inline, nil
			}
			i++
			if i >= len(args) {
				return "", fail(1, "Missing argument for option '"+name+"'.\nError splitting the argument list: Invalid argument")
			}
			return args[i], nil
		}
		switch name {
		case "y":
			j.overwrite = 1
		case "n":
			j.overwrite = -1
		case "nostdin":
			j.nostdin = true
		case "hide_banner":
			j.hide = true
		case "stats":
			j.stats = true
		case "bitexact":
			j.bitexact = true
		case "version":
			j.version = true
		case "license", "L":
			j.license = true
		case "buildconf":
			j.buildconf = true
		case "formats", "codecs", "decoders", "encoders", "filters", "muxers", "demuxers", "pix_fmts", "sample_fmts", "layouts":
			j.list = name
		case "h", "help", "?":
			if hasInline {
				j.help = inline
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				j.help = args[i]
				continue
			}
			j.help = "basic"
		case "loglevel", "v":
			value, err := take()
			if err != nil {
				return j, err
			}
			level, err := parseLevel(value)
			if err != nil {
				return j, err
			}
			j.level = level
			j.levelSet = true
		case "progress":
			value, err := take()
			if err != nil {
				return j, err
			}
			j.progress = value
		case "cpuflags":
			value, err := take()
			if err != nil {
				return j, err
			}
			j.cpu = value
		case "filter_complex":
			value, err := take()
			if err != nil {
				return j, err
			}
			j.filter = value
		case "filter_complex_script":
			value, err := take()
			if err != nil {
				return j, err
			}
			body, err := os.ReadFile(value)
			if err != nil {
				return j, fail(1, err.Error())
			}
			j.filter = string(body)
		case "i":
			value, err := take()
			if err != nil {
				return j, err
			}
			j.in = append(j.in, inputFile{url: value, opt: cur})
			cur = blankOpt()
		case "f":
			value, err := take()
			if err != nil {
				return j, err
			}
			cur.format = value
		case "c", "codec":
			value, err := take()
			if err != nil {
				return j, err
			}
			cur.codec[spec] = value
		case "vcodec":
			value, err := take()
			if err != nil {
				return j, err
			}
			cur.vcodec = value
		case "acodec":
			value, err := take()
			if err != nil {
				return j, err
			}
			cur.acodec = value
		case "map":
			value, err := take()
			if err != nil {
				return j, err
			}
			cur.maps = append(cur.maps, value)
		case "ss", "t", "to":
			value, err := take()
			if err != nil {
				return j, err
			}
			us, err := avutil.ParseTimeUS(value)
			if err != nil {
				return j, fail(1, err.Error())
			}
			switch name {
			case "ss":
				cur.ss, cur.ssSet = us, true
			case "t":
				cur.dur, cur.durSet = us, true
			default:
				cur.to, cur.toSet = us, true
			}
		case "frames":
			value, err := take()
			if err != nil {
				return j, err
			}
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 {
				return j, fail(1, "frames "+value)
			}
			switch spec {
			case "a":
				cur.framesA, cur.framesASet = n, true
			default:
				cur.framesV, cur.framesVSet = n, true
			}
		case "r":
			value, err := take()
			if err != nil {
				return j, err
			}
			rate, err := avutil.ParseRate(value)
			if err != nil {
				return j, fail(1, err.Error())
			}
			cur.rate, cur.rateSet = rate, true
		case "s":
			value, err := take()
			if err != nil {
				return j, err
			}
			w, h, err := avutil.ParseSize(value)
			if err != nil {
				return j, fail(1, err.Error())
			}
			cur.width, cur.height = w, h
		case "pix_fmt":
			value, err := take()
			if err != nil {
				return j, err
			}
			pix, err := avutil.ParsePixelFormat(value)
			if err != nil {
				return j, fail(1, err.Error())
			}
			cur.pix, cur.pixSet = pix, true
		case "ar":
			value, err := take()
			if err != nil {
				return j, err
			}
			n, err := strconv.Atoi(value)
			if err != nil || n <= 0 {
				return j, fail(1, "sample rate "+value)
			}
			cur.ar = n
		case "ac":
			value, err := take()
			if err != nil {
				return j, err
			}
			layout, err := avutil.ParseLayout(value)
			if err != nil {
				return j, fail(1, err.Error())
			}
			cur.layout, cur.layoutSet = layout, true
		case "sample_fmt":
			value, err := take()
			if err != nil {
				return j, err
			}
			sample, err := avutil.ParseSampleFormat(value)
			if err != nil {
				return j, fail(1, err.Error())
			}
			cur.sample, cur.sampleSet = sample, true
		case "b":
			value, err := take()
			if err != nil {
				return j, err
			}
			cur.bitrate = value
		case "vn":
			cur.vn = true
		case "an":
			cur.an = true
		case "sn":
			cur.sn = true
		case "shortest":
			cur.shortest = true
		case "vf":
			value, err := take()
			if err != nil {
				return j, err
			}
			cur.vf = value
		case "af":
			value, err := take()
			if err != nil {
				return j, err
			}
			cur.af = value
		case "filter":
			value, err := take()
			if err != nil {
				return j, err
			}
			if spec == "a" {
				cur.af = value
			} else {
				cur.vf = value
			}
		case "metadata":
			value, err := take()
			if err != nil {
				return j, err
			}
			key, val, _ := strings.Cut(value, "=")
			cur.metadata = append(cur.metadata, [2]string{key, val})
		case "sws_flags":
			value, err := take()
			if err != nil {
				return j, err
			}
			cur.sws, cur.swsSet = parseSWS(value), true
		case "q", "qscale":
			value, err := take()
			if err != nil {
				return j, err
			}
			n, err := strconv.Atoi(value)
			if err != nil {
				return j, fail(1, "qscale "+value)
			}
			cur.qscale = n
		default:
			return j, fail(1, "Unrecognized option '"+name+"'.\nError splitting the argument list: Option not found")
		}
	}
	return j, nil
}

func splitOpt(arg string) (name, spec, inline string, hasInline bool) {
	body := strings.TrimPrefix(arg, "-")
	if eq := strings.IndexByte(body, '='); eq >= 0 {
		inline = body[eq+1:]
		hasInline = true
		body = body[:eq]
	}
	name = body
	if i := strings.IndexByte(body, ':'); i >= 0 {
		name = body[:i]
		spec = body[i+1:]
	}
	return name, spec, inline, hasInline
}

func parseLevel(text string) (avutil.Level, error) {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "quiet":
		return avutil.LevelQuiet, nil
	case "panic":
		return avutil.LevelPanic, nil
	case "fatal":
		return avutil.LevelFatal, nil
	case "error":
		return avutil.LevelError, nil
	case "warning":
		return avutil.LevelWarning, nil
	case "info":
		return avutil.LevelInfo, nil
	case "verbose":
		return avutil.LevelVerbose, nil
	case "debug":
		return avutil.LevelDebug, nil
	case "trace":
		return avutil.LevelTrace, nil
	}
	n, err := strconv.Atoi(text)
	if err != nil {
		return 0, fail(1, "loglevel "+text)
	}
	return avutil.Level(n), nil
}

func parseSWS(text string) swscale.Flags {
	var flags swscale.Flags
	for _, part := range strings.Split(text, "+") {
		switch strings.TrimSpace(part) {
		case "fast_bilinear":
			flags |= swscale.FastBilinear
		case "bilinear":
			flags |= swscale.Bilinear
		case "bicubic":
			flags |= swscale.Bicubic
		case "neighbor", "point":
			flags |= swscale.Point
		case "area":
			flags |= swscale.Area
		case "lanczos":
			flags |= swscale.Lanczos
		case "spline":
			flags |= swscale.Spline
		case "gauss", "gaussian":
			flags |= swscale.Gauss
		case "accurate_rnd":
			flags |= swscale.AccurateRnd
		}
	}
	return flags
}

const usageText = "Hyper fast Audio and Video encoder\nusage: ffmpeg [options] [[infile options] -i infile]... {[outfile options] outfile}...\n\nUse -h to get full help or, even better, run 'man ffmpeg'\n"

func helpText(topic string) string {
	var b strings.Builder
	b.WriteString(usageText)
	b.WriteString("\nThis build transcribes the FFmpeg 9.0.2 libraries that lewkit ships.\n")
	b.WriteString("Enabled codecs, filters, and formats are listed by -codecs, -filters, and -formats.\n")
	if topic != "" && topic != "basic" {
		fmt.Fprintf(&b, "\nNo extra help topic %q is compiled into this build.\n", topic)
	}
	b.WriteString(`
Getting help:
    -h                  print basic options
    -version            show version
    -formats            show available formats
    -codecs             show available codecs
    -decoders           show available decoders
    -encoders           show available encoders
    -filters            show available filters
    -muxers             show available muxers
    -demuxers           show available demuxers
    -pix_fmts           show available pixel formats
    -sample_fmts        show available sample formats
    -layouts            show standard channel layouts
    -buildconf          show build configuration
    -L                  show license

Global options:
    -y                  overwrite output files
    -n                  never overwrite output files
    -nostdin            do not read interactive input
    -hide_banner        do not print the banner
    -loglevel level     set logging level
    -stats              print a short summary
    -cpuflags flags     0 or none forces the scalar kernels
    -bitexact           omit the muxer software tag
    -filter_complex g   build a filtergraph
    -progress url       accept a progress target

Per-file options:
    -f fmt              force format
    -i url              input url
    -c[:s] codec        codec name, or copy
    -vcodec/-acodec c   video or audio codec
    -map spec           choose a stream or [label]
    -ss/-t/-to time     seek, duration, and stop
    -frames[:v|:a] n    frame count
    -r rate             frame rate
    -s size             frame size
    -pix_fmt fmt        pixel format
    -ar/-ac/-sample_fmt audio rate, channels, and sample format
    -vn/-an/-sn         disable video, audio, or subtitles
    -vf/-af graph       video or audio filtergraph
    -sws_flags flags    swscale flags
    -q/-qscale n        quantizer scale
`)
	return b.String()
}

func versionText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "ffmpeg version %s Copyright (c) 2000-2026 the FFmpeg developers\n", Version)
	fmt.Fprintf(&b, "built with %s\n", runtime.Version())
	fmt.Fprintf(&b, "configuration: %s\n", configuration())
	for _, name := range []string{"libavutil", "libavcodec", "libavformat", "libavfilter", "libswscale", "libswresample"} {
		fmt.Fprintf(&b, "%-15s %s\n", name, Version)
	}
	return b.String()
}

func writeBanner(w io.Writer, full bool, level avutil.Level) {
	if level <= avutil.LevelError && !full {
		return
	}
	fmt.Fprint(w, versionText())
}

func configuration() string {
	var parts []string
	parts = append(parts, "--enable-gpl", "--disable-avdevice", "--disable-postproc", "--enable-swscale", "--enable-swresample")
	for _, item := range avfilter.Filters() {
		parts = append(parts, "--enable-filter="+item.Name)
	}
	for _, item := range avcodec.Codecs() {
		if item.Decode {
			parts = append(parts, "--enable-decoder="+item.Name)
		}
		if item.Encode {
			parts = append(parts, "--enable-encoder="+item.Name)
		}
	}
	for _, item := range avformat.Formats() {
		if item.Demux {
			parts = append(parts, "--enable-demuxer="+item.Name)
		}
		if item.Mux {
			parts = append(parts, "--enable-muxer="+item.Name)
		}
	}
	return strings.Join(parts, " ")
}

func infoText(j *job) string {
	if j.license {
		return "lewkit ffmpeg is GPL. The transcribed FFmpeg algorithms are covered by that license.\nSee LICENSE.md.\n"
	}
	if j.buildconf {
		return configuration() + "\n"
	}
	switch j.list {
	case "formats":
		return listFormats(true, true)
	case "muxers":
		return listFormats(false, true)
	case "demuxers":
		return listFormats(true, false)
	case "codecs":
		return listCodecs(true, true)
	case "decoders":
		return listCodecs(true, false)
	case "encoders":
		return listCodecs(false, true)
	case "filters":
		return listFilters()
	case "pix_fmts":
		return listPix()
	case "sample_fmts":
		return listSamples()
	case "layouts":
		return listLayouts()
	default:
		return ""
	}
}

func listFormats(demux, mux bool) string {
	var b strings.Builder
	b.WriteString("File formats:\n D. = Demuxing supported\n .E = Muxing supported\n --\n")
	for _, item := range avformat.Formats() {
		if demux && !mux && !item.Demux {
			continue
		}
		if mux && !demux && !item.Mux {
			continue
		}
		flag := ".."
		if item.Demux && item.Mux {
			flag = "DE"
		} else if item.Demux {
			flag = "D."
		} else if item.Mux {
			flag = ".E"
		}
		fmt.Fprintf(&b, " %s %-16s %s\n", flag, item.Name, item.Long)
	}
	return b.String()
}

func listCodecs(dec, enc bool) string {
	var b strings.Builder
	b.WriteString("Codecs:\n D..... = Decoding supported\n .E.... = Encoding supported\n ..V... = Video codec\n ..A... = Audio codec\n ...I.. = Intra frame-only codec\n ....L. = Lossy compression\n .....S = Lossless compression\n -------\n")
	for _, item := range avcodec.Codecs() {
		if dec && !enc && !item.Decode {
			continue
		}
		if enc && !dec && !item.Encode {
			continue
		}
		fmt.Fprintf(&b, " %s %-18s %s\n", codecFlags(item), item.Name, item.Long)
	}
	return b.String()
}

func codecFlags(item avcodec.Codec) string {
	b := []byte{'.', '.', '.', '.', '.', '.'}
	if item.Decode {
		b[0] = 'D'
	}
	if item.Encode {
		b[1] = 'E'
	}
	switch item.Type {
	case avutil.MediaVideo:
		b[2] = 'V'
	case avutil.MediaAudio:
		b[2] = 'A'
	case avutil.MediaSubtitle:
		b[2] = 'S'
	}
	if item.Intra {
		b[3] = 'I'
	}
	if item.Lossy {
		b[4] = 'L'
	}
	if item.Lossless {
		b[5] = 'S'
	}
	return string(b)
}

func listFilters() string {
	var b strings.Builder
	b.WriteString("Filters:\n")
	for _, item := range avfilter.Filters() {
		ioMark := "V->V"
		if item.Audio && !item.Video {
			ioMark = "A->A"
		}
		if item.Source {
			if item.Audio {
				ioMark = "|->A"
			} else {
				ioMark = "|->V"
			}
		}
		fmt.Fprintf(&b, " ... %-16s %-8s %s\n", item.Name, ioMark, item.Desc)
	}
	return b.String()
}

func listPix() string {
	var b strings.Builder
	b.WriteString("Pixel formats:\n")
	for _, item := range avutil.PixelFormat(0).Values() {
		flags := "IO"
		if desc, ok := avutil.Desc(item); ok && desc.RGB {
			flags = "IORGB"
		}
		fmt.Fprintf(&b, "%-8s %s\n", flags, item)
	}
	return b.String()
}

func listSamples() string {
	var b strings.Builder
	b.WriteString("Sample formats:\n")
	for _, item := range avutil.SampleFormat(0).Values() {
		fmt.Fprintf(&b, "%-8s %d\n", item, avutil.BytesPerSample(item))
	}
	return b.String()
}

func listLayouts() string {
	var b strings.Builder
	b.WriteString("Individual channels:\nNAME           DESCRIPTION\n")
	b.WriteString("FL             front left\nFR             front right\nFC             front center\n")
	b.WriteString("Standard channel layouts:\nNAME           DECOMPOSITION\n")
	for _, item := range avutil.LayoutName(0).Values() {
		fmt.Fprintf(&b, "%-14s %s\n", item, item)
	}
	return b.String()
}
