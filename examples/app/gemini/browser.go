package main

import (
	"context"
	"fmt"
	"image"
	"math"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/lewtec/lewkit/examples/app/internal/icon"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/http/asset/lucide"
	lewimage "github.com/lewtec/lewkit/x/image"
	"github.com/lewtec/lewkit/x/net/gemini"
	"github.com/lewtec/lewkit/x/ui/gui"
	"golang.org/x/image/font"
)

type fetchedMsg struct {
	target    string
	direction int
	resp      gemini.Response
	err       error
}

type pageState struct {
	url    string
	title  string
	status int
	meta   string
	doc    gemini.Document
	scroll float32
}

type browserPalette struct {
	bg        gui.RGB
	surface   gui.RGB
	card      gui.RGB
	border    gui.RGB
	text      gui.RGB
	muted     gui.RGB
	accent    gui.RGB
	accentDim gui.RGB
	quoteBg   gui.RGB
	preBg     gui.RGB
	scrim     gui.RGB
}

func paletteFor(t float32) browserPalette {
	dark := browserPalette{
		bg:        gui.RGB{14, 17, 28, 255},
		surface:   gui.RGB{22, 27, 44, 255},
		card:      gui.RGB{30, 36, 58, 255},
		border:    gui.RGB{52, 62, 96, 255},
		text:      gui.RGB{236, 240, 252, 255},
		muted:     gui.RGB{148, 158, 190, 255},
		accent:    gui.RGB{98, 178, 255, 255},
		accentDim: gui.RGB{36, 62, 104, 255},
		quoteBg:   gui.RGB{24, 32, 52, 255},
		preBg:     gui.RGB{10, 12, 20, 255},
		scrim:     gui.RGB{6, 8, 14, 170},
	}
	light := browserPalette{
		bg:        gui.RGB{245, 246, 250, 255},
		surface:   gui.RGB{232, 235, 244, 255},
		card:      gui.RGB{255, 255, 255, 255},
		border:    gui.RGB{202, 208, 224, 255},
		text:      gui.RGB{24, 28, 42, 255},
		muted:     gui.RGB{102, 110, 134, 255},
		accent:    gui.RGB{28, 108, 218, 255},
		accentDim: gui.RGB{214, 228, 252, 255},
		quoteBg:   gui.RGB{236, 240, 250, 255},
		preBg:     gui.RGB{228, 232, 242, 255},
		scrim:     gui.RGB{18, 22, 34, 110},
	}
	return browserPalette{
		bg:        gui.LerpRGB(dark.bg, light.bg, t),
		surface:   gui.LerpRGB(dark.surface, light.surface, t),
		card:      gui.LerpRGB(dark.card, light.card, t),
		border:    gui.LerpRGB(dark.border, light.border, t),
		text:      gui.LerpRGB(dark.text, light.text, t),
		muted:     gui.LerpRGB(dark.muted, light.muted, t),
		accent:    gui.LerpRGB(dark.accent, light.accent, t),
		accentDim: gui.LerpRGB(dark.accentDim, light.accentDim, t),
		quoteBg:   gui.LerpRGB(dark.quoteBg, light.quoteBg, t),
		preBg:     gui.LerpRGB(dark.preBg, light.preBg, t),
		scrim:     gui.LerpRGB(dark.scrim, light.scrim, t),
	}
}

// Browser is a Gemini protocol client and gemtext viewer built on [gui.Model].
type Browser struct {
	gui.Dirty
	Client *gemini.Client

	size       image.Point
	lastTick   time.Duration
	booted     bool
	loading    bool
	loadPhase  float32
	mode       daynight.Mode
	address    []rune
	cursor     int
	addrFocus  bool
	caret      bool
	drawerOpen bool

	promptOpen bool
	promptURL  string
	promptMeta string
	promptText []rune

	current   pageState
	previous  *pageState
	routeDir  int
	history   []string
	histIndex int
	bookmarks []string
	statusMsg string
	links     []string

	routeProgress gui.Tween
	drawerSlide   gui.Tween
	promptScale   gui.Tween
	addrGlow      gui.Tween
	themeMix      gui.Tween
	scrollSpring  gui.Spring

	docColumn *gui.ColumnView
	addrText  *gui.Text
}

// NewBrowser creates a Gemini browser model ready to run.
func NewBrowser(_ context.Context) (*Browser, error) {
	b := &Browser{
		Client:        newDefaultClient(),
		size:          image.Pt(960, 700),
		mode:          daynight.Dark,
		address:       []rune(defaultHomeURL),
		cursor:        len(defaultHomeURL),
		histIndex:     -1,
		bookmarks:     []string{"gemini://home/", "gemini://home/spec", "gemini://home/animations", "gemini://home/search", "gemini://home/poetry"},
		routeProgress: gui.NewTween(1),
		drawerSlide:   gui.NewTween(0),
		promptScale:   gui.NewTween(0),
		addrGlow:      gui.NewTween(0),
		themeMix:      gui.NewTween(0),
		scrollSpring:  gui.Spring{Stiffness: 280, Damping: 34},
	}
	return b, nil
}

func (b *Browser) Init() gui.Cmd { return gui.Tick() }

func (b *Browser) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if b == nil {
		return b, nil
	}
	if size, ok := browserSize(msg); ok && size.X > 0 && size.Y > 0 {
		b.Dirty, b.size = gui.See(b.Dirty, b.size, size)
	}
	switch event := msg.(type) {
	case gui.TickMsg:
		return b.onTick(event)
	case gui.ModeMsg:
		b.setMode(event.Mode)
		return b, gui.Tick()
	case fetchedMsg:
		return b.onFetched(event)
	case window.Pointer:
		return b.onPointer(event)
	case window.Scroll:
		return b.onScroll(event)
	case window.Key:
		return b.onKey(event)
	}
	return b, nil
}

func (b *Browser) onTick(tick gui.TickMsg) (gui.Model, gui.Cmd) {
	if !b.booted {
		b.booted = true
		b.Dirty = gui.Touch(b.Dirty)
		return b, b.navigate(defaultHomeURL, 0, true)
	}
	dt := time.Duration(0)
	if b.lastTick > 0 && tick.Elapsed > b.lastTick {
		dt = tick.Elapsed - b.lastTick
	} else {
		dt = 16 * time.Millisecond
	}
	b.lastTick = tick.Elapsed

	animating := false
	if b.routeProgress.Step(dt) {
		animating = true
	} else if b.previous != nil {
		b.previous = nil
		b.Dirty = gui.Touch(b.Dirty)
	}
	if b.drawerSlide.Step(dt) {
		animating = true
	}
	if b.promptScale.Step(dt) {
		animating = true
	}
	if b.addrGlow.Step(dt) {
		animating = true
	}
	if b.themeMix.Step(dt) {
		animating = true
	}
	if b.scrollSpring.Step(dt) {
		b.current.scroll = b.scrollSpring.Value
		animating = true
	}
	if b.loading {
		b.loadPhase = float32(math.Mod(float64(b.loadPhase)+dt.Seconds()*1.6, 1.0))
		animating = true
	}
	if b.addrFocus || b.promptOpen {
		blink := tick.Elapsed.Milliseconds()/450%2 == 0
		if blink != b.caret {
			b.caret = blink
			b.Dirty = gui.Touch(b.Dirty)
		}
		animating = true
	}
	if animating {
		b.Dirty = gui.Touch(b.Dirty)
		return b, gui.Every(tick.Period)
	}
	return b, nil
}

func (b *Browser) setMode(mode daynight.Mode) {
	b.mode = mode
	target := float32(0)
	if mode == daynight.Light {
		target = 1
	}
	b.themeMix.To(target, 280*time.Millisecond, gui.EaseInOutCubic)
	b.Dirty = gui.Touch(b.Dirty)
}

func (b *Browser) toggleTheme() {
	if b.mode == daynight.Light {
		b.setMode(daynight.Dark)
	} else {
		b.setMode(daynight.Light)
	}
}

func (b *Browser) navigate(rawURL string, direction int, pushHistory bool) gui.Cmd {
	resolved := strings.TrimSpace(rawURL)
	if b.current.url != "" && !strings.Contains(resolved, "://") && !strings.HasPrefix(resolved, "about:") {
		if next, err := gemini.ResolveURL(b.current.url, resolved); err == nil {
			resolved = next
		}
	}
	normalized, err := gemini.NormalizeURL(resolved)
	if err == nil {
		resolved = normalized.String()
	}
	b.loading = true
	b.statusMsg = "Connecting to " + resolved
	b.address = []rune(resolved)
	b.cursor = len(b.address)
	b.addrFocus = false
	b.addrGlow.To(0, 180*time.Millisecond, gui.EaseOut)
	b.Dirty = gui.Touch(b.Dirty)
	if pushHistory {
		if b.histIndex+1 < len(b.history) {
			b.history = b.history[:b.histIndex+1]
		}
		if len(b.history) == 0 || b.history[len(b.history)-1] != resolved {
			b.history = append(b.history, resolved)
			b.histIndex = len(b.history) - 1
		}
	}
	client := b.Client
	if client == nil {
		client = newDefaultClient()
	}
	return func(ctx context.Context) gui.Msg {
		resp, fetchErr := client.Fetch(ctx, resolved)
		return fetchedMsg{
			target:    resolved,
			direction: direction,
			resp:      resp,
			err:       fetchErr,
		}
	}
}

func (b *Browser) onFetched(msg fetchedMsg) (gui.Model, gui.Cmd) {
	b.loading = false
	b.Dirty = gui.Touch(b.Dirty)
	if msg.err != nil {
		b.applyDocument(msg.target, 59, msg.err.Error(), fmt.Sprintf("# Connection Error\n\nCould not load `%s`:\n> %s\n\n=> gemini://home/ Return to Geminispace Observatory\n", msg.target, msg.err.Error()), msg.direction)
		return b, gui.Tick()
	}
	cat := gemini.Category(msg.resp.Status)
	switch cat {
	case gemini.StatusInput:
		b.promptOpen = true
		b.promptURL = msg.resp.URL
		if b.promptURL == "" {
			b.promptURL = msg.target
		}
		b.promptMeta = msg.resp.Meta
		b.promptText = nil
		b.promptScale.FromTo(0, 1, 240*time.Millisecond, gui.EaseOutBack)
		b.statusMsg = fmt.Sprintf("Status %d — Input requested", msg.resp.Status)
		return b, gui.Tick()
	case gemini.StatusSuccess:
		b.applyDocument(msg.resp.URL, int(msg.resp.Status), msg.resp.Meta, string(msg.resp.Body), msg.direction)
		return b, gui.Tick()
	default:
		body := fmt.Sprintf("# Gemini Status %d\n## %s\n\nThe capsule returned status %d (%s).\n\n=> gemini://home/ Return to Geminispace Observatory\n", msg.resp.Status, msg.resp.Meta, msg.resp.Status, cat.String())
		b.applyDocument(msg.target, int(msg.resp.Status), msg.resp.Meta, body, msg.direction)
		return b, gui.Tick()
	}
}

func (b *Browser) applyDocument(targetURL string, status int, meta, rawBody string, direction int) {
	if b.current.url != "" {
		prev := b.current
		b.previous = &prev
		if direction == 0 {
			direction = 1
		}
		b.routeDir = direction
		b.routeProgress.FromTo(0, 1, 280*time.Millisecond, gui.FastOutSlowIn)
	} else {
		b.routeProgress.Set(1)
	}
	doc := gemini.ParseGemtext(rawBody)
	title := doc.Title()
	if title == "" {
		title = targetURL
	}
	var links []string
	for _, line := range doc.Lines {
		if line.Kind == gemini.KindLink {
			links = append(links, line.URL)
		}
	}
	b.links = links
	b.current = pageState{
		url:    targetURL,
		title:  title,
		status: status,
		meta:   meta,
		doc:    doc,
		scroll: 0,
	}
	b.scrollSpring.Set(0)
	b.address = []rune(targetURL)
	b.cursor = len(b.address)
	b.statusMsg = fmt.Sprintf("%d %s · %d lines", status, meta, len(doc.Lines))
}

func (b *Browser) onPointer(event window.Pointer) (gui.Model, gui.Cmd) {
	if event.Button != 1 || !event.Pressed {
		return b, nil
	}
	root := b.View()
	if root == nil {
		return b, nil
	}
	size := gui.Size{Width: float32(b.size.X), Height: float32(b.size.Y)}
	key, _, _ := gui.Hit(root, size, event.Pos)

	if key != "address" && b.addrFocus {
		b.addrFocus = false
		b.addrGlow.To(0, 180*time.Millisecond, gui.EaseOut)
		b.Dirty = gui.Touch(b.Dirty)
	}

	switch {
	case key == "back":
		return b.goBack()
	case key == "forward":
		return b.goForward()
	case key == "home":
		return b, b.navigate(defaultHomeURL, -1, true)
	case key == "reload":
		if b.current.url != "" {
			return b, b.navigate(b.current.url, 0, false)
		}
	case key == "bookmark":
		b.toggleBookmark()
		return b, gui.Tick()
	case key == "drawer":
		b.setDrawer(!b.drawerOpen)
		return b, gui.Tick()
	case key == "scrim":
		if b.promptOpen {
			b.closePrompt()
			return b, gui.Tick()
		}
		if b.drawerOpen {
			b.setDrawer(false)
			return b, gui.Tick()
		}
	case key == "theme":
		b.toggleTheme()
		return b, gui.Tick()
	case key == "address":
		b.addrFocus = true
		b.caret = true
		if b.addrText != nil {
			b.cursor = b.addrText.IndexAt(event.Pos)
		} else {
			b.cursor = len(b.address)
		}
		b.addrGlow.To(1, 200*time.Millisecond, gui.EaseOutCubic)
		b.Dirty = gui.Touch(b.Dirty)
		return b, gui.Tick()
	case key == "prompt-cancel":
		b.closePrompt()
		return b, gui.Tick()
	case key == "prompt-submit":
		return b.submitPrompt()
	case strings.HasPrefix(key, "link:"):
		idx, err := strconv.Atoi(strings.TrimPrefix(key, "link:"))
		if err == nil && idx >= 0 && idx < len(b.links) {
			return b, b.navigate(b.links[idx], 1, true)
		}
	case strings.HasPrefix(key, "jump:"):
		target := strings.TrimPrefix(key, "jump:")
		b.setDrawer(false)
		return b, b.navigate(target, 1, true)
	}
	return b, nil
}

func (b *Browser) onScroll(event window.Scroll) (gui.Model, gui.Cmd) {
	delta := -float32(event.Delta.Y) * 28
	if delta == 0 {
		delta = -float32(event.Delta.X) * 28
	}
	b.scrollBy(delta)
	return b, gui.Tick()
}

func (b *Browser) scrollBy(delta float32) {
	target := b.scrollSpring.Target + delta
	maxScroll := float32(2400)
	if b.docColumn != nil {
		maxScroll = b.docColumn.MaxOffset()
	}
	if target < 0 {
		target = 0
	}
	if target > maxScroll {
		target = maxScroll
	}
	b.scrollSpring.Target = target
	b.Dirty = gui.Touch(b.Dirty)
}

func (b *Browser) onKey(key window.Key) (gui.Model, gui.Cmd) {
	if !key.Pressed {
		return b, nil
	}
	if b.promptOpen {
		return b.onPromptKey(key)
	}
	if b.addrFocus {
		return b.onAddressKey(key)
	}
	switch {
	case key.Rune == '/' || (key.Mod&window.ModCtrl != 0 && (key.Rune == 'l' || key.Rune == 'L')):
		b.addrFocus = true
		b.caret = true
		b.cursor = len(b.address)
		b.addrGlow.To(1, 200*time.Millisecond, gui.EaseOutCubic)
		b.Dirty = gui.Touch(b.Dirty)
		return b, gui.Tick()
	case key.Rune == 'b' || key.Rune == 'B':
		b.setDrawer(!b.drawerOpen)
		return b, gui.Tick()
	case key.Rune == 't' || key.Rune == 'T':
		b.toggleTheme()
		return b, gui.Tick()
	case key.Rune == 'j' || isDownKey(key):
		b.scrollBy(72)
		return b, gui.Tick()
	case key.Rune == 'k' || isUpKey(key):
		b.scrollBy(-72)
		return b, gui.Tick()
	case key.Rune == ' ':
		b.scrollBy(220)
		return b, gui.Tick()
	case isLeftKey(key) && key.Mod&window.ModAlt != 0:
		return b.goBack()
	case isRightKey(key) && key.Mod&window.ModAlt != 0:
		return b.goForward()
	}
	return b, nil
}

func (b *Browser) onPromptKey(key window.Key) (gui.Model, gui.Cmd) {
	switch {
	case key.Code == 53 || key.Code == 27 || key.Rune == 27:
		b.closePrompt()
		return b, gui.Tick()
	case isBackspace(key):
		if len(b.promptText) > 0 {
			b.promptText = b.promptText[:len(b.promptText)-1]
			b.Dirty = gui.Touch(b.Dirty)
		}
		return b, gui.Tick()
	case isEnter(key):
		return b.submitPrompt()
	case key.Rune >= 32 && key.Rune != 127:
		b.promptText = append(b.promptText, key.Rune)
		b.Dirty = gui.Touch(b.Dirty)
		return b, gui.Tick()
	}
	return b, nil
}

func (b *Browser) onAddressKey(key window.Key) (gui.Model, gui.Cmd) {
	switch {
	case key.Code == 53 || key.Code == 27 || key.Rune == 27:
		b.addrFocus = false
		b.addrGlow.To(0, 180*time.Millisecond, gui.EaseOut)
		b.Dirty = gui.Touch(b.Dirty)
		return b, gui.Tick()
	case isBackspace(key):
		if b.cursor > 0 && len(b.address) > 0 {
			b.address = append(b.address[:b.cursor-1], b.address[b.cursor:]...)
			b.cursor--
			b.Dirty = gui.Touch(b.Dirty)
		}
		return b, gui.Tick()
	case isLeftKey(key):
		if b.cursor > 0 {
			b.cursor--
			b.Dirty = gui.Touch(b.Dirty)
		}
		return b, gui.Tick()
	case isRightKey(key):
		if b.cursor < len(b.address) {
			b.cursor++
			b.Dirty = gui.Touch(b.Dirty)
		}
		return b, gui.Tick()
	case isEnter(key):
		target := strings.TrimSpace(string(b.address))
		if target == "" {
			return b, nil
		}
		return b, b.navigate(target, 1, true)
	case key.Rune >= 32 && key.Rune != 127:
		b.address = slices.Insert(b.address, b.cursor, key.Rune)
		b.cursor++
		b.Dirty = gui.Touch(b.Dirty)
		return b, gui.Tick()
	}
	return b, nil
}

func (b *Browser) closePrompt() {
	b.promptOpen = false
	b.promptScale.To(0, 180*time.Millisecond, gui.EaseIn)
	b.Dirty = gui.Touch(b.Dirty)
}

func (b *Browser) submitPrompt() (gui.Model, gui.Cmd) {
	query := strings.TrimSpace(string(b.promptText))
	base := b.promptURL
	b.closePrompt()
	if base == "" {
		return b, nil
	}
	u, err := gemini.NormalizeURL(base)
	if err != nil {
		return b, nil
	}
	u.RawQuery = url.QueryEscape(query)
	return b, b.navigate(u.String(), 1, true)
}

func (b *Browser) goBack() (gui.Model, gui.Cmd) {
	if b.histIndex <= 0 {
		return b, nil
	}
	b.histIndex--
	return b, b.navigate(b.history[b.histIndex], -1, false)
}

func (b *Browser) goForward() (gui.Model, gui.Cmd) {
	if b.histIndex+1 >= len(b.history) {
		return b, nil
	}
	b.histIndex++
	return b, b.navigate(b.history[b.histIndex], 1, false)
}

func (b *Browser) setDrawer(open bool) {
	b.drawerOpen = open
	target := float32(0)
	if open {
		target = 1
	}
	b.drawerSlide.To(target, 260*time.Millisecond, gui.FastOutSlowIn)
	b.Dirty = gui.Touch(b.Dirty)
}

func (b *Browser) isBookmarked(target string) bool {
	return slices.Contains(b.bookmarks, target)
}

func (b *Browser) toggleBookmark() {
	u := b.current.url
	if u == "" {
		return
	}
	if idx := slices.Index(b.bookmarks, u); idx >= 0 {
		b.bookmarks = slices.Delete(b.bookmarks, idx, idx+1)
		b.statusMsg = "Removed bookmark"
	} else {
		b.bookmarks = append(b.bookmarks, u)
		b.statusMsg = "Bookmarked " + u
	}
	b.Dirty = gui.Touch(b.Dirty)
}

func (b *Browser) face(mul float32) font.Face {
	return lewimage.FaceSize(float64(16 * mul))
}

func (b *Browser) View() gui.Node {
	if b == nil {
		return nil
	}
	pal := paletteFor(b.themeMix.Value())
	mainColumn := &gui.Flex{
		Axis: gui.Vertical,
		Children: []gui.FlexChild{
			{Child: b.toolbarView(pal)},
			{Child: b.progressBarView(pal)},
			gui.Expanded(b.stageView(pal)),
			{Child: b.statusBarView(pal)},
		},
	}

	layers := []gui.Node{mainColumn}
	drawerVal := b.drawerSlide.Value()
	if drawerVal > 0.01 {
		scrim := pal.scrim
		scrim.Alpha = uint8(float32(scrim.Alpha) * drawerVal)
		layers = append(layers,
			&gui.Box{Key: "scrim", Fill: &scrim},
			b.drawerView(pal, drawerVal),
		)
	}
	promptVal := b.promptScale.Value()
	if b.promptOpen || promptVal > 0.01 {
		scrim := pal.scrim
		scrim.Alpha = uint8(float32(scrim.Alpha) * min(float32(1), promptVal))
		layers = append(layers,
			&gui.Box{Key: "scrim", Fill: &scrim},
			b.promptModalView(pal, promptVal),
		)
	}

	return &gui.Stage{
		Scene: &gui.Box{Fill: &pal.bg},
		Child: &gui.Stack{Clip: true, Children: layers},
	}
}

func (b *Browser) toolbarView(pal browserPalette) gui.Node {
	canBack := b.histIndex > 0
	canFwd := b.histIndex+1 < len(b.history)

	backBtn := b.navButton("back", lucide.NameChevronLeft, canBack, pal)
	fwdBtn := b.navButton("forward", lucide.NameChevronRight, canFwd, pal)
	homeBtn := b.navButton("home", lucide.NameHouse, true, pal)
	reloadBtn := b.navButton("reload", lucide.NameRotateCcw, true, pal)

	glow := b.addrGlow.Value()
	addrBorder := gui.LerpRGB(pal.border, pal.accent, glow)
	addrBg := gui.LerpRGB(pal.bg, pal.card, glow*0.6)
	cursor := -1
	if b.addrFocus {
		cursor = b.cursor
	}
	b.addrText = &gui.Text{
		Value:  string(b.address),
		Face:   b.face(1.0),
		Ink:    pal.text,
		Cursor: cursor,
		Caret:  b.addrFocus && b.caret,
	}
	addrBox := &gui.Box{
		Key:     "address",
		Height:  38,
		Radius:  10,
		Fill:    &addrBorder,
		Padding: gui.EdgeInsets{Left: 2, Top: 2, Right: 2, Bottom: 2},
		Child: &gui.Box{
			Height:  34,
			Radius:  8,
			Fill:    &addrBg,
			Padding: gui.EdgeInsets{Left: 12, Right: 12},
			Align:   gui.Alignment{Y: 0.5},
			Clip:    true,
			Child:   b.addrText,
		},
	}

	mark := lucide.NameBookmark
	starFill := pal.card
	if b.isBookmarked(b.current.url) {
		mark = lucide.NameBookmarkCheck
		starFill = pal.accentDim
	}
	starBtn := &gui.Box{
		Key:    "bookmark",
		Width:  36,
		Height: 36,
		Radius: 10,
		Fill:   &starFill,
		Align:  gui.Alignment{X: 0.5, Y: 0.5},
		Child:  icon.Node(mark, pal.accent, 18),
	}

	theme := lucide.NameMoon
	if b.mode == daynight.Light {
		theme = lucide.NameSun
	}
	themeBtn := b.navButton("theme", theme, true, pal)
	drawerBtn := b.navButton("drawer", lucide.NameMenu, true, pal)

	row := &gui.Flex{
		Axis:  gui.Horizontal,
		Gap:   8,
		Cross: gui.CrossCenter,
		Children: []gui.FlexChild{
			{Child: backBtn},
			{Child: fwdBtn},
			{Child: homeBtn},
			{Child: reloadBtn},
			gui.Expanded(addrBox),
			{Child: starBtn},
			{Child: themeBtn},
			{Child: drawerBtn},
		},
	}
	return &gui.Box{
		Height:  56,
		Fill:    &pal.surface,
		Padding: gui.EdgeInsets{Left: 14, Top: 9, Right: 14, Bottom: 9},
		Child:   row,
	}
}

func (b *Browser) navButton(key string, name lucide.Name, enabled bool, pal browserPalette) gui.Node {
	ink := pal.text
	fill := pal.card
	if !enabled {
		ink = pal.muted
		fill = pal.surface
	}
	return &gui.Box{
		Key:    key,
		Width:  36,
		Height: 36,
		Radius: 10,
		Fill:   &fill,
		Align:  gui.Alignment{X: 0.5, Y: 0.5},
		Child:  icon.Node(name, ink, 18),
	}
}

func (b *Browser) progressBarView(pal browserPalette) gui.Node {
	if !b.loading {
		return &gui.Box{Height: 3, Fill: &pal.border}
	}
	totalW := float32(max(b.size.X, 320))
	barW := totalW * 0.28
	x := (totalW+barW)*b.loadPhase - barW
	accent := pal.accent
	return &gui.Box{
		Height: 3,
		Fill:   &pal.border,
		Clip:   true,
		Child: &gui.Stack{
			Clip: true,
			Children: []gui.Node{
				&gui.Positioned{
					X: x,
					Child: &gui.Box{
						Width:  barW,
						Height: 3,
						Radius: 1.5,
						Fill:   &accent,
					},
				},
			},
		},
	}
}

func (b *Browser) stageView(pal browserPalette) gui.Node {
	p := b.routeProgress.Value()
	currDoc := b.renderPage(b.current, true, pal)
	if p >= 0.999 || b.previous == nil {
		return currDoc
	}
	dir := float32(b.routeDir)
	if dir == 0 {
		dir = 1
	}
	prevDoc := b.renderPage(*b.previous, false, pal)
	outgoing := &gui.Opacity{
		Value: 1 - p,
		Child: &gui.Slide{
			Fractional: gui.Offset{X: -0.18 * dir * p},
			Child:      prevDoc,
		},
	}
	incoming := &gui.Opacity{
		Value: p,
		Child: &gui.Slide{
			Fractional: gui.Offset{X: 0.22 * dir * (1 - p)},
			Child:      currDoc,
		},
	}
	return &gui.Stack{
		Clip:     true,
		Children: []gui.Node{outgoing, incoming},
	}
}

func (b *Browser) renderPage(page pageState, active bool, pal browserPalette) gui.Node {
	var items []gui.Node
	linkIndex := 0
	for _, line := range page.doc.Lines {
		switch line.Kind {
		case gemini.KindHeading:
			mul := float32(1.65)
			topPad := float32(12)
			ink := pal.text
			switch line.Level {
			case 2:
				mul = 1.32
				topPad = 8
				ink = pal.accent
			case 3:
				mul = 1.12
				topPad = 6
			}
			items = append(items, &gui.Box{
				Padding: gui.EdgeInsets{Top: topPad, Bottom: 4},
				Child:   &gui.Text{Value: line.Text, Face: b.face(mul), Ink: ink},
			})
		case gemini.KindLink:
			key := ""
			if active {
				key = fmt.Sprintf("link:%d", linkIndex)
			}
			linkIndex++
			items = append(items, b.linkCard(key, line.Text, line.URL, pal))
		case gemini.KindListItem:
			row := &gui.Flex{
				Axis:  gui.Horizontal,
				Gap:   10,
				Cross: gui.CrossStart,
				Children: []gui.FlexChild{
					{Child: &gui.Box{
						Width: 16, Height: 18,
						Align: gui.Alignment{X: 0.5, Y: 0.5},
						Child: icon.Node(lucide.NameCircleSmall, pal.accent, 12),
					}},
					gui.Expanded(&gui.Text{Value: line.Text, Face: b.face(1.0), Ink: pal.text}),
				},
			}
			items = append(items, &gui.Box{
				Padding: gui.EdgeInsets{Left: 8, Top: 2, Bottom: 2},
				Child:   row,
			})
		case gemini.KindQuote:
			bar := pal.accent
			quoteBg := pal.quoteBg
			quote := &gui.Box{
				Fill:    &quoteBg,
				Radius:  8,
				Padding: gui.EdgeInsets{Left: 12, Top: 8, Right: 12, Bottom: 8},
				Child: &gui.Flex{
					Axis:  gui.Horizontal,
					Gap:   10,
					Cross: gui.CrossStart,
					Children: []gui.FlexChild{
						{Child: &gui.Box{Width: 4, Height: 22, Radius: 2, Fill: &bar}},
						gui.Expanded(&gui.Text{Value: line.Text, Face: b.face(0.98), Ink: pal.muted}),
					},
				},
			}
			items = append(items, quote)
		case gemini.KindPreformatted:
			preBg := pal.preBg
			children := []gui.Node{}
			if line.Alt != "" {
				children = append(children, &gui.Text{Value: line.Alt, Face: b.face(0.82), Ink: pal.accent})
			}
			children = append(children, &gui.Text{Value: line.Text, Face: b.face(0.92), Ink: pal.text})
			items = append(items, &gui.Box{
				Fill:    &preBg,
				Radius:  10,
				Padding: gui.EdgeInsets{Left: 14, Top: 10, Right: 14, Bottom: 10},
				Child:   gui.WithGap(6, gui.WithCross(gui.CrossStart, gui.Column(children...))),
			})
		default:
			if strings.TrimSpace(line.Text) == "" {
				items = append(items, &gui.Box{Height: 6})
			} else {
				items = append(items, &gui.Text{Value: line.Text, Face: b.face(1.0), Ink: pal.text})
			}
		}
	}
	thumb := pal.border
	col := &gui.ColumnView{
		Gap:      8,
		Padding:  gui.EdgeInsets{Left: 28, Top: 20, Right: 28, Bottom: 28},
		Offset:   page.scroll,
		Thumb:    &thumb,
		Children: items,
	}
	if active {
		b.docColumn = col
	}
	return col
}

func (b *Browser) linkCard(key, label, target string, pal browserPalette) gui.Node {
	cardFill := pal.card
	badgeFill := pal.accentDim
	row := &gui.Flex{
		Axis:  gui.Horizontal,
		Gap:   10,
		Cross: gui.CrossCenter,
		Children: []gui.FlexChild{
			{Child: &gui.Box{
				Width:  28,
				Height: 24,
				Radius: 6,
				Fill:   &badgeFill,
				Align:  gui.Alignment{X: 0.5, Y: 0.5},
				Child:  icon.Node(lucide.NameArrowRight, pal.accent, 16),
			}},
			gui.Expanded(gui.WithGap(2, gui.WithCross(gui.CrossStart, gui.Column(
				&gui.Text{Value: label, Face: b.face(1.02), Ink: pal.accent},
				&gui.Text{Value: target, Face: b.face(0.82), Ink: pal.muted},
			)))),
		},
	}
	return &gui.Box{
		Key:     key,
		Fill:    &cardFill,
		Radius:  10,
		Padding: gui.EdgeInsets{Left: 12, Top: 8, Right: 12, Bottom: 8},
		Child:   row,
	}
}

func (b *Browser) drawerView(pal browserPalette, progress float32) gui.Node {
	drawerW := float32(290)
	var items []gui.Node
	items = append(items, &gui.Text{Value: "Bookmarks", Face: b.face(1.2), Ink: pal.accent})
	for _, bm := range b.bookmarks {
		fill := pal.card
		items = append(items, &gui.Box{
			Key:     "jump:" + bm,
			Fill:    &fill,
			Radius:  8,
			Padding: gui.EdgeInsets{Left: 10, Top: 8, Right: 10, Bottom: 8},
			Child:   &gui.Text{Value: bm, Face: b.face(0.92), Ink: pal.text},
		})
	}
	items = append(items,
		&gui.Box{Height: 8},
		&gui.Text{Value: "History", Face: b.face(1.2), Ink: pal.accent},
	)
	for i := len(b.history) - 1; i >= 0; i-- {
		entry := b.history[i]
		fill := pal.surface
		if i == b.histIndex {
			fill = pal.accentDim
		}
		items = append(items, &gui.Box{
			Key:     "jump:" + entry,
			Fill:    &fill,
			Radius:  8,
			Padding: gui.EdgeInsets{Left: 10, Top: 7, Right: 10, Bottom: 7},
			Child:   &gui.Text{Value: entry, Face: b.face(0.9), Ink: pal.text},
		})
	}
	panelFill := pal.surface
	panel := &gui.Box{
		Width: drawerW,
		Fill:  &panelFill,
		Clip:  true,
		Child: &gui.ColumnView{
			Gap:      6,
			Padding:  gui.EdgeInsets{Left: 16, Top: 18, Right: 16, Bottom: 18},
			Children: items,
		},
	}
	return &gui.Box{
		Align: gui.Alignment{X: 1, Y: 0},
		Child: &gui.Opacity{
			Value: progress,
			Child: &gui.Slide{
				Fractional: gui.Offset{X: 1 - progress},
				Child:      panel,
			},
		},
	}
}

func (b *Browser) promptModalView(pal browserPalette, progress float32) gui.Node {
	cardFill := pal.surface
	inputFill := pal.bg
	accentFill := pal.accent
	cancelFill := pal.card

	inputBox := &gui.Box{
		Height:  38,
		Radius:  8,
		Fill:    &inputFill,
		Padding: gui.EdgeInsets{Left: 12, Right: 12},
		Align:   gui.Alignment{Y: 0.5},
		Child: &gui.Text{
			Value:  string(b.promptText),
			Face:   b.face(1.0),
			Ink:    pal.text,
			Cursor: len(b.promptText),
			Caret:  b.caret,
		},
	}
	actions := &gui.Flex{
		Axis:  gui.Horizontal,
		Gap:   10,
		Cross: gui.CrossCenter,
		Children: []gui.FlexChild{
			gui.Expanded(&gui.Box{}),
			{Child: &gui.Box{
				Key:    "prompt-cancel",
				Width:  88,
				Height: 34,
				Radius: 8,
				Fill:   &cancelFill,
				Align:  gui.Alignment{X: 0.5, Y: 0.5},
				Child:  &gui.Text{Value: "Cancel", Face: b.face(0.95), Ink: pal.text},
			}},
			{Child: &gui.Box{
				Key:    "prompt-submit",
				Width:  96,
				Height: 34,
				Radius: 8,
				Fill:   &accentFill,
				Align:  gui.Alignment{X: 0.5, Y: 0.5},
				Child:  &gui.Text{Value: "Submit", Face: b.face(0.95), Ink: pal.bg},
			}},
		},
	}
	card := &gui.Box{
		Width:   440,
		Radius:  14,
		Fill:    &cardFill,
		Padding: gui.EdgeInsets{Left: 20, Top: 18, Right: 20, Bottom: 18},
		Child: gui.WithGap(12, gui.WithCross(gui.CrossStart, gui.Column(
			&gui.Text{Value: "Gemini Input Request (Status 10)", Face: b.face(1.15), Ink: pal.accent},
			&gui.Text{Value: b.promptMeta, Face: b.face(0.95), Ink: pal.text},
			inputBox,
			actions,
		))),
	}
	return &gui.Box{
		Align: gui.Alignment{X: 0.5, Y: 0.5},
		Child: &gui.Opacity{
			Value: min(float32(1), progress),
			Child: &gui.Slide{
				Offset: gui.Offset{Y: (1 - progress) * 28},
				Child:  card,
			},
		},
	}
}

func (b *Browser) statusBarView(pal browserPalette) gui.Node {
	left := b.statusMsg
	if left == "" {
		left = b.current.url
	}
	right := fmt.Sprintf("%d bookmarks · %d history", len(b.bookmarks), len(b.history))
	row := &gui.Flex{
		Axis:  gui.Horizontal,
		Cross: gui.CrossCenter,
		Children: []gui.FlexChild{
			{Child: &gui.Text{Value: left, Face: b.face(0.85), Ink: pal.muted}},
			gui.Expanded(&gui.Box{}),
			{Child: &gui.Text{Value: right, Face: b.face(0.85), Ink: pal.muted}},
		},
	}
	return &gui.Box{
		Height:  28,
		Fill:    &pal.surface,
		Padding: gui.EdgeInsets{Left: 14, Right: 14},
		Align:   gui.Alignment{Y: 0.5},
		Child:   row,
	}
}

func browserSize(msg gui.Msg) (image.Point, bool) {
	switch message := msg.(type) {
	case gui.TickMsg:
		return message.Size, message.Size.X > 0
	case window.Resize:
		return message.Size, true
	default:
		return image.Point{}, false
	}
}

func isBackspace(key window.Key) bool {
	return key.Rune == 8 || key.Rune == 127 || key.Code == 51 || key.Code == 8 || key.Code == 22
}

func isEnter(key window.Key) bool {
	return key.Rune == '\r' || key.Rune == '\n' || key.Code == 36 || key.Code == 13 || key.Code == 24
}

func isLeftKey(key window.Key) bool {
	return key.Code == 123 || key.Code == 0x25 || key.Code == 113
}

func isRightKey(key window.Key) bool {
	return key.Code == 124 || key.Code == 0x27 || key.Code == 114
}

func isUpKey(key window.Key) bool {
	return key.Code == 126 || key.Code == 0x26 || key.Code == 111
}

func isDownKey(key window.Key) bool {
	return key.Code == 125 || key.Code == 0x28 || key.Code == 116
}
