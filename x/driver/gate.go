package driver

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
)

// AppMode reports whether this process is a packaged app.
// Android and iOS are app mode, except Termux. LEWKIT_NO_UI and
// ELETROCROMO_NO_UI mark a desktop host that owns the window.
func AppMode() bool {
	if IsTermux() {
		return false
	}
	switch runtime.GOOS {
	case "android", "ios":
		return true
	}
	return envFlag("LEWKIT_NO_UI") || envFlag("ELETROCROMO_NO_UI")
}

func envFlag(key string) bool {
	value := strings.TrimSpace(os.Getenv(key))
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}

// MemoryGate keeps in-memory test drivers incompatible unless
// LEWKIT_ENABLE_MEMORY_DRIVER is set.
func MemoryGate() error {
	if os.Getenv("LEWKIT_ENABLE_MEMORY_DRIVER") == "" {
		return fmt.Errorf("%w: LEWKIT_ENABLE_MEMORY_DRIVER not set", ErrIncompatible)
	}
	return nil
}

// TerminalGate keeps drivers that read or write a terminal incompatible
// unless stdin and stdout are both terminals and the process is not an app.
// A pipe or discarded stdio is not a terminal.
func TerminalGate() error {
	if AppMode() {
		return fmt.Errorf("%w: app mode", ErrIncompatible)
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return fmt.Errorf("%w: stdio is not a terminal", ErrIncompatible)
	}
	return nil
}
