package android

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/driver/filedialog"
)

var errDialog = errors.New("android file dialog")

type requestJSON struct {
	Title      string   `json:"title"`
	Directory  string   `json:"directory,omitempty"`
	Name       string   `json:"name,omitempty"`
	Extensions []string `json:"extensions,omitempty"`
	Multiple   bool     `json:"multiple,omitempty"`
	Folder     bool     `json:"folder,omitempty"`
	Save       bool     `json:"save,omitempty"`
}

type outcomeJSON struct {
	Paths    []string `json:"paths,omitempty"`
	Canceled bool     `json:"canceled,omitempty"`
	Error    string   `json:"error,omitempty"`
}

func encodeRequest(req filedialog.Request) (string, error) {
	raw, err := json.Marshal(requestJSON{
		Title:      req.TitleOrDefault(),
		Directory:  req.Directory,
		Name:       req.Name,
		Extensions: filedialog.Extensions(req.Filters),
		Multiple:   req.Multiple,
		Folder:     req.Folder,
		Save:       req.Save,
	})
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodeOutcome(raw string) ([]string, error) {
	var out outcomeJSON
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("%w: %v", errDialog, err)
	}
	if out.Canceled {
		return nil, filedialog.ErrCanceled
	}
	if out.Error != "" {
		return nil, fmt.Errorf("%w: %s", errDialog, out.Error)
	}
	if len(out.Paths) == 0 {
		return nil, fmt.Errorf("%w: no path", errDialog)
	}
	return out.Paths, nil
}
