package avfilter

import (
	"fmt"
	"io"
	"strings"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/ffmpeg/swscale"
)

// Link pulls frames until io.EOF.
type Link interface {
	Next() (*avutil.Frame, error)
}

// Info is one filter this build can run.
type Info struct {
	Name   string
	Desc   string
	Source bool
	Video  bool
	Audio  bool
}

// Filters is the implemented set, in CLI order.
func Filters() []Info {
	return []Info{
		{Name: "testsrc", Desc: "generate a test pattern", Source: true, Video: true},
		{Name: "sine", Desc: "generate a sine wave", Source: true, Audio: true},
		{Name: "color", Desc: "provide an uniformly colored input", Source: true, Video: true},
		{Name: "nullsrc", Desc: "null video source", Source: true, Video: true},
		{Name: "anullsrc", Desc: "null audio source", Source: true, Audio: true},
		{Name: "scale", Desc: "scale the input video", Video: true},
		{Name: "format", Desc: "convert the video pixel format", Video: true},
		{Name: "crop", Desc: "crop the input video", Video: true},
		{Name: "pad", Desc: "pad the input video", Video: true},
		{Name: "hflip", Desc: "horizontally flip the input video", Video: true},
		{Name: "negate", Desc: "negate video", Video: true},
		{Name: "vflip", Desc: "vertically flip the input video", Video: true},
		{Name: "transpose", Desc: "transpose the input video", Video: true},
		{Name: "fps", Desc: "force constant framerate", Video: true},
		{Name: "trim", Desc: "pick a video interval", Video: true},
		{Name: "atrim", Desc: "pick an audio interval", Audio: true},
		{Name: "volume", Desc: "change audio volume", Audio: true},
		{Name: "aresample", Desc: "resample audio", Audio: true},
		{Name: "aformat", Desc: "convert audio format", Audio: true},
		{Name: "pan", Desc: "remix audio channels", Audio: true},
		{Name: "setsar", Desc: "set the sample aspect ratio", Video: true},
		{Name: "setpts", Desc: "set the presentation timestamp", Video: true, Audio: true},
		{Name: "null", Desc: "pass the video source unchanged", Video: true},
		{Name: "anull", Desc: "pass the audio source unchanged", Audio: true},
	}
}

// Graph is a parsed filtergraph.
type Graph struct {
	def     Link
	links   map[string]Link
	inputs  map[string]*feed
	outputs []string
	sws     swscale.Flags
}

// Parse compiles a filtergraph. sws flags are the scale default.
func Parse(spec string, sws swscale.Flags) (*Graph, error) {
	g := &Graph{links: map[string]Link{}, inputs: map[string]*feed{}, sws: sws}
	var unlabeled []Link
	for _, chain := range splitTop(spec, ';') {
		chain = strings.TrimSpace(chain)
		if chain == "" {
			continue
		}
		var cur Link
		for _, part := range splitTop(chain, ',') {
			item, err := parseFilter(part)
			if err != nil {
				return nil, err
			}
			built, err := g.make(item.name, item.args)
			if err != nil {
				return nil, err
			}
			var next Link
			if built.src != nil {
				next = built.src
			} else {
				var up Link
				if len(item.ins) > 0 {
					up = g.resolve(item.ins[0])
				} else if cur != nil {
					up = cur
				} else {
					up = g.resolve("in")
				}
				next = &stage{up: up, run: built.run}
			}
			cur = next
			for _, label := range item.outs {
				g.links[label] = next
				g.outputs = append(g.outputs, label)
			}
		}
		if cur != nil {
			unlabeled = append(unlabeled, cur)
		}
	}
	if len(unlabeled) > 0 {
		g.def = unlabeled[len(unlabeled)-1]
	} else if len(g.links) > 0 {
		for _, link := range g.links {
			g.def = link
			break
		}
	}
	if g.def == nil {
		return nil, fmt.Errorf("%w: empty graph", avutil.ErrOption)
	}
	return g, nil
}

// Labels are the named outputs, in graph order.
func (g *Graph) Labels() []string { return append([]string(nil), g.outputs...) }

// InputLabels are pads the graph reads, such as 0:v or in.
func (g *Graph) InputLabels() []string {
	labels := make([]string, 0, len(g.inputs))
	for name := range g.inputs {
		labels = append(labels, name)
	}
	return labels
}

// SetInput queues frames for a labeled input pad such as 0:v or in.
func (g *Graph) SetInput(label string, frames []*avutil.Frame) {
	feed := g.resolve(label).(*feed)
	feed.frames = frames
	feed.i = 0
}

// Next pulls the default output.
func (g *Graph) Next() (*avutil.Frame, error) { return g.def.Next() }

// NextLabel pulls one named output.
func (g *Graph) NextLabel(label string) (*avutil.Frame, error) {
	link, ok := g.links[label]
	if !ok {
		return nil, fmt.Errorf("%w: output %s", avutil.ErrOption, label)
	}
	return link.Next()
}

// Filter pushes one frame through a chain and returns the frames it emits.
func (g *Graph) Filter(in *avutil.Frame) ([]*avutil.Frame, error) {
	feed, ok := g.inputs["in"]
	if !ok {
		return nil, fmt.Errorf("%w: filter input", avutil.ErrOption)
	}
	feed.frames = []*avutil.Frame{in}
	feed.i = 0
	var out []*avutil.Frame
	for {
		frame, err := g.def.Next()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, frame)
	}
}

type built struct {
	src Link
	run func(*avutil.Frame) ([]*avutil.Frame, error)
}

func (g *Graph) make(name, args string) (built, error) {
	parsed := parseArgs(args)
	switch name {
	case "testsrc":
		src, err := newTestSrc(parsed)
		return built{src: src}, err
	case "sine":
		src, err := newSine(parsed)
		return built{src: src}, err
	case "color":
		src, err := newColor(parsed, g.sws)
		return built{src: src}, err
	case "nullsrc":
		src, err := newNullSrc(parsed)
		return built{src: src}, err
	case "anullsrc":
		src, err := newAnull(parsed)
		return built{src: src}, err
	case "scale":
		return built{run: newScale(parsed, g.sws)}, nil
	case "format":
		return built{run: newFormat(parsed, g.sws)}, nil
	case "crop":
		return built{run: newCrop(parsed, g.sws)}, nil
	case "pad":
		return built{run: newPad(parsed, g.sws)}, nil
	case "hflip":
		return built{run: newFlip(false, g.sws)}, nil
	case "negate":
		return built{run: newNegate(parsed)}, nil
	case "vflip":
		return built{run: newFlip(true, g.sws)}, nil
	case "transpose":
		run := newTranspose(parsed, g.sws)
		if run == nil {
			return built{}, fmt.Errorf("%w: transpose", avutil.ErrOption)
		}
		return built{run: run}, nil
	case "fps":
		return built{run: newFPS(parsed)}, nil
	case "trim":
		return built{run: newTrim(false, parsed)}, nil
	case "atrim":
		return built{run: newTrim(true, parsed)}, nil
	case "volume":
		return built{run: newVolume(parsed)}, nil
	case "aresample":
		return built{run: newAresample(parsed)}, nil
	case "aformat":
		return built{run: newAformat(parsed)}, nil
	case "pan":
		return built{run: newPan(parsed)}, nil
	case "setsar":
		return built{run: newSetSAR(parsed)}, nil
	case "setpts":
		return built{run: newSetPTS(parsed)}, nil
	case "null", "anull":
		return built{run: newNull()}, nil
	default:
		return built{}, badFilter(name)
	}
}

func (g *Graph) resolve(label string) Link {
	if link, ok := g.links[label]; ok {
		return link
	}
	if feed, ok := g.inputs[label]; ok {
		return feed
	}
	feed := &feed{}
	g.inputs[label] = feed
	return feed
}

type feed struct {
	frames []*avutil.Frame
	i      int
}

func (f *feed) Next() (*avutil.Frame, error) {
	if f.i >= len(f.frames) {
		return nil, io.EOF
	}
	frame := f.frames[f.i]
	f.i++
	return frame, nil
}

type stage struct {
	up  Link
	run func(*avutil.Frame) ([]*avutil.Frame, error)
	q   []*avutil.Frame
}

func (s *stage) Next() (*avutil.Frame, error) {
	for len(s.q) == 0 {
		in, err := s.up.Next()
		if err != nil {
			return nil, err
		}
		out, err := s.run(in)
		if err != nil {
			return nil, err
		}
		if len(out) == 0 {
			continue
		}
		s.q = out
	}
	frame := s.q[0]
	s.q = s.q[1:]
	return frame, nil
}
