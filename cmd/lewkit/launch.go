package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lewtec/lewkit/x/taskgroup"
)

func launchApp(ctx context.Context, goos, path, id string) error {
	switch goos {
	case "android":
		adb := adbBin()
		if err := runTool(ctx, adb, "install", "-r", path); err != nil {
			return err
		}
		return runTool(ctx, adb, "shell", "am", "start", "-n", id+"/.MainActivity")
	case "darwin":
		return runTool(ctx, "open", path)
	case "ios":
		if err := runTool(ctx, "xcrun", "simctl", "install", "booted", path); err != nil {
			return err
		}
		return runTool(ctx, "xcrun", "simctl", "launch", "booted", id)
	default:
		return fmt.Errorf("%s has no app to launch", goos)
	}
}

func adbBin() string {
	home := os.Getenv("ANDROID_HOME")
	if home == "" {
		return "adb"
	}
	bin := filepath.Join(home, "platform-tools", "adb")
	if _, err := os.Stat(bin); err != nil {
		return "adb"
	}
	return bin
}

func runTool(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	out := taskgroup.LineWriterFrom(ctx)
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}
