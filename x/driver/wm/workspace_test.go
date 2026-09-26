package wm

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

type fakeDriver struct {
	outputs    []Output
	workspaces []Workspace
	moves      [][2]string
	switched   []string
}

func (f *fakeDriver) SwitchToWorkspace(_ context.Context, ws string, _ bool) error {
	f.switched = append(f.switched, ws)
	return nil
}
func (f *fakeDriver) ToggleScratchpad(context.Context) error { return nil }
func (f *fakeDriver) GetFocusedOutput(context.Context) (string, *Rect, error) {
	return "", nil, nil
}
func (f *fakeDriver) GetFocusedWindowRect(context.Context) (*Rect, error) { return nil, nil }
func (f *fakeDriver) GetOutputs(context.Context) ([]Output, error)        { return f.outputs, nil }
func (f *fakeDriver) GetWorkspaces(context.Context) ([]Workspace, error) {
	return f.workspaces, nil
}
func (f *fakeDriver) MoveWorkspaceToOutput(_ context.Context, workspace string, output string) error {
	f.moves = append(f.moves, [2]string{workspace, output})
	return nil
}

type fakeFactory struct{}

func (fakeFactory) ID() string                               { return "wm_workspace_fake" }
func (fakeFactory) Name() string                             { return "fake" }
func (fakeFactory) CheckCompatibility(context.Context) error { return nil }
func (fakeFactory) New(context.Context) (Driver, error) {
	fakeMu.Lock()
	defer fakeMu.Unlock()
	return activeFake, nil
}

var (
	fakeOnce   sync.Once
	fakeMu     sync.Mutex
	activeFake *fakeDriver
)

func useFake(t *testing.T, d *fakeDriver) {
	t.Helper()
	fakeOnce.Do(func() { driver.Register[Driver](fakeFactory{}) })
	fakeMu.Lock()
	activeFake = d
	fakeMu.Unlock()
	t.Cleanup(func() {
		fakeMu.Lock()
		activeFake = nil
		fakeMu.Unlock()
	})
}

func TestRotateWorkspaces(t *testing.T) {
	one := &fakeDriver{outputs: []Output{{Name: "A", CurrentWorkspace: "1"}}}
	useFake(t, one)
	require.NoError(t, RotateWorkspaces(t.Context()))
	require.Empty(t, one.moves)

	two := &fakeDriver{
		outputs: []Output{
			{Name: "A", CurrentWorkspace: "1"},
			{Name: "B", CurrentWorkspace: "2"},
		},
		workspaces: []Workspace{{Name: "1", Focused: true}, {Name: "2"}},
	}
	useFake(t, two)
	require.NoError(t, RotateWorkspaces(t.Context()))
	require.Equal(t, [][2]string{{"1", "B"}, {"2", "A"}}, two.moves)
	require.Equal(t, []string{"1"}, two.switched)
}

func TestNextWorkspace(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	next, err := AdvanceWorkspace()
	require.NoError(t, err)
	require.Equal(t, "11", next)
	got, err := os.ReadFile(filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "lewkit", "last-workspace"))
	require.NoError(t, err)
	require.Equal(t, "11", string(got))
}
