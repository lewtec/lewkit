package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/audio_play"
	"github.com/lewtec/lewkit/x/driver/battery"
	"github.com/lewtec/lewkit/x/driver/brightness"
	"github.com/lewtec/lewkit/x/driver/bundle"
	"github.com/lewtec/lewkit/x/driver/camera"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/dirs"
	"github.com/lewtec/lewkit/x/driver/httpclient"
	"github.com/lewtec/lewkit/x/driver/media"
	"github.com/lewtec/lewkit/x/driver/screen"
	"github.com/lewtec/lewkit/x/driver/thread"
	"github.com/lewtec/lewkit/x/driver/treesitter"
	"github.com/lewtec/lewkit/x/driver/volume"
	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/driver/wm"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/release"
)

type panel struct {
	rows []row
	acts []act
}

type spec struct {
	load func(ctx context.Context, _ *page) (panel, error)
	run  func(context.Context, *page, string, *http.Request) (string, error)
}

var (
	errNoPanel         = errors.New("no panel")
	errNoDriverOp      = errors.New("no driver op")
	errNoBrightness    = errors.New("no brightness device")
	errNoMetadata      = errors.New("no metadata")
	errNoHTTPClient    = errors.New("no http client")
	errLevel           = errors.New("level must be between 0 and 1")
	errCameraID        = errors.New("camera id is empty")
	errCameraMissing   = errors.New("camera is not listed")
	errWorkspaceName   = errors.New("workspace name is empty")
	errWorkspaceOutput = errors.New("workspace and output are required")
	errPathEmpty       = errors.New("path is empty")
	errNoItems         = errors.New("add at least one item")
	errNoChoice        = errors.New("no item chosen")
	errCommandEmpty    = errors.New("command is empty")
	errTargetEmpty     = errors.New("target is empty")
	errURLScheme       = errors.New("url must be http or https")
	errHTTPMethod      = errors.New("method must be GET, HEAD, or POST")
	errLanguageEmpty   = errors.New("language is empty")
	errEmpty           = errors.New("empty")
	errWindowOpen      = errors.New("a window is already open")
	errNoWindow        = errors.New("no window open")
	errWebOpen         = errors.New("a web view is already open")
	errNoWeb           = errors.New("no web view open")
	errTrayOpen        = errors.New("a tray is already open")
	errNoTray          = errors.New("no tray open")
	errSwapOpen        = errors.New("a vulkan window is already open")
	errNoSwap          = errors.New("no vulkan window open")
)

var driverSpecs = map[string]spec{
	"audio_play":         {load: loadAudio, run: runAudio},
	"battery":            {load: loadBattery},
	"brightness":         {load: loadBrightness, run: runBrightness},
	"bundle":             {load: loadBundle},
	"camera":             {load: loadCamera, run: runCamera},
	"clipboard":          {load: loadClipboard, run: runClipboard},
	"daynight":           {load: loadDaynight},
	"dirs":               {load: loadDirs},
	"exec":               {load: loadExec, run: runExec},
	"fetchurl":           {load: loadFetch, run: runFetch},
	"filedialog":         {load: loadFile, run: runFile},
	"httpclient":         {load: loadHTTP, run: runHTTP},
	"launcher":           {load: loadLauncher, run: runLauncher},
	"launcher.Chooser":   {load: loadChooser, run: runChooser},
	"launcher.Prompter":  {load: loadPrompter, run: runPrompter},
	"launcher.Confirmer": {load: loadConfirmer, run: runConfirmer},
	"media":              {load: loadMedia, run: runMedia},
	"ndarray.Evaluator":  {load: loadEval},
	"notification":       {load: loadNotification, run: runNotification},
	"opener":             {load: loadOpener, run: runOpener},
	"power":              {load: loadPower, run: runPower},
	"screen":             {load: loadScreen, run: runScreen},
	"screenshot":         {load: loadScreenshot, run: runScreenshot},
	"share":              {load: loadShare, run: runShare},
	"terminal":           {load: loadTerminal, run: runTerminal},
	"thread":             {load: loadThread, run: runThread},
	"tray":               {load: loadTray, run: runTray},
	"treesitter":         {load: loadTreesitter, run: runTreesitter},
	"volume":             {load: loadVolume, run: runVolume},
	"vulkan.Device":      {load: loadVulkan},
	"vulkanwindow":       {load: loadSwap, run: runSwap},
	"wallpaper":          {load: loadWallpaper, run: runWallpaper},
	"webview":            {load: loadWeb, run: runWeb},
	"window":             {load: loadWindow, run: runWindow},
	"wm":                 {load: loadWM, run: runWM},
}

func actOf(op, label string, fields ...field) act {
	return act{Op: op, Label: label, Fields: fields}
}

func text(name, label, value, placeholder string) field {
	return field{Name: name, Label: label, Kind: "text", Value: value, Placeholder: placeholder}
}

func area(name, label, value, placeholder string) field {
	return field{Name: name, Label: label, Kind: "area", Value: value, Placeholder: placeholder}
}

func number(name, label, value, min, max, step string) field {
	return field{Name: name, Label: label, Kind: "number", Value: value, Min: min, Max: max, Step: step}
}

func level(name, label, value string) field {
	return number(name, label, value, "0", "1", "0.01")
}

func whole(name, label, value string) field {
	return number(name, label, value, "1", "10000", "1")
}

func check(name, label string) field {
	return field{Name: name, Label: label, Kind: "check"}
}

func hidden(name, value string) field {
	return field{Name: name, Kind: "hidden", Value: value}
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func exampleAppID() string {
	id, err := release.AppID()
	if err != nil || id == "" {
		return "br.tec.lew.drivers"
	}
	return id
}

func loadBattery(ctx context.Context, _ *page) (panel, error) {
	status, err := battery.BatteryStatus(ctx)
	if err != nil {
		return panel{}, err
	}
	return panel{rows: []row{{Label: "Status", Value: string(status)}}}, nil
}

func loadDaynight(ctx context.Context, _ *page) (panel, error) {
	mode, err := daynight.Current(ctx)
	if err != nil {
		return panel{}, err
	}
	return panel{rows: []row{{Label: "Mode", Value: mode.String()}}}, nil
}

func loadDirs(ctx context.Context, _ *page) (panel, error) {
	tree, err := dirs.Resolve(ctx, exampleAppID())
	if err != nil {
		return panel{}, err
	}
	return panel{rows: []row{
		{Label: "App", Value: exampleAppID()},
		{Label: "Data", Value: tree.Data},
		{Label: "Cache", Value: tree.Cache},
		{Label: "Config", Value: tree.Config},
		{Label: "Inbox", Value: tree.Inbox},
	}}, nil
}

func loadBundle(ctx context.Context, _ *page) (panel, error) {
	root, err := bundle.ResolveID(ctx, exampleAppID())
	if err != nil {
		return panel{}, err
	}
	return panel{rows: []row{
		{Label: "ID", Value: root.ID},
		{Label: "Data", Value: root.Data},
		{Label: "Cache", Value: root.Cache},
		{Label: "Config", Value: root.Config},
		{Label: "Profile", Value: root.Profile},
		{Label: "Share", Value: bundle.SharePath(root)},
	}}, nil
}

func loadBrightness(ctx context.Context, _ *page) (panel, error) {
	dev, err := brightness.Status(ctx)
	prefill := ""
	if err == nil && dev != nil {
		prefill = fmt.Sprintf("%.2f", dev.Brightness)
	}
	acts := []act{
		actOf("up", "Increase"),
		actOf("down", "Decrease"),
		actOf("set", "Set brightness", level("level", "Level", prefill)),
	}
	if err != nil {
		return panel{acts: acts}, err
	}
	if dev == nil {
		return panel{acts: acts}, errNoBrightness
	}
	return panel{
		rows: []row{
			{Label: "Device", Value: dev.Name},
			{Label: "Brightness", Value: fmt.Sprintf("%.0f%% (%s)", dev.Brightness*100, prefill)},
		},
		acts: acts,
	}, nil
}

func loadVolume(ctx context.Context, _ *page) (panel, error) {
	value, verr := volume.GetVolume(ctx)
	muted, merr := volume.GetMute(ctx)
	sink, serr := volume.SinkName(ctx)
	prefill := ""
	if verr == nil {
		prefill = fmt.Sprintf("%.2f", value)
	}
	acts := []act{
		actOf("up", "Increase"),
		actOf("down", "Decrease"),
		actOf("mute", "Toggle mute"),
		actOf("set", "Set volume", level("level", "Level", prefill)),
	}
	if verr != nil && merr != nil && serr != nil {
		return panel{acts: acts}, verr
	}
	rows := []row{
		{Label: "Sink", Value: valueOrErr(sink, serr)},
		{Label: "Volume", Value: valueOrErr(prefill, verr)},
		{Label: "Mute", Value: boolOrErr(muted, merr, "muted", "not muted")},
	}
	return panel{rows: rows, acts: acts}, nil
}

func loadScreen(ctx context.Context, _ *page) (panel, error) {
	acts := []act{
		actOf("on", "Turn display on"),
		actOf("off", "Turn display off"),
		actOf("toggle", "Toggle display power"),
		actOf("reset", "Reset layout"),
	}
	on, err := screen.IsDPMSOn(ctx)
	if err != nil {
		return panel{acts: acts}, err
	}
	return panel{rows: []row{{Label: "Display", Value: onOff(on)}}, acts: acts}, nil
}

func loadClipboard(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("text", "Copy text", text("text", "Text", "lewkit", "")),
		actOf("image", "Copy an image"),
	}}, nil
}

func loadShare(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("out", "Share",
			text("title", "Title", "", ""),
			text("text", "Text", "lewkit", ""),
			text("url", "URL", "", "https://"),
			area("paths", "Files", "", "absolute path, one per line"),
		),
	}}, nil
}

func loadNotification(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("post", "Post",
			text("title", "Title", "lewkit", ""),
			area("message", "Message", "drivers", ""),
			text("urgency", "Urgency", "normal", "low, normal, or critical"),
		),
	}}, nil
}

func loadMedia(ctx context.Context, _ *page) (panel, error) {
	acts := []act{
		actOf("previous", "Previous"),
		actOf("play", "Play or pause"),
		actOf("next", "Next"),
		actOf("stop", "Stop"),
	}
	meta, err := media.GetMetadata(ctx)
	if err != nil {
		return panel{acts: acts}, err
	}
	if meta == nil {
		return panel{acts: acts}, errNoMetadata
	}
	return panel{rows: []row{
		{Label: "Player", Value: meta.Player},
		{Label: "Status", Value: string(meta.Status)},
		{Label: "Title", Value: meta.Title},
		{Label: "Artist", Value: meta.Artist},
		{Label: "Position", Value: seconds(meta.Position)},
		{Label: "Length", Value: seconds(meta.Length)},
	}, acts: acts}, nil
}

func loadPower(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("lock", "Lock session"),
		actOf("logout", "Log out"),
		actOf("suspend", "Suspend"),
		actOf("hibernate", "Hibernate"),
		actOf("reboot", "Reboot"),
		actOf("shutdown", "Shut down"),
		actOf("wake", "Wake on LAN", text("mac", "MAC", "", "aa:bb:cc:dd:ee:ff")),
	}}, nil
}

func loadCamera(ctx context.Context, _ *page) (panel, error) {
	cams, err := camera.List(ctx)
	if err != nil {
		return panel{}, err
	}
	if len(cams) == 0 {
		return panel{rows: []row{{Label: "Cameras", Value: "none"}}}, nil
	}
	rows := make([]row, 0, len(cams))
	acts := make([]act, 0, len(cams))
	for _, cam := range cams {
		rows = append(rows, row{Label: cam.Name(), Value: cam.ID()})
		acts = append(acts, actOf("capture", "Capture "+cam.Name(), hidden("id", cam.ID())))
	}
	return panel{rows: rows, acts: acts}, nil
}

func loadWM(ctx context.Context, _ *page) (panel, error) {
	acts := wmActs()
	outputs, oerr := wm.GetOutputs(ctx)
	workspaces, werr := wm.GetWorkspaces(ctx)
	name, rect, ferr := wm.GetFocusedOutput(ctx)
	windowRect, winerr := wm.GetFocusedWindowRect(ctx)
	if oerr != nil && werr != nil && ferr != nil && winerr != nil {
		return panel{acts: acts}, oerr
	}
	focused := name
	if ferr != nil {
		focused = ferr.Error()
	} else if rect != nil {
		focused = name + " " + rectText(*rect)
	}
	return panel{rows: []row{
		{Label: "Outputs", Value: outputText(outputs, oerr)},
		{Label: "Workspaces", Value: workspaceText(workspaces, werr)},
		{Label: "Focused output", Value: focused},
		{Label: "Focused window", Value: rectPtr(windowRect, winerr)},
	}, acts: acts}, nil
}

func wmActs() []act {
	return []act{
		actOf("scratch", "Toggle scratchpad"),
		actOf("rotate", "Rotate workspaces"),
		actOf("switch", "Switch workspace",
			text("workspace", "Workspace", "", "1"),
			check("move", "Move the focused window"),
		),
		actOf("move", "Move workspace to output",
			text("workspace", "Workspace", "", ""),
			text("output", "Output", "", ""),
		),
	}
}

func loadScreenshot(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("all", "Capture all outputs"),
		actOf("output", "Capture the focused output"),
		actOf("window", "Capture the focused window"),
		actOf("selection", "Select an area and capture"),
	}}, nil
}

func loadWallpaper(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("set", "Set wallpaper", text("path", "Image", "", "absolute path")),
	}}, nil
}

func loadLauncher(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("run", "Open the app launcher"),
		actOf("switch", "Open the window switcher"),
	}}, nil
}

func loadChooser(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("choose", "Choose",
			text("prompt", "Prompt", "Open", ""),
			area("items", "Items", "one\ntwo", "one item per line"),
		),
	}}, nil
}

func loadPrompter(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("ask", "Ask", text("prompt", "Prompt", "Name", "")),
	}}, nil
}

func loadConfirmer(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("ask", "Ask", text("message", "Message", "Continue?", "")),
	}}, nil
}

func loadExec(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("run", "Run",
			text("command", "Command", "", "name"),
			area("args", "Arguments", "", "one argument per line"),
		),
	}}, nil
}

func loadOpener(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("open", "Open", text("target", "Target", "", "path or URL")),
	}}, nil
}

func loadTerminal(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("open", "Open terminal",
			text("title", "Title", "lewkit", ""),
			text("command", "Command", "", "empty opens the emulator"),
			area("args", "Arguments", "", "one argument per line"),
		),
	}}, nil
}

func loadFile(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("choose", "Choose",
			text("title", "Title", "Open", ""),
			text("directory", "Directory", "", ""),
			text("name", "Name", "", ""),
			check("multiple", "Multiple"),
			check("folder", "Folder"),
			check("save", "Save"),
		),
	}}, nil
}

func loadFetch(ctx context.Context, _ *page) (panel, error) {
	return panel{acts: []act{
		actOf("fetch", "Fetch",
			area("urls", "URLs", "", "one URL per line"),
			text("algo", "Algorithm", "sha256", ""),
			text("hash", "Hash", "", ""),
		),
	}}, nil
}

func loadHTTP(ctx context.Context, _ *page) (panel, error) {
	acts := []act{actOf("request", "Send",
		text("method", "Method", "GET", ""),
		text("url", "URL", "", "https://example.com"),
	)}
	source, err := driver.Get[httpclient.Driver](ctx)
	if err != nil {
		return panel{acts: acts}, err
	}
	client := source.Client()
	if client == nil {
		return panel{acts: acts}, errNoHTTPClient
	}
	timeout := "none"
	if client.Timeout > 0 {
		timeout = client.Timeout.String()
	}
	return panel{rows: []row{
		{Label: "Client", Value: "ready"},
		{Label: "Timeout", Value: timeout},
	}, acts: acts}, nil
}

func loadAudio(ctx context.Context, _ *page) (panel, error) {
	acts := []act{actOf("play", "Play 100 ms", text("sink", "Sink", "", "default"))}
	sinks, err := audio_play.Sinks(ctx)
	if err != nil {
		return panel{acts: acts}, err
	}
	if len(sinks) == 0 {
		return panel{rows: []row{{Label: "Sinks", Value: "none"}}, acts: acts}, nil
	}
	rows := make([]row, 0, len(sinks))
	for _, sink := range sinks {
		label := sink.Name
		if label == "" {
			label = sink.ID
		}
		rows = append(rows, row{Label: label, Value: sink.ID})
	}
	return panel{rows: rows, acts: acts}, nil
}

func loadWindow(ctx context.Context, p *page) (panel, error) {
	open, size := p.held.windowView()
	if open {
		return panel{
			rows: []row{{Label: "Window", Value: "open"}, {Label: "Size", Value: size}},
			acts: []act{actOf("close", "Close")},
		}, nil
	}
	return panel{
		rows: []row{{Label: "Window", Value: "closed"}},
		acts: []act{actOf("open", "Open",
			text("title", "Title", "lewkit", ""),
			whole("width", "Width", "640"),
			whole("height", "Height", "480"),
		)},
	}, nil
}

func loadWeb(ctx context.Context, p *page) (panel, error) {
	open, title := p.held.webView()
	if open {
		label := title
		if label == "" {
			label = "open"
		}
		return panel{
			rows: []row{{Label: "Web view", Value: label}},
			acts: []act{
				actOf("close", "Close"),
				actOf("eval", "Evaluate", area("script", "JavaScript", "1+1", "")),
			},
		}, nil
	}
	return panel{
		rows: []row{{Label: "Web view", Value: "closed"}},
		acts: []act{actOf("open", "Open",
			text("title", "Title", "lewkit", ""),
			whole("width", "Width", "640"),
			whole("height", "Height", "480"),
			area("html", "HTML", "<p>lewkit</p>", ""),
		)},
	}, nil
}

func loadTray(ctx context.Context, p *page) (panel, error) {
	open, tip := p.held.trayView()
	if open {
		return panel{
			rows: []row{{Label: "Tray", Value: "open"}, {Label: "Tip", Value: tip}},
			acts: []act{
				actOf("close", "Close"),
				actOf("update", "Update",
					text("title", "Title", "lewkit", ""),
					text("tooltip", "Tooltip", tip, ""),
				),
			},
		}, nil
	}
	return panel{
		rows: []row{{Label: "Tray", Value: "closed"}},
		acts: []act{actOf("open", "Open",
			text("title", "Title", "lewkit", ""),
			text("tooltip", "Tooltip", "", ""),
		)},
	}, nil
}

func loadTreesitter(ctx context.Context, _ *page) (panel, error) {
	names, err := treesitter.Names(ctx)
	acts := []act{actOf("parse", "Parse",
		text("language", "Language", treeLang(names), ""),
		area("source", "Source", "{\n  \"ok\": true\n}", ""),
	)}
	if err != nil {
		return panel{acts: acts}, err
	}
	listed := "none"
	if len(names) > 0 {
		listed = strings.Join(names, ", ")
	}
	return panel{rows: []row{
		{Label: "Grammars", Value: strconv.Itoa(len(names))},
		{Label: "Names", Value: listed},
	}, acts: acts}, nil
}

func loadThread(ctx context.Context, _ *page) (panel, error) {
	acts := []act{actOf("ping", "Run on the UI thread")}
	ui, err := driver.Get[thread.Driver](ctx)
	if err != nil {
		return panel{acts: acts}, err
	}
	return panel{rows: []row{
		{Label: "UI thread bound", Value: yesNo(ui.Bound())},
		{Label: "Request on UI thread", Value: yesNo(ui.On())},
	}, acts: acts}, nil
}

func loadVulkan(ctx context.Context, _ *page) (panel, error) {
	return handleRows(vulkan.List(ctx))
}

func loadEval(ctx context.Context, _ *page) (panel, error) {
	return handleRows(driver.List[ndarray.Evaluator](ctx))
}

func loadSwap(ctx context.Context, p *page) (panel, error) {
	open, size := p.held.swapView()
	if open {
		return panel{
			rows: []row{{Label: "Screen", Value: "open"}, {Label: "Size", Value: size}},
			acts: []act{actOf("close", "Close")},
		}, nil
	}
	return panel{
		rows: []row{{Label: "Screen", Value: "closed"}},
		acts: []act{actOf("open", "Open",
			whole("width", "Width", "640"),
			whole("height", "Height", "480"),
		)},
	}, nil
}

func handleRows[T any](handles []driver.Handle[T], err error) (panel, error) {
	if err != nil {
		return panel{}, err
	}
	if len(handles) == 0 {
		return panel{rows: []row{{Label: "Drivers", Value: "none"}}}, nil
	}
	rows := make([]row, 0, len(handles))
	for i, handle := range handles {
		value := fmt.Sprintf("%s · weight %d", handle.ID, handle.Weight)
		if i == 0 {
			value += " · selected"
		}
		rows = append(rows, row{Label: handle.Name, Value: value})
	}
	return panel{rows: rows}, nil
}

func treeLang(names []string) string {
	for _, name := range names {
		if name == "json" {
			return name
		}
	}
	if len(names) > 0 {
		return names[0]
	}
	return ""
}

func seconds(us int64) string {
	return fmt.Sprintf("%.1f s", float64(us)/1e6)
}

func valueOrErr(value string, err error) string {
	if err != nil {
		return err.Error()
	}
	if value == "" {
		return "none"
	}
	return value
}

func boolOrErr(v bool, err error, yes, no string) string {
	if err != nil {
		return err.Error()
	}
	if v {
		return yes
	}
	return no
}

func outputText(outputs []wm.Output, err error) string {
	if err != nil {
		return err.Error()
	}
	if len(outputs) == 0 {
		return "none"
	}
	var b strings.Builder
	for i, out := range outputs {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s %s %s", out.Name, out.CurrentWorkspace, rectText(out.Rect))
		if out.Focused {
			b.WriteString(" focused")
		}
	}
	return b.String()
}

func workspaceText(workspaces []wm.Workspace, err error) string {
	if err != nil {
		return err.Error()
	}
	if len(workspaces) == 0 {
		return "none"
	}
	var b strings.Builder
	for i, space := range workspaces {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%s on %s", space.Name, space.Output)
		if space.Focused {
			b.WriteString(" focused")
		}
	}
	return b.String()
}

func rectText(r wm.Rect) string {
	return fmt.Sprintf("%d,%d %dx%d", r.X, r.Y, r.Width, r.Height)
}

func rectPtr(r *wm.Rect, err error) string {
	if err != nil {
		return err.Error()
	}
	if r == nil {
		return "none"
	}
	return rectText(*r)
}
