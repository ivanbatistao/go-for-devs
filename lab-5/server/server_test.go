package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testToken = "Bearer secret"

func newTestAPI(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(New().Handler())
}

func doReq(t *testing.T, ts *httptest.Server, method, path, body string, authed bool) (*http.Response, []byte) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, ts.URL+path, reader)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if authed {
		req.Header.Set("Authorization", testToken)
	}

	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	return resp, data
}

func TestCreateAndGetNote(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	resp, body := doReq(t, ts, http.MethodPost, "/notes",
		`{"title":"learning Go","body":"middleware and JSON"}`, true)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d, want 201 (body: %s)", resp.StatusCode, body)
	}

	var created Note
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("unmarshal created note: %v", err)
	}
	if created.ID != 1 {
		t.Errorf("created.ID = %d, want 1", created.ID)
	}
	if created.Title != "learning Go" {
		t.Errorf("created.Title = %q, want %q", created.Title, "learning Go")
	}

	resp, body = doReq(t, ts, http.MethodGet, "/notes/1", "", false)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d, want 200 (body: %s)", resp.StatusCode, body)
	}

	var got Note
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal fetched note: %v", err)
	}
	if got != created {
		t.Errorf("fetched note = %+v, want %+v", got, created)
	}
}

func TestGetNoteDoesNotRequireAuth(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	resp, _ := doReq(t, ts, http.MethodGet, "/notes/1", "", false)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestCreateNoteRequiresAuth(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	resp, body := doReq(t, ts, http.MethodPost, "/notes", `{"title":"x"}`, false)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body: %s)", resp.StatusCode, body)
	}
}

func TestCreateNoteInvalidJSON(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	for _, body := range []string{"", "{", `{"title":`} {
		resp, data := doReq(t, ts, http.MethodPost, "/notes", body, true)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400 (response: %s)",
				body, resp.StatusCode, data)
		}
	}
}

func TestCreateNoteUnknownField(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	resp, body := doReq(t, ts, http.MethodPost, "/notes",
		`{"title":"x","nope":1}`, true)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (body: %s)", resp.StatusCode, body)
	}
}

func TestCreateNoteTrailingContent(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	resp, body := doReq(t, ts, http.MethodPost, "/notes",
		`{"title":"x"}{"title":"y"}`, true)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (body: %s)", resp.StatusCode, body)
	}
}

func TestCreateNoteMissingTitle(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	resp, body := doReq(t, ts, http.MethodPost, "/notes", `{"body":"x"}`, true)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422 (body: %s)", resp.StatusCode, body)
	}
}

func TestCreateNoteBodyTooLarge(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	big := fmt.Sprintf(`{"title": "%s"}`, strings.Repeat("a", maxBodyBytes+1))
	resp, body := doReq(t, ts, http.MethodPost, "/notes", big, true)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413 (body len: %d)", resp.StatusCode, len(body))
	}
}

func TestGetNoteNotFound(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	resp, _ := doReq(t, ts, http.MethodGet, "/notes/999", "", false)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestGetNoteInvalidID(t *testing.T) {
	ts := newTestAPI(t)
	defer ts.Close()

	resp, _ := doReq(t, ts, http.MethodGet, "/notes/abc", "", false)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestNewServerTimeouts(t *testing.T) {
	srv := NewServer()

	tests := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{name: "ReadHeaderTimeout", got: srv.ReadHeaderTimeout, want: 5 * time.Second},
		{name: "ReadTimeout", got: srv.ReadTimeout, want: 10 * time.Second},
		{name: "WriteTimeout", got: srv.WriteTimeout, want: 10 * time.Second},
		{name: "IdleTimeout", got: srv.IdleTimeout, want: 60 * time.Second},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
