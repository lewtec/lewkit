package gui

import (
	"context"
	"errors"
	"image"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/driver/window"
	lewpath "github.com/lewtec/lewkit/x/path"
)

const welcomeWidth = 460

// Welcome is the start screen: the LEWTEC TECNOLOGIA lockup and recent folders.
type Welcome struct {
	title  string
	dirs   []Directory
	mode   daynight.Mode
	cursor int
	note   string
	picked string
	rows   []*Box
	browse *Box
}

// NewWelcome lists dirs under title. An empty title is lewkit.
func NewWelcome(title string, dirs []Directory) *Welcome {
	if title == "" {
		title = "lewkit"
	}
	if len(dirs) > recentLimit {
		dirs = dirs[:recentLimit]
	}
	return &Welcome{title: title, dirs: append([]Directory(nil), dirs...), mode: daynight.Dark}
}

// Picked is the folder the user chose. It is empty until then.
func (welcome *Welcome) Picked() string {
	if welcome == nil {
		return ""
	}
	return welcome.picked
}

func (welcome *Welcome) Init() Cmd { return nil }

func (welcome *Welcome) Update(msg Msg) (Model, Cmd) {
	if welcome == nil {
		return welcome, nil
	}
	if mode, ok := msg.(ModeMsg); ok {
		welcome.mode = mode.Mode
	}
	switch event := msg.(type) {
	case window.Pointer:
		return welcome.point(event)
	case window.Key:
		if event.Pressed {
			return welcome.key(event)
		}
	case folderPicked:
		return welcome.folder(event)
	}
	return welcome, nil
}

func (welcome *Welcome) point(event window.Pointer) (Model, Cmd) {
	if event.Button == 0 {
		if index, ok := welcome.hit(event.Pos); ok {
			welcome.cursor = index
		}
		return welcome, nil
	}
	if event.Button != 1 || !event.Pressed {
		return welcome, nil
	}
	index, ok := welcome.hit(event.Pos)
	if !ok {
		return welcome, nil
	}
	welcome.cursor = index
	return welcome.activate()
}

func (welcome *Welcome) key(event window.Key) (Model, Cmd) {
	switch {
	case welcome.escape(event):
		return welcome.finish()
	case welcome.up(event) && welcome.cursor > 0:
		welcome.cursor--
	case welcome.down(event) && welcome.cursor < welcome.browseAt():
		welcome.cursor++
	case welcome.enter(event):
		return welcome.activate()
	}
	return welcome, nil
}

func (welcome *Welcome) activate() (Model, Cmd) {
	if welcome.cursor == welcome.browseAt() {
		return welcome, welcome.openFolder()
	}
	if welcome.cursor >= 0 && welcome.cursor < len(welcome.dirs) {
		welcome.picked = welcome.dirs[welcome.cursor].Path
		return welcome.finish()
	}
	return welcome, nil
}

func (welcome *Welcome) openFolder() Cmd {
	return func(ctx context.Context) Msg {
		paths, err := filedialog.Choose(ctx, filedialog.Request{
			Title:  "Open folder",
			Folder: true,
		})
		return folderPicked{paths: paths, err: err}
	}
}

func (welcome *Welcome) folder(msg folderPicked) (Model, Cmd) {
	if errors.Is(msg.err, filedialog.ErrCanceled) {
		return welcome, nil
	}
	if msg.err != nil {
		if errors.Is(msg.err, driver.ErrUnavailable) {
			welcome.note = "No folder dialog on this system."
		} else {
			welcome.note = msg.err.Error()
		}
		return welcome, nil
	}
	if len(msg.paths) == 0 {
		return welcome, nil
	}
	welcome.picked = msg.paths[0]
	return welcome.finish()
}

func (welcome *Welcome) finish() (Model, Cmd) {
	return welcome, func(context.Context) Msg { return quitMsg{} }
}

func (welcome *Welcome) hit(pos image.Point) (int, bool) {
	for i, row := range welcome.rows {
		if row.Contains(pos) {
			return i, true
		}
	}
	if welcome.browse != nil && welcome.browse.Contains(pos) {
		return welcome.browseAt(), true
	}
	return 0, false
}

func (welcome *Welcome) browseAt() int {
	if welcome == nil {
		return 0
	}
	return len(welcome.dirs)
}

func (welcome *Welcome) View() Node {
	if welcome == nil {
		return nil
	}
	background, ink := Palette(welcome.mode)
	card, selected := welcome.cards()
	muted := fade(ink, background)
	welcome.rows = welcome.rows[:0]
	children := []Node{
		welcome.logo(),
		gap(18),
		welcome.centerText(welcome.title, ink),
		gap(6),
		welcome.centerText("Choose a folder", muted),
		gap(28),
		line("Recent", muted),
		gap(8),
	}
	if len(welcome.dirs) == 0 {
		children = append(children, line("No recent folders", muted), gap(8))
	}
	for i, dir := range welcome.dirs {
		fill := card
		if i == welcome.cursor {
			fill = selected
		}
		row := welcome.row(lewpath.New(dir.Path).Name(), dir.Path, fill, ink, muted)
		welcome.rows = append(welcome.rows, row)
		children = append(children, row, gap(8))
	}
	children = append(children, gap(8))
	welcome.browse = welcome.row("Open a folder", "Browse this computer", Color{240, 162, 2, 255}, Color{28, 22, 8, 255}, Color{80, 58, 12, 255})
	if welcome.cursor == welcome.browseAt() {
		welcome.browse.Fill = &Color{255, 186, 48, 255}
	}
	children = append(children, welcome.browse)
	if welcome.note != "" {
		children = append(children, gap(12), line(welcome.note, muted))
	}
	return &Box{
		Fill:  &background,
		Align: Alignment{0.5, 0.5},
		Child: Column(children...),
	}
}

func (welcome *Welcome) row(name, detail string, fill, ink, muted Color) *Box {
	return &Box{
		Width:   welcomeWidth,
		Height:  58,
		Radius:  12,
		Fill:    &fill,
		Clip:    true,
		Padding: EdgeInsets{16, 8, 16, 8},
		Child: Column(
			textLine(name, ink),
			textLine(detail, muted),
		),
	}
}

func (welcome *Welcome) cards() (card, selected Color) {
	if welcome.mode == daynight.Light {
		return Color{255, 255, 255, 255}, Color{255, 228, 186, 255}
	}
	return Color{20, 22, 30, 255}, Color{48, 56, 78, 255}
}

func (welcome *Welcome) center(child Node, height float32) *Box {
	return &Box{Width: welcomeWidth, Height: height, Align: Alignment{0.5, 0.5}, Child: child}
}

func (welcome *Welcome) centerText(value string, ink Color) *Box {
	return &Box{Width: welcomeWidth, Align: Alignment{0.5, 0}, Child: &Text{Value: value, Ink: ink}}
}

func line(value string, ink Color) *Box {
	return &Box{Width: welcomeWidth, Child: &Text{Value: value, Ink: ink}}
}

func textLine(value string, ink Color) *Box {
	return &Box{Width: welcomeWidth - 32, Child: &Text{Value: value, Ink: ink}}
}

func gap(height float32) *Box {
	return &Box{Width: welcomeWidth, Height: height}
}

func (welcome *Welcome) logo() Node {
	lockup := logoImage()
	width := float32(welcomeWidth - 40)
	height := width
	if bounds := lockup.Bounds(); bounds.Dx() > 0 {
		height = width * float32(bounds.Dy()) / float32(bounds.Dx())
	}
	imageNode := &Image{Src: lockup, Width: width, Height: height}
	if welcome.mode == daynight.Light {
		return &Box{Width: welcomeWidth, Align: Alignment{0.5, 0.5}, Child: imageNode}
	}
	return &Box{
		Width:   welcomeWidth,
		Align:   Alignment{0.5, 0.5},
		Padding: EdgeInsets{20, 16, 20, 16},
		Radius:  16,
		Fill:    &Color{255, 255, 255, 255},
		Child:   imageNode,
	}
}

func fade(ink, background Color) Color {
	mix := func(front, back uint8) uint8 {
		return uint8((int(front) + int(back)*2) / 3)
	}
	return Color{mix(ink.Red, background.Red), mix(ink.Green, background.Green), mix(ink.Blue, background.Blue), 255}
}

type folderPicked struct {
	paths []string
	err   error
}

func (*Welcome) enter(key window.Key) bool {
	return key.Rune == '\r' || key.Rune == '\n' || key.Code == 36 || key.Code == 13 || key.Code == 24
}

func (*Welcome) up(key window.Key) bool {
	return key.Code == 126 || key.Code == 0x26 || key.Code == 111
}

func (*Welcome) down(key window.Key) bool {
	return key.Code == 125 || key.Code == 0x28 || key.Code == 116
}

func (*Welcome) escape(key window.Key) bool {
	return key.Rune == 0x1b || key.Code == 53 || key.Code == 27 || key.Code == 9
}
