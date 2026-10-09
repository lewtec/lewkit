package github

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	lewrelease "github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/singleton"
)

const (
	apiVersion = "2022-11-28"

	tokenProbeEnv = "LEWKIT_GITHUB_TOKEN_PROBE"
	tokenProbeVal = "1"
	tokenStop     = "STOP"

	githubHost       = "github.com"
	ghKeyringService = "gh:" + githubHost
)

// tokenStrategy is one place a github.com token can already be stored.
// The order matches `gh auth token`: environment, hosts.yml, the OS keyring,
// then the gh binary.
type tokenStrategy struct {
	name string
	find func(ctx context.Context) (string, error)
}

var tokenStrategies = []tokenStrategy{
	{name: "environment", find: tokenFromEnv},
	{name: "gh hosts.yml", find: tokenFromHosts},
	{name: "keyring", find: tokenFromKeyring},
	{name: "gh auth token", find: tokenFromGH},
}

var githubToken = singleton.NewSingleton(func(ctx context.Context) (string, error) {
	return Lookup(ctx), nil
})

// Token returns a cached GitHub token. An empty string means anonymous requests.
func Token(ctx context.Context) string {
	if tokenProbe() {
		return ""
	}
	value, err := githubToken.GetContext(ctx)
	if err != nil {
		return ""
	}
	return value
}

// Lookup walks the token strategies once and does not cache the result.
func Lookup(ctx context.Context) string {
	if tokenProbe() {
		return ""
	}
	return resolveToken(ctx)
}

func tokenProbe() bool {
	return strings.TrimSpace(os.Getenv(tokenProbeEnv)) == tokenProbeVal
}

func resolveToken(ctx context.Context) string {
	for _, strategy := range tokenStrategies {
		token, err := strategy.find(ctx)
		if err != nil {
			slog.WarnContext(ctx, "github token strategy failed", "strategy", strategy.name, "error", err)
			continue
		}
		token = cleanToken(token)
		if token == "" {
			continue
		}
		slog.InfoContext(ctx, "using github token", "strategy", strategy.name)
		return token
	}
	return ""
}

func tokenFromEnv(context.Context) (string, error) {
	for _, key := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if token := cleanToken(os.Getenv(key)); token != "" {
			return token, nil
		}
	}
	return "", nil
}

func tokenFromGH(ctx context.Context) (string, error) {
	command := execdriver.MustCommand(ctx, "gh", "auth", "token")
	command.Env = append(os.Environ(), tokenProbeEnv+"="+tokenProbeVal)
	output, err := execdriver.Output(ctx, command)
	if err != nil {
		slog.DebugContext(ctx, "gh auth token failed", "error", err)
		return "", nil
	}
	return cleanToken(string(output)), nil
}

func cleanToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" || token == tokenStop {
		return ""
	}
	return token
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
	request.Header.Set("User-Agent", lewrelease.Name()+" (+https://github.com/lewtec/lewkit)")
	request.Header.Set("X-GitHub-Api-Version", apiVersion)
	if value := Token(ctx); value != "" {
		request.Header.Set("Authorization", "Bearer "+value)
	}
}
