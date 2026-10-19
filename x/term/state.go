package term

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type termState struct {
	Facts map[string]map[string]string `json:"facts,omitempty"`
}

// useState loads facts saved beside the socket and writes them back when
// they change. A missing file is an empty set. The directory is not created
// until the first save.
func (s *Session) useState(path string) {
	if path == "" {
		return
	}
	s.mu.Lock()
	s.statePath = path
	s.mu.Unlock()
	blob, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var saved termState
	if json.Unmarshal(blob, &saved) != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for path, bag := range saved.Facts {
		for key, value := range bag {
			s.setFact(path, key, value)
		}
	}
}

func (s *Session) saveState() {
	s.mu.Lock()
	path := s.statePath
	saved := termState{Facts: s.factBags()}
	s.mu.Unlock()
	if path == "" {
		return
	}
	blob, err := json.Marshal(saved)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, blob, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

func (s *Session) factBags() map[string]map[string]string {
	if len(s.facts) == 0 {
		return nil
	}
	out := make(map[string]map[string]string, len(s.facts))
	for path, bag := range s.facts {
		copied := make(map[string]string, len(bag))
		for key, value := range bag {
			copied[key] = value
		}
		out[path] = copied
	}
	return out
}
