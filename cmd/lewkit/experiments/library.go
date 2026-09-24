package experiments

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lewtec/lewkit/x/db/sqlite"
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
	db     *sql.DB
	covers string
}

// OpenLibrary returns an empty in-memory catalog.
func OpenLibrary() (*Library, error) {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1)
	_, err = conn.Exec(`CREATE TABLE tracks (
		id INTEGER PRIMARY KEY,
		path TEXT NOT NULL UNIQUE,
		title TEXT NOT NULL,
		artist TEXT NOT NULL,
		album TEXT NOT NULL,
		cover TEXT NOT NULL,
		duration_ms INTEGER NOT NULL
	)`)
	if err != nil {
		return nil, errors.Join(err, conn.Close())
	}
	return &Library{db: conn}, nil
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
		return 0, sql.ErrConnDone
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
	_, err = lib.db.ExecContext(ctx, `INSERT INTO tracks(path, title, artist, album, cover, duration_ms)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			title = excluded.title,
			artist = excluded.artist,
			album = excluded.album,
			cover = excluded.cover,
			duration_ms = excluded.duration_ms`,
		path, title, artist, album, cover, duration.Milliseconds())
	return err == nil, err
}

// Albums lists folders that match query. An empty query lists every album.
func (lib *Library) Albums(ctx context.Context, query string) ([]Album, error) {
	like := likeQuery(query)
	rows, err := lib.db.QueryContext(ctx, `SELECT album, MIN(artist), MAX(cover), COUNT(*)
		FROM tracks
		WHERE ? = '' OR title LIKE ? OR artist LIKE ? OR album LIKE ?
		GROUP BY album
		ORDER BY album`, query, like, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Album
	for rows.Next() {
		var album Album
		if err := rows.Scan(&album.Name, &album.Artist, &album.Cover, &album.Tracks); err != nil {
			return nil, err
		}
		out = append(out, album)
	}
	return out, rows.Err()
}

// Tracks lists files in album. An empty album lists every track.
func (lib *Library) Tracks(ctx context.Context, album, query string) ([]Track, error) {
	like := likeQuery(query)
	rows, err := lib.db.QueryContext(ctx, `SELECT id, path, title, artist, album, cover, duration_ms
		FROM tracks
		WHERE (? = '' OR album = ?)
		  AND (? = '' OR title LIKE ? OR artist LIKE ?)
		ORDER BY title`, album, album, query, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Track
	for rows.Next() {
		var track Track
		var ms int64
		if err := rows.Scan(&track.ID, &track.Path, &track.Title, &track.Artist, &track.Album, &track.Cover, &ms); err != nil {
			return nil, err
		}
		track.Duration = time.Duration(ms) * time.Millisecond
		out = append(out, track)
	}
	return out, rows.Err()
}

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
