package sarif

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestComposeRoundTrip(t *testing.T) {
	t.Parallel()
	file := sample()
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	printed := buf.String()
	require.Equal(t, sampleJSON, printed)

	taken, err := Take(strings.NewReader(printed))
	require.NoError(t, err)
	require.Equal(t, file, taken)

	var again bytes.Buffer
	require.NoError(t, taken.Make(&again))
	require.Equal(t, printed, again.String())
}

func TestStoresChildren(t *testing.T) {
	t.Parallel()
	file := New().SetSchema(SchemaURI)
	run := file.Run()
	driver := run.Tool("lewkit").SetVersion("1.2.3")
	rule := run.Rule("S1001").SetName("unused").SetLevel(LevelNote)
	result := run.Result("S1001").SetRuleIndex(0).SetMessage("found")
	location := result.Location("main.go").SetStartLine(4)
	fix := result.Fix().SetDescription("repair")
	change := fix.Change("main.go")
	replacement := change.Replace().SetStartLine(4).SetInsert("ok")

	require.Equal(t, SchemaURI, file.Schema)
	require.Same(t, run, file.Runs[0])
	require.Same(t, driver, run.Driver)
	require.Equal(t, "lewkit", driver.Name)
	require.Equal(t, "1.2.3", driver.Version)
	require.Same(t, rule, driver.Rules[0])
	require.Equal(t, LevelNote, rule.Level)
	require.Same(t, result, run.Results[0])
	require.Equal(t, 0, *result.RuleIndex)
	require.Same(t, location, result.Locations[0])
	require.Equal(t, 4, location.Region.StartLine)
	require.Same(t, fix, result.Fixes[0])
	require.Same(t, change, fix.Changes[0])
	require.Same(t, replacement, change.Replacements[0])
	require.Equal(t, "ok", *replacement.Insert)
}

func TestRuleIndexZeroAndNegative(t *testing.T) {
	t.Parallel()
	zero, zeroPrinted := takeMake(t, `{
  "version": "2.1.0",
  "runs": [
    {
      "tool": {"driver": {"name": "t"}},
      "results": [
        {"ruleIndex": 0, "message": {"text": "m"}}
      ]
    }
  ]
}
`)
	require.NotNil(t, zero.Runs[0].Results[0].RuleIndex)
	require.Equal(t, 0, *zero.Runs[0].Results[0].RuleIndex)
	require.Contains(t, zeroPrinted, `"ruleIndex": 0`)

	negative, negativePrinted := takeMake(t, `{
  "version": "2.1.0",
  "runs": [
    {
      "tool": {"driver": {"name": "t"}},
      "results": [
        {"ruleIndex": -1, "message": {"text": "m"}}
      ]
    }
  ]
}
`)
	require.Equal(t, -1, *negative.Runs[0].Results[0].RuleIndex)
	require.Contains(t, negativePrinted, `"ruleIndex": -1`)

	absent, absentPrinted := takeMake(t, `{
  "version": "2.1.0",
  "runs": [
    {
      "tool": {"driver": {"name": "t"}},
      "results": [
        {"message": {"text": "m"}}
      ]
    }
  ]
}
`)
	require.Nil(t, absent.Runs[0].Results[0].RuleIndex)
	require.NotContains(t, absentPrinted, "ruleIndex")
}

func TestDeletionOmitsInsert(t *testing.T) {
	t.Parallel()
	file := New()
	run := file.Run()
	run.Tool("t").SetVersion("1")
	found := run.Result("R").SetMessage("m")
	found.Fix().Change("a.go").Replace().SetStartLine(2).SetStartColumn(1)
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	printed := buf.String()
	require.NotContains(t, printed, "insertedContent")
	require.Contains(t, printed, `"startLine": 2`)
	taken, err := Take(strings.NewReader(printed))
	require.NoError(t, err)
	require.Nil(t, taken.Runs[0].Results[0].Fixes[0].Changes[0].Replacements[0].Insert)
	require.Equal(t, file, taken)
}

func TestEmptyInsertSurvives(t *testing.T) {
	t.Parallel()
	file := New()
	file.Run().Tool("t")
	file.Runs[0].Result("R").SetMessage("m").Fix().Change("a.go").Replace().SetInsert("")
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	require.Contains(t, buf.String(), `"insertedContent"`)
	require.Contains(t, buf.String(), `"text": ""`)
	taken, err := Take(&buf)
	require.NoError(t, err)
	require.NotNil(t, taken.Runs[0].Results[0].Fixes[0].Changes[0].Replacements[0].Insert)
	require.Empty(t, *taken.Runs[0].Results[0].Fixes[0].Changes[0].Replacements[0].Insert)
}

func TestSnippetDoesNotEscapeHTML(t *testing.T) {
	t.Parallel()
	file := New()
	file.Run().Tool("t")
	file.Runs[0].Result("R").SetMessage("a < b & c").Location("a.go").SetSnippet("a<b>&c")
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	require.Contains(t, buf.String(), `"text": "a < b & c"`)
	require.Contains(t, buf.String(), `"text": "a<b>&c"`)
	taken, err := Take(&buf)
	require.NoError(t, err)
	require.Equal(t, "a<b>&c", taken.Runs[0].Results[0].Locations[0].Region.Snippet)
}

func TestMarkdownOnlyMessage(t *testing.T) {
	t.Parallel()
	taken, printed := takeMake(t, `{
  "version": "2.1.0",
  "runs": [
    {
      "tool": {"driver": {"name": "t"}},
      "results": [
        {"message": {"markdown": "say **hi**"}}
      ]
    }
  ]
}
`)
	require.Empty(t, taken.Runs[0].Results[0].Message.Text)
	require.Equal(t, "say **hi**", taken.Runs[0].Results[0].Message.Markdown)
	require.Contains(t, printed, `"markdown": "say **hi**"`)
	require.NotContains(t, printed, `"text"`)
}

func TestEmptyLog(t *testing.T) {
	t.Parallel()
	file, printed := takeMake(t, "{\"version\":\"2.1.0\",\"runs\":[]}\n")
	require.Equal(t, New(), file)
	require.Equal(t, "{\n  \"version\": \"2.1.0\",\n  \"runs\": []\n}\n", printed)
	var buf bytes.Buffer
	require.NoError(t, New().Make(&buf))
	require.Equal(t, printed, buf.String())
}

func TestTwoRuns(t *testing.T) {
	t.Parallel()
	file := New()
	file.Run().Tool("one").SetInformationURI("https://example.com/one")
	file.Run().Tool("two")
	second := file.Runs[1].Result("R").SetKind(KindPass).SetMessage("ok")
	second.Location("b.go").SetURIBase("SRC").SetEndLine(3)
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	taken, err := Take(&buf)
	require.NoError(t, err)
	require.Equal(t, file, taken)
	require.Equal(t, []string{"one", "two"}, []string{taken.Runs[0].Driver.Name, taken.Runs[1].Driver.Name})
	require.Equal(t, KindPass, taken.Runs[1].Results[0].Kind)
}

func TestLevelsAndKinds(t *testing.T) {
	t.Parallel()
	taken, _ := takeMake(t, `{
  "version": "2.1.0",
  "runs": [
    {
      "columnKind": "utf16CodeUnits",
      "tool": {
        "driver": {
          "name": "t",
          "rules": [
            {
              "id": "N",
              "defaultConfiguration": {"level": "none"}
            }
          ]
        }
      },
      "results": [
        {"ruleId": "N", "kind": "notApplicable", "level": "note", "message": {"text": "n"}},
        {"ruleId": "N", "kind": "review", "level": "error", "message": {"text": "e"}},
        {"ruleId": "N", "kind": "open", "message": {"text": "o"}}
      ]
    }
  ]
}
`)
	require.Equal(t, ColumnKindUTF16, taken.Runs[0].Column)
	require.Equal(t, LevelNone, taken.Runs[0].Driver.Rules[0].Level)
	require.Equal(t, KindNotApplicable, taken.Runs[0].Results[0].Kind)
	require.Equal(t, LevelNote, taken.Runs[0].Results[0].Level)
	require.Equal(t, KindReview, taken.Runs[0].Results[1].Kind)
	require.Equal(t, LevelError, taken.Runs[0].Results[1].Level)
	require.Equal(t, KindOpen, taken.Runs[0].Results[2].Kind)
	require.Equal(t, Level(0), taken.Runs[0].Results[2].Level)
}

func TestRejects(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		src  string
		err  error
	}{
		{name: "json", src: "{", err: ErrSyntax},
		{name: "trailing", src: "{\"version\":\"2.1.0\",\"runs\":[]}[]", err: ErrSyntax},
		{name: "version", src: "{\"version\":\"2.0.0\",\"runs\":[]}", err: ErrForm},
		{name: "missing version", src: "{\"runs\":[]}", err: ErrForm},
		{name: "missing runs", src: "{\"version\":\"2.1.0\"}", err: ErrSyntax},
		{name: "null runs", src: "{\"version\":\"2.1.0\",\"runs\":null}", err: ErrSyntax},
		{name: "driver", src: "{\"version\":\"2.1.0\",\"runs\":[{}]}", err: ErrSyntax},
		{name: "unknown", src: "{\"version\":\"2.1.0\",\"runs\":[],\"properties\":{}}", err: ErrForm},
		{name: "char offset", src: "{\"version\":\"2.1.0\",\"runs\":[{\"tool\":{\"driver\":{\"name\":\"t\"}},\"results\":[{\"message\":{\"text\":\"m\"},\"locations\":[{\"physicalLocation\":{\"region\":{\"charOffset\":1}}}]}]}]}", err: ErrForm},
		{name: "level", src: "{\"version\":\"2.1.0\",\"runs\":[{\"tool\":{\"driver\":{\"name\":\"t\"}},\"results\":[{\"level\":\"fatal\",\"message\":{\"text\":\"m\"}}]}]}", err: ErrLevel},
		{name: "kind", src: "{\"version\":\"2.1.0\",\"runs\":[{\"tool\":{\"driver\":{\"name\":\"t\"}},\"results\":[{\"kind\":\"nope\",\"message\":{\"text\":\"m\"}}]}]}", err: ErrKind},
		{name: "column", src: "{\"version\":\"2.1.0\",\"runs\":[{\"tool\":{\"driver\":{\"name\":\"t\"}},\"columnKind\":\"bytes\"}]}", err: ErrColumnKind},
		{name: "message", src: "{\"version\":\"2.1.0\",\"runs\":[{\"tool\":{\"driver\":{\"name\":\"t\"}},\"results\":[{}]}]}", err: ErrSyntax},
		{name: "rule id", src: "{\"version\":\"2.1.0\",\"runs\":[{\"tool\":{\"driver\":{\"name\":\"t\",\"rules\":[{}]}}]}", err: ErrSyntax},
		{name: "fix", src: "{\"version\":\"2.1.0\",\"runs\":[{\"tool\":{\"driver\":{\"name\":\"t\"}},\"results\":[{\"message\":{\"text\":\"m\"},\"fixes\":[{}]}]}]}", err: ErrSyntax},
		{name: "uri", src: "{\"version\":\"2.1.0\",\"runs\":[{\"tool\":{\"driver\":{\"name\":\"t\"}},\"results\":[{\"message\":{\"text\":\"m\"},\"fixes\":[{\"artifactChanges\":[{\"replacements\":[{\"deletedRegion\":{}}]}]}]}]}]}", err: ErrSyntax},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Take(strings.NewReader(tc.src))
			require.ErrorIs(t, err, tc.err)
		})
	}
}

func TestMakeRejects(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, ErrValue)
	require.ErrorIs(t, err, errs.ErrInvalid)
	var file *File
	require.ErrorIs(t, file.Make(io.Discard), ErrValue)
	require.ErrorIs(t, New().Make(nil), ErrValue)

	badVersion := New().SetVersion("2.0.0")
	require.ErrorIs(t, badVersion.Make(io.Discard), ErrForm)
	require.ErrorIs(t, badVersion.Make(io.Discard), errs.ErrUnsupported)

	nameless := New()
	nameless.Run()
	require.ErrorIs(t, nameless.Make(io.Discard), ErrValue)

	quiet := New()
	quiet.Run().Tool("t")
	quiet.Runs[0].Result("R")
	require.ErrorIs(t, quiet.Make(io.Discard), ErrValue)

	brokenLevel := New()
	brokenLevel.Run().Tool("t")
	brokenLevel.Runs[0].Result("R").SetMessage("m").SetLevel(Level(99))
	require.ErrorIs(t, brokenLevel.Make(io.Discard), ErrLevel)
	require.ErrorIs(t, brokenLevel.Make(io.Discard), errs.ErrUnknown)

	emptyFix := New()
	emptyFix.Run().Tool("t")
	emptyFix.Runs[0].Result("R").SetMessage("m").Fix()
	require.ErrorIs(t, emptyFix.Make(io.Discard), ErrValue)

	emptyRule := New()
	emptyRule.Run().Tool("t")
	emptyRule.Runs[0].Rule("")
	require.ErrorIs(t, emptyRule.Make(io.Discard), ErrValue)
}

func TestNilReceivers(t *testing.T) {
	t.Parallel()
	var file *File
	require.Nil(t, file.Run())
	require.Nil(t, file.SetSchema(SchemaURI))
	var run *Run
	require.Nil(t, run.Tool("t"))
	require.Nil(t, run.Rule("id"))
	require.Nil(t, run.Result("id"))
	var result *Result
	require.Nil(t, result.Location("a"))
	require.Nil(t, result.Fix())
	var fix *Fix
	require.Nil(t, fix.Change("a"))
	var change *Change
	require.Nil(t, change.Replace())
}

func TestReadError(t *testing.T) {
	t.Parallel()
	_, err := Take(failReader{})
	require.ErrorIs(t, err, errs.ErrUnavailable)
	require.ErrorIs(t, err, ErrValue)
}

func TestMakeWritesThrough(t *testing.T) {
	t.Parallel()
	var got countWriter
	require.NoError(t, New().Make(&got))
	require.Greater(t, got.writes, 0)
	require.Contains(t, got.String(), `"version": "2.1.0"`)
}

func sample() *File {
	file := New().SetSchema(SchemaURI)
	run := file.Run()
	run.Tool("lewkit").SetVersion("1.2.3").SetInformationURI("https://github.com/lewtec/lewkit")
	run.SetColumn(ColumnKindUnicode)
	rule := run.Rule("S1001").
		SetName("unused").
		SetShortDescription("unused import").
		SetFullDescription("the import is not used").
		SetHelp("delete the import").
		SetHelpURI("https://example.com/S1001").
		SetLevel(LevelNote)
	rule.FullDescription.Markdown = "the import is **not** used"
	run.Rule("S1002").SetName("empty").SetShortDescription("empty")
	result := run.Result("S1001").
		SetRuleIndex(0).
		SetKind(KindFail).
		SetLevel(LevelWarning).
		SetMessage("imported and not used").
		SetMarkdown("imported and **not** used")
	result.Location("main.go").
		SetURIBase("%SRCROOT%").
		SetStartLine(4).
		SetStartColumn(1).
		SetEndLine(4).
		SetEndColumn(8).
		SetSnippet(`"os"`).
		SetMessage("import")
	result.Fix().SetDescription("remove the import").
		Change("main.go").
		SetURIBase("%SRCROOT%").
		Replace().
		SetStartLine(4).
		SetStartColumn(1).
		SetEndLine(5).
		SetEndColumn(1).
		SetInsert("")
	run.Result("S1002").SetLevel(LevelNone).SetMessage("blank")
	return file
}

const sampleJSON = `{
  "version": "2.1.0",
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "lewkit",
          "version": "1.2.3",
          "informationUri": "https://github.com/lewtec/lewkit",
          "rules": [
            {
              "id": "S1001",
              "name": "unused",
              "shortDescription": {
                "text": "unused import"
              },
              "fullDescription": {
                "text": "the import is not used",
                "markdown": "the import is **not** used"
              },
              "help": {
                "text": "delete the import"
              },
              "helpUri": "https://example.com/S1001",
              "defaultConfiguration": {
                "level": "note"
              }
            },
            {
              "id": "S1002",
              "name": "empty",
              "shortDescription": {
                "text": "empty"
              }
            }
          ]
        }
      },
      "columnKind": "unicodeCodePoints",
      "results": [
        {
          "ruleId": "S1001",
          "ruleIndex": 0,
          "kind": "fail",
          "level": "warning",
          "message": {
            "text": "imported and not used",
            "markdown": "imported and **not** used"
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "main.go",
                  "uriBaseId": "%SRCROOT%"
                },
                "region": {
                  "startLine": 4,
                  "startColumn": 1,
                  "endLine": 4,
                  "endColumn": 8,
                  "snippet": {
                    "text": "\"os\""
                  }
                }
              },
              "message": {
                "text": "import"
              }
            }
          ],
          "fixes": [
            {
              "description": {
                "text": "remove the import"
              },
              "artifactChanges": [
                {
                  "artifactLocation": {
                    "uri": "main.go",
                    "uriBaseId": "%SRCROOT%"
                  },
                  "replacements": [
                    {
                      "deletedRegion": {
                        "startLine": 4,
                        "startColumn": 1,
                        "endLine": 5,
                        "endColumn": 1
                      },
                      "insertedContent": {
                        "text": ""
                      }
                    }
                  ]
                }
              ]
            }
          ]
        },
        {
          "ruleId": "S1002",
          "level": "none",
          "message": {
            "text": "blank"
          }
        }
      ]
    }
  ]
}
`

func takeMake(t *testing.T, src string) (*File, string) {
	t.Helper()
	file, err := Take(strings.NewReader(src))
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	printed := buf.String()
	again, err := Take(strings.NewReader(printed))
	require.NoError(t, err)
	require.Equal(t, file, again)
	return file, printed
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) {
	return 0, errs.ErrUnavailable
}

type countWriter struct {
	bytes.Buffer
	writes int
}

func (c *countWriter) Write(p []byte) (int, error) {
	c.writes++
	return c.Buffer.Write(p)
}
