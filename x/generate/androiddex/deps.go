package androiddex

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	mavenAndroid = "https://dl.google.com/dl/android/maven2/"
	maxArtifact  = 64 << 20
)

// hostLibraries is the bytecode the packed host calls outside the framework.
// swiperefreshlayout 1.1.0 reaches core, interpolator, collection, lifecycle,
// versionedparcelable, annotation, and arch core-common.
var hostLibraries = []mavenArtifact{
	{"androidx.swiperefreshlayout", "swiperefreshlayout", "1.1.0", "aar"},
	{"androidx.core", "core", "1.1.0", "aar"},
	{"androidx.interpolator", "interpolator", "1.0.0", "aar"},
	{"androidx.collection", "collection", "1.0.0", "jar"},
	{"androidx.lifecycle", "lifecycle-runtime", "2.0.0", "aar"},
	{"androidx.lifecycle", "lifecycle-common", "2.0.0", "jar"},
	{"androidx.versionedparcelable", "versionedparcelable", "1.1.0", "aar"},
	{"androidx.annotation", "annotation", "1.1.0", "jar"},
	{"androidx.arch.core", "core-common", "2.0.0", "jar"},
}

type mavenArtifact struct {
	group   string
	name    string
	version string
	ext     string
}

func (art mavenArtifact) fileName() string {
	return art.name + "-" + art.version + "." + art.ext
}

func (art mavenArtifact) url() string {
	return mavenAndroid + strings.ReplaceAll(art.group, ".", "/") + "/" + art.name + "/" + art.version + "/" + art.fileName()
}

type libraries struct {
	classpath []string
	sources   map[string]string
}

func cacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "lewkit", "androiddex"), nil
}

func fetchLibraries(ctx context.Context, cache, work string) (libraries, error) {
	if err := ctx.Err(); err != nil {
		return libraries{}, err
	}
	out := libraries{sources: map[string]string{}}
	for _, art := range hostLibraries {
		if err := ctx.Err(); err != nil {
			return libraries{}, err
		}
		path, err := ensureArtifact(ctx, cache, art)
		if err != nil {
			return libraries{}, err
		}
		classes, err := art.classesJar(ctx, path, work)
		if err != nil {
			return libraries{}, err
		}
		out.classpath = append(out.classpath, classes)
		source, rel, err := libraryResource(ctx, path)
		if err != nil {
			return libraries{}, err
		}
		if rel != "" {
			out.sources[rel] = source
		}
	}
	return out, nil
}

func (art mavenArtifact) classesJar(ctx context.Context, path, work string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if art.ext == "jar" {
		return path, nil
	}
	raw, err := zipEntry(path, "classes.jar")
	if err != nil {
		return "", err
	}
	dest := filepath.Join(work, art.name+"-"+art.version+"-classes.jar")
	if err := os.WriteFile(dest, raw, 0o644); err != nil {
		return "", err
	}
	return dest, nil
}

func libraryResource(ctx context.Context, path string) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	text, err := zipEntry(path, "R.txt")
	if err != nil {
		if isMissingEntry(err) {
			return "", "", nil
		}
		return "", "", err
	}
	manifest, err := zipEntry(path, "AndroidManifest.xml")
	if err != nil {
		return "", "", err
	}
	pkg, err := manifestPackage(manifest)
	if err != nil {
		return "", "", err
	}
	source, err := resourceJavaFromText(pkg, string(text))
	if err != nil {
		return "", "", err
	}
	rel := strings.ReplaceAll(pkg, ".", "/") + "/R.java"
	return source, rel, nil
}

func manifestPackage(raw []byte) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(raw)))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("%w: manifest: %w", ErrDependency, err)
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "manifest" {
			continue
		}
		for _, attr := range start.Attr {
			if attr.Name.Local == "package" && attr.Value != "" {
				return attr.Value, nil
			}
		}
	}
	return "", fmt.Errorf("%w: manifest package", ErrDependency)
}

func zipEntry(path, name string) ([]byte, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDependency, path)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		if file.UncompressedSize64 > maxArtifact {
			return nil, fmt.Errorf("%w: %s", ErrDependency, name)
		}
		opened, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrDependency, name)
		}
		body, err := io.ReadAll(io.LimitReader(opened, maxArtifact+1))
		opened.Close()
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrDependency, name)
		}
		if len(body) > maxArtifact {
			return nil, fmt.Errorf("%w: %s", ErrDependency, name)
		}
		return body, nil
	}
	return nil, fmt.Errorf("%w: %s", errMissingEntry, name)
}

func isMissingEntry(err error) bool {
	return errors.Is(err, errMissingEntry)
}

func ensureArtifact(ctx context.Context, cache string, art mavenArtifact) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	dest := filepath.Join(cache, filepath.FromSlash(strings.ReplaceAll(art.group, ".", "/")), art.name, art.version, art.fileName())
	if zipFile(dest) {
		return dest, nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, art.url(), nil)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrDependency, art.fileName())
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrDependency, art.fileName(), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %s: %s", ErrDependency, art.fileName(), resp.Status)
	}
	tmp := dest + ".partial"
	file, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	n, copyErr := io.Copy(file, io.LimitReader(resp.Body, maxArtifact+1))
	closeErr := file.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("%w: %s: %w", ErrDependency, art.fileName(), copyErr)
	}
	if closeErr != nil {
		os.Remove(tmp)
		return "", closeErr
	}
	if n > maxArtifact {
		os.Remove(tmp)
		return "", fmt.Errorf("%w: %s", ErrDependency, art.fileName())
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if !zipFile(dest) {
		os.Remove(dest)
		return "", fmt.Errorf("%w: %s", ErrDependency, art.fileName())
	}
	return dest, nil
}

func zipFile(path string) bool {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return false
	}
	reader.Close()
	return true
}
