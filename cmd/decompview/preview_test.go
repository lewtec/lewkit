package decompview

import (
	"context"
	"debug/elf"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/decomp"
	"github.com/lewtec/lewkit/x/disasm"
	"github.com/lewtec/lewkit/x/http/asset/sakuracss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreviewShowsTheReopenedFunction(t *testing.T) {
	path := writeProject(t)
	project, err := decomp.Open(t.Context(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = project.Close() })
	handler := Handler(project)

	index := get(t, handler, "/")
	assert.Equal(t, http.StatusOK, index.Code)
	assert.Contains(t, index.Body.String(), "branch.elf")
	assert.Contains(t, index.Body.String(), "data-cmdpalette")
	assert.Contains(t, index.Body.String(), "Ctrl+K")
	assert.Contains(t, index.Body.String(), `event.key === "k"`)
	assert.Contains(t, index.Body.String(), "data-keywords")
	assert.Contains(t, index.Body.String(), "start 0x1000")
	assert.Contains(t, index.Body.String(), "/artifact/branch.elf")
	assert.Contains(t, index.Body.String(), sakuracss.Path)

	artifact := get(t, handler, "/artifact/branch.elf")
	assert.Equal(t, http.StatusOK, artifact.Code)
	assert.Contains(t, artifact.Body.String(), "functions 1")
	assert.Contains(t, artifact.Body.String(), "/artifact/branch.elf/function/0x1000")

	function := get(t, handler, "/artifact/branch.elf/function/0x1000")
	assert.Equal(t, http.StatusOK, function.Code)
	body := function.Body.String()
	assert.Contains(t, body, "void start(void)")
	assert.Contains(t, body, "entry point")
	assert.Contains(t, body, "ret")
	assert.Contains(t, body, "void start(void) {")
	assert.Contains(t, body, "Matched source")

	missing := get(t, handler, "/artifact/missing.elf")
	assert.Equal(t, http.StatusNotFound, missing.Code)
	unknown := get(t, handler, "/artifact/branch.elf/function/0x1")
	assert.Equal(t, http.StatusNotFound, unknown.Code)
	bad := get(t, handler, "/artifact/branch.elf/function/nope")
	assert.Equal(t, http.StatusBadRequest, bad.Code)

	style := get(t, handler, sakuracss.Path)
	assert.Equal(t, http.StatusOK, style.Code)
	assert.Contains(t, style.Header().Get("Content-Type"), "text/css")
}

func TestPreviewStoresNameAndNote(t *testing.T) {
	path := writeProject(t)
	project, err := decomp.Open(t.Context(), path)
	require.NoError(t, err)
	handler := Handler(project)

	renamed := post(t, handler, "/artifact/branch.elf/function/0x1000/name", url.Values{"name": {"greet"}})
	assert.Equal(t, http.StatusSeeOther, renamed.Code)
	assert.Equal(t, "/artifact/branch.elf/function/0x1000", renamed.Header().Get("Location"))

	noted := post(t, handler, "/artifact/branch.elf/note", url.Values{
		"address": {"0x1000"},
		"text":    {"saved clue"},
		"next":    {"/artifact/branch.elf/function/0x1000"},
	})
	assert.Equal(t, http.StatusSeeOther, noted.Code)

	page := get(t, handler, "/artifact/branch.elf/function/0x1000")
	assert.Equal(t, http.StatusOK, page.Code)
	assert.Contains(t, page.Body.String(), "greet")
	assert.Contains(t, page.Body.String(), "saved clue")
	assert.Contains(t, page.Body.String(), "Save name")
	assert.Contains(t, page.Body.String(), "Save note")
	assert.Contains(t, page.Body.String(), `name="signature"`)

	table := get(t, handler, "/artifact/branch.elf")
	assert.Equal(t, http.StatusOK, table.Code)
	assert.Contains(t, table.Body.String(), "greet")
	assert.Contains(t, table.Body.String(), "0x1000 1")
	require.NoError(t, project.Close())

	reopened, err := decomp.Open(t.Context(), path)
	require.NoError(t, err)
	defer reopened.Close()
	artifact, ok := reopened.Artifact("branch.elf")
	require.True(t, ok)
	function, ok := artifact.Function(0x1000)
	require.True(t, ok)
	assert.Equal(t, "greet", function.Name)
	text, ok := artifact.Note(0x1000)
	require.True(t, ok)
	assert.Equal(t, "saved clue", text)
}

func TestPreviewFollowsCallsNotesAndSignature(t *testing.T) {
	path := filepath.Join(t.TempDir(), "program.decomp")
	project, err := decomp.Open(t.Context(), path)
	require.NoError(t, err)
	artifact, err := project.AddBytes(t.Context(), "setup.exe", disasm.ELF64(elf.EM_X86_64, []byte{0xe8, 0x01, 0x00, 0x00, 0x00, 0xc3, 0xc3}))
	require.NoError(t, err)
	require.NoError(t, artifact.SetNote(t.Context(), 0x1001, "kernel32!CloseHandle"))
	require.NoError(t, artifact.SetNote(t.Context(), 0x1006, "owns-close"))
	handler := Handler(project)

	function := get(t, handler, "/artifact/setup.exe/function/0x1000")
	assert.Equal(t, http.StatusOK, function.Code)
	assert.Contains(t, function.Body.String(), `/artifact/setup.exe/function/0x1006`)

	page := get(t, handler, "/artifact/setup.exe")
	assert.Contains(t, page.Body.String(), "0x1000 6")
	assert.Contains(t, page.Body.String(), "0x1006 1")

	hits := paletteHits(function.Body.String(), "closehandle")
	require.Len(t, hits, 1)
	assert.Equal(t, "/artifact/setup.exe/function/0x1000", hits[0])
	owners := paletteHits(function.Body.String(), "owns-close")
	require.Len(t, owners, 1)
	assert.Equal(t, "/artifact/setup.exe/function/0x1006", owners[0])

	renamed := post(t, handler, "/artifact/setup.exe/function/0x1006/name", url.Values{"name": {"StringAssign"}})
	assert.Equal(t, http.StatusSeeOther, renamed.Code)
	assert.Equal(t, "/artifact/setup.exe/function/0x1006", renamed.Header().Get("Location"))
	page = get(t, handler, "/artifact/setup.exe")
	assert.Contains(t, page.Body.String(), "StringAssign")

	signed := post(t, handler, "/artifact/setup.exe/function/0x1000/signature", url.Values{"signature": {"void entry(void)"}})
	assert.Equal(t, http.StatusSeeOther, signed.Code)
	function = get(t, handler, "/artifact/setup.exe/function/0x1000")
	assert.Contains(t, function.Body.String(), `name="signature"`)
	assert.Contains(t, function.Body.String(), "void entry(void)")
	assert.Contains(t, function.Body.String(), "signature void entry(void)")
	require.NoError(t, project.Close())
}

func TestListenServesTheIndex(t *testing.T) {
	path := writeProject(t)
	project, err := decomp.Open(t.Context(), path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = project.Close() })

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	errc := make(chan error, 1)
	go func() {
		errc <- Serve(ctx, listener, Handler(project))
	}()

	address := "http://" + listener.Addr().String() + "/"
	var response *http.Response
	deadline := time.Now().Add(2 * time.Second)
	for {
		response, err = http.Get(address)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	body, err := io.ReadAll(response.Body)
	require.NoError(t, response.Body.Close())
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Contains(t, string(body), "branch.elf")

	cancel()
	require.NoError(t, <-errc)
}

func paletteHits(body, word string) []string {
	pattern := regexp.MustCompile(`data-keywords="([^"]*)" role="option"><a href="([^"]+)"`)
	var hrefs []string
	for _, match := range pattern.FindAllStringSubmatch(body, -1) {
		if strings.Contains(match[1], word) {
			hrefs = append(hrefs, match[2])
		}
	}
	return hrefs
}

func post(t *testing.T, handler http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func get(t *testing.T, handler http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func writeProject(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "program.decomp")
	project, err := decomp.Open(t.Context(), path)
	require.NoError(t, err)
	artifact, err := project.AddBytes(t.Context(), "branch.elf", disasm.ELF64(elf.EM_X86_64, []byte{0xc3}))
	require.NoError(t, err)
	function, ok := artifact.Function(0x1000)
	require.True(t, ok)
	artifact.Attach(fixedModel{reply: candidate(function)})
	result, err := artifact.Match(t.Context(), 0x1000, decomp.MatchConfig{Rounds: 1})
	require.NoError(t, err)
	require.True(t, result.Matched)
	require.NoError(t, artifact.SetComment(t.Context(), 0x1000, "entry point"))
	require.NoError(t, project.Close())
	return path
}

type fixedModel struct {
	reply string
}

func (model fixedModel) Complete(context.Context, decomp.Prompt) (string, error) {
	return model.reply, nil
}

func candidate(function decomp.Function) string {
	var blocks strings.Builder
	for index, block := range function.Blocks {
		if index > 0 {
			blocks.WriteByte(',')
		}
		blocks.WriteString(`{"address":"0x`)
		blocks.WriteString(strconv.FormatUint(block.Address, 16))
		blocks.WriteString(`","kind":"`)
		blocks.WriteString(block.Kind)
		blocks.WriteByte('"')
		if block.Target != 0 {
			blocks.WriteString(`,"target":"0x`)
			blocks.WriteString(strconv.FormatUint(block.Target, 16))
			blocks.WriteByte('"')
		}
		if block.Fall != 0 {
			blocks.WriteString(`,"fall":"0x`)
			blocks.WriteString(strconv.FormatUint(block.Fall, 16))
			blocks.WriteByte('"')
		}
		blocks.WriteByte('}')
	}
	source := strings.ReplaceAll("void start(void) {\n  return;\n}\n", "\n", `\n`)
	return `{"name":"start","signature":"void start(void)","source":"` + source + `","blocks":[` + blocks.String() + `]}`
}
