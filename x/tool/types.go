package tool

import "context"

// Backend turns a package ref into a Tool.
// It is registered under the id that appears before ':' in a spec.
type Backend interface {
	Name() string
	Tool(ref string) (Tool, error)
}

// Tool is one package from a backend.
// ListVersions returns versions newest-first. Install writes that version into destination.
type Tool interface {
	ListVersions(ctx context.Context) ([]string, error)
	Install(ctx context.Context, version, destination string) error
}

// Artifact is one downloadable platform build.
type Artifact struct {
	OS   string
	Arch string
	URL  string
	Hash string
	Size int64

	// GitHubAssetID and GitHubAssetAPIURL select the authenticated asset
	// endpoint. Install rewrites the request. URL stays the browser download
	// URL so the temp file keeps its archive extension.
	GitHubAssetID     int64
	GitHubAssetAPIURL string
}

// ArtifactTool lists and installs one platform build.
type ArtifactTool interface {
	Tool
	ListArtifacts(ctx context.Context, version string) ([]Artifact, error)
	InstallArtifact(ctx context.Context, artifact Artifact, destination string) error
}

// BinaryTool installs one named binary for a version.
type BinaryTool interface {
	Tool
	EnsureBinary(ctx context.Context, version, binaryName, destination string) (string, error)
}

// Fixer rewrites an install directory after extraction.
// Fix is idempotent and cheap when the tree is already valid.
type Fixer interface {
	Fix(ctx context.Context, destination string) error
}

// Checker declares expectations for an install directory.
type Checker interface {
	InstallChecks() []Check
}

// Check is one expectation about an installed tree.
type Check interface {
	Name() string
	Check(ctx context.Context, destination string) error
}

// Pin is lock metadata a tool knows about itself.
// The caller stores the resolved version. Pin does not.
type Pin struct {
	Name           string
	Datasource     string
	Versioning     string
	ExtractVersion string
}

// Pinner supplies a Pin for the locker.
type Pinner interface {
	Pin() Pin
}
