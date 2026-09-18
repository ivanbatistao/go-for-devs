package store

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func openTempStore(t *testing.T) (*NoteStore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "notes.db")
	st, err := Open(path)
	if err != nil {
		t.Fatalf("Open(%q): %v", path, err)
	}
	t.Cleanup(func() { st.Close() })
	return st, path
}

func TestOpenMigratesToNewestSchema(t *testing.T) {
	st, _ := openTempStore(t)

	version, err := schemaVersion(st.db)
	if err != nil {
		t.Fatalf("schemaVersion: %v", err)
	}
	if version != len(migrations) {
		t.Errorf("user_version = %d, want %d", version, len(migrations))
	}

	var name string
	err = st.db.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'notes'").
		Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		t.Fatal("notes table missing after migration")
	}
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.db")

	st, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	st.Close()

	if _, err := Open(path); err != nil {
		t.Fatalf("second Open: %v", err)
	}
}

func TestAddAndGetRoundtrip(t *testing.T) {
	st, _ := openTempStore(t)

	created, err := st.Add(context.Background(),
		Note{Title: "learning Go", Body: "SQLite persistence"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if created.ID != 1 {
		t.Errorf("created.ID = %d, want 1", created.ID)
	}

	got, err := st.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get(%d): %v", created.ID, err)
	}
	if got != created {
		t.Errorf("got = %+v, want %+v", got, created)
	}
}

func TestGetNotFound(t *testing.T) {
	st, _ := openTempStore(t)

	_, err := st.Get(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(999) error = %v, want %v", err, ErrNotFound)
	}
}

func TestPersistenceAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.db")

	st, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	created, err := st.Add(context.Background(), Note{Title: "survives"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	st.Close()

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()

	got, err := reopened.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got != created {
		t.Errorf("got = %+v, want %+v", got, created)
	}
}

func TestConcurrentAdds(t *testing.T) {
	st, _ := openTempStore(t)

	const n = 20
	var wg sync.WaitGroup
	ids := make(chan int, n)
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			created, err := st.Add(context.Background(), Note{Title: "concurrent"})
			if err != nil {
				errs <- err
				return
			}
			ids <- created.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)

	for err := range errs {
		t.Errorf("concurrent Add: %v", err)
	}

	seen := make(map[int]bool)
	for id := range ids {
		if seen[id] {
			t.Errorf("duplicate id %d assigned", id)
		}
		seen[id] = true
	}
	if len(seen) != n {
		t.Errorf("got %d unique ids, want %d", len(seen), n)
	}
}