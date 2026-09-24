package music

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lewtec/lewkit/cmd/lewkit/experiments/music/musicdb"
	"github.com/lewtec/lewkit/x/db"
	"github.com/lewtec/lewkit/x/sound"
)

// Track is one ingested file.
type Track struct {
	ID       int64
	Path     string
	Title    string
	Artist   string
	Album    string
	Cover    string
	Duration time.Duration
}

// Album is a group of tracks that share a folder name.
type Album struct {
	Name   string
	Artist string
	Cover  string
	Tracks int
}

// Library is an in-memory catalog of audio files.
type Library struct {
	db     *db.Conn[musicdb.Queries]
	covers string
}

// OpenLibrary migrates an in-memory catalog and returns it.
func OpenLibrary(ctx context.Context) (*Library, error) {
	var arg musicdb.DBArg
	if err := arg.Parse(":memory:"); err != nil {
		return nil, err
	}
	if err := arg.Open(ctx); err != nil {
		return nil, err
	}
	return &Library{db: arg.Value()}, nil
}

// Close releases the catalog.
func (lib *Library) Close() error {
	if lib == nil {
		return nil
	}
	var err error
	if lib.covers != "" {
		err = os.RemoveAll(lib.covers)
	}
	if lib.db != nil {
		err = errors.Join(err, lib.db.Close())
	}
	return err
}

// Ingest walks root. A file adds itself. A folder adds audio under it.
// One bad file is skipped. The count is files added or replaced.
func (lib *Library) Ingest(ctx context.Context, root string) (int, error) {
	if lib == nil || lib.db == nil {
		return 0, errNotOpen
	}
	info, err := os.Stat(root)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		ok, err := lib.ingestFile(ctx, filepath.Dir(root), root)
		if !ok {
			return 0, err
		}
		return 1, nil
	}
	var added int
	var failed error
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && strings.HasPrefix(entry.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") {
			return nil
		}
		ok, addErr := lib.ingestFile(ctx, root, path)
		if addErr != nil {
			failed = addErr
			return nil
		}
		if ok {
			added++
		}
		return nil
	})
	if err != nil {
		return added, err
	}
	if added == 0 {
		return 0, failed
	}
	return added, nil
}

func (lib *Library) ingestFile(ctx context.Context, root, path string) (bool, error) {
	if !audioExt(path) {
		return false, nil
	}
	title, artist, album := trackMeta(root, path)
	duration, err := audioDuration(path)
	if err != nil {
		return false, err
	}
	cover := lib.trackCover(filepath.Dir(path), path)
	err = lib.db.Queries().UpsertTrack(ctx, musicdb.UpsertTrackParams{
		Path: path, Title: title, Artist: artist, Album: album, Cover: cover,
		DurationMs: duration.Milliseconds(),
	})
	return err == nil, err
}

// Albums lists folders that match query. An empty query lists every album.
func (lib *Library) Albums(ctx context.Context, query string) ([]Album, error) {
	like := likeQuery(query)
	rows, err := lib.db.Queries().ListAlbums(ctx, musicdb.ListAlbumsParams{
		Column1: query, Title: like, Artist: like, Album: like,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Album, len(rows))
	for i, row := range rows {
		out[i] = Album{Name: row.Album, Artist: sqlText(row.Artist), Cover: sqlText(row.Cover), Tracks: int(row.Tracks)}
	}
	return out, nil
}

// Tracks lists files in album. An empty album lists every track.
func (lib *Library) Tracks(ctx context.Context, album, query string) ([]Track, error) {
	like := likeQuery(query)
	rows, err := lib.db.Queries().ListTracks(ctx, musicdb.ListTracksParams{
		Column1: album, Album: album, Column3: query, Title: like, Artist: like,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Track, len(rows))
	for i, row := range rows {
		out[i] = Track{
			ID: row.ID, Path: row.Path, Title: row.Title, Artist: row.Artist,
			Album: row.Album, Cover: row.Cover,
			Duration: time.Duration(row.DurationMs) * time.Millisecond,
		}
	}
	return out, nil
}

func sqlText(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprint(v)
	}
}

var errNotOpen = errors.New("database not open")

func likeQuery(query string) string {
	if query == "" {
		return ""
	}
	return "%" + query + "%"
}

func audioExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".wav", ".mp3", ".ogg":
		return true
	default:
		return false
	}
}

func trackMeta(root, path string) (title, artist, album string) {
	title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	dir := filepath.Dir(path)
	album = filepath.Base(dir)
	if samePath(dir, root) {
		album = filepath.Base(root)
		artist = "Unknown"
		return
	}
	parent := filepath.Dir(dir)
	if samePath(parent, root) {
		artist = "Unknown"
		return
	}
	artist = filepath.Base(parent)
	return
}

func samePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func audioDuration(path string) (time.Duration, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	pipe, err := sound.Decode(path, file)
	if err != nil {
		return 0, err
	}
	if pipe.Frames() < 0 {
		return 0, nil
	}
	return pipe.Duration(), nil
}
