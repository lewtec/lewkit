package swscale

import (
	"math"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
)

type kernel int

const (
	kernelPoint kernel = iota
	kernelBilinear
	kernelBicubic
	kernelLanczos
	kernelArea
	kernelGauss
)

func kernelOf(flags Flags) kernel {
	switch {
	case flags&Point != 0:
		return kernelPoint
	case flags&Area != 0:
		return kernelArea
	case flags&Lanczos != 0:
		return kernelLanczos
	case flags&Gauss != 0:
		return kernelGauss
	case flags&(Bilinear|FastBilinear) != 0:
		return kernelBilinear
	default:
		return kernelBicubic
	}
}

func scalePlanar(dst, src *avutil.Frame, kind kernel) error {
	planes := 3
	if dst.Format == avutil.PixGray {
		planes = 1
	}
	for plane := 0; plane < planes; plane++ {
		scalePlane(dst, src, plane, kind)
	}
	return nil
}

func scalePlane(dst, src *avutil.Frame, plane int, kind kernel) {
	dw, dh := dst.Width, dst.Height
	sw, sh := src.Width, src.Height
	for y := 0; y < dh; y++ {
		row := dst.PlaneRow(plane, y, dw)
		fy := float64(y) + 0.5
		sy := fy*float64(sh)/float64(dh) - 0.5
		for x := 0; x < dw && x < len(row); x++ {
			fx := float64(x) + 0.5
			sx := fx*float64(sw)/float64(dw) - 0.5
			row[x] = samplePlane(src, plane, sx, sy, sw, sh, kind)
		}
	}
}

func samplePlane(src *avutil.Frame, plane int, sx, sy float64, sw, sh int, kind kernel) byte {
	switch kind {
	case kernelPoint:
		return at(src, plane, int(math.Round(sx)), int(math.Round(sy)), sw, sh)
	case kernelBilinear:
		return bilinear(src, plane, sx, sy, sw, sh)
	case kernelArea:
		return area(src, plane, sx, sy, sw, sh)
	case kernelLanczos:
		return taps(src, plane, sx, sy, sw, sh, 3, lanczos)
	case kernelGauss:
		return taps(src, plane, sx, sy, sw, sh, 3, gauss)
	default:
		return taps(src, plane, sx, sy, sw, sh, 2, bicubic)
	}
}

func at(src *avutil.Frame, plane, x, y, w, h int) byte {
	if x < 0 {
		x = 0
	} else if x >= w {
		x = w - 1
	}
	if y < 0 {
		y = 0
	} else if y >= h {
		y = h - 1
	}
	row := src.PlaneRow(plane, y, w)
	if x >= len(row) {
		return 0
	}
	return row[x]
}

func bilinear(src *avutil.Frame, plane int, sx, sy float64, sw, sh int) byte {
	x0 := int(math.Floor(sx))
	y0 := int(math.Floor(sy))
	fx := sx - float64(x0)
	fy := sy - float64(y0)
	if fx < 0 {
		fx = 0
	}
	if fy < 0 {
		fy = 0
	}
	a := float64(at(src, plane, x0, y0, sw, sh))
	b := float64(at(src, plane, x0+1, y0, sw, sh))
	c := float64(at(src, plane, x0, y0+1, sw, sh))
	d := float64(at(src, plane, x0+1, y0+1, sw, sh))
	top := a*(1-fx) + b*fx
	bot := c*(1-fx) + d*fx
	return avutil.ClipUint8(int(math.Round(top*(1-fy) + bot*fy)))
}

func area(src *avutil.Frame, plane int, sx, sy float64, sw, sh int) byte {
	return bilinear(src, plane, sx, sy, sw, sh)
}

func taps(src *avutil.Frame, plane int, sx, sy float64, sw, sh, radius int, weight func(float64) float64) byte {
	x0 := int(math.Floor(sx))
	y0 := int(math.Floor(sy))
	var sum, norm float64
	for yy := y0 - radius + 1; yy <= y0+radius; yy++ {
		wy := weight(sy - float64(yy))
		for xx := x0 - radius + 1; xx <= x0+radius; xx++ {
			w := wy * weight(sx-float64(xx))
			sum += w * float64(at(src, plane, xx, yy, sw, sh))
			norm += w
		}
	}
	if norm == 0 {
		return at(src, plane, x0, y0, sw, sh)
	}
	return avutil.ClipUint8(int(math.Round(sum / norm)))
}

func bicubic(x float64) float64 {
	x = math.Abs(x)
	if x < 1 {
		return (1.5*x-2.5)*x*x + 1
	}
	if x < 2 {
		return ((-0.5*x+2.5)*x-4)*x + 2
	}
	return 0
}

func lanczos(x float64) float64 {
	x = math.Abs(x)
	if x < 1e-8 {
		return 1
	}
	if x >= 3 {
		return 0
	}
	return 3 * math.Sin(math.Pi*x) * math.Sin(math.Pi*x/3) / (math.Pi * math.Pi * x * x)
}

func gauss(x float64) float64 {
	return math.Exp(-2 * x * x)
}
