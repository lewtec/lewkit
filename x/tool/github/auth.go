package github

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

const (
	userAgent  = "lewkit (+https://github.com/lewtec/lewkit)"
	apiVersion = "2022-11-28"

	tokenProbeEnv = "LEWKIT_GITHUB_TOKEN_PROBE"
	tokenProbeVal = "1"
	tokenStop     = "STOP"
)

var (
	tokenOnce sync.Once
	token     string
)

// Token returns a GitHub token from GITHUB_TOKEN, GH_TOKEN, or `gh auth token`.
// An empty string means anonymous requests.
func Token(ctx context.Context) string {
	if strings.TrimSpace(os.Getenv(tokenProbeEnv)) == tokenProbeVal {
		return ""
	}
	tokenOnce.Do(func() {
		token = resolveToken(ctx)
	})
	return token
}

func resolveToken(ctx context.Context) string {
	for _, key := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" && value != tokenStop {
			return value
		}
	}
	command := exec.CommandContext(ctx, "gh", "auth", "token")
	command.Env = append(os.Environ(), tokenProbeEnv+"="+tokenProbeVal)
	output, err := command.Output()
	if err != nil {
		return ""
	}
	value := strings.TrimSpace(string(output))
	if value == "" || value == tokenStop {
		return ""
	}
	return value
}

// NewAPIRequest builds a GitHub REST request with User-Agent, API version, and auth.
func NewAPIRequest(ctx context.Context, method, rawURL string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	ApplyAPIHeaders(ctx, request)
	return request, nil
}

// ApplyAPIHeaders sets User-Agent, X-GitHub-Api-Version, and Authorization when a token is available.
func ApplyAPIHeaders(ctx context.Context, request *http.Request) {
	if request == nil {
		return
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("X-GitHub-Api-Version", apiVersion)
	if value := Token(ctx); value != "" {
		request.Header.Set("Authorization", "Bearer "+value)
	}
}
