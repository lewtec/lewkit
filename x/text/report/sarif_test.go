package report

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteSARIFFix(t *testing.T) {
	t.Parallel()
	src := []byte("func Hello() {}\n")
	f := Finding{
		RuleID:  "demo/hello",
		Level:   LevelError,
		Message: "don't greet",
		File:    "main.go",
		Line:    1,
		Column:  6,
		EndLine: 1,
		EndCol:  11,
		Snippet: "Hello",
		Fixable: true,
		Source:  src,
		Edits: []Edit{{
			File:      "main.go",
			StartByte: 5,
			EndByte:   10,
			NewText:   "Bye",
		}},
	}
	var buf bytes.Buffer
	tool := Tool{Name: "demo", Version: "dev", InformationURI: "https://github.com/lewtec/lewkit"}
	require.NoError(t, WriteSARIF(&buf, "", tool, []Finding{f}, nil))
	var log struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name  string `json:"name"`
					Rules []struct {
						ID string `json:"id"`
					} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
			Results []struct {
				RuleID    string `json:"ruleId"`
				Level     string `json:"level"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine   int `json:"startLine"`
							StartColumn int `json:"startColumn"`
							Snippet     struct {
								Text string `json:"text"`
							} `json:"snippet"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
				Fixes []struct {
					ArtifactChanges []struct {
						Replacements []struct {
							InsertedContent struct {
								Text string `json:"text"`
							} `json:"insertedContent"`
							DeletedRegion struct {
								StartColumn int `json:"startColumn"`
								EndColumn   int `json:"endColumn"`
							} `json:"deletedRegion"`
						} `json:"replacements"`
					} `json:"artifactChanges"`
				} `json:"fixes"`
			} `json:"results"`
		} `json:"runs"`
	}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &log))
	require.Equal(t, "2.1.0", log.Version)
	require.Len(t, log.Runs, 1)
	run := log.Runs[0]
	require.Equal(t, "demo", run.Tool.Driver.Name)
	require.Equal(t, "demo/hello", run.Tool.Driver.Rules[0].ID)
	res := run.Results[0]
	require.Equal(t, "demo/hello", res.RuleID)
	require.Equal(t, "error", res.Level)
	reg := res.Locations[0].PhysicalLocation
	require.Equal(t, "main.go", reg.ArtifactLocation.URI)
	require.Equal(t, 6, reg.Region.StartColumn)
	require.Equal(t, "Hello", reg.Region.Snippet.Text)
	repl := res.Fixes[0].ArtifactChanges[0].Replacements[0]
	require.Equal(t, "Bye", repl.InsertedContent.Text)
	require.Equal(t, 6, repl.DeletedRegion.StartColumn)
	require.Equal(t, 11, repl.DeletedRegion.EndColumn)
}

func TestWriteSARIFReadsFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abc\n"), 0o644))
	f := Finding{
		RuleID:  "r",
		Level:   LevelNote,
		Message: "trim",
		File:    "a.txt",
		Line:    1,
		Column:  1,
		Fixable: true,
		Edits: []Edit{{
			File:      "a.txt",
			StartByte: 0,
			EndByte:   1,
			NewText:   "z",
		}},
	}
	var buf bytes.Buffer
	require.NoError(t, FormatSARIF.Render(&buf, dir, Tool{Name: "kit", Version: "1"}, []Finding{f}, nil))
	require.Contains(t, buf.String(), `"text": "z"`)
}
