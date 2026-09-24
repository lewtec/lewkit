//go:build linux

package portal

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
)

var (
	errPortal = errors.New("file dialog portal")
	errURI    = errors.New("file dialog uri")
)

const (
	responseSuccess = 0
	responseCancel  = 1
	globRule        = uint32(0)
)

type rule struct {
	Kind uint32
	Glob string
}

type filter struct {
	Name  string
	Rules []rule
}

var handles atomic.Uint64

// Available reports whether service owns the file chooser on the session bus.
func Available(ctx context.Context, service string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	defer conn.Close()
	var owned bool
	err = conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus").
		CallWithContext(ctx, "org.freedesktop.DBus.NameHasOwner", 0, service).
		Store(&owned)
	if err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	if !owned {
		return fmt.Errorf("%w: %s", driver.ErrIncompatible, service)
	}
	return nil
}

// Choose shows the dialog on service and returns filesystem paths.
func Choose(ctx context.Context, service string, req filedialog.Request) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", filedialog.ErrRequest, err)
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", driver.ErrUnavailable, err)
	}
	defer conn.Close()

	method := iface + ".OpenFile"
	if req.Save {
		method = iface + ".SaveFile"
	}
	var response uint32
	var results map[string]dbus.Variant
	err = conn.Object(service, objectPath).CallWithContext(
		ctx,
		method,
		0,
		nextHandle(),
		appID,
		"",
		req.TitleOrDefault(),
		options(req),
	).Store(&response, &results)
	if err != nil {
		return nil, err
	}
	return pathsFromResults(response, results)
}

func nextHandle() dbus.ObjectPath {
	return dbus.ObjectPath(fmt.Sprintf("/org/lewtec/lewkit/filedialog/%d", handles.Add(1)))
}

func options(req filedialog.Request) map[string]dbus.Variant {
	out := map[string]dbus.Variant{
		"modal":     dbus.MakeVariant(true),
		"multiple":  dbus.MakeVariant(req.Multiple),
		"directory": dbus.MakeVariant(req.Folder),
	}
	if req.Directory != "" {
		out["current_folder"] = dbus.MakeVariant(append([]byte(req.Directory), 0))
	}
	if req.Save && req.Name != "" {
		out["current_name"] = dbus.MakeVariant(req.Name)
	}
	if filters := filtersOf(req); len(filters) > 0 {
		out["filters"] = dbus.MakeVariant(filters)
	}
	return out
}

func filtersOf(req filedialog.Request) []filter {
	var out []filter
	for _, item := range req.Filters {
		var rules []rule
		for _, pattern := range item.Patterns {
			if pattern == "" {
				continue
			}
			rules = append(rules, rule{Kind: globRule, Glob: pattern})
		}
		if len(rules) == 0 {
			continue
		}
		name := item.Name
		if name == "" {
			name = rules[0].Glob
		}
		out = append(out, filter{Name: name, Rules: rules})
	}
	return out
}

func pathsFromResults(response uint32, results map[string]dbus.Variant) ([]string, error) {
	if response == responseCancel {
		return nil, filedialog.ErrCanceled
	}
	if response != responseSuccess {
		return nil, fmt.Errorf("%w: response %d", errPortal, response)
	}
	raw, ok := results["uris"]
	if !ok {
		return nil, fmt.Errorf("%w: no uris", errPortal)
	}
	uris, err := uriList(raw)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(uris))
	for _, item := range uris {
		path, err := pathFromURI(item)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: no uris", errPortal)
	}
	return paths, nil
}

func uriList(v dbus.Variant) ([]string, error) {
	switch got := v.Value().(type) {
	case []string:
		return got, nil
	default:
		return nil, fmt.Errorf("%w: %T", errURI, v.Value())
	}
}

func pathFromURI(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errURI, err)
	}
	if parsed.Scheme != "file" || (parsed.Host != "" && parsed.Host != "localhost") || parsed.Path == "" {
		return "", fmt.Errorf("%w: %q", errURI, raw)
	}
	return parsed.Path, nil
}
