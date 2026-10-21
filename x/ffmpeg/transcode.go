package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avfilter"
	"github.com/lewtec/lewkit/x/ffmpeg/avformat"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/ffmpeg/swresample"
	"github.com/lewtec/lewkit/x/ffmpeg/swscale"
)

type elementary struct {
	par    avcodec.Parameters
	frames []*avutil.Frame
	pkts   []*avutil.Packet
}

type opened struct {
	streams []elementary
}

type filtered struct {
	label  string
	frames []*avutil.Frame
}

type streamRef struct {
	filter int
	input  int
	index  int
	from   string
}

func transcode(ctx context.Context, job *job, in io.Reader, out, errw io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if job.cpu == "0" || job.cpu == "none" {
		avutil.SetScalar(true)
	}
	inputs, err := readInputs(ctx, job, in)
	if err != nil {
		return err
	}
	filts, err := runFilter(ctx, job, inputs)
	if err != nil {
		return err
	}
	if len(job.out) == 0 {
		return fail(1, "At least one output file must be specified")
	}
	var encoded int
	for _, item := range job.out {
		n, err := writeOutput(ctx, job, item, inputs, filts, in, out)
		if err != nil {
			return err
		}
		encoded += n
	}
	if job.stats {
		fmt.Fprintf(errw, "encoded %d frames\n", encoded)
	}
	_ = job.progress
	return nil
}

func readInputs(ctx context.Context, job *job, in io.Reader) ([]opened, error) {
	var inputs []opened
	for _, item := range job.in {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		openedOne, err := readOne(ctx, item, job.bitexact, in)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, openedOne)
	}
	return inputs, nil
}

func readOne(ctx context.Context, item inputFile, bitexact bool, in io.Reader) (opened, error) {
	opt := demuxOpt(item.opt, bitexact, in, nil)
	demux, err := avformat.OpenInput(ctx, pipeName(item.url), item.opt.format, opt)
	if err != nil {
		return opened{}, err
	}
	defer demux.Close()
	streams := demux.Streams()
	groups := make([][]*avutil.Packet, len(streams))
	for {
		if err := ctx.Err(); err != nil {
			return opened{}, err
		}
		pkt, err := demux.ReadPacket()
		if err == io.EOF {
			break
		}
		if err != nil {
			return opened{}, err
		}
		if pkt == nil {
			continue
		}
		n := pkt.Stream
		if n < 0 || n >= len(groups) {
			n = 0
		}
		groups[n] = append(groups[n], pkt)
	}
	out := opened{streams: make([]elementary, len(streams))}
	for i, st := range streams {
		frames, err := decodeAll(st, groups[i])
		if err != nil {
			return opened{}, err
		}
		frames, pkts := trimInput(frames, groups[i], st, item.opt)
		par := st.Par
		par.TimeBase = st.TimeBase
		par.SAR = st.SAR
		out.streams[i] = elementary{par: par, frames: frames, pkts: pkts}
	}
	return out, nil
}

func decodeAll(st avformat.Stream, pkts []*avutil.Packet) ([]*avutil.Frame, error) {
	if len(pkts) == 0 {
		return nil, nil
	}
	dec, err := avcodec.OpenDecoder(st.Par)
	if err != nil {
		return nil, err
	}
	defer dec.Close()
	var frames []*avutil.Frame
	for _, pkt := range pkts {
		got, err := dec.Decode(pkt)
		if err != nil {
			return nil, err
		}
		for _, frame := range got {
			if frame == nil {
				continue
			}
			if pkt.TimeBase.Den != 0 {
				frame.TimeBase = pkt.TimeBase
			} else if st.TimeBase.Den != 0 {
				frame.TimeBase = st.TimeBase
			}
			if frame.SAR.Den == 0 {
				frame.SAR = st.SAR
			}
			if frame.Media == 0 {
				frame.Media = st.Par.Type
			}
			frames = append(frames, frame)
		}
	}
	return frames, nil
}

func trimInput(frames []*avutil.Frame, pkts []*avutil.Packet, st avformat.Stream, opt fileOpt) ([]*avutil.Frame, []*avutil.Packet) {
	var keptF []*avutil.Frame
	for _, frame := range frames {
		tb := frame.TimeBase
		if tb.Den == 0 {
			tb = st.TimeBase
		}
		if !inWindow(frame.PTS, tb, opt) {
			continue
		}
		if opt.ssSet {
			frame.PTS -= avutil.RescaleQ(opt.ss, avutil.Q(1, 1000000), tb)
			if frame.PTS < 0 {
				frame.PTS = 0
			}
		}
		keptF = append(keptF, frame)
	}
	var keptP []*avutil.Packet
	for _, pkt := range pkts {
		tb := pkt.TimeBase
		if tb.Den == 0 {
			tb = st.TimeBase
		}
		if !inWindow(pkt.PTS, tb, opt) {
			continue
		}
		cp := pkt.Clone()
		if opt.ssSet {
			cp.PTS -= avutil.RescaleQ(opt.ss, avutil.Q(1, 1000000), tb)
			cp.DTS = cp.PTS
			if cp.PTS < 0 {
				cp.PTS, cp.DTS = 0, 0
			}
		}
		keptP = append(keptP, cp)
	}
	return keptF, keptP
}

func inWindow(pts int64, tb avutil.Rational, opt fileOpt) bool {
	us := stampUS(pts, tb)
	origin := int64(0)
	if opt.ssSet {
		if us < opt.ss {
			return false
		}
		origin = opt.ss
	}
	if opt.toSet && us >= opt.to {
		return false
	}
	if opt.durSet && us >= origin+opt.dur {
		return false
	}
	return true
}

func stampUS(pts int64, tb avutil.Rational) int64 {
	if tb.Den == 0 {
		tb = avutil.Q(1, 25)
	}
	return avutil.RescaleQ(pts, tb, avutil.Q(1, 1000000))
}

func runFilter(ctx context.Context, job *job, inputs []opened) ([]filtered, error) {
	if job.filter == "" {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	flags := swscale.Flags(0)
	if len(job.out) > 0 && job.out[0].opt.swsSet {
		flags = job.out[0].opt.sws
	}
	graph, err := avfilter.Parse(job.filter, flags)
	if err != nil {
		return nil, err
	}
	for _, label := range graph.InputLabels() {
		frames, err := framesForLabel(label, inputs)
		if err != nil {
			return nil, err
		}
		graph.SetInput(label, frames)
	}
	labels := graph.Labels()
	if len(labels) == 0 {
		frames, err := pullAll(graph.Next)
		if err != nil {
			return nil, err
		}
		return []filtered{{frames: frames}}, nil
	}
	var out []filtered
	for _, label := range labels {
		frames, err := pullAll(func() (*avutil.Frame, error) { return graph.NextLabel(label) })
		if err != nil {
			return nil, err
		}
		out = append(out, filtered{label: label, frames: frames})
	}
	return out, nil
}

func pullAll(next func() (*avutil.Frame, error)) ([]*avutil.Frame, error) {
	var frames []*avutil.Frame
	for {
		frame, err := next()
		if err == io.EOF {
			return frames, nil
		}
		if err != nil {
			return nil, err
		}
		if frame != nil {
			frames = append(frames, frame)
		}
	}
}

func framesForLabel(label string, inputs []opened) ([]*avutil.Frame, error) {
	spec := strings.Trim(label, "[]")
	refs, err := resolveMap(spec, inputs, nil)
	if err != nil {
		return nil, err
	}
	var frames []*avutil.Frame
	for _, ref := range refs {
		if ref.input < 0 || ref.input >= len(inputs) || ref.index >= len(inputs[ref.input].streams) {
			return nil, fail(1, "filter input "+label)
		}
		frames = append(frames, inputs[ref.input].streams[ref.index].frames...)
	}
	return frames, nil
}

func writeOutput(ctx context.Context, job *job, item outputFile, inputs []opened, filts []filtered, in io.Reader, out io.Writer) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	url := pipeName(item.url)
	if url != "-" {
		if _, err := os.Stat(url); err == nil && job.overwrite != 1 {
			return 0, fail(1, "File '"+url+"' already exists. Exiting.")
		}
	}
	refs, err := selectStreams(item.opt, inputs, filts)
	if err != nil {
		return 0, err
	}
	if len(refs) == 0 {
		return 0, fail(1, "Output file does not contain any stream")
	}
	format := item.opt.format
	if format == "" {
		format = avformat.Guess(url)
	}
	if format == "" {
		return 0, fail(1, "Unable to find a suitable output format for '"+item.url+"'")
	}
	muxOpt := demuxOpt(item.opt, job.bitexact, in, out)
	mux, err := avformat.Create(ctx, url, format, muxOpt)
	if err != nil {
		return 0, err
	}
	type encoded struct {
		index int
		pkts  []*avutil.Packet
	}
	var tracks []encoded
	count := 0
	limit := shortestLimit(item.opt, refs, inputs, filts)
	for _, ref := range refs {
		media, frames, pkts, par := refMaterial(ref, inputs, filts)
		if item.opt.vn && media == avutil.MediaVideo {
			continue
		}
		if item.opt.an && media == avutil.MediaAudio {
			continue
		}
		if item.opt.sn && media == avutil.MediaSubtitle {
			continue
		}
		copying := codecName(item.opt, media, ref.index) == "copy"
		outPar := par
		var packets []*avutil.Packet
		if copying {
			packets = pkts
		} else {
			id, err := chooseCodec(item.opt, format, url, media, ref.index)
			if err != nil {
				_ = mux.Close()
				return 0, err
			}
			frames, err = applyChain(frames, item.opt, media)
			if err != nil {
				_ = mux.Close()
				return 0, err
			}
			frames = limitFrames(frames, item.opt, media, limit)
			outPar, packets, err = encodeFrames(id, frames, item.opt)
			if err != nil {
				_ = mux.Close()
				return 0, err
			}
		}
		if len(packets) == 0 && len(frames) == 0 && !copying {
			continue
		}
		stream, err := mux.AddStream(outPar)
		if err != nil {
			_ = mux.Close()
			return 0, err
		}
		for _, pkt := range packets {
			pkt.Stream = stream
		}
		tracks = append(tracks, encoded{index: stream, pkts: packets})
		count += len(packets)
	}
	if len(tracks) == 0 {
		_ = mux.Close()
		return 0, fail(1, "Output file does not contain any stream")
	}
	if err := mux.WriteHeader(); err != nil {
		_ = mux.Close()
		return 0, err
	}
	for _, track := range tracks {
		for _, pkt := range track.pkts {
			if err := ctx.Err(); err != nil {
				_ = mux.Close()
				return 0, err
			}
			if err := mux.WritePacket(pkt); err != nil {
				_ = mux.Close()
				return 0, err
			}
		}
	}
	if err := mux.Close(); err != nil {
		return 0, err
	}
	return count, nil
}

func refMaterial(ref streamRef, inputs []opened, filts []filtered) (avutil.MediaType, []*avutil.Frame, []*avutil.Packet, avcodec.Parameters) {
	if ref.filter >= 0 {
		frames := filts[ref.filter].frames
		par := parFromFrames(frames)
		return par.Type, frames, nil, par
	}
	st := inputs[ref.input].streams[ref.index]
	return st.par.Type, st.frames, st.pkts, st.par
}

func parFromFrames(frames []*avutil.Frame) avcodec.Parameters {
	if len(frames) == 0 {
		return avcodec.Parameters{}
	}
	frame := frames[0]
	if frame.Width > 0 {
		tb := frame.TimeBase
		if tb.Den == 0 {
			tb = avutil.Q(1, 25)
		}
		return avcodec.Parameters{
			Type: avutil.MediaVideo, Width: frame.Width, Height: frame.Height,
			Format: frame.Format, TimeBase: tb, SAR: frame.SAR,
		}
	}
	rate := frame.SampleRate
	if rate <= 0 {
		rate = 44100
	}
	tb := frame.TimeBase
	if tb.Den == 0 {
		tb = avutil.Q(1, int32(rate))
	}
	layout := frame.Layout
	if layout.Channels() <= 0 {
		layout = avutil.Mono()
	}
	return avcodec.Parameters{
		Type: avutil.MediaAudio, SampleFormat: frame.SampleFmt, SampleRate: rate,
		Layout: layout, TimeBase: tb,
	}
}

func selectStreams(opt fileOpt, inputs []opened, filts []filtered) ([]streamRef, error) {
	if len(opt.maps) == 0 {
		if len(filts) > 0 {
			refs := make([]streamRef, len(filts))
			for i := range filts {
				refs[i] = streamRef{filter: i, input: -1}
			}
			return refs, nil
		}
		return defaultMaps(inputs), nil
	}
	var refs []streamRef
	for _, spec := range opt.maps {
		got, err := resolveMap(spec, inputs, filts)
		if err != nil {
			return nil, err
		}
		refs = append(refs, got...)
	}
	return refs, nil
}

func defaultMaps(inputs []opened) []streamRef {
	var refs []streamRef
	seen := map[avutil.MediaType]bool{}
	for i, in := range inputs {
		for j, st := range in.streams {
			if st.par.Type != avutil.MediaVideo && st.par.Type != avutil.MediaAudio && st.par.Type != avutil.MediaSubtitle {
				continue
			}
			if seen[st.par.Type] {
				continue
			}
			seen[st.par.Type] = true
			refs = append(refs, streamRef{filter: -1, input: i, index: j})
		}
	}
	return refs
}

func resolveMap(spec string, inputs []opened, filts []filtered) ([]streamRef, error) {
	if strings.HasPrefix(spec, "[") && strings.HasSuffix(spec, "]") {
		label := strings.Trim(spec, "[]")
		for i, item := range filts {
			if item.label == label {
				return []streamRef{{filter: i, input: -1}}, nil
			}
		}
		return nil, fail(1, "Output with label "+label+" does not exist")
	}
	parts := strings.Split(spec, ":")
	input, err := strconv.Atoi(parts[0])
	if err != nil || input < 0 || input >= len(inputs) {
		return nil, fail(1, "Invalid stream specifier '"+spec+"'")
	}
	streams := inputs[input].streams
	if len(parts) == 1 {
		refs := make([]streamRef, len(streams))
		for i := range streams {
			refs[i] = streamRef{filter: -1, input: input, index: i}
		}
		return refs, nil
	}
	if n, err := strconv.Atoi(parts[1]); err == nil && len(parts) == 2 {
		if n < 0 || n >= len(streams) {
			return nil, fail(1, "Invalid stream specifier '"+spec+"'")
		}
		return []streamRef{{filter: -1, input: input, index: n}}, nil
	}
	kind := parts[1]
	nth := -1
	if len(parts) >= 3 {
		nth, err = strconv.Atoi(parts[2])
		if err != nil {
			return nil, fail(1, "Invalid stream specifier '"+spec+"'")
		}
	}
	var match []int
	for i, st := range streams {
		if kindLetter(st.par.Type) == kind {
			match = append(match, i)
		}
	}
	if nth >= 0 {
		if nth >= len(match) {
			return nil, fail(1, "Invalid stream specifier '"+spec+"'")
		}
		return []streamRef{{filter: -1, input: input, index: match[nth]}}, nil
	}
	refs := make([]streamRef, len(match))
	for i, index := range match {
		refs[i] = streamRef{filter: -1, input: input, index: index}
	}
	return refs, nil
}

func kindLetter(media avutil.MediaType) string {
	switch media {
	case avutil.MediaVideo:
		return "v"
	case avutil.MediaAudio:
		return "a"
	case avutil.MediaSubtitle:
		return "s"
	default:
		return "d"
	}
}

func codecName(opt fileOpt, media avutil.MediaType, index int) string {
	keys := []string{strconv.Itoa(index)}
	keys = append(keys, kindLetter(media), "")
	for _, key := range keys {
		if name, ok := opt.codec[key]; ok {
			return name
		}
	}
	if media == avutil.MediaVideo && opt.vcodec != "" {
		return opt.vcodec
	}
	if media == avutil.MediaAudio && opt.acodec != "" {
		return opt.acodec
	}
	return ""
}

func parseBitrate(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	mul := 1
	switch {
	case strings.HasSuffix(s, "k") || strings.HasSuffix(s, "K"):
		mul = 1000
		s = s[:len(s)-1]
	case strings.HasSuffix(s, "M"):
		mul = 1000000
		s = s[:len(s)-1]
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0
	}
	return n * mul
}

func chooseCodec(opt fileOpt, format, url string, media avutil.MediaType, index int) (avcodec.ID, error) {
	if name := codecName(opt, media, index); name != "" && name != "copy" {
		item, ok := avcodec.FindByName(name)
		if !ok || !item.Encode {
			return 0, fail(1, "Unknown encoder '"+name+"'")
		}
		return item.ID, nil
	}
	if media == avutil.MediaVideo {
		switch format {
		case "image2", "image2pipe":
			return avformat.ImageCodec(url), nil
		case "avi", "mp4", "mov":
			return avcodec.IDMJPEG, nil
		case "h264":
			return avcodec.IDH264, nil
		case "hevc":
			return avcodec.IDHEVC, nil
		default:
			return avcodec.IDRawVideo, nil
		}
	}
	if media == avutil.MediaAudio {
		switch format {
		case "gsm":
			return avcodec.IDGSM, nil
		case "g722":
			return avcodec.IDG722, nil
		case "g726":
			return avcodec.IDG726, nil
		case "g726le":
			return avcodec.IDG726LE, nil
		case "mp2":
			return avcodec.IDMP2, nil
		case "wsaud":
			return avcodec.IDADPCMIMAWS, nil
		case "flac":
			return avcodec.IDFLAC, nil
		case "u8":
			return avcodec.IDPCMU8, nil
		case "s16be":
			return avcodec.IDPCMS16BE, nil
		case "s24le":
			return avcodec.IDPCMS24LE, nil
		case "s32le":
			return avcodec.IDPCMS32LE, nil
		case "f32le":
			return avcodec.IDPCMF32LE, nil
		case "alaw":
			return avcodec.IDPCMALAW, nil
		case "mulaw":
			return avcodec.IDPCMMULAW, nil
		default:
			return avcodec.IDPCMS16LE, nil
		}
	}
	return 0, fail(1, "unsupported stream type "+media.String())
}

func applyChain(frames []*avutil.Frame, opt fileOpt, media avutil.MediaType) ([]*avutil.Frame, error) {
	spec := chainSpec(opt, media)
	if spec == "" || len(frames) == 0 {
		return frames, nil
	}
	flags := swscale.Flags(0)
	if opt.swsSet {
		flags = opt.sws
	}
	graph, err := avfilter.Parse(spec, flags)
	if err != nil {
		return nil, err
	}
	var out []*avutil.Frame
	for _, frame := range frames {
		got, err := graph.Filter(frame)
		if err != nil {
			return nil, err
		}
		out = append(out, got...)
	}
	return out, nil
}

func chainSpec(opt fileOpt, media avutil.MediaType) string {
	var parts []string
	if media == avutil.MediaVideo {
		if opt.vf != "" {
			parts = append(parts, opt.vf)
		}
		if opt.width > 0 && opt.height > 0 {
			parts = append(parts, fmt.Sprintf("scale=%d:%d", opt.width, opt.height))
		}
		if opt.pixSet {
			parts = append(parts, "format="+opt.pix.String())
		}
		if opt.rateSet {
			parts = append(parts, "fps="+rateText(opt.rate))
		}
	}
	if media == avutil.MediaAudio {
		if opt.af != "" {
			parts = append(parts, opt.af)
		}
		if opt.ar > 0 || opt.layoutSet || opt.sampleSet {
			var args []string
			if opt.ar > 0 {
				args = append(args, "sample_rates="+strconv.Itoa(opt.ar))
			}
			if opt.sampleSet {
				args = append(args, "sample_fmts="+opt.sample.String())
			}
			if opt.layoutSet {
				args = append(args, "channel_layouts="+avutil.LayoutString(opt.layout))
			}
			parts = append(parts, "aformat="+strings.Join(args, ":"))
		}
	}
	return strings.Join(parts, ",")
}

func limitFrames(frames []*avutil.Frame, opt fileOpt, media avutil.MediaType, limit int64) []*avutil.Frame {
	var out []*avutil.Frame
	for _, frame := range frames {
		if !inWindow(frame.PTS, frame.TimeBase, opt) {
			continue
		}
		if limit >= 0 && stampUS(frame.PTS, frame.TimeBase) >= limit {
			break
		}
		if media == avutil.MediaVideo && opt.framesVSet && len(out) >= opt.framesV {
			break
		}
		if media == avutil.MediaAudio && opt.framesASet && len(out) >= opt.framesA {
			break
		}
		out = append(out, frame)
	}
	return out
}

func shortestLimit(opt fileOpt, refs []streamRef, inputs []opened, filts []filtered) int64 {
	if !opt.shortest {
		return -1
	}
	limit := int64(-1)
	for _, ref := range refs {
		_, frames, _, _ := refMaterial(ref, inputs, filts)
		if len(frames) == 0 {
			return 0
		}
		last := frames[len(frames)-1]
		end := stampUS(last.PTS+last.Duration, last.TimeBase)
		if limit < 0 || end < limit {
			limit = end
		}
	}
	return limit
}

func encodeFrames(id avcodec.ID, frames []*avutil.Frame, opt fileOpt) (avcodec.Parameters, []*avutil.Packet, error) {
	if len(frames) == 0 {
		return avcodec.Parameters{ID: id}, nil, nil
	}
	par := parFromFrames(frames)
	par.ID = id
	par.QScale = opt.qscale
	if par.Type == avutil.MediaVideo {
		if opt.pixSet && id == avcodec.IDRawVideo {
			par.Format = opt.pix
		}
		if id == avcodec.IDH264 || id == avcodec.IDHEVC {
			par.Format = avutil.PixYUV420P
		}
		if opt.width > 0 {
			par.Width, par.Height = opt.width, opt.height
		}
	}
	if par.Type == avutil.MediaAudio {
		par.SampleFormat = sampleFor(id, par.SampleFormat, opt)
		if opt.ar > 0 {
			par.SampleRate = opt.ar
		}
		if n := parseBitrate(opt.bitrate); n > 0 {
			par.BitRate = n
		}
		if id == avcodec.IDGSM || id == avcodec.IDGSMMS {
			par.SampleFormat = avutil.SampleS16
			par.SampleRate = 8000
			par.Layout = avutil.Mono()
		}
		if id == avcodec.IDG722 {
			par.SampleFormat = avutil.SampleS16
			par.SampleRate = 16000
			par.Layout = avutil.Mono()
		}
		if opt.layoutSet {
			par.Layout = opt.layout
		}
	}
	enc, err := avcodec.OpenEncoder(par)
	if err != nil {
		return par, nil, err
	}
	defer enc.Close()
	var packets []*avutil.Packet
	flags := swscale.Flags(0)
	if opt.swsSet {
		flags = opt.sws
	}
	for _, frame := range frames {
		fitted, err := fitFrame(frame, par, flags)
		if err != nil {
			return par, nil, err
		}
		got, err := enc.Encode(fitted)
		if err != nil {
			return par, nil, err
		}
		packets = append(packets, stampPackets(got, fitted)...)
	}
	flushed, err := enc.Flush()
	if err != nil {
		return par, nil, err
	}
	if len(flushed) > 0 {
		last := frames[len(frames)-1]
		packets = append(packets, stampPackets(flushed, last)...)
	}
	if par.Type == avutil.MediaVideo && len(frames) > 0 {
		par.Width = frames[len(frames)-1].Width
		par.Height = frames[len(frames)-1].Height
		if id == avcodec.IDRawVideo {
			par.Format = frames[len(frames)-1].Format
		}
		par.TimeBase = frames[0].TimeBase
		par.SAR = frames[0].SAR
	}
	if par.Type == avutil.MediaAudio && len(frames) > 0 {
		if id == avcodec.IDGSM || id == avcodec.IDGSMMS {
			par.SampleRate = 8000
			par.SampleFormat = avutil.SampleS16
			par.Layout = avutil.Mono()
			par.TimeBase = avutil.Q(1, 8000)
		} else if id == avcodec.IDG722 {
			par.SampleRate = 16000
			par.SampleFormat = avutil.SampleS16
			par.Layout = avutil.Mono()
			par.TimeBase = avutil.Q(1, 16000)
		} else {
			par.SampleRate = frames[len(frames)-1].SampleRate
			par.SampleFormat = frames[len(frames)-1].SampleFmt
			par.Layout = frames[len(frames)-1].Layout
			par.TimeBase = frames[0].TimeBase
		}
	}
	return par, packets, nil
}

func stampPackets(pkts []*avutil.Packet, frame *avutil.Frame) []*avutil.Packet {
	for _, pkt := range pkts {
		if pkt.Duration == 0 {
			pkt.Duration = frame.Duration
		}
		if pkt.TimeBase.Den == 0 {
			pkt.TimeBase = frame.TimeBase
		}
		if pkt.PTS == 0 && frame.PTS != 0 {
			pkt.PTS = frame.PTS
			pkt.DTS = frame.PTS
		}
	}
	return pkts
}

func sampleFor(id avcodec.ID, fallback avutil.SampleFormat, opt fileOpt) avutil.SampleFormat {
	if opt.sampleSet {
		return opt.sample
	}
	switch id {
	case avcodec.IDPCMU8:
		return avutil.SampleU8
	case avcodec.IDPCMS16LE, avcodec.IDPCMS16BE, avcodec.IDPCMALAW, avcodec.IDPCMMULAW, avcodec.IDFLAC, avcodec.IDGSM, avcodec.IDGSMMS, avcodec.IDMP2, avcodec.IDADPCMIMAWAV, avcodec.IDADPCMMS, avcodec.IDADPCMYAMAHA, avcodec.IDADPCMIMAQT, avcodec.IDG722, avcodec.IDLIBSHINE, avcodec.IDG726, avcodec.IDG726LE, avcodec.IDADPCMSWF, avcodec.IDLIBTWOLAME, avcodec.IDADPCMIMAWS, avcodec.IDLIBSPEEX:
		return avutil.SampleS16
	case avcodec.IDPCMS24LE, avcodec.IDPCMS32LE:
		return avutil.SampleS32
	case avcodec.IDPCMF32LE:
		return avutil.SampleFLT
	default:
		return fallback
	}
}

func fitFrame(frame *avutil.Frame, par avcodec.Parameters, flags swscale.Flags) (*avutil.Frame, error) {
	if frame.Width > 0 {
		format := par.Format
		if format == 0 {
			format = frame.Format
		}
		w, h := par.Width, par.Height
		if w <= 0 || h <= 0 {
			w, h = frame.Width, frame.Height
		}
		if format == frame.Format && w == frame.Width && h == frame.Height {
			return frame, nil
		}
		dst, err := avutil.NewVideo(format, w, h)
		if err != nil {
			return nil, err
		}
		sw, err := swscale.New(frame.Width, frame.Height, frame.Format, w, h, format, flags)
		if err != nil {
			return nil, err
		}
		if err := sw.Scale(dst, frame); err != nil {
			return nil, err
		}
		dst.PTS, dst.Duration, dst.TimeBase, dst.SAR = frame.PTS, frame.Duration, frame.TimeBase, frame.SAR
		return dst, nil
	}
	format := par.SampleFormat
	if format == 0 {
		format = frame.SampleFmt
	}
	rate := par.SampleRate
	if rate <= 0 {
		rate = frame.SampleRate
	}
	layout := par.Layout
	if layout.Channels() <= 0 {
		layout = frame.Layout
	}
	if layout.Channels() <= 0 {
		layout = avutil.Mono()
	}
	if format == frame.SampleFmt && rate == frame.SampleRate && layout == frame.Layout {
		return frame, nil
	}
	sw, err := swresample.New(frame.SampleRate, frame.SampleFmt, frame.Layout, rate, format, layout)
	if err != nil {
		return nil, err
	}
	outN := frame.NbSamples
	if frame.SampleRate != rate && frame.SampleRate > 0 {
		outN = int(int64(frame.NbSamples) * int64(rate) / int64(frame.SampleRate))
		if outN < 1 && frame.NbSamples > 0 {
			outN = 1
		}
	}
	dst, err := avutil.NewAudio(format, rate, layout, outN)
	if err != nil {
		return nil, err
	}
	if err := sw.Convert(dst, frame); err != nil {
		return nil, err
	}
	dst.PTS = frame.PTS
	if frame.SampleRate > 0 && rate != frame.SampleRate {
		dst.PTS = avutil.Rescale(frame.PTS, int64(rate), int64(frame.SampleRate))
	}
	dst.Duration = int64(dst.NbSamples)
	dst.TimeBase = avutil.Q(1, int32(rate))
	return dst, nil
}

func demuxOpt(opt fileOpt, bitexact bool, in io.Reader, out io.Writer) avformat.Options {
	layout := opt.layout
	return avformat.Options{
		Width: opt.width, Height: opt.height, Format: opt.pix, FrameRate: opt.rate,
		SampleRate: opt.ar, Layout: layout, SampleFormat: opt.sample, Bitexact: bitexact,
		Stdin: in, Stdout: out,
	}
}

func rateText(rate avutil.Rational) string {
	if rate.Den == 1 {
		return strconv.Itoa(int(rate.Num))
	}
	return strconv.Itoa(int(rate.Num)) + "/" + strconv.Itoa(int(rate.Den))
}

func pipeName(url string) string {
	switch url {
	case "pipe:", "pipe:0", "pipe:1":
		return "-"
	default:
		return url
	}
}
