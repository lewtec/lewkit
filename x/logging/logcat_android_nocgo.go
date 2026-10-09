//go:build android && !cgo

package logging

import (
	"io"
	"runtime"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/lewtec/lewkit/x/release"
)

// Priorities match the bionic android/log.h constants.
const (
	logDebug = 3
	logInfo  = 4
	logWarn  = 5
	logError = 6
)

// Logcat writes process logs to Android logcat under the tag [release.Name].
// __android_log_write is resolved from liblog.so.
func Logcat() io.Writer { return logcatWriter{} }

type logcatWriter struct{}

func (logcatWriter) Write(p []byte) (int, error) {
	text := strings.TrimRight(string(p), "\n")
	if text == "" {
		return len(p), nil
	}
	fn, err := androidLog()
	if err != nil {
		return len(p), nil
	}
	tag := append([]byte(release.Name()), 0)
	for _, line := range strings.Split(text, "\n") {
		body := append([]byte(line), 0)
		fn(int32(logPriority(line)), &tag[0], &body[0])
		runtime.KeepAlive(body)
	}
	runtime.KeepAlive(tag)
	return len(p), nil
}

var _ io.Writer = logcatWriter{}

var (
	logOnce sync.Once
	logFn   func(prio int32, tag, text *byte) int32
	logErr  error
)

func androidLog() (func(prio int32, tag, text *byte) int32, error) {
	logOnce.Do(func() {
		lib, err := native.Open("liblog.so", native.Now)
		if err != nil {
			logErr = err
			return
		}
		sym, err := native.Symbol(lib, "__android_log_write")
		if err != nil {
			logErr = err
			return
		}
		native.Register(&logFn, sym)
	})
	return logFn, logErr
}

func logPriority(line string) int {
	switch {
	case strings.HasPrefix(line, "E"):
		return logError
	case strings.HasPrefix(line, "W"):
		return logWarn
	case strings.HasPrefix(line, "D"):
		return logDebug
	default:
		return logInfo
	}
}
