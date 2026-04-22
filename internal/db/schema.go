package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS burials (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    buried_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    error_text  TEXT NOT NULL,
    error_hash  TEXT UNIQUE,
    fix_text    TEXT NOT NULL,
    context     TEXT,
    tags        TEXT,
    times_dug   INTEGER DEFAULT 0,
    last_dug    DATETIME
);

CREATE VIRTUAL TABLE IF NOT EXISTS burial_fts USING fts5(
    error_text,
    fix_text,
    context,
    tags,
    content='burials',
    content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS burials_ai AFTER INSERT ON burials BEGIN
    INSERT INTO burial_fts(rowid, error_text, fix_text, context, tags)
    VALUES (new.id, new.error_text, new.fix_text, new.context, new.tags);
END;

CREATE TRIGGER IF NOT EXISTS burials_au AFTER UPDATE ON burials BEGIN
    INSERT INTO burial_fts(burial_fts, rowid, error_text, fix_text, context, tags)
    VALUES ('delete', old.id, old.error_text, old.fix_text, old.context, old.tags);
    INSERT INTO burial_fts(rowid, error_text, fix_text, context, tags)
    VALUES (new.id, new.error_text, new.fix_text, new.context, new.tags);
END;

CREATE TRIGGER IF NOT EXISTS burials_ad AFTER DELETE ON burials BEGIN
    INSERT INTO burial_fts(burial_fts, rowid, error_text, fix_text, context, tags)
    VALUES ('delete', old.id, old.error_text, old.fix_text, old.context, old.tags);
END;

CREATE TABLE IF NOT EXISTS embeddings (
    burial_id   INTEGER PRIMARY KEY REFERENCES burials(id) ON DELETE CASCADE,
    vector_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS comments (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    burial_id    INTEGER NOT NULL REFERENCES burials(id) ON DELETE CASCADE,
    comment_text TEXT NOT NULL,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
