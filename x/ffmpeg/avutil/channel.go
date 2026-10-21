package avutil

import (
	"fmt"
	"strconv"
	"strings"
)

// Channel bits follow FFmpeg's AV_CH_* order.
const (
	ChanFrontLeft          uint64 = 1 << 0
	ChanFrontRight         uint64 = 1 << 1
	ChanFrontCenter        uint64 = 1 << 2
	ChanLowFrequency       uint64 = 1 << 3
	ChanBackLeft           uint64 = 1 << 4
	ChanBackRight          uint64 = 1 << 5
	ChanFrontLeftOfCenter  uint64 = 1 << 6
	ChanFrontRightOfCenter uint64 = 1 << 7
	ChanBackCenter         uint64 = 1 << 8
	ChanSideLeft           uint64 = 1 << 9
	ChanSideRight          uint64 = 1 << 10
)

// ChannelLayout is a mask plus the channel count the mask describes.
type ChannelLayout struct {
	Mask  uint64
	Count int
}

// Channels reports how many channels the layout carries.
func (layout ChannelLayout) Channels() int {
	if layout.Count > 0 {
		return layout.Count
	}
	return bitsOnes(layout.Mask)
}

func bitsOnes(mask uint64) int {
	n := 0
	for mask != 0 {
		n += int(mask & 1)
		mask >>= 1
	}
	return n
}

// Mono is FC.
func Mono() ChannelLayout { return ChannelLayout{Mask: ChanFrontCenter, Count: 1} }

// Stereo is FL+FR.
func Stereo() ChannelLayout {
	return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight, Count: 2}
}

func preset(name LayoutName) ChannelLayout {
	switch name {
	case LayoutMono:
		return Mono()
	case LayoutStereo:
		return Stereo()
	case LayoutTwoOne:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanLowFrequency, Count: 3}
	case LayoutThree:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanFrontCenter, Count: 3}
	case LayoutThreeOne:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanFrontCenter | ChanLowFrequency, Count: 4}
	case LayoutQuad:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanBackLeft | ChanBackRight, Count: 4}
	case LayoutQuadSide:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanSideLeft | ChanSideRight, Count: 4}
	case LayoutFive:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanFrontCenter | ChanBackLeft | ChanBackRight, Count: 5}
	case LayoutFiveSide:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanFrontCenter | ChanSideLeft | ChanSideRight, Count: 5}
	case LayoutFiveOne:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanFrontCenter | ChanLowFrequency | ChanBackLeft | ChanBackRight, Count: 6}
	case LayoutFiveOneSide:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanFrontCenter | ChanLowFrequency | ChanSideLeft | ChanSideRight, Count: 6}
	case LayoutSixOne:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanFrontCenter | ChanLowFrequency | ChanBackCenter | ChanSideLeft | ChanSideRight, Count: 7}
	case LayoutSevenOne:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanFrontCenter | ChanLowFrequency | ChanBackLeft | ChanBackRight | ChanSideLeft | ChanSideRight, Count: 8}
	case LayoutSevenOneWide:
		return ChannelLayout{Mask: ChanFrontLeft | ChanFrontRight | ChanFrontCenter | ChanLowFrequency | ChanBackLeft | ChanBackRight | ChanFrontLeftOfCenter | ChanFrontRightOfCenter, Count: 8}
	default:
		return ChannelLayout{}
	}
}

// ParseLayout reads a preset, a channel count, or a 0x mask.
func ParseLayout(text string) (ChannelLayout, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return ChannelLayout{}, fmt.Errorf("%w: empty channel layout", ErrOption)
	}
	if name, err := ParseLayoutName(text); err == nil {
		return preset(name), nil
	}
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		mask, err := strconv.ParseUint(text, 0, 64)
		if err != nil {
			return ChannelLayout{}, fmt.Errorf("%w: %s", ErrOption, text)
		}
		return ChannelLayout{Mask: mask, Count: bitsOnes(mask)}, nil
	}
	count, err := strconv.Atoi(text)
	if err != nil || count <= 0 || count > 64 {
		return ChannelLayout{}, fmt.Errorf("%w: %s", ErrOption, text)
	}
	if count == 1 {
		return Mono(), nil
	}
	if count == 2 {
		return Stereo(), nil
	}
	var mask uint64
	for i := 0; i < count; i++ {
		mask |= 1 << i
	}
	return ChannelLayout{Mask: mask, Count: count}, nil
}

// LayoutString is the preset name, or the channel count when no preset matches.
func LayoutString(layout ChannelLayout) string {
	for _, name := range LayoutName(0).Values() {
		if preset(name) == layout {
			return name.String()
		}
	}
	if layout.Mask != 0 && layout.Channels() == bitsOnes(layout.Mask) {
		return fmt.Sprintf("0x%x", layout.Mask)
	}
	return strconv.Itoa(layout.Channels())
}
