package android

import (
	"strconv"

	"github.com/lewtec/lewkit/x/driver/notification"
)

// notifySig is lewkit.Host.notify(String id, String title, String message,
// String urgency, String progress) -> String. An empty progress string means
// the alert has no progress bar. An empty return is success.
const notifySig = "(Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;Ljava/lang/String;)Ljava/lang/String;"

func notifyArgs(n notification.Notification) []string {
	return []string{
		strconv.FormatUint(uint64(n.ID), 10),
		n.Title,
		n.Message,
		n.Urgency,
		progressArg(n),
	}
}

func progressArg(n notification.Notification) string {
	if !n.HasProgress {
		return ""
	}
	v := int(n.Progress * 100)
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	return strconv.Itoa(v)
}
