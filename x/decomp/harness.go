package decomp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
)

// harnessSchema is the decompilation object [Artifact.Match] parses.
// The harness constrains the model to this object.
const harnessSchema = `{"type":"object","additionalProperties":false,"required":["name","signature","source","blocks"],"properties":{"name":{"type":"string"},"signature":{"type":"string"},"source":{"type":"string"},"blocks":{"type":"array","items":{"type":"object","additionalProperties":false,"required":["address","kind"],"properties":{"address":{"type":"string"},"kind":{"type":"string"},"target":{"type":"string"},"fall":{"type":"string"}}}}}}`

// Harness completes one prompt by running the Grok harness.
// The harness process uses the signed-in subscription. An API key is its
// fallback when no session is signed in. The subscription file stays with
// the harness. Each call is one headless prompt in a private directory,
// with a read-only sandbox and dontAsk permissions.
type Harness struct {
	// Path is the grok executable. Empty means grok on PATH.
	Path string
	// Model is the harness model id. Empty keeps the signed-in default.
	Model string
	// Dir is the working directory. Empty uses a private temporary directory.
	Dir string
	// Run starts one harness process. Nil runs Path through x/driver/exec.
	Run func(ctx context.Context, call HarnessCall) ([]byte, error)
}

// HarnessCall is one grok invocation.
type HarnessCall struct {
	Path string
	Args []string
	Dir  string
}

type harnessFailure struct {
	message string
}

func (failure harnessFailure) Error() string {
	return failure.message
}

// Complete runs one headless harness prompt and returns the response text.
func (harness Harness) Complete(ctx context.Context, prompt Prompt) (string, error) {
	if ctx == nil {
		return "", errors.New("grok harness: nil context")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path := harness.Path
	if path == "" {
		path = "grok"
	}
	dir := harness.Dir
	if dir == "" {
		made, err := os.MkdirTemp("", "lewkit-decomp-")
		if err != nil {
			return "", fmt.Errorf("grok harness directory: %w", err)
		}
		dir = made
		defer os.RemoveAll(made)
	}
	promptPath := filepath.Join(dir, "prompt.txt")
	if err := os.WriteFile(promptPath, []byte(prompt.User), 0o600); err != nil {
		return "", fmt.Errorf("grok harness prompt: %w", err)
	}
	if harness.Dir != "" {
		defer os.Remove(promptPath)
	}
	call := HarnessCall{
		Path: path,
		Dir:  dir,
		Args: harness.arguments(dir, promptPath, prompt),
	}
	run := harness.Run
	if run == nil {
		run = runGrok
	}
	stdout, runErr := run(ctx, call)
	text, decErr := decodeHarness(stdout)
	if decErr == nil {
		return text, nil
	}
	var failed harnessFailure
	if errors.As(decErr, &failed) {
		return "", fmt.Errorf("grok harness: %w", decErr)
	}
	if runErr != nil {
		return "", fmt.Errorf("grok harness: %w", runErr)
	}
	return "", fmt.Errorf("grok harness: %w", decErr)
}

func (harness Harness) arguments(dir, promptPath string, prompt Prompt) []string {
	args := make([]string, 0, 24)
	if harness.Model != "" {
		args = append(args, "--model", harness.Model)
	}
	args = append(args,
		"--cwd", dir,
		"--sandbox", "read-only",
		"--permission-mode", "dontAsk",
		"--verbatim",
		"--output-format", "json",
		"--json-schema", harnessSchema,
		"--max-turns", "1",
		"--no-subagents",
		"--no-plan",
		"--disable-web-search",
		"--prompt-file", promptPath,
	)
	if prompt.System != "" {
		args = append(args, "--system-prompt-override", prompt.System)
	}
	return args
}

func runGrok(ctx context.Context, call HarnessCall) ([]byte, error) {
	if _, err := execdriver.Which(ctx, call.Path); err != nil {
		return nil, err
	}
	command := execdriver.MustCommand(ctx, call.Path, call.Args...)
	command.Dir = call.Dir
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := execdriver.Output(ctx, command)
	if err != nil {
		detail := strings.TrimSpace(stderr.String())
		if len(detail) > 512 {
			detail = detail[:512]
		}
		if detail != "" {
			return stdout, fmt.Errorf("%w: %s", err, detail)
		}
		return stdout, err
	}
	return stdout, nil
}

func decodeHarness(stdout []byte) (string, error) {
	payload := harnessPayload(stdout)
	if len(bytes.TrimSpace(payload)) == 0 {
		return "", errors.New("empty harness response")
	}
	var doc struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Text    string `json:"text"`
		Source  string `json:"source"`
		Blocks  []any  `json:"blocks"`
	}
	if err := json.Unmarshal(payload, &doc); err != nil {
		return "", fmt.Errorf("output: %w", err)
	}
	if doc.Type == "error" {
		if doc.Message == "" {
			return "", harnessFailure{message: "harness error"}
		}
		return "", harnessFailure{message: doc.Message}
	}
	if strings.TrimSpace(doc.Text) != "" {
		return doc.Text, nil
	}
	if doc.Source != "" || doc.Blocks != nil {
		return string(payload), nil
	}
	return "", errors.New("empty harness response")
}

func harnessPayload(stdout []byte) []byte {
	trimmed := bytes.TrimSpace(stdout)
	if len(trimmed) == 0 || trimmed[0] == '{' {
		return trimmed
	}
	for _, line := range bytes.Split(trimmed, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) > 0 && line[0] == '{' {
			return line
		}
	}
	return trimmed
}
