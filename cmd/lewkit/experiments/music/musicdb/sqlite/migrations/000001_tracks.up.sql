CREATE TABLE tracks (
    id INTEGER PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    artist TEXT NOT NULL,
    album TEXT NOT NULL,
    cover TEXT NOT NULL,
    duration_ms INTEGER NOT NULL
);
