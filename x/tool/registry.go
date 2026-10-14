package tool

import (
	"fmt"
	"sync"

	"github.com/lewtec/lewkit/x/errs"
)

// ErrBackendNotFound is returned when a spec names a backend that is not registered.
var ErrBackendNotFound = errs.Wrap(errs.ErrNotFound, "tool backend not found")

var (
	backendMu sync.RWMutex
	backends  = map[string]Backend{}
)

// Register installs a backend under id.
// id is the text before ':' in a spec. A second registration of id panics.
func Register(id string, backend Backend) {
	backendMu.Lock()
	defer backendMu.Unlock()
	if _, exists := backends[id]; exists {
		panic("tool backend " + id + " already registered")
	}
	backends[id] = backend
}

// Get returns the backend registered under id.
func Get(id string) (Backend, error) {
	backendMu.RLock()
	defer backendMu.RUnlock()
	backend, ok := backends[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrBackendNotFound, id)
	}
	return backend, nil
}
