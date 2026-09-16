// Package server exposes a small REST API built only with the Go standard
// library. It shows the core net/http building blocks: an http.Server with
// sensible timeouts, http.Handler and http.HandlerFunc, middleware that wraps
// handlers, and JSON encoding/decoding with request size limits.
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
)

// Server is the REST API. It owns the note store.
type Server struct {
	notes *noteStore
}

// New returns a Server with an empty note store.
func New() *Server {
	return &Server{notes: newNoteStore()}
}

// Handler returns the full HTTP handler with every middleware applied. It is
// safe to call multiple times.
func (s *Server) Handler() http.Handler {
	return s.routes()
}

// NewServer returns an *http.Server with hardened timeouts on top of the API
// handler. ReadHeaderTimeout and IdleTimeout are the two most important:
// they keep slow or idle connections from pinning goroutines and sockets.
func NewServer() *http.Server {
	return &http.Server{
		Addr:              ":8080",
		Handler:           New().Handler(),
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

	note, ok := s.notes.get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, newErrorResponse("note not found"))
		return
	}

	writeJSON(w, http.StatusOK, note)
}

func (s *Server) createNote(w http.ResponseWriter, r *http.Request) {
	user, _ := r.Context().Value(authKey).(string)
	log.Printf("creating note as %s", user)

	var note Note
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

	writeJSON(w, http.StatusCreated, s.notes.add(note))
}
