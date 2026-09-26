// Package gocmd runs go build -v and logs each compiler line.
package gocmd

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log/slog"
	"os/exec"
	"sync"
)

// Build runs go build -v with args after the verb. Each output line is slog.Info.
func Build(ctx context.Context, dir string, env []string, args ...string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	argv := append([]string{"build", "-v"}, args...)
	cmd := exec.CommandContext(ctx, "go", argv...)
	cmd.Dir = dir
	cmd.Env = env
	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = writer
	var scanErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanErr = slogLines(reader)
	}()
	err := cmd.Run()
	_ = writer.Close()
	wg.Wait()
	_ = reader.Close()
	if err != nil {
		return err
	}
	return scanErr
}

func slogLines(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		slog.Info(string(line))
	}
	return scanner.Err()
}
