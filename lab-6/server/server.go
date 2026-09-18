// Package server exposes a small REST API built only with the Go standard
// library on top of a SQLite-backed store. It shows the core net/http
// building blocks: an http.Server with sensible timeouts, http.Handler and
// http.HandlerFunc, middleware that wraps handlers, and JSON
// encoding/decoding with request size limits.
//
// gRPC is worth considering when you control both endpoints, want strongly
// typed contracts (protobuf), streaming, or low-latency calls between
// internal services. REST over net/http remains the better default when you
// need broad client compatibility, simple HTTP tooling (curl, browsers), or
// easy proxying and load balancing. This module sticks to REST with the
// standard library.
package server

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"lab-6/store"
)

// Server is the REST API. It owns the note store.
type Server struct {
	notes *store.NoteStore
}

// New returns a Server backed by the given SQLite store.
func New(notes *store.NoteStore) *Server {
	return &Server{notes: notes}
}

// Handler returns the full HTTP handler with every middleware applied. It is
// safe to call multiple times.
func (s *Server) Handler() http.Handler {
	return s.routes()
}

// NewServer returns an *http.Server with hardened timeouts on top of the API
// handler. ReadHeaderTimeout and IdleTimeout are the two most important:
// they keep slow or idle connections from pinning goroutines and sockets.
func NewServer(notes *store.NoteStore) *http.Server {
	return &http.Server{
		Addr:              ":8080",
		Handler:           New(notes).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /notes/{id}", s.getNote)
	// Only note creation requires authentication.
	mux.Handle("POST /notes", chain(http.HandlerFunc(s.createNote), WithAuth))
	// Logging and panic recovery apply to every request.
	return chain(mux, WithLogging, WithRecover)
}

func (s *Server) getNote(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, newErrorResponse("invalid note id"))
		return
	}

	note, err := s.notes.Get(r.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusNotFound, newErrorResponse("note not found"))
	case err != nil:
		log.Printf("getNote: %v", err)
		writeJSON(w, http.StatusInternalServerError, newErrorResponse("internal server error"))
	default:
		writeJSON(w, http.StatusOK, note)
	}
}

func (s *Server) createNote(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(authKey).(string)
	log.Printf("creating note as %s", user)

	var note store.Note
	if err := decodeJSON(w, r, &note); err != nil {
		switch {
		case errors.Is(err, errBodyTooLarge):
			writeJSON(w, http.StatusRequestEntityTooLarge, newErrorResponse("request body too large"))
		case errors.Is(err, errMultipleObjects):
			writeJSON(w, http.StatusBadRequest, newErrorResponse("request body must be a single JSON object"))
		default:
			writeJSON(w, http.StatusBadRequest, newErrorResponse("invalid JSON: "+err.Error()))
		}
		return
	}

	if note.Title == "" {
		writeJSON(w, http.StatusUnprocessableEntity, newErrorResponse("title is required"))
		return
	}

	created, err := s.notes.Add(r.Context(), note)
	if err != nil {
		log.Printf("createNote: %v", err)
		writeJSON(w, http.StatusInternalServerError, newErrorResponse("internal server error"))
		return
	}

	writeJSON(w, http.StatusCreated, created)
}