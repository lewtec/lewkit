package progress

import (
	"io"
	"log"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinePrinterSplitsOnNewline(t *testing.T) {
	var got []string
	w := &linePrinter{print: func(s string) { got = append(got, s) }}
	_, err := w.Write([]byte("hello\nwor"))
	require.NoError(t, err)
	_, err = w.Write([]byte("ld\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"hello", "world"}, got)
}

func TestLinePrinterHoldsPartial(t *testing.T) {
	var got []string
	w := &linePrinter{print: func(s string) { got = append(got, s) }}
	_, err := w.Write([]byte("no-nl"))
	require.NoError(t, err)
	assert.Empty(t, got)
	w.close()
	_, err = w.Write([]byte("after\n"))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestHijackSlogRestoreDropsLaterLogs(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var mu sync.Mutex
	var got []string
	restore := hijackSlog(func(s string) {
		mu.Lock()
		got = append(got, s)
		mu.Unlock()
	})
	t.Cleanup(restore)

	slog.Info("progress-hang-one")
	log.Println("progress-hang-std")
	restore()
	slog.Info("progress-hang-two")
	log.Println("progress-hang-std-two")

	mu.Lock()
	defer mu.Unlock()
	joined := strings.Join(got, "\n")
	assert.Contains(t, joined, "I progress-hang-one")
	assert.Contains(t, joined, "progress-hang-std")
	assert.NotContains(t, joined, "progress-hang-two")
	assert.NotContains(t, joined, "progress-hang-std-two")
}

func TestPrintAfterProgramExitReturns(t *testing.T) {
	var u teaUI
	u.p = tea.NewProgram(quitNow{}, tea.WithOutput(io.Discard), tea.WithInput(nil))
	_, err := u.p.Run()
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		u.print("late")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		require.FailNow(t, "print blocked after Program.Run returned")
	}
}

func TestUpdatePrintLineMsgReturnsCmd(t *testing.T) {
	m := newModel(nil)
	_, cmd := m.Update(printLineMsg("hello"))
	require.NotNil(t, cmd)
	require.NotNil(t, cmd())
}

type quitNow struct{}

func (quitNow) Init() tea.Cmd { return tea.Quit }

func (quitNow) Update(tea.Msg) (tea.Model, tea.Cmd) { return quitNow{}, nil }

func (quitNow) View() (v tea.View) { return }
