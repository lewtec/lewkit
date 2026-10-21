package avutil

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// ParseRate reads a frame rate: a ratio, a number, or a name such as ntsc.
func ParseRate(text string) (Rational, error) {
	text = strings.TrimSpace(strings.ToLower(text))
	switch text {
	case "ntsc", "qntsc", "sntsc":
		return Q(30000, 1001), nil
	case "pal", "qpal", "spal":
		return Q(25, 1), nil
	case "film":
		return Q(24, 1), nil
	case "ntsc-film":
		return Q(24000, 1001), nil
	}
	if num, den, ok := strings.Cut(text, "/"); ok {
		n, errN := strconv.ParseInt(strings.TrimSpace(num), 10, 32)
		d, errD := strconv.ParseInt(strings.TrimSpace(den), 10, 32)
		if errN != nil || errD != nil || d == 0 {
			return Rational{}, fmt.Errorf("%w: rate %s", ErrOption, text)
		}
		r, _ := Reduce(n, d, math.MaxInt32)
		return r, nil
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || value <= 0 {
		return Rational{}, fmt.Errorf("%w: rate %s", ErrOption, text)
	}
	r, _ := Reduce(int64(math.Round(value*1000)), 1000, 1<<30)
	if r.Den == 0 {
		return Rational{}, fmt.Errorf("%w: rate %s", ErrOption, text)
	}
	return r, nil
}

// ParseSize reads WxH or a name such as vga or hd720.
func ParseSize(text string) (int, int, error) {
	text = strings.TrimSpace(strings.ToLower(text))
	if width, height, ok := sizeName(text); ok {
		return width, height, nil
	}
	sep := "x"
	if !strings.Contains(text, "x") && strings.Contains(text, "X") {
		sep = "X"
	}
	left, right, ok := strings.Cut(text, sep)
	if !ok {
		return 0, 0, fmt.Errorf("%w: size %s", ErrOption, text)
	}
	width, errW := strconv.Atoi(strings.TrimSpace(left))
	height, errH := strconv.Atoi(strings.TrimSpace(right))
	if errW != nil || errH != nil || width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("%w: size %s", ErrOption, text)
	}
	return width, height, nil
}

func sizeName(text string) (int, int, bool) {
	sizes := map[string][2]int{
		"ntsc": {720, 480}, "pal": {720, 576}, "qntsc": {352, 240}, "qpal": {352, 288},
		"sntsc": {640, 480}, "spal": {768, 576}, "film": {352, 240}, "ntsc-film": {352, 240},
		"sqcif": {128, 96}, "qcif": {176, 144}, "cif": {352, 288}, "4cif": {704, 576},
		"16cif": {1408, 1152}, "qqvga": {160, 120}, "qvga": {320, 240}, "vga": {640, 480},
		"svga": {800, 600}, "xga": {1024, 768}, "uxga": {1600, 1200}, "qxga": {2048, 1536},
		"sxga": {1280, 1024}, "qsxga": {2560, 2048}, "hsxga": {5120, 4096}, "wvga": {852, 480},
		"wxga": {1366, 768}, "wsxga": {1600, 1024}, "wuxga": {1920, 1200}, "woxga": {2560, 1600},
		"wqhd": {2560, 1440}, "wqsxga": {3200, 2048}, "wquxga": {3840, 2400}, "whsxga": {6400, 4096},
		"whuxga": {7680, 4800}, "cga": {320, 200}, "ega": {640, 350}, "hd480": {852, 480},
		"hd720": {1280, 720}, "hd1080": {1920, 1080}, "quadhd": {2560, 1440}, "2k": {2048, 1080},
		"2kdci": {2048, 1080}, "2kflat": {1998, 1080}, "2kscope": {2048, 858}, "4k": {4096, 2160},
		"4kdci": {4096, 2160}, "4kflat": {3996, 2160}, "4kscope": {4096, 1716}, "nhd": {640, 360},
		"hqvga": {240, 160}, "wqvga": {400, 240}, "fwqvga": {432, 240}, "hvga": {480, 320},
		"qhd": {960, 540}, "uhd2160": {3840, 2160}, "uhd4320": {7680, 4320},
	}
	pair, ok := sizes[text]
	if !ok {
		return 0, 0, false
	}
	return pair[0], pair[1], true
}

// ParseTimeUS reads a timestamp into microseconds.
// Forms are seconds, or hh:mm:ss[.frac].
func ParseTimeUS(text string) (int64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("%w: empty time", ErrOption)
	}
	sign := int64(1)
	if text[0] == '-' {
		sign = -1
		text = text[1:]
	} else if text[0] == '+' {
		text = text[1:]
	}
	if strings.Contains(text, ":") {
		us, err := parseClock(text)
		if err != nil {
			return 0, err
		}
		return sign * us, nil
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: time %s", ErrOption, text)
	}
	return sign * int64(math.Round(value*1e6)), nil
}

func parseClock(text string) (int64, error) {
	parts := strings.Split(text, ":")
	if len(parts) > 3 {
		return 0, fmt.Errorf("%w: time %s", ErrOption, text)
	}
	var fields [3]float64
	for i := range parts {
		value, err := strconv.ParseFloat(parts[len(parts)-1-i], 64)
		if err != nil {
			return 0, fmt.Errorf("%w: time %s", ErrOption, text)
		}
		fields[i] = value
	}
	seconds := fields[0] + fields[1]*60 + fields[2]*3600
	return int64(math.Round(seconds * 1e6)), nil
}

// ParseColor reads a name or a 0xRRGGBB / #RRGGBB value into RGBA.
func ParseColor(text string) ([4]byte, error) {
	text = strings.TrimSpace(strings.ToLower(text))
	if text == "" {
		return [4]byte{}, fmt.Errorf("%w: empty color", ErrOption)
	}
	named := map[string][4]byte{
		"black": {0, 0, 0, 255}, "white": {255, 255, 255, 255}, "red": {255, 0, 0, 255},
		"green": {0, 128, 0, 255}, "blue": {0, 0, 255, 255}, "yellow": {255, 255, 0, 255},
		"cyan": {0, 255, 255, 255}, "magenta": {255, 0, 255, 255}, "gray": {128, 128, 128, 255},
		"grey": {128, 128, 128, 255}, "orange": {255, 165, 0, 255}, "purple": {128, 0, 128, 255},
		"brown": {165, 42, 42, 255}, "pink": {255, 192, 203, 255}, "lime": {0, 255, 0, 255},
		"navy": {0, 0, 128, 255}, "teal": {0, 128, 128, 255}, "olive": {128, 128, 0, 255},
		"maroon": {128, 0, 0, 255}, "silver": {192, 192, 192, 255},
	}
	if color, ok := named[text]; ok {
		return color, nil
	}
	hex := strings.TrimPrefix(text, "0x")
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 && len(hex) != 8 {
		return [4]byte{}, fmt.Errorf("%w: color %s", ErrOption, text)
	}
	for _, r := range hex {
		if !unicode.Is(unicode.ASCII_Hex_Digit, r) {
			return [4]byte{}, fmt.Errorf("%w: color %s", ErrOption, text)
		}
	}
	value, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return [4]byte{}, fmt.Errorf("%w: color %s", ErrOption, text)
	}
	if len(hex) == 6 {
		return [4]byte{byte(value >> 16), byte(value >> 8), byte(value), 255}, nil
	}
	return [4]byte{byte(value >> 24), byte(value >> 16), byte(value >> 8), byte(value)}, nil
}

// ParseSI reads an integer with an optional k, M, or G suffix.
func ParseSI(text string) (int64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("%w: empty number", ErrOption)
	}
	mult := int64(1)
	last := text[len(text)-1]
	switch last {
	case 'k', 'K':
		mult = 1000
		text = text[:len(text)-1]
	case 'm', 'M':
		mult = 1000 * 1000
		text = text[:len(text)-1]
	case 'g', 'G':
		mult = 1000 * 1000 * 1000
		text = text[:len(text)-1]
	}
	if strings.Contains(text, ".") {
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %s", ErrOption, text)
		}
		return int64(math.Round(value * float64(mult))), nil
	}
	value, err := strconv.ParseInt(text, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrOption, text)
	}
	return value * mult, nil
}
