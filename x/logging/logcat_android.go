//go:build android && cgo

package logging

import (
	"io"
	"strings"

	"github.com/lewtec/lewkit/x/release"
)

/*
#cgo LDFLAGS: -llog
#include <android/log.h>
#include <stdlib.h>

static void log_write(int prio, char *tag, char *text) {
	__android_log_write(prio, tag, text);
	free(tag);
	free(text);
}
*/
import "C"

// Logcat writes process logs to Android logcat under the tag [release.Name].
// This build has cgo, which is what calls __android_log_write.
func Logcat() io.Writer { return logcatWriter{} }

type logcatWriter struct{}

func (logcatWriter) Write(p []byte) (int, error) {
	text := strings.TrimRight(string(p), "\n")
	if text == "" {
		return len(p), nil
	}
	for _, line := range strings.Split(text, "\n") {
		tag := C.CString(release.Name())
		c := C.CString(line)
		C.log_write(C.int(logPriority(line)), tag, c)
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
