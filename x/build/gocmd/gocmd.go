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

// Command is one go build -v. Each output line is slog.Info.
type Command struct {
	Context context.Context
	Dir     string
	Env     []string
	Args    []string
}

// Run runs go build -v with Args after the verb.
func (c Command) Run() error {
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}
	argv := append([]string{"build", "-v"}, c.Args...)
	cmd := exec.CommandContext(ctx, "go", argv...)
	cmd.Dir = c.Dir
	cmd.Env = c.Env
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
