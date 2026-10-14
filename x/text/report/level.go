package report

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum level.go

// Level is a diagnostic severity. The wire form matches SARIF result.level.
// An empty text is warning.
//
//lewkit:enum fold empty=warning
type Level string

const (
	LevelError   Level = "error"
	LevelWarning Level = "warning"
	LevelNote    Level = "note"
)
