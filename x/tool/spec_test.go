package tool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Spec
		wantErr bool
	}{
		{name: "full spec with version", input: "github:denoland/deno@1.40.0", want: Spec{Backend: "github", Package: "denoland/deno", Version: "1.40.0"}},
		{name: "spec with latest", input: "github:denoland/deno@latest", want: Spec{Backend: "github", Package: "denoland/deno", Version: "latest"}},
		{name: "spec without version defaults to latest", input: "github:denoland/deno", want: Spec{Backend: "github", Package: "denoland/deno", Version: "latest"}},
		{name: "spec with v prefix in version", input: "github:golang/go@v1.21.0", want: Spec{Backend: "github", Package: "golang/go", Version: "v1.21.0"}},
		{name: "omit provider uses registry default", input: "denoland/deno@1.40.0", want: Spec{Backend: "registry", Package: "denoland/deno", Version: "1.40.0"}},
		{name: "omit both provider and version", input: "deno", want: Spec{Backend: "registry", Package: "deno", Version: "latest"}},
		{name: "omit provider with version", input: "ripgrep@14.0.0", want: Spec{Backend: "registry", Package: "ripgrep", Version: "14.0.0"}},
		{name: "empty string", input: "", wantErr: true},
		{name: "only provider colon creates empty package", input: "registry:", want: Spec{Backend: "registry", Package: "", Version: "latest"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Parse(test.input)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestSpecStringAndDirectory(t *testing.T) {
	spec := Spec{Backend: "github", Package: "denoland/deno", Version: "1.40.0"}
	require.Equal(t, "github:denoland/deno@1.40.0", spec.String())
	require.Equal(t, "github-denoland-deno", spec.Directory())
	require.Equal(t, "http-example.com-8080-path", DirectoryName("http", "example.com:8080/path"))
}

func TestCompareVersions(t *testing.T) {
	require.Negative(t, CompareVersions("1.2.0", "1.10.0"))
	require.Zero(t, CompareVersions("v1.2.3", "1.2.3"))
	require.Positive(t, CompareVersions("latest", "9.0.0"))
}
