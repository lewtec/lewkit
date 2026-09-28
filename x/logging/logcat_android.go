//go:build android

package logging

import (
	"io"
	"strings"
)

/*
#cgo LDFLAGS: -llog
#include <android/log.h>
#include <stdlib.h>

static void lewkit_log(int prio, char *text) {
	__android_log_write(prio, "lewkit", text);
	free(text);
}
*/
import "C"

// Logcat writes process logs to Android logcat under the tag lewkit.
func Logcat() io.Writer { return logcatWriter{} }

type logcatWriter struct{}

func (logcatWriter) Write(p []byte) (int, error) {
	text := strings.TrimRight(string(p), "\n")
	if text == "" {
		return len(p), nil
	}
	for _, line := range strings.Split(text, "\n") {
		c := C.CString(line)
		C.lewkit_log(C.int(logPriority(line)), c)
	}
	return len(p), nil
}

var _ io.Writer = logcatWriter{}

func logPriority(line string) int {
	switch {
	case strings.HasPrefix(line, "E"):
		return C.ANDROID_LOG_ERROR
	case strings.HasPrefix(line, "W"):
		return C.ANDROID_LOG_WARN
	case strings.HasPrefix(line, "D"):
		return C.ANDROID_LOG_DEBUG
	default:
		return C.ANDROID_LOG_INFO
	}
}
