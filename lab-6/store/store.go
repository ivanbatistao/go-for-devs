// Package store persists notes in a SQLite database. It owns the schema
// migrations (see migrate.go) and exposes a small concurrency-safe API on
// top of database/sql.
package store

import (
	"context"
	"database/sql"
	"errors"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned by Get when the requested note does not exist.
var ErrNotFound = errors.New("note not found")

// Note is a single note. The struct tags control how fields are encoded and
// decoded with encoding/json, and match the columns in the notes table.
type Note struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// NoteStore is a SQLite-backed notes store.
type NoteStore struct {
	db *sql.DB
}

// Open opens (creating if necessary) the SQLite database at path and applies
// any pending schema migrations. The returned store is safe for concurrent
// use; WAL journaling and a busy timeout keep readers from blocking writers.
func Open(path string) (*NoteStore, error) {
	// modernc.org/sqlite reads PRAGMAs from the DSN so they apply to every
	// pooled connection, not just the first one opened.
	dsn := path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=foreign_keys(1)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return &NoteStore{db: db}, nil
}

// Close releases the underlying database connection.
func (s *NoteStore) Close() error {
	return s.db.Close()
}

// Add inserts note and returns it with its assigned ID.
func (s *NoteStore) Add(ctx context.Context, note Note) (Note, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO notes (title, body) VALUES (?, ?)", note.Title, note.Body)
	if err != nil {
		return Note{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Note{}, err
	}

	note.ID = int(id)
	return note, nil
}

// Get returns the note with the given id, or ErrNotFound.
func (s *NoteStore) Get(ctx context.Context, id int) (Note, error) {
	var note Note
	err := s.db.QueryRowContext(ctx,
		"SELECT id, title, body FROM notes WHERE id = ?", id).
		Scan(&note.ID, &note.Title, &note.Body)
	if errors.Is(err, sql.ErrNoRows) {
		return Note{}, ErrNotFound
	}
	if err != nil {
		return Note{}, err
	}
	return note, nil
}