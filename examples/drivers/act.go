package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/audio_play"
	"github.com/lewtec/lewkit/x/driver/brightness"
	"github.com/lewtec/lewkit/x/driver/camera"
	"github.com/lewtec/lewkit/x/driver/clipboard"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/fetchurl"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/driver/httpclient"
	"github.com/lewtec/lewkit/x/driver/launcher"
	"github.com/lewtec/lewkit/x/driver/media"
	"github.com/lewtec/lewkit/x/driver/messagebox"
	"github.com/lewtec/lewkit/x/driver/notification"
	"github.com/lewtec/lewkit/x/driver/opener"
	"github.com/lewtec/lewkit/x/driver/power"
	"github.com/lewtec/lewkit/x/driver/screen"
	"github.com/lewtec/lewkit/x/driver/screenshot"
	"github.com/lewtec/lewkit/x/driver/share"
	"github.com/lewtec/lewkit/x/driver/terminal"
	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/driver/tray"
	"github.com/lewtec/lewkit/x/driver/treesitter"
	"github.com/lewtec/lewkit/x/driver/volume"
	"github.com/lewtec/lewkit/x/driver/wallpaper"
	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/driver/wm"
	"github.com/lewtec/lewkit/x/http/asset/hastad_nha"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/sound"
	_ "github.com/lewtec/lewkit/x/sound/mp3"
)

func runBrightness(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	switch op {
	case "set":
		level, err := formFloat(r, "level")
		if err != nil {
			return "", err
		}
		if level < 0 || level > 1 {
			return "", errLevel
		}
		return okNote("set", brightness.SetBrightness(ctx, level))
	case "up":
		return okNote("increased", brightness.Increase(ctx))
	case "down":
		return okNote("decreased", brightness.Decrease(ctx))
	default:
		return "", errNoDriverOp
	}
}

func runVolume(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	switch op {
	case "set":
		level, err := formFloat(r, "level")
		if err != nil {
			return "", err
		}
		if level < 0 || level > 1 {
			return "", errLevel
		}
		return okNote("set", volume.SetVolume(ctx, level))
	case "up":
		return okNote("increased", volume.Increase(ctx))
	case "down":
		return okNote("decreased", volume.Decrease(ctx))
	case "mute":
		return okNote("toggled", volume.ToggleMute(ctx))
	default:
		return "", errNoDriverOp
	}
}

func runScreen(ctx context.Context, _ *page, op string, _ *http.Request) (string, error) {
	switch op {
	case "on":
		return okNote("on", screen.SetDPMS(ctx, true))
	case "off":
		return okNote("off", screen.SetDPMS(ctx, false))
	case "toggle":
		return okNote("toggled", screen.ToggleDPMS(ctx))
	case "reset":
		return okNote("reset", screen.Reset(ctx))
	default:
		return "", errNoDriverOp
	}
}

func runClipboard(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	switch op {
	case "text":
		return okNote("copied", clipboard.WriteText(ctx, r.FormValue("text")))
	case "image":
		img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
		img.Set(0, 0, color.NRGBA{R: 200, A: 255})
		return okNote("copied", clipboard.WriteImage(ctx, img))
	default:
		return "", errNoDriverOp
	}
}

func runShare(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "out" {
		return "", errNoDriverOp
	}
	return okNote("shared", share.Out(ctx, share.Item{
		Title: r.FormValue("title"),
		Text:  r.FormValue("text"),
		URL:   r.FormValue("url"),
		Paths: formLines(r.FormValue("paths")),
	}))
}

func runNotification(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "post" {
		return "", errNoDriverOp
	}
	return okNote("posted", notification.Notify(ctx, notification.Notification{
		Title:   r.FormValue("title"),
		Message: r.FormValue("message"),
		Urgency: r.FormValue("urgency"),
	}))
}

func runMessagebox(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "show" {
		return "", errNoDriverOp
	}
	return okNote("shown", messagebox.ShowNotice(ctx, messagebox.Notice{
		Title:   r.FormValue("title"),
		Message: r.FormValue("message"),
		Style:   r.FormValue("style"),
	}))
}

func runMedia(ctx context.Context, _ *page, op string, _ *http.Request) (string, error) {
	switch op {
	case "next":
		return okNote("next", media.Next(ctx))
	case "previous":
		return okNote("previous", media.Previous(ctx))
	case "play":
		return okNote("toggled", media.PlayPause(ctx))
	case "stop":
		return okNote("stopped", media.Stop(ctx))
	default:
		return "", errNoDriverOp
	}
}

func runPower(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	switch op {
	case "lock":
		return okNote("locked", power.Lock(ctx))
	case "logout":
		return okNote("logged out", power.Logout(ctx))
	case "suspend":
		return okNote("suspended", power.Suspend(ctx))
	case "hibernate":
		return okNote("hibernated", power.Hibernate(ctx))
	case "reboot":
		return okNote("rebooting", power.Reboot(ctx))
	case "shutdown":
		return okNote("shutting down", power.Shutdown(ctx))
	case "wake":
		return okNote("sent", power.Wake(ctx, strings.TrimSpace(r.FormValue("mac"))))
	default:
		return "", errNoDriverOp
	}
}

func runCamera(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "capture" {
		return "", errNoDriverOp
	}
	id := r.FormValue("id")
	if id == "" {
		return "", errCameraID
	}
	cams, err := camera.List(ctx)
	if err != nil {
		return "", err
	}
	for _, cam := range cams {
		if cam.ID() != id {
			continue
		}
		ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		img, err := cam.Capture(ctx)
		if err != nil {
			return "", err
		}
		bounds := img.Bounds()
		return fmt.Sprintf("%s %dx%d", cam.Name(), bounds.Dx(), bounds.Dy()), nil
	}
	return "", errCameraMissing
}

func runWM(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	switch op {
	case "scratch":
		return okNote("toggled", wm.ToggleScratchpad(ctx))
	case "rotate":
		return okNote("rotated", wm.RotateWorkspaces(ctx))
	case "switch":
		name := strings.TrimSpace(r.FormValue("workspace"))
		if name == "" {
			return "", errWorkspaceName
		}
		return okNote("switched", wm.SwitchToWorkspace(ctx, name, formOn(r, "move")))
	case "move":
		name := strings.TrimSpace(r.FormValue("workspace"))
		output := strings.TrimSpace(r.FormValue("output"))
		if name == "" || output == "" {
			return "", errWorkspaceOutput
		}
		return okNote("moved", wm.MoveWorkspaceToOutput(ctx, name, output))
	default:
		return "", errNoDriverOp
	}
}

func runScreenshot(ctx context.Context, _ *page, op string, _ *http.Request) (string, error) {
	var kind screenshot.TargetType
	switch op {
	case "all":
		kind = screenshot.TargetAll
	case "output":
		kind = screenshot.TargetOutput
	case "window":
		kind = screenshot.TargetWindow
	case "selection":
		return captureTarget(ctx, screenshot.TargetSelection)
	default:
		return "", errNoDriverOp
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return captureTarget(ctx, kind)
}

func captureTarget(ctx context.Context, kind screenshot.TargetType) (string, error) {
	rect, err := screenshot.ResolveRect(ctx, kind)
	if err != nil {
		return "", err
	}
	img, err := screenshot.Capture(ctx, rect)
	if err != nil {
		return "", err
	}
	bounds := img.Bounds()
	return fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy()), nil
}

func runWallpaper(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "set" {
		return "", errNoDriverOp
	}
	path := strings.TrimSpace(r.FormValue("path"))
	if path == "" {
		return "", errPathEmpty
	}
	return okNote("set", wallpaper.SetStatic(ctx, path))
}

func runLauncher(ctx context.Context, _ *page, op string, _ *http.Request) (string, error) {
	switch op {
	case "run":
		return okNote("opened", launcher.RunApp(ctx))
	case "switch":
		return okNote("opened", launcher.SwitchWindow(ctx))
	default:
		return "", errNoDriverOp
	}
}

func runChooser(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "choose" {
		return "", errNoDriverOp
	}
	var items []launcher.Item
	for _, line := range formLines(r.FormValue("items")) {
		items = append(items, launcher.Item{Label: line, Value: line})
	}
	if len(items) == 0 {
		return "", errNoItems
	}
	item, err := launcher.Choose(ctx, launcher.ChooseOptions{
		Prompt: r.FormValue("prompt"),
		Items:  items,
	})
	if err != nil {
		return "", err
	}
	if item == nil {
		return "", errNoChoice
	}
	if item.Label != "" {
		return item.Label, nil
	}
	return item.Value, nil
}

func runPrompter(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "ask" {
		return "", errNoDriverOp
	}
	return launcher.Prompt(ctx, r.FormValue("prompt"))
}

func runConfirmer(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "ask" {
		return "", errNoDriverOp
	}
	ok, err := launcher.Confirm(ctx, r.FormValue("message"))
	if err != nil {
		return "", err
	}
	if ok {
		return "yes", nil
	}
	return "no", nil
}

func runExec(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "run" {
		return "", errNoDriverOp
	}
	name := strings.TrimSpace(r.FormValue("command"))
	if name == "" {
		return "", errCommandEmpty
	}
	cmd, err := execdriver.Command(name, formLines(r.FormValue("args"))...)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := execdriver.Output(ctx, cmd)
	text := strings.TrimRight(string(out), "\n")
	if err != nil {
		if text == "" {
			return "", err
		}
		return "", fmt.Errorf("%s: %w", text, err)
	}
	if text == "" {
		return "exited", nil
	}
	return text, nil
}

func runOpener(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "open" {
		return "", errNoDriverOp
	}
	target := strings.TrimSpace(r.FormValue("target"))
	if target == "" {
		return "", errTargetEmpty
	}
	return okNote("opened", opener.Open(ctx, target))
}

func runTerminal(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "open" {
		return "", errNoDriverOp
	}
	return okNote("opened", terminal.Open(ctx, terminal.Options{
		Title:   r.FormValue("title"),
		Command: strings.TrimSpace(r.FormValue("command")),
		Args:    formLines(r.FormValue("args")),
	}))
}

func runFile(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "choose" {
		return "", errNoDriverOp
	}
	paths, err := filedialog.Choose(ctx, filedialog.Request{
		Title:     r.FormValue("title"),
		Directory: r.FormValue("directory"),
		Name:      r.FormValue("name"),
		Multiple:  formOn(r, "multiple"),
		Folder:    formOn(r, "folder"),
		Save:      formOn(r, "save"),
	})
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "none", nil
	}
	return strings.Join(paths, "\n"), nil
}

func runFetch(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "fetch" {
		return "", errNoDriverOp
	}
	urls := formLines(r.FormValue("urls"))
	if len(urls) == 0 {
		return "", fetchurl.ErrNoURLs
	}
	source, err := driver.Get[fetchurl.Driver](ctx)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var buf bytes.Buffer
	err = source.Fetch(ctx, fetchurl.FetchOptions{
		URLs: urls,
		Algo: strings.TrimSpace(r.FormValue("algo")),
		Hash: strings.TrimSpace(r.FormValue("hash")),
		Out:  &buf,
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d bytes", buf.Len()), nil
}

func runHTTP(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "request" {
		return "", errNoDriverOp
	}
	raw := strings.TrimSpace(r.FormValue("url"))
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errURLScheme
	}
	method := strings.ToUpper(strings.TrimSpace(r.FormValue("method")))
	if method == "" {
		method = http.MethodGet
	}
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost:
	default:
		return "", errHTTPMethod
	}
	source, err := driver.Get[httpclient.Driver](ctx)
	if err != nil {
		return "", err
	}
	client := source.Client()
	if client == nil {
		return "", errNoHTTPClient
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, raw, nil)
	if err != nil {
		return "", err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	n, err := io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s %d bytes", res.Status, n), nil
}

func runAudio(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "play" {
		return "", errNoDriverOp
	}
	pipe, err := sound.Decode(hastad_nha.Name, bytes.NewReader(hastad_nha.Bytes()))
	if err != nil {
		return "", err
	}
	pcm, err := io.ReadAll(pipe)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	writer, err := audio_play.Open(ctx, audio_play.Config{
		Sink:   strings.TrimSpace(r.FormValue("sink")),
		Format: pipe.Format(),
		Name:   hastad_nha.Name,
	})
	if err != nil {
		return "", err
	}
	_, writeErr := writer.Write(pcm)
	closeErr := writer.Close()
	if writeErr != nil {
		return "", writeErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return "played " + pipe.Duration().Round(time.Millisecond).String(), nil
}

func runWindow(ctx context.Context, p *page, op string, r *http.Request) (string, error) {
	switch op {
	case "open":
		width, err := formInt(r, "width", 640)
		if err != nil {
			return "", err
		}
		height, err := formInt(r, "height", 480)
		if err != nil {
			return "", err
		}
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			title = release.Name()
		}
		size, err := p.held.openWindow(p.ctx, window.Config{Title: title, Width: width, Height: height})
		if err != nil {
			return "", err
		}
		return "open " + size, nil
	case "close":
		return okNote("closed", p.held.closeWindow())
	default:
		return "", errNoDriverOp
	}
}

func runWeb(ctx context.Context, p *page, op string, r *http.Request) (string, error) {
	switch op {
	case "open":
		width, err := formInt(r, "width", 640)
		if err != nil {
			return "", err
		}
		height, err := formInt(r, "height", 480)
		if err != nil {
			return "", err
		}
		title := strings.TrimSpace(r.FormValue("title"))
		if title == "" {
			title = release.Name()
		}
		return okNote("open", p.held.openWeb(p.ctx, webview.Config{
			Title:  title,
			Width:  width,
			Height: height,
			HTML:   r.FormValue("html"),
		}))
	case "close":
		return okNote("closed", p.held.closeWeb())
	case "eval":
		return p.held.evalWeb(ctx, r.FormValue("script"))
	default:
		return "", errNoDriverOp
	}
}

func runTray(ctx context.Context, p *page, op string, r *http.Request) (string, error) {
	switch op {
	case "open":
		return okNote("open", p.held.openTray(p.ctx, trayConfig(r)))
	case "update":
		return okNote("updated", p.held.updateTray(trayConfig(r)))
	case "close":
		return okNote("closed", p.held.closeTray())
	default:
		return "", errNoDriverOp
	}
}

func trayConfig(r *http.Request) tray.Config {
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			img.Set(x, y, color.NRGBA{R: 32, G: 160, B: 96, A: 255})
		}
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = release.Name()
	}
	return tray.Config{
		Title:   title,
		Tooltip: r.FormValue("tooltip"),
		Icon:    tray.Icon{Image: img},
	}
}

func runTreesitter(ctx context.Context, _ *page, op string, r *http.Request) (string, error) {
	if op != "parse" {
		return "", errNoDriverOp
	}
	name := strings.TrimSpace(r.FormValue("language"))
	if name == "" {
		return "", errLanguageEmpty
	}
	tree, err := treesitter.Parse(ctx, name, []byte(r.FormValue("source")))
	if err != nil {
		return "", err
	}
	if err := tree.Parsed(); err != nil {
		return "", err
	}
	root := tree.RootNode()
	return fmt.Sprintf("%s children=%d", root.Type(), root.ChildCount()), nil
}

func runThread(ctx context.Context, _ *page, op string, _ *http.Request) (string, error) {
	if op != "ping" {
		return "", errNoDriverOp
	}
	ui, err := driver.Get[thread.Driver](ctx)
	if err != nil {
		return "", err
	}
	var on bool
	ui.Do(func() { on = ui.On() })
	if on {
		return "ran on the ui thread", nil
	}
	return "ran, but not on the ui thread", nil
}

func okNote(note string, err error) (string, error) {
	if err != nil {
		return "", err
	}
	if note == "" {
		return "done", nil
	}
	return note, nil
}

func formFloat(r *http.Request, name string) (float64, error) {
	text := strings.TrimSpace(r.FormValue(name))
	if text == "" {
		return 0, fmt.Errorf("%s: %w", name, errEmpty)
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return value, nil
}

func formInt(r *http.Request, name string, fallback int) (int, error) {
	text := strings.TrimSpace(r.FormValue(name))
	if text == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return value, nil
}

func formOn(r *http.Request, name string) bool {
	switch r.FormValue(name) {
	case "on", "true", "1":
		return true
	default:
		return false
	}
}

func formLines(text string) []string {
	var out []string
	for line := range strings.SplitSeq(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
