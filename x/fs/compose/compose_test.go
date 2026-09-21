package compose

import (
	iofs "io/fs"
	"testing"
	"testing/fstest"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/path"
)

func TestLinesMergeAndOpen(t *testing.T) {
	t.Parallel()
	tree := New()
	require.NoError(t, tree.Add(path.New(".bashrc"), File{
		Type: TypeLines,
		Values: map[string]Slot{
			"20-alias": Text("alias ll=ls"),
			"00-umask": Text("umask 022"),
		},
	}))
	require.NoError(t, tree.Add(path.New(".bashrc"), File{
		Type:   TypeLines,
		Values: map[string]Slot{"00-umask": Text("umask 022"), "10-path": Text("export PATH=$HOME/bin:$PATH")},
	}))

	filesystem, err := tree.FS(nil)
	require.NoError(t, err)
	got, err := iofs.ReadFile(filesystem, ".bashrc")
	require.NoError(t, err)
	assert.Equal(t, "umask 022\nexport PATH=$HOME/bin:$PATH\nalias ll=ls", string(got))

	_, err = filesystem.Open("00-umask")
	require.ErrorIs(t, err, iofs.ErrNotExist)

	entries, err := filesystem.ReadDir(".")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, ".bashrc", entries[0].Name())
}

func TestMergeOrderIndependent(t *testing.T) {
	t.Parallel()
	left := New()
	require.NoError(t, left.Add(path.New(".bashrc"), File{
		Type:   TypeLines,
		Values: map[string]Slot{"b": Text("B")},
	}))
	right := New()
	require.NoError(t, right.Add(path.New(".bashrc"), File{
		Type:   TypeLines,
		Values: map[string]Slot{"a": Text("A")},
	}))

	forward := New()
	require.NoError(t, forward.Merge(left))
	require.NoError(t, forward.Merge(right))
	backward := New()
	require.NoError(t, backward.Merge(right))
	require.NoError(t, backward.Merge(left))

	forwardFS, err := forward.FS(nil)
	require.NoError(t, err)
	backwardFS, err := backward.FS(nil)
	require.NoError(t, err)
	forwardBody, err := iofs.ReadFile(forwardFS, ".bashrc")
	require.NoError(t, err)
	backwardBody, err := iofs.ReadFile(backwardFS, ".bashrc")
	require.NoError(t, err)
	assert.Equal(t, "A\nB", string(forwardBody))
	assert.Equal(t, forwardBody, backwardBody)
}

func TestSlotConflict(t *testing.T) {
	t.Parallel()
	tree := New()
	require.NoError(t, tree.Add(path.New(".bashrc"), File{
		Type:   TypeLines,
		Values: map[string]Slot{"20-alias": Text("alias a")},
	}))
	err := tree.Add(path.New(".bashrc"), File{
		Type:   TypeLines,
		Values: map[string]Slot{"20-alias": Text("alias b")},
	})
	require.ErrorIs(t, err, ErrSlot)
}

func TestTypeConflict(t *testing.T) {
	t.Parallel()
	tree := New()
	require.NoError(t, tree.Add(path.New(".bashrc"), File{
		Type:   TypeLines,
		Values: map[string]Slot{"a": Text("A")},
	}))
	err := tree.Add(path.New(".bashrc"), File{
		Type:   TypeText,
		Values: map[string]Slot{"content": Text("nope")},
	})
	require.ErrorIs(t, err, ErrType)
}

func TestTextArity(t *testing.T) {
	t.Parallel()
	tree := New()
	err := tree.Add(path.New("a.txt"), File{
		Type: TypeText,
		Values: map[string]Slot{
			"x": Text("one"),
			"y": Text("two"),
		},
	})
	require.ErrorIs(t, err, ErrArity)
}

func TestModeConflictAndDefault(t *testing.T) {
	t.Parallel()
	tree := New()
	require.NoError(t, tree.Add(path.New("bin/tool"), File{
		Type:   TypeText,
		Mode:   0o755,
		Values: map[string]Slot{"content": Text("#!/bin/sh\n")},
	}))
	err := tree.Add(path.New("bin/tool"), File{
		Type:   TypeText,
		Mode:   0o644,
		Values: map[string]Slot{"content": Text("#!/bin/sh\n")},
	})
	require.ErrorIs(t, err, ErrMode)

	require.NoError(t, tree.Add(path.New("bin/tool"), File{
		Type:   TypeText,
		Values: map[string]Slot{"content": Text("#!/bin/sh\n")},
	}))
	filesystem, err := tree.FS(nil)
	require.NoError(t, err)
	info, err := filesystem.Stat("bin/tool")
	require.NoError(t, err)
	assert.Equal(t, iofs.FileMode(0o755), info.Mode())

	entries, err := filesystem.ReadDir("bin")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "tool", entries[0].Name())
	assert.False(t, entries[0].IsDir())
}

func TestPathClash(t *testing.T) {
	t.Parallel()
	tree := New()
	require.NoError(t, tree.Add(path.New("a/b"), File{
		Type:   TypeText,
		Values: map[string]Slot{"content": Text("b")},
	}))
	err := tree.Add(path.New("a"), File{
		Type:   TypeText,
		Values: map[string]Slot{"content": Text("a")},
	})
	require.ErrorIs(t, err, ErrPath)

	err = tree.Add(path.New("../x"), File{
		Type:   TypeText,
		Values: map[string]Slot{"content": Text("x")},
	})
	require.ErrorIs(t, err, ErrPath)
	err = tree.Add(path.New("~/.bashrc"), File{
		Type:   TypeText,
		Values: map[string]Slot{"content": Text("x")},
	})
	require.ErrorIs(t, err, ErrPath)
}

func TestJSONMerge(t *testing.T) {
	t.Parallel()
	tree := New()
	require.NoError(t, tree.Add(path.New("cfg.json"), File{
		Type: TypeJSON,
		Data: map[string]any{
			"port":  8080,
			"name":  "foo",
			"flags": map[string]any{"a": true},
			"extra": []any{"x"},
		},
	}))
	require.NoError(t, tree.Add(path.New("cfg.json"), File{
		Type: TypeJSON,
		Data: map[string]any{
			"name":  "foo",
			"flags": map[string]any{"b": false},
		},
	}))
	filesystem, err := tree.FS(nil)
	require.NoError(t, err)
	got, err := iofs.ReadFile(filesystem, "cfg.json")
	require.NoError(t, err)
	assert.Equal(t, "{\n  \"extra\": [\n    \"x\"\n  ],\n  \"flags\": {\n    \"a\": true,\n    \"b\": false\n  },\n  \"name\": \"foo\",\n  \"port\": 8080\n}\n", string(got))

	err = tree.Add(path.New("cfg.json"), File{
		Type: TypeJSON,
		Data: map[string]any{"port": 9090},
	})
	require.ErrorIs(t, err, ErrData)

	err = tree.Add(path.New("cfg.json"), File{
		Type: TypeJSON,
		Data: map[string]any{"extra": []any{"y"}},
	})
	require.ErrorIs(t, err, ErrData)
}

func TestEncodeStructured(t *testing.T) {
	t.Parallel()
	iniBody, err := Encode(File{
		Type: TypeINI,
		Data: map[string]any{
			"editor": "vim",
			"core":   map[string]any{"bare": true, "filemode": true},
		},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "editor = vim\n[core]\nbare = true\nfilemode = true\n", string(iniBody))

	_, err = Encode(File{
		Type: TypeINI,
		Data: map[string]any{"core": map[string]any{"deep": map[string]any{"x": "y"}}},
	}, nil)
	require.ErrorIs(t, err, ErrData)

	xmlBody, err := Encode(File{
		Type: TypeXML,
		Data: map[string]any{"cfg": map[string]any{
			"name":   "x",
			"nested": map[string]any{"a": true},
			"port":   8080,
		}},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<cfg>\n  <name>x</name>\n  <nested>\n    <a>true</a>\n  </nested>\n  <port>8080</port>\n</cfg>\n", string(xmlBody))

	listBody, err := Encode(File{
		Type: TypeXML,
		Data: map[string]any{"items": map[string]any{"item": []any{"a", "b"}}},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<items>\n  <item>a</item>\n  <item>b</item>\n</items>\n", string(listBody))

	escaped, err := Encode(File{Type: TypeXML, Data: map[string]any{"note": "a<b>&c"}}, nil)
	require.NoError(t, err)
	assert.Equal(t, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<note>a&lt;b&gt;&amp;c</note>\n", string(escaped))

	_, err = Encode(File{Type: TypeXML, Data: map[string]any{"a": "1", "b": "2"}}, nil)
	require.ErrorIs(t, err, ErrData)
	_, err = Encode(File{Type: TypeXML, Data: map[string]any{"1cfg": "x"}}, nil)
	require.ErrorIs(t, err, ErrData)

	yamlBody, err := Encode(File{
		Type: TypeYAML,
		Data: map[string]any{"name": "x", "nested": map[string]any{"a": true}},
	}, nil)
	require.NoError(t, err)
	assert.Contains(t, string(yamlBody), "name: x")
	assert.Contains(t, string(yamlBody), "a: true")

	tomlBody, err := Encode(File{
		Type: TypeTOML,
		Data: map[string]any{"name": "x", "port": 8080},
	}, nil)
	require.NoError(t, err)
	assert.Contains(t, string(tomlBody), "name")
	assert.Contains(t, string(tomlBody), "8080")
}

func TestRefOpen(t *testing.T) {
	t.Parallel()
	base := fstest.MapFS{
		"extra.sh": &fstest.MapFile{Data: []byte("echo hi")},
	}
	tree := New()
	require.NoError(t, tree.Add(path.New(".profile"), File{
		Type: TypeLines,
		Values: map[string]Slot{
			"head": Text("umask 022"),
			"tail": Ref("extra.sh"),
		},
	}))
	filesystem, err := tree.FS(base)
	require.NoError(t, err)
	got, err := iofs.ReadFile(filesystem, ".profile")
	require.NoError(t, err)
	assert.Equal(t, "umask 022\necho hi", string(got))

	bare, err := tree.FS(nil)
	require.NoError(t, err)
	_, err = bare.Open(".profile")
	require.ErrorIs(t, err, ErrRef)
}

func TestSquashDotDirectoryAndCue(t *testing.T) {
	t.Parallel()
	source := fstest.MapFS{
		".bashrc.d.tmpl/00-umask.sh":     &fstest.MapFile{Data: []byte("umask 022")},
		".bashrc.d.tmpl/20-alias.sh":     &fstest.MapFile{Data: []byte("alias ll=ls")},
		".bashrc.d.tmpl/30-skip.sh.tmpl": &fstest.MapFile{Data: []byte("no")},
		".gitconfig":                     &fstest.MapFile{Data: []byte("[user]\n"), Mode: 0o640},
		"skip.tmpl":                      &fstest.MapFile{Data: []byte("no")},
	}
	squashed, err := Squash(source)
	require.NoError(t, err)

	cueValue := cuecontext.New().CompileString(`
dest: ".bashrc": {
	type: "lines"
	values: "10-path": "export PATH=$HOME/bin:$PATH"
}
`, cue.Filename("dest.cue"))
	require.NoError(t, cueValue.Err())
	parsed, err := Parse(cueValue.LookupPath(cue.ParsePath("dest")))
	require.NoError(t, err)
	require.NoError(t, squashed.Merge(parsed))

	filesystem, err := squashed.FS(source)
	require.NoError(t, err)
	bashrc, err := iofs.ReadFile(filesystem, ".bashrc")
	require.NoError(t, err)
	assert.Equal(t, "umask 022\nexport PATH=$HOME/bin:$PATH\nalias ll=ls", string(bashrc))

	gitconfig, err := iofs.ReadFile(filesystem, ".gitconfig")
	require.NoError(t, err)
	assert.Equal(t, "[user]\n", string(gitconfig))
	info, err := filesystem.Stat(".gitconfig")
	require.NoError(t, err)
	assert.Equal(t, iofs.FileMode(0o640), info.Mode().Perm())

	_, err = filesystem.Open("skip")
	require.ErrorIs(t, err, iofs.ErrNotExist)
	_, err = filesystem.Open(".bashrc.d.tmpl/00-umask.sh")
	require.ErrorIs(t, err, iofs.ErrNotExist)
	_, err = filesystem.Open("30-skip.sh")
	require.ErrorIs(t, err, iofs.ErrNotExist)
}

func TestSquashTwoModules(t *testing.T) {
	t.Parallel()
	leftSource := fstest.MapFS{
		".bashrc.d.tmpl/20-alias.sh": &fstest.MapFile{Data: []byte("alias ll=ls")},
	}
	rightSource := fstest.MapFS{
		".bashrc.d.tmpl/00-umask.sh": &fstest.MapFile{Data: []byte("umask 022")},
	}
	left, err := Squash(leftSource)
	require.NoError(t, err)
	right, err := Squash(rightSource)
	require.NoError(t, err)
	require.NoError(t, left.Merge(right))
	filesystem, err := left.FS(nil)
	require.NoError(t, err)
	got, err := iofs.ReadFile(filesystem, ".bashrc")
	require.NoError(t, err)
	assert.Equal(t, "umask 022\nalias ll=ls", string(got))

	clashSource := fstest.MapFS{
		".bashrc.d.tmpl/20-alias.sh": &fstest.MapFile{Data: []byte("alias other")},
	}
	clash, err := Squash(clashSource)
	require.NoError(t, err)
	err = left.Merge(clash)
	require.ErrorIs(t, err, ErrSlot)

	same, err := Squash(leftSource)
	require.NoError(t, err)
	require.NoError(t, left.Merge(same))
}

func TestSquashRejectsNestedAndSiblingFile(t *testing.T) {
	t.Parallel()
	nested := fstest.MapFS{
		".bashrc.d.tmpl/nested/10.sh": &fstest.MapFile{Data: []byte("x")},
	}
	_, err := Squash(nested)
	require.ErrorIs(t, err, ErrPath)

	sibling := fstest.MapFS{
		".bashrc":              &fstest.MapFile{Data: []byte("plain")},
		".bashrc.d.tmpl/10.sh": &fstest.MapFile{Data: []byte("slot")},
	}
	_, err = Squash(sibling)
	require.ErrorIs(t, err, ErrType)

	fileNamedDirectory := fstest.MapFS{
		"notes.d.tmpl": &fstest.MapFile{Data: []byte("x")},
	}
	_, err = Squash(fileNamedDirectory)
	require.ErrorIs(t, err, ErrPath)
}

func TestParseCueRefAndMount(t *testing.T) {
	t.Parallel()
	source, err := Mount("app.dest")
	require.NoError(t, err)
	assert.Contains(t, source, "app: {")
	assert.Contains(t, source, "dest?: _compose.#Tree")

	_, err = Mount("")
	require.ErrorIs(t, err, ErrMount)
	_, err = Mount("foo..bar")
	require.ErrorIs(t, err, ErrMount)
	_, err = Mount("foo-bar.x")
	require.ErrorIs(t, err, ErrMount)

	cueContext := cuecontext.New()
	user := cueContext.CompileString(`
app: dest: {
	".profile": {
		type: "lines"
		values: {
			head: "umask 022"
			tail: {kind: "ref", ref: "extra.sh"}
		}
	}
	"cfg.json": {type: "json", mode: 420, values: {ok: true, port: 8080}}
}
`, cue.Filename("app.cue"))
	require.NoError(t, user.Err())
	constrained, err := Constrain(user, "app.dest")
	require.NoError(t, err)
	tree, err := Parse(constrained.LookupPath(cue.ParsePath("app.dest")))
	require.NoError(t, err)

	base := fstest.MapFS{"extra.sh": &fstest.MapFile{Data: []byte("echo hi")}}
	filesystem, err := tree.FS(base)
	require.NoError(t, err)
	profile, err := iofs.ReadFile(filesystem, ".profile")
	require.NoError(t, err)
	assert.Equal(t, "umask 022\necho hi", string(profile))
	config, err := iofs.ReadFile(filesystem, "cfg.json")
	require.NoError(t, err)
	assert.Equal(t, "{\n  \"ok\": true,\n  \"port\": 8080\n}\n", string(config))
	info, err := filesystem.Stat("cfg.json")
	require.NoError(t, err)
	assert.Equal(t, iofs.FileMode(420), info.Mode())

	rejected := cueContext.CompileString(`
app: dest: "a.txt": {
	type: "text"
	values: x: {kind: "env", env: "EDITOR"}
}
`, cue.Filename("bad.cue"))
	require.NoError(t, rejected.Err())
	_, err = Constrain(rejected, "app.dest")
	require.Error(t, err)

	empty, err := Parse(cueContext.CompileString("_").LookupPath(cue.ParsePath("missing")))
	require.NoError(t, err)
	filesystem, err = empty.FS(nil)
	require.NoError(t, err)
	entries, err := iofs.ReadDir(filesystem, ".")
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRefFileRequiresRefSlot(t *testing.T) {
	t.Parallel()
	tree := New()
	err := tree.Add(path.New("plain"), File{
		Type:   TypeRef,
		Values: map[string]Slot{"src": Text("nope")},
	})
	require.ErrorIs(t, err, ErrSlot)

	base := fstest.MapFS{"plain": &fstest.MapFile{Data: []byte("body")}}
	require.NoError(t, tree.Add(path.New("plain"), File{
		Type:   TypeRef,
		Values: map[string]Slot{"src": Ref("plain")},
	}))
	filesystem, err := tree.FS(base)
	require.NoError(t, err)
	got, err := iofs.ReadFile(filesystem, "plain")
	require.NoError(t, err)
	assert.Equal(t, "body", string(got))
}

func TestRegisterFormat(t *testing.T) {
	t.Parallel()
	err := Register(TypeJSON, encodeJSON)
	require.ErrorIs(t, err, ErrRegistered)
	err = Register(TypeLines, encodeJSON)
	require.ErrorIs(t, err, ErrType)
	err = Register("demo", nil)
	require.ErrorIs(t, err, ErrType)

	err = Register("demo", func(data map[string]any) ([]byte, error) {
		text, _ := data["k"].(string)
		return []byte(text), nil
	})
	if err != nil {
		require.ErrorIs(t, err, ErrRegistered)
	}
	body, err := Encode(File{Type: "demo", Data: map[string]any{"k": "hi"}}, nil)
	require.NoError(t, err)
	assert.Equal(t, "hi\n", string(body))
	assert.Contains(t, Formats(), Type("demo"))

	source, err := Mount("app.dest")
	require.NoError(t, err)
	assert.Contains(t, source, `"json"`)
	assert.Contains(t, source, `"demo"`)

	cueContext := cuecontext.New()
	rejected := cueContext.CompileString(`
app: dest: "a.json": {type: "nope", values: {a: 1}}
`, cue.Filename("nope.cue"))
	require.NoError(t, rejected.Err())
	_, err = Constrain(rejected, "app.dest")
	require.Error(t, err)

	accepted := cueContext.CompileString(`
app: dest: "a.demo": {type: "demo", values: {k: "hi"}}
`, cue.Filename("demo.cue"))
	require.NoError(t, accepted.Err())
	constrained, err := Constrain(accepted, "app.dest")
	require.NoError(t, err)
	tree, err := Parse(constrained.LookupPath(cue.ParsePath("app.dest")))
	require.NoError(t, err)
	filesystem, err := tree.FS(nil)
	require.NoError(t, err)
	got, err := iofs.ReadFile(filesystem, "a.demo")
	require.NoError(t, err)
	assert.Equal(t, "hi\n", string(got))
}

func TestNestedDirectoryFromStructuredPath(t *testing.T) {
	t.Parallel()
	source := fstest.MapFS{
		"a/b.d.tmpl/10.sh": &fstest.MapFile{Data: []byte("echo")},
	}
	tree, err := Squash(source)
	require.NoError(t, err)
	filesystem, err := tree.FS(nil)
	require.NoError(t, err)
	got, err := iofs.ReadFile(filesystem, "a/b")
	require.NoError(t, err)
	assert.Equal(t, "echo", string(got))
	info, err := filesystem.Stat("a")
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}
