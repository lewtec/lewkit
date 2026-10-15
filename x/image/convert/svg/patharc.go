package svg

import (
	"fmt"
	"math"
)

func appendQuad(cmds []PathCmd, x0, y0, x1, y1, x, y float64) ([]PathCmd, error) {
	return appendCubic(cmds,
		x0+(2.0/3.0)*(x1-x0), y0+(2.0/3.0)*(y1-y0),
		x+(2.0/3.0)*(x1-x), y+(2.0/3.0)*(y1-y),
		x, y,
	)
}

func appendArc(cmds []PathCmd, x0, y0, rx, ry, rot float64, large, sweep bool, x, y float64) ([]PathCmd, error) {
	if x0 == x && y0 == y {
		return cmds, nil
	}
	arc, ok := svgArcToArc(x0, y0, x, y, rx, ry, rot*math.Pi/180, large, sweep)
	if !ok {
		return appendLine(cmds, x, y)
	}
	n := len(cmds)
	var err error
	arc.toCubics(0.1, func(x1, y1, x2, y2, x3, y3 float64) {
		if err != nil {
			return
		}
		cmds, err = appendCubic(cmds, x1, y1, x2, y2, x3, y3)
	})
	if err != nil {
		return nil, err
	}
	if len(cmds) == n {
		return appendLine(cmds, x, y)
	}
	return cmds, nil
}

func appendLine(cmds []PathCmd, x, y float64) ([]PathCmd, error) {
	if len(cmds)+1 > maxPathCmds {
		return nil, fmt.Errorf("path: more than %d commands", maxPathCmds)
	}
	return append(cmds, PathCmd{Kind: CmdLine, X: x, Y: y}), nil
}

func appendCubic(cmds []PathCmd, x1, y1, x2, y2, x, y float64) ([]PathCmd, error) {
	if len(cmds)+1 > maxPathCmds {
		return nil, fmt.Errorf("path: more than %d commands", maxPathCmds)
	}
	return append(cmds, PathCmd{Kind: CmdCubic, X: x, Y: y, X1: x1, Y1: y1, X2: x2, Y2: y2}), nil
}

func svgArcToArc(x0, y0, x1, y1, rx, ry, xrot float64, large, sweep bool) (ellArc, bool) {
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx <= 1e-5 || ry <= 1e-5 || (x0 == x1 && y0 == y1) {
		return ellArc{}, false
	}
	sinPhi, cosPhi := math.Sincos(math.Mod(xrot, 2*math.Pi))
	hdx := (x0 - x1) * 0.5
	hdy := (y0 - y1) * 0.5
	px := cosPhi*hdx + sinPhi*hdy
	py := -sinPhi*hdx + cosPhi*hdy
	rf := px*px/(rx*rx) + py*py/(ry*ry)
	if rf > 1 {
		s := math.Sqrt(rf)
		rx *= s
		ry *= s
	}
	rxpy := rx * py
	rypx := ry * px
	sum := rxpy*rxpy + rypx*rypx
	if sum == 0 {
		return ellArc{}, false
	}
	sign := 1.0
	if large == sweep {
		sign = -1
	}
	rxry := rx * ry
	coe := sign * math.Sqrt(math.Abs((rxry*rxry-sum)/sum))
	tcx := coe * rxpy / ry
	tcy := -coe * rypx / rx
	cx := cosPhi*tcx - sinPhi*tcy + (x0+x1)*0.5
	cy := sinPhi*tcx + cosPhi*tcy + (y0+y1)*0.5
	startV := [2]float64{(px - tcx) / rx, (py - tcy) / ry}
	endV := [2]float64{(-px - tcx) / rx, (-py - tcy) / ry}
	start := math.Atan2(startV[1], startV[0])
	sweepA := math.Mod(math.Atan2(endV[1], endV[0])-start, 2*math.Pi)
	if sweep && sweepA < 0 {
		sweepA += 2 * math.Pi
	} else if !sweep && sweepA > 0 {
		sweepA -= 2 * math.Pi
	}
	return ellArc{cx, cy, rx, ry, start, sweepA, xrot}, true
}

type ellArc struct {
	cx, cy, rx, ry float64
	start, sweep   float64
	xrot           float64
}

func (a ellArc) toCubics(tol float64, emit func(x1, y1, x2, y2, x3, y3 float64)) {
	sign := 1.0
	if a.sweep < 0 {
		sign = -1
	}
	scaled := math.Max(a.rx, a.ry) / tol
	nErr := math.Max(math.Pow(1.1163*scaled, 1.0/6.0), 3.999999)
	n := math.Ceil(nErr * math.Abs(a.sweep) / (2 * math.Pi))
	if n < 1 {
		return
	}
	step := a.sweep / n
	ni := int(n)
	arm := (4.0 / 3.0) * math.Tan(math.Abs(0.25*step)) * sign
	ang0 := a.start
	p0x, p0y := sampleEllipse(a.rx, a.ry, a.xrot, ang0)
	for i := 0; i < ni; i++ {
		ang1 := ang0 + step
		q1x, q1y := sampleEllipse(a.rx, a.ry, a.xrot, ang0+math.Pi/2)
		p1x, p1y := p0x+arm*q1x, p0y+arm*q1y
		p3x, p3y := sampleEllipse(a.rx, a.ry, a.xrot, ang1)
		q2x, q2y := sampleEllipse(a.rx, a.ry, a.xrot, ang1+math.Pi/2)
		p2x, p2y := p3x-arm*q2x, p3y-arm*q2y
		emit(a.cx+p1x, a.cy+p1y, a.cx+p2x, a.cy+p2y, a.cx+p3x, a.cy+p3y)
		ang0, p0x, p0y = ang1, p3x, p3y
	}
}

func sampleEllipse(rx, ry, rot, ang float64) (float64, float64) {
	s, c := math.Sincos(ang)
	u, v := rx*c, ry*s
	rs, rc := math.Sincos(rot)
	return u*rc - v*rs, u*rs + v*rc
}
