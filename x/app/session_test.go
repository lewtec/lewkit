package app

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"testing/synctest"

	"github.com/lewtec/lewkit/x/driver/webview"
	"github.com/stretchr/testify/require"
)

var errFirst = errors.New("first failed")

type gateWindow struct {
	release chan struct{}
	err     error
	width   int
	height  int
	profile string
}

func (g *gateWindow) httpHandler() http.Handler { return nil }

func (g *gateWindow) open(ctx context.Context, _ string, width, height int, profile string) error {
	g.width = width
	g.height = height
	g.profile = profile
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-g.release:
		return g.err
	}
}

func resetSession() {
	appMu.Lock()
	s := appWindows
	appWindows = nil
	appMu.Unlock()
	if s != nil {
		s.cancel()
	}
}

func TestSessionWaitsForLastWindow(t *testing.T) {
	resetSession()
	t.Cleanup(resetSession)
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()
		first := &gateWindow{release: make(chan struct{}), err: errFirst}
		runDone := make(chan error, 1)
		go func() {
			runDone <- runSession(ctx, first, openCall{title: "First", width: 800, height: 600, profile: "prof"})
		}()
		synctest.Wait()

		second := &gateWindow{release: make(chan struct{})}
		openDone := make(chan error, 1)
		go func() {
			openDone <- Open(ctx, second, "Second", 0, 0)
		}()
		synctest.Wait()
		require.Equal(t, "prof", second.profile)
		require.Equal(t, 800, second.width)
		require.Equal(t, 600, second.height)

		close(first.release)
		synctest.Wait()
		select {
		case err := <-runDone:
			require.Fail(t, "run returned while a window was open", "%v", err)
		default:
		}

		close(second.release)
		synctest.Wait()
		require.ErrorIs(t, <-runDone, errFirst)
		require.NoError(t, <-openDone)

		again := &gateWindow{release: make(chan struct{})}
		close(again.release)
		require.NoError(t, runSession(ctx, again, openCall{title: "Again", width: 4, height: 4, profile: "next"}))
		require.Equal(t, "next", again.profile)
	})
}

func TestSessionParentCancelClosesWindows(t *testing.T) {
	resetSession()
	t.Cleanup(resetSession)
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := contextWithCancel(t)
		first := &gateWindow{release: make(chan struct{})}
		runDone := make(chan error, 1)
		go func() {
			runDone <- runSession(ctx, first, openCall{title: "First", width: 8, height: 8, profile: "p"})
		}()
		synctest.Wait()
		second := &gateWindow{release: make(chan struct{})}
		openDone := make(chan error, 1)
		go func() {
			openDone <- Open(ctx, second, "Second", 8, 8)
		}()
		synctest.Wait()
		cancel()
		synctest.Wait()
		require.ErrorIs(t, <-runDone, context.Canceled)
		require.ErrorIs(t, <-openDone, context.Canceled)
	})
}

func TestOpenCallerCancelClosesThatWindow(t *testing.T) {
	resetSession()
	t.Cleanup(resetSession)
	synctest.Test(t, func(t *testing.T) {
		first := &gateWindow{release: make(chan struct{})}
		runDone := make(chan error, 1)
		go func() {
			runDone <- runSession(t.Context(), first, openCall{title: "First", width: 8, height: 8, profile: "p"})
		}()
		synctest.Wait()
		ctx, cancel := contextWithCancel(t)
		second := &gateWindow{release: make(chan struct{})}
		openDone := make(chan error, 1)
		go func() {
			openDone <- Open(ctx, second, "Second", 8, 8)
		}()
		synctest.Wait()
		cancel()
		synctest.Wait()
		require.ErrorIs(t, <-openDone, context.Canceled)
		select {
		case err := <-runDone:
			require.Fail(t, "caller cancel ended the app", "%v", err)
		default:
		}
		close(first.release)
		synctest.Wait()
		require.NoError(t, <-runDone)
	})
}

func TestSecondRunWhileSessionIsOpen(t *testing.T) {
	resetSession()
	t.Cleanup(resetSession)
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()
		first := &gateWindow{release: make(chan struct{})}
		runDone := make(chan error, 1)
		go func() {
			runDone <- runSession(ctx, first, openCall{title: "First", width: 8, height: 8, profile: "p"})
		}()
		synctest.Wait()
		err := runSession(ctx, &gateWindow{release: make(chan struct{})}, openCall{title: "Other", width: 8, height: 8, profile: "p"})
		require.ErrorIs(t, err, errAppRunning)
		close(first.release)
		synctest.Wait()
		require.NoError(t, <-runDone)
	})
}

func TestOpenWithoutSession(t *testing.T) {
	resetSession()
	t.Cleanup(resetSession)
	require.ErrorIs(t, Open(t.Context(), nil, "t", 1, 1), webview.ErrPage)
	gate := &gateWindow{release: make(chan struct{})}
	close(gate.release)
	require.NoError(t, Open(t.Context(), gate, "Solo", 10, 12))
	require.Equal(t, "", gate.profile)
	require.Equal(t, 10, gate.width)
	require.Equal(t, 12, gate.height)
}
