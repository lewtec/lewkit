package mise

import (
	"context"
	"os/exec"
	"sync"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/stretchr/testify/require"
)

func TestNewToolRejectsEmptyRef(t *testing.T) {
	_, err := NewTool("  ")
	require.ErrorIs(t, err, ErrEmptyMiseRef)
}

type recordingExec struct {
	mu    sync.Mutex
	calls [][]string
}

func (r *recordingExec) Command(name string, args ...string) *exec.Cmd {
	r.mu.Lock()
	r.calls = append(r.calls, append([]string{name}, args...))
	r.mu.Unlock()
	return exec.Command("sh", "-c", "printf 22")
}

func (r *recordingExec) Which(context.Context, string) (string, error) {
	return "/usr/bin/mise", nil
}

func (r *recordingExec) args() [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([][]string, len(r.calls))
	copy(out, r.calls)
	return out
}

type recordingFactory struct{}

func (recordingFactory) ID() string                               { return "exec_mise_test" }
func (recordingFactory) Name() string                             { return "mise test" }
func (recordingFactory) Weight() int                              { return 100 }
func (recordingFactory) CheckCompatibility(context.Context) error { return nil }
func (recordingFactory) New(context.Context) (execdriver.Driver, error) {
	return &recorder, nil
}

var recorder recordingExec

func init() {
	driver.Register[execdriver.Driver](recordingFactory{})
}

func TestListVersionsUsesContextDriver(t *testing.T) {
	recorder.mu.Lock()
	recorder.calls = nil
	recorder.mu.Unlock()

	installed, err := NewTool("node")
	require.NoError(t, err)
	versions, err := installed.ListVersions(t.Context())
	require.NoError(t, err)
	require.Equal(t, []string{"22"}, versions)
	require.Equal(t, [][]string{{"mise", "latest", "node"}}, recorder.args())
}
