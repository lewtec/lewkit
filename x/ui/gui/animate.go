package gui

import (
	"math"
	"time"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Curve maps a unit progress [0, 1] onto an eased progress.
// A nil Curve is [Linear].
type Curve func(float32) float32

// Transform evaluates curve at t, clamping t to [0, 1] first.
func (curve Curve) Transform(t float32) float32 {
	t = clamp01(t)
	if curve == nil {
		return t
	}
	return curve(t)
}

var (
	Linear         Curve = func(t float32) float32 { return t }
	Ease           Curve = Cubic(0.25, 0.1, 0.25, 1.0)
	EaseIn         Curve = Cubic(0.42, 0.0, 1.0, 1.0)
	EaseOut        Curve = Cubic(0.0, 0.0, 0.58, 1.0)
	EaseInOut      Curve = Cubic(0.42, 0.0, 0.58, 1.0)
	FastOutSlowIn  Curve = Cubic(0.4, 0.0, 0.2, 1.0)
	EaseOutCubic   Curve = Cubic(0.215, 0.61, 0.355, 1.0)
	EaseInOutCubic Curve = Cubic(0.645, 0.045, 0.355, 1.0)
	EaseOutQuart   Curve = Cubic(0.165, 0.84, 0.44, 1.0)
	EaseOutBack    Curve = Cubic(0.175, 0.885, 0.32, 1.275)
)

// Cubic returns a cubic Bézier curve with control points (a, b) and (c, d).
func Cubic(a, b, c, d float32) Curve {
	return func(t float32) float32 {
		if t <= 0 {
			return 0
		}
		if t >= 1 {
			return 1
		}
		low := float32(0)
		high := float32(1)
		mid := t
		for range 16 {
			x := cubicEval(a, c, mid)
			if math.Abs(float64(x-t)) < 1e-4 {
				break
			}
			if t > x {
				low = mid
			} else {
				high = mid
			}
			mid = (low + high) * 0.5
		}
		return cubicEval(b, d, mid)
	}
}

func cubicEval(p1, p2, m float32) float32 {
	u := 1 - m
	return 3*u*u*m*p1 + 3*u*m*m*p2 + m*m*m
}

func clamp01(t float32) float32 {
	if t <= 0 || math.IsNaN(float64(t)) {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t
}

// Lerp linearly interpolates from a to b by t.
func Lerp(a, b, t float32) float32 {
	return a + (b-a)*t
}

// LerpRGB linearly interpolates each channel of a and b by t.
func LerpRGB(a, b RGB, t float32) RGB {
	t = clamp01(t)
	return RGB{
		Red:   lerpByte(a.Red, b.Red, t),
		Green: lerpByte(a.Green, b.Green, t),
		Blue:  lerpByte(a.Blue, b.Blue, t),
		Alpha: lerpByte(a.Alpha, b.Alpha, t),
	}
}

func lerpByte(a, b uint8, t float32) uint8 {
	v := float32(a) + (float32(b)-float32(a))*t
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}

// LerpOffset linearly interpolates between two offsets.
func LerpOffset(a, b Offset, t float32) Offset {
	return Offset{
		X: Lerp(a.X, b.X, t),
		Y: Lerp(a.Y, b.Y, t),
	}
}

// LerpSize linearly interpolates between two sizes.
func LerpSize(a, b Size, t float32) Size {
	return Size{
		Width:  Lerp(a.Width, b.Width, t),
		Height: Lerp(a.Height, b.Height, t),
	}
}

// LerpInsets linearly interpolates between two [EdgeInsets].
func LerpInsets(a, b EdgeInsets, t float32) EdgeInsets {
	return EdgeInsets{
		Left:   Lerp(a.Left, b.Left, t),
		Top:    Lerp(a.Top, b.Top, t),
		Right:  Lerp(a.Right, b.Right, t),
		Bottom: Lerp(a.Bottom, b.Bottom, t),
	}
}

// Tween drives a scalar value from a starting point to a target over a duration.
type Tween struct {
	from     float32
	to       float32
	value    float32
	elapsed  time.Duration
	duration time.Duration
	curve    Curve
	active   bool
}

// NewTween returns a settled [Tween] at initial.
func NewTween(initial float32) Tween {
	return Tween{from: initial, to: initial, value: initial}
}

// Set snaps the tween to v and stops any in-flight motion.
func (tween *Tween) Set(v float32) {
	if tween == nil {
		return
	}
	tween.from = v
	tween.to = v
	tween.value = v
	tween.elapsed = 0
	tween.duration = 0
	tween.active = false
}

// To starts a transition from the current value to target over duration using curve.
// If the tween is already targeting target, it continues without restarting.
func (tween *Tween) To(target float32, duration time.Duration, curve Curve) {
	if tween == nil {
		return
	}
	if duration <= 0 {
		tween.Set(target)
		return
	}
	if tween.active && tween.to == target && tween.duration == duration {
		return
	}
	if !tween.active && tween.value == target {
		return
	}
	tween.from = tween.value
	tween.to = target
	tween.elapsed = 0
	tween.duration = duration
	tween.curve = curve
	tween.active = true
}

// FromTo starts a transition explicitly from start to target over duration.
func (tween *Tween) FromTo(start, target float32, duration time.Duration, curve Curve) {
	if tween == nil {
		return
	}
	tween.Set(start)
	tween.To(target, duration, curve)
}

// Step advances the tween by dt and returns true while still animating.
func (tween *Tween) Step(dt time.Duration) bool {
	if tween == nil || !tween.active {
		return false
	}
	if dt > 0 {
		tween.elapsed += dt
	}
	if tween.duration <= 0 || tween.elapsed >= tween.duration {
		tween.value = tween.to
		tween.from = tween.to
		tween.elapsed = tween.duration
		tween.active = false
		return false
	}
	raw := float32(tween.elapsed) / float32(tween.duration)
	eased := tween.curve.Transform(raw)
	tween.value = Lerp(tween.from, tween.to, eased)
	return true
}

// Value returns the current interpolated scalar.
func (tween Tween) Value() float32 { return tween.value }

// Target returns the destination scalar.
func (tween Tween) Target() float32 { return tween.to }

// Animating reports whether the tween has not yet reached its target.
func (tween Tween) Animating() bool { return tween.active }

// Progress returns the raw unit progress in [0, 1].
func (tween Tween) Progress() float32 {
	if !tween.active || tween.duration <= 0 {
		return 1
	}
	return clamp01(float32(tween.elapsed) / float32(tween.duration))
}

// Spring simulates a damped harmonic oscillator towards Target.
// Zero Stiffness or Damping uses a critically damped default (Stiffness 260, Damping 32).
type Spring struct {
	Value     float32
	Velocity  float32
	Target    float32
	Stiffness float32
	Damping   float32
}

// Set snaps the spring to v with zero velocity.
func (spring *Spring) Set(v float32) {
	if spring == nil {
		return
	}
	spring.Value = v
	spring.Target = v
	spring.Velocity = 0
}

// Animating reports whether the spring is still in motion.
func (spring Spring) Animating() bool {
	return math.Abs(float64(spring.Target-spring.Value)) > 0.25 || math.Abs(float64(spring.Velocity)) > 2.0
}

// Step advances the spring by dt and returns true while still in motion.
func (spring *Spring) Step(dt time.Duration) bool {
	if spring == nil {
		return false
	}
	if !spring.Animating() {
		spring.Value = spring.Target
		spring.Velocity = 0
		return false
	}
	seconds := float32(dt.Seconds())
	if seconds <= 0 {
		return true
	}
	if seconds > 0.064 {
		seconds = 0.064
	}
	stiffness := spring.Stiffness
	if stiffness <= 0 {
		stiffness = 260
	}
	damping := spring.Damping
	if damping <= 0 {
		damping = 32
	}
	const maxStep = float32(1.0 / 240.0)
	for seconds > 0 {
		step := min(seconds, maxStep)
		displacement := spring.Value - spring.Target
		force := -stiffness*displacement - damping*spring.Velocity
		spring.Velocity += force * step
		spring.Value += spring.Velocity * step
		seconds -= step
	}
	if !spring.Animating() {
		spring.Value = spring.Target
		spring.Velocity = 0
		return false
	}
	return true
}

// Opacity scales the alpha channel of every fill and glyph mark its Child paints.
// Value is clamped to [0, 1]; at 0 the child is laid out but not painted.
type Opacity struct {
	Value float32
	Child Node
}

func (node *Opacity) Layout(constraints BoxConstraints) Size {
	if node == nil || node.Child == nil {
		return Size{}
	}
	return node.Child.Layout(constraints)
}

func (node *Opacity) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if node == nil || node.Child == nil {
		return accumulatorOf(picture)
	}
	alpha := clamp01(node.Value)
	if alpha <= 0 {
		return accumulatorOf(picture)
	}
	if alpha >= 1 || picture == nil {
		return node.Child.Paint(origin, clip, picture)
	}
	prevYield := picture.yield
	start := len(picture.marks)
	picture.yield = func(mark Mark) bool {
		scaled := scaleMarkAlpha(mark, alpha)
		if start < len(picture.marks) {
			picture.marks[len(picture.marks)-1] = scaled
		}
		if prevYield != nil {
			return prevYield(scaled)
		}
		return true
	}
	out := node.Child.Paint(origin, clip, picture)
	picture.yield = prevYield
	if prevYield == nil {
		picture.adopt(lower(func(yield func(Mark) bool) {
			for _, m := range picture.marks {
				if !yield(m) {
					return
				}
			}
		}))
	}
	return out
}

func scaleMarkAlpha(mark Mark, alpha float32) Mark {
	switch mark.Kind {
	case MarkFill, MarkText:
		mark.Color.Alpha = lerpByte(0, mark.Color.Alpha, alpha)
	}
	return mark
}

// Slide translates Child during Paint while preserving incoming layout constraints.
// Offset is in logical pixels and scales with window density; Fractional is multiplied
// by the child's laid-out size.
type Slide struct {
	Offset     Offset
	Fractional Offset
	Child      Node

	size  Size
	scale float32
}

func (node *Slide) Layout(constraints BoxConstraints) Size {
	if node == nil || node.Child == nil {
		return Size{}
	}
	node.scale = constraints.scale
	node.size = node.Child.Layout(constraints)
	return node.size
}

func (node *Slide) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if node == nil || node.Child == nil {
		return accumulatorOf(picture)
	}
	f := factor(node.scale)
	dx := node.Offset.X*f + node.Fractional.X*node.size.Width
	dy := node.Offset.Y*f + node.Fractional.Y*node.size.Height
	return node.Child.Paint(origin.Add(Offset{X: dx, Y: dy}), clip, picture)
}

// ColumnView is a vertically scrollable [Flex] column with an optional
// animated scrollbar thumb. Offset is the scroll position in logical pixels.
type ColumnView struct {
	Gap       float32
	Padding   EdgeInsets
	Offset    float32
	Thumb     *RGB
	Children  []Node
	size      Size
	contentH  float32
	scale     float32
	innerFlow *Flex
}

// ContentHeight returns the laid-out height of the column contents in logical pixels.
func (view *ColumnView) ContentHeight() float32 {
	if view == nil {
		return 0
	}
	f := factor(view.scale)
	if f <= 0 {
		f = 1
	}
	return view.contentH / f
}

// MaxOffset returns the maximum scroll offset in logical pixels.
func (view *ColumnView) MaxOffset() float32 {
	if view == nil {
		return 0
	}
	f := factor(view.scale)
	if f <= 0 {
		f = 1
	}
	return max(float32(0), (view.contentH-view.size.Height)/f)
}

func (view *ColumnView) Layout(constraints BoxConstraints) Size {
	if view == nil {
		return Size{}
	}
	view.scale = constraints.scale
	view.size = constraints.Constrain(Size{Width: constraints.MaxWidth, Height: constraints.MaxHeight})
	pad := scaleInsets(view.Padding, view.scale)
	innerW := max(float32(0), view.size.Width-pad.Horizontal())
	flow := WithCross(CrossStart, WithGap(view.Gap, Column(view.Children...)))
	childConstraints := BoxConstraints{
		MinWidth:  innerW,
		MaxWidth:  innerW,
		MaxHeight: unbounded,
		scale:     constraints.scale,
		font:      constraints.font,
	}
	measured := flow.Layout(childConstraints)
	view.contentH = measured.Height + pad.Vertical()
	view.innerFlow = flow
	return view.size
}

func (view *ColumnView) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if view == nil || view.innerFlow == nil {
		return accumulatorOf(picture)
	}
	bounds := Rect{origin.X, origin.Y, view.size.Width, view.size.Height}
	clipped := clip.Intersect(bounds)
	if clipped.Width <= 0 || clipped.Height <= 0 {
		return accumulatorOf(picture)
	}
	f := factor(view.scale)
	pad := scaleInsets(view.Padding, view.scale)
	scrollY := view.Offset * f
	childOrigin := origin.Add(Offset{X: pad.Left, Y: pad.Top - scrollY})
	acc := view.innerFlow.Paint(childOrigin, clipped, picture)
	if view.Thumb != nil && picture != nil && view.contentH > view.size.Height && view.size.Height > 0 {
		ratio := view.size.Height / view.contentH
		thumbH := max(24*f, view.size.Height*ratio)
		maxScroll := view.contentH - view.size.Height
		progress := clamp01(scrollY / maxScroll)
		trackRoom := max(float32(0), view.size.Height-thumbH-8*f)
		thumbW := 5 * f
		thumbX := origin.X + view.size.Width - thumbW - 4*f
		thumbY := origin.Y + 4*f + trackRoom*progress
		acc = picture.Fill(Rect{thumbX, thumbY, thumbW, thumbH}, thumbW/2, *view.Thumb, clipped)
	}
	return acc
}
