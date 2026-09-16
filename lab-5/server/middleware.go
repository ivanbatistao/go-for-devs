package server

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// middleware wraps an http.Handler and returns a new one. This is the core
// net/http composition pattern: every middleware is an http.HandlerFunc that
// performs its work and then calls the next handler in the chain.
type middleware func(http.Handler) http.Handler

// chain applies mws to h, outermost first, so the last middleware in the list
// runs closest to the original handler.
func chain(h http.Handler, mws ...middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

type ctxKey string

const authKey ctxKey = "auth-user"

// statusRecorder captures the status code written by a handler so it can be
// reported after the response has finished. Handlers that never call
// WriteHeader implicitly write 200, so that is the default.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// WithLogging logs each request's method, path, status code and duration.
// It wraps the ResponseWriter to observe the status written downstream.
func WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		log.Printf("%s %s -> %d (%s)", r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}

// WithRecover catches panics from downstream handlers and converts them into
// a 500 JSON response, so a single bad request cannot take the server down.
func WithRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic serving %s: %v\n%s", r.URL.Path, err, debug.Stack())
				writeJSON(w, http.StatusInternalServerError, newErrorResponse("internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// WithAuth requires a Bearer token. Once verified, it places the user id in
// the request context so downstream handlers can read it.
func WithAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			writeJSON(w, http.StatusUnauthorized, newErrorResponse("unauthorized"))
			return
		}

		ctx := context.WithValue(r.Context(), authKey, "demo-user")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
