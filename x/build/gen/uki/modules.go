package uki

import (
	"path"
	"slices"
	"strings"
)

// moduleStems are the devices a guest window and a serial-less net need.
// Their dependencies are pulled from modules.dep.
//
// simpledrm is the EFI GOP / ramfb console UTM shows by default.
// virtio-gpu and bochs cover a virtio display and VGA. The USB host
// controllers and usbhid are how UTM delivers the keyboard and tablet.
// evdev is loaded before those drivers so /dev/input exists when they probe.
var moduleStems = []string{
	"simpledrm",
	"virtio-gpu",
	"bochs",
	"evdev",
	"virtio-input",
	"hid-generic",
	"usbhid",
	"xhci-pci",
	"ehci-pci",
	"uhci-hcd",
	"virtio-net",
}

func moduleStem(name string) string {
	base := path.Base(name)
	base = strings.TrimSuffix(base, ".gz")
	base = strings.TrimSuffix(base, ".ko")
	return strings.ReplaceAll(base, "_", "-")
}

func parseDep(text string) map[string][]string {
	graph := map[string][]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		left, right, _ := strings.Cut(line, ":")
		left = strings.TrimSpace(left)
		var deps []string
		for _, dep := range strings.Fields(right) {
			deps = append(deps, dep)
		}
		graph[left] = deps
	}
	return graph
}

func parseBuiltin(text string) map[string]bool {
	set := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			set[line] = true
			set[line+".gz"] = true
		}
	}
	return set
}

// moduleOrder returns ko paths, dependencies first, for the wanted stems.
func moduleOrder(depText, builtinText string) ([]string, error) {
	graph := parseDep(depText)
	builtin := parseBuiltin(builtinText)
	var roots []string
	for _, want := range moduleStems {
		names := make([]string, 0, len(graph))
		for name := range graph {
			names = append(names, name)
		}
		slices.Sort(names)
		var found string
		for _, name := range names {
			if moduleStem(name) == want {
				found = name
				break
			}
		}
		if found == "" {
			return nil, errMissing(want)
		}
		roots = append(roots, found)
	}
	state := map[string]int{}
	var out []string
	var walk func(string)
	walk = func(name string) {
		switch state[name] {
		case 1, 2:
			return
		}
		state[name] = 1
		for _, dep := range graph[name] {
			walk(dep)
		}
		state[name] = 2
		if builtin[name] || builtin[strings.TrimSuffix(name, ".gz")] {
			return
		}
		out = append(out, name)
	}
	for _, root := range roots {
		walk(root)
	}
	return out, nil
}

type missingModule string

func errMissing(stem string) error { return missingModule(stem) }

func (m missingModule) Error() string {
	return "uki: kernel module " + string(m) + " is not in this package"
}
