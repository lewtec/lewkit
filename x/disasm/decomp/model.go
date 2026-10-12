package decomp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Prompt is one completion request.
type Prompt struct {
	System string
	User   string
}

// Model writes a decompilation for one prompt.
// An implementation may call a remote model. The match itself stays in this process.
type Model interface {
	Complete(ctx context.Context, prompt Prompt) (string, error)
}

// Chat is an OpenAI-compatible chat completions client.
// Endpoint for xAI is https://api.x.ai/v1/chat/completions.
type Chat struct {
	Endpoint string
	Model    string
	APIKey   string
	Client   *http.Client
}

// Complete posts one chat completion and returns the assistant text.
func (chat Chat) Complete(ctx context.Context, prompt Prompt) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if chat.Endpoint == "" {
		return "", errors.New("empty chat endpoint")
	}
	if chat.Model == "" {
		return "", errors.New("empty chat model")
	}
	if chat.APIKey == "" {
		return "", errors.New("empty chat api key")
	}
	body, err := json.Marshal(chatRequest{
		Model: chat.Model,
		Messages: []chatMessage{
			{Role: "system", Content: prompt.System},
			{Role: "user", Content: prompt.User},
		},
	})
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, chat.Endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+chat.APIKey)
	client := chat.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("chat status %s: %s", response.Status, strings.TrimSpace(string(payload)))
	}
	var decoded chatResponse
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return "", fmt.Errorf("chat response: %w", err)
	}
	if decoded.Error.Message != "" {
		return "", errors.New(decoded.Error.Message)
	}
	if len(decoded.Choices) == 0 || decoded.Choices[0].Message.Content == "" {
		return "", errors.New("empty chat response")
	}
	return decoded.Choices[0].Message.Content, nil
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}
