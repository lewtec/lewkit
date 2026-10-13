-- name: UpsertTrack :exec
INSERT INTO tracks (path, title, artist, album, cover, duration_ms)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(path) DO UPDATE SET
    title = excluded.title,
    artist = excluded.artist,
    album = excluded.album,
    cover = excluded.cover,
    duration_ms = excluded.duration_ms;

-- name: ListAlbums :many
SELECT album, MIN(artist) AS artist, MAX(cover) AS cover, COUNT(*) AS tracks
FROM tracks
WHERE ? = '' OR title LIKE ? OR artist LIKE ? OR album LIKE ?
GROUP BY album
ORDER BY album;

-- name: ListTracks :many
SELECT id, path, title, artist, album, cover, duration_ms
FROM tracks
WHERE (? = '' OR album = ?)
  AND (? = '' OR title LIKE ? OR artist LIKE ?)
ORDER BY title;
