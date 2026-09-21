// Package github is the GitHub Releases backend.
//
// A spec github:owner/repo@version lists release tags and installs the
// asset whose name best matches this OS, architecture, and binary hint.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/report"
	"github.com/lewtec/lewkit/x/tool"
)

var (
	// ErrEmptyGitHubRef is returned when a github ref is blank.
	ErrEmptyGitHubRef = errors.New("github ref cannot be empty (expected owner/repo)")
	// ErrInvalidGitHubRef is returned when a github ref is not owner/repo.
	ErrInvalidGitHubRef = errors.New("invalid github ref (expected owner/repo)")
	// ErrAPIError is returned when the GitHub API responds with a non-OK status.
	ErrAPIError = errors.New("github api error")
	// ErrAPIRateLimit is returned when the API body says the rate limit was exceeded.
	ErrAPIRateLimit = errors.New("github api rate limit exceeded")
	// ErrNoArtifact is returned when no asset matches this OS and architecture.
	ErrNoArtifact = errors.New("no suitable artifact")
)

// Backend is the github releases backend.
type Backend struct{}

// Name returns a short description of the backend.
func (backend *Backend) Name() string { return "GitHub Releases" }

// Tool returns the tool for ref (owner/repo).
func (backend *Backend) Tool(ref string) (tool.Tool, error) {
	return NewTool(ref)
}

// GitHubTool installs one owner/repo from GitHub Releases.
type GitHubTool struct {
	repo       string
	binaryHint string
	backend    *Backend
}

// NewTool constructs a GitHubTool for ref ("owner/repo").
// binaryHint, when set, biases asset selection toward that filename token.
// The default hint is the repository name.
func NewTool(ref string, binaryHint ...string) (tool.Tool, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, ErrEmptyGitHubRef
	}
	parts := strings.Split(ref, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("%w: %q", ErrInvalidGitHubRef, ref)
	}
	hint := path.Base(ref)
	if len(binaryHint) > 0 && strings.TrimSpace(binaryHint[0]) != "" {
		hint = strings.TrimSpace(binaryHint[0])
	}
	return &GitHubTool{repo: ref, binaryHint: hint, backend: &Backend{}}, nil
}

// ListVersions returns non-draft release tags, newest first as the API returns them.
// Prereleases are included only when the stable list is empty.
func (installed *GitHubTool) ListVersions(ctx context.Context) ([]string, error) {
	return installed.backend.listVersions(ctx, installed.repo)
}

// Install selects an artifact for this OS and architecture and extracts it into destination.
func (installed *GitHubTool) Install(ctx context.Context, version, destination string) error {
	artifacts, err := installed.backend.artifacts(ctx, installed.repo, version)
	if err != nil {
		return err
	}
	artifact := tool.SelectArtifact(artifacts, runtime.GOOS, runtime.GOARCH, installed.binaryHint)
	if artifact == nil {
		return fmt.Errorf("no suitable artifact found for %s/%s for github:%s@%s: %w", runtime.GOOS, runtime.GOARCH, installed.repo, version, ErrNoArtifact)
	}
	return installed.backend.install(ctx, *artifact, destination)
}

// ListArtifacts returns platform-matched assets for version.
func (installed *GitHubTool) ListArtifacts(ctx context.Context, version string) ([]tool.Artifact, error) {
	return installed.backend.artifacts(ctx, installed.repo, version)
}

// InstallArtifact downloads one asset into destination.
func (installed *GitHubTool) InstallArtifact(ctx context.Context, artifact tool.Artifact, destination string) error {
	return installed.backend.install(ctx, artifact, destination)
}

// Pin names the repository as a github-releases dependency.
func (installed *GitHubTool) Pin() tool.Pin {
	return tool.Pin{Name: installed.repo, Datasource: "github-releases"}
}

func (backend *Backend) listVersions(ctx context.Context, repo string) ([]string, error) {
	requestURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", repo)
	slog.DebugContext(ctx, "fetching versions", "url", requestURL)
	response, err := doAPI(ctx, requestURL)
	if err != nil {
		return nil, err
	}
	defer func() { report.Report(response.Body.Close()) }()
	if response.StatusCode != http.StatusOK {
		return nil, apiErrorFromResponse(requestURL, response)
	}
	var releases []release
	if err := json.NewDecoder(response.Body).Decode(&releases); err != nil {
		return nil, err
	}
	versions := releaseTags(releases, false)
	if len(versions) == 0 {
		versions = releaseTags(releases, true)
	}
	slog.DebugContext(ctx, "found versions", "count", len(versions))
	return versions, nil
}

func (backend *Backend) artifacts(ctx context.Context, repo, version string) ([]tool.Artifact, error) {
	requestURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, version)
	if version == "latest" {
		requestURL = fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	}
	slog.DebugContext(ctx, "fetching release info", "url", requestURL)
	response, err := doAPI(ctx, requestURL)
	if err != nil {
		return nil, err
	}
	defer func() { report.Report(response.Body.Close()) }()
	if response.StatusCode != http.StatusOK {
		return nil, apiErrorFromResponse(requestURL, response)
	}
	var body release
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return nil, err
	}
	var artifacts []tool.Artifact
	for _, asset := range body.Assets {
		osName, arch, ok := parseAssetName(asset.Name)
		if !ok {
			continue
		}
		hash := ""
		if asset.Digest != "" {
			if algorithm, sum, ok := strings.Cut(asset.Digest, ":"); ok {
				hash = asset.Digest
				prefix := sum
				if len(prefix) > 16 {
					prefix = prefix[:16]
				}
				slog.DebugContext(ctx, "found checksum", "asset", asset.Name, "algorithm", algorithm, "hash", prefix)
			}
		}
		artifacts = append(artifacts, tool.Artifact{
			OS:                osName,
			Arch:              arch,
			URL:               asset.BrowserDownloadURL,
			Hash:              hash,
			Size:              asset.Size,
			GitHubAssetID:     asset.ID,
			GitHubAssetAPIURL: asset.APIURL,
		})
	}
	slog.DebugContext(ctx, "found assets", "total", len(body.Assets), "matched", len(artifacts))
	return artifacts, nil
}

func (backend *Backend) install(ctx context.Context, artifact tool.Artifact, destination string) error {
	browserURL := artifact.URL
	configure := func(request *http.Request) {
		ApplyAPIHeaders(ctx, request)
	}
	if artifact.GitHubAssetID != 0 && artifact.GitHubAssetAPIURL != "" {
		if Token(ctx) != "" {
			if apiURL, err := url.Parse(artifact.GitHubAssetAPIURL); err == nil {
				configure = func(request *http.Request) {
					request.URL = apiURL
					request.Host = apiURL.Host
					request.Header.Set("Accept", "application/octet-stream")
					ApplyAPIHeaders(ctx, request)
				}
			}
		}
	}
	return tool.InstallArtifact(ctx, tool.Artifact{
		OS:   artifact.OS,
		Arch: artifact.Arch,
		URL:  browserURL,
		Hash: artifact.Hash,
		Size: artifact.Size,
	}, destination, tool.DownloadOptions{ConfigureRequest: configure})
}

type release struct {
	TagName    string  `json:"tag_name"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []asset `json:"assets"`
}

type asset struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
	BrowserDownloadURL string `json:"browser_download_url"`
	APIURL             string `json:"url"`
}

func releaseTags(releases []release, includePrerelease bool) []string {
	var versions []string
	for _, item := range releases {
		if item.Draft {
			continue
		}
		if item.Prerelease && !includePrerelease {
			continue
		}
		tag := strings.TrimSpace(item.TagName)
		if tag == "" || strings.EqualFold(tag, "latest") {
			continue
		}
		versions = append(versions, tag)
	}
	return versions
}

func doAPI(ctx context.Context, rawURL string) (*http.Response, error) {
	request, err := NewAPIRequest(ctx, http.MethodGet, rawURL)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(request)
}

func apiErrorFromResponse(requestURL string, response *http.Response) error {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("github api error for %s: %s (reading body): %w: %w", requestURL, response.Status, err, ErrAPIError)
	}
	message := strings.TrimSpace(string(body))
	if response.StatusCode == http.StatusForbidden && strings.Contains(message, "rate limit") {
		return fmt.Errorf("%w for %s (consider setting GITHUB_TOKEN or logging in with 'gh auth login'): %s: %w", ErrAPIRateLimit, requestURL, response.Status, ErrAPIError)
	}
	if message != "" {
		return fmt.Errorf("github api error for %s: %s: %s: %w", requestURL, response.Status, message, ErrAPIError)
	}
	return fmt.Errorf("github api error for %s: %s: %w", requestURL, response.Status, ErrAPIError)
}

func parseAssetName(name string) (osName, arch string, ok bool) {
	name = strings.ToLower(name)
	if tool.ContainsAnyOf(name, "android") {
		osName = "android"
	} else if tool.ContainsAnyOf(name, "linux", "ubuntu") {
		osName = "linux"
	} else if tool.ContainsAnyOf(name, "darwin", "macos", "apple") {
		osName = "darwin"
	} else if tool.ContainsAnyOf(name, "windows") {
		osName = "windows"
	} else {
		return "", "", false
	}
	if tool.ContainsAnyOf(name, "amd64", "x86_64", "x64") {
		arch = "amd64"
	} else if tool.ContainsAnyOf(name, "arm64", "aarch64") {
		arch = "arm64"
	} else if tool.ContainsAnyOf(name, "386", "x86") {
		arch = "386"
	} else if tool.ContainsAnyOf(name, "riscv") {
		arch = "riscv"
	} else {
		return "", "", false
	}
	return osName, arch, true
}
