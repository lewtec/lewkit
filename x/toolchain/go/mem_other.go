//go:build !linux

package gotool

import "os"

func openMem(name string) (*memFile, error) {
	f, err := os.CreateTemp("", "lewkit-"+name+"-")
	if err != nil {
		return nil, err
	}
	return &memFile{f: f, path: f.Name()}, nil
}
