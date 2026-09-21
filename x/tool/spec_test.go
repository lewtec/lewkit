package tool

import "testing"

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
			if (err != nil) != test.wantErr {
				t.Fatalf("Parse(%q) error = %v, wantErr %v", test.input, err, test.wantErr)
			}
			if !test.wantErr && got != test.want {
				t.Fatalf("Parse(%q) = %+v, want %+v", test.input, got, test.want)
			}
		})
	}
}

func TestSpecStringAndDirectory(t *testing.T) {
	spec := Spec{Backend: "github", Package: "denoland/deno", Version: "1.40.0"}
	if got, want := spec.String(), "github:denoland/deno@1.40.0"; got != want {
		t.Fatalf("String() = %s, want %s", got, want)
	}
	if got, want := spec.Directory(), "github-denoland-deno"; got != want {
		t.Fatalf("Directory() = %s, want %s", got, want)
	}
	if got, want := DirectoryName("http", "example.com:8080/path"), "http-example.com-8080-path"; got != want {
		t.Fatalf("DirectoryName() = %s, want %s", got, want)
	}
}

func TestCompareVersions(t *testing.T) {
	if CompareVersions("1.2.0", "1.10.0") >= 0 {
		t.Fatal("1.2.0 should be older than 1.10.0")
	}
	if CompareVersions("v1.2.3", "1.2.3") != 0 {
		t.Fatal("v prefix should not change order")
	}
	if CompareVersions("latest", "9.0.0") <= 0 {
		t.Fatal("latest should sort after a concrete version")
	}
}
