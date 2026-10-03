package android

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
)

const (
	statusCanceled   = "canceled"
	statusNoActivity = "no activity"
	statusNoPicker   = "no picker"
)

var (
	errBusy       = errors.New("file dialog already open")
	errMainLooper = errors.New("main looper")
	errDialog     = errors.New("file dialog")
	errURI        = errors.New("file dialog uri")
)

type pick struct {
	status string
	paths  string
}

type waiter struct {
	ch  chan pick
	gen uint64
}

var (
	waitMu  sync.Mutex
	wait    *waiter
	waitGen uint64
)

func beginPick() (<-chan pick, uint64, error) {
	waitMu.Lock()
	defer waitMu.Unlock()
	if wait != nil {
		return nil, 0, errBusy
	}
	waitGen++
	ch := make(chan pick, 1)
	wait = &waiter{ch: ch, gen: waitGen}
	return ch, waitGen, nil
}

func endPick() {
	waitMu.Lock()
	wait = nil
	waitMu.Unlock()
}

func deliverPick(gen uint64, status, paths string) {
	waitMu.Lock()
	defer waitMu.Unlock()
	if wait == nil || wait.gen != gen {
		return
	}
	select {
	case wait.ch <- pick{status: status, paths: paths}:
	default:
	}
}

func extensionList(filters []filedialog.Filter) string {
	return strings.Join(filedialog.Extensions(filters), ",")
}

func listenerArgs(args []any) (string, string, error) {
	if len(args) != 2 {
		return "", "", fmt.Errorf("%w: %d arguments", errDialog, len(args))
	}
	status, err := listenerString(args[0])
	if err != nil {
		return "", "", err
	}
	paths, err := listenerString(args[1])
	if err != nil {
		return "", "", err
	}
	return status, paths, nil
}

func listenerString(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%w: %T", errDialog, v)
	}
	return s, nil
}

func resultOf(status, joined string) ([]string, error) {
	switch status {
	case "":
		paths := splitPaths(joined)
		if len(paths) == 0 {
			return nil, errURI
		}
		return paths, nil
	case statusCanceled:
		return nil, filedialog.ErrCanceled
	case statusNoActivity, statusNoPicker:
		return nil, fmt.Errorf("%w: %s", driver.ErrUnavailable, status)
	default:
		return nil, fmt.Errorf("%w: %s", errDialog, status)
	}
}

func splitPaths(joined string) []string {
	if joined == "" {
		return nil
	}
	var out []string
	for part := range strings.SplitSeq(joined, "\n") {
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
