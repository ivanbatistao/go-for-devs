package server

import "sync"

// Note is a single note in the API. The struct tags control how fields are
// encoded and decoded with encoding/json: exported fields map to JSON names,
// unexported fields are ignored.
type Note struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// noteStore is a minimal concurrency-safe in-memory store for notes. The
// mutex is needed because the HTTP server serves requests concurrently.
type noteStore struct {
	mu    sync.Mutex
	notes map[int]Note
	next  int
}

func newNoteStore() *noteStore {
	return &noteStore{notes: make(map[int]Note)}
}

func (s *noteStore) add(note Note) Note {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	note.ID = s.next
	s.notes[note.ID] = note
	return note
}

func (s *noteStore) get(id int) (Note, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notes[id]
	return note, ok
}
