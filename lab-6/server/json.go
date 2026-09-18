package server

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

// maxBodyBytes caps the size of request bodies read with decodeJSON.
const maxBodyBytes = 1 << 20 // 1 MiB

var (
	errBodyTooLarge    = errors.New("request body too large")
	errMultipleObjects = errors.New("request body must be a single JSON object")
)

// errorResponse is the JSON shape returned for failed requests.
type errorResponse struct {
	Error string `json:"error"`
}

func newErrorResponse(msg string) errorResponse {
	return errorResponse{Error: msg}
}

// decodeJSON decodes a single JSON object from the request body into dst.
// The body is capped with http.MaxBytesReader so a client cannot exhaust
// server memory, unknown fields are rejected, and trailing content after the
// first object is treated as an error.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return errBodyTooLarge
		}
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errMultipleObjects
		}
		return err
	}
	return nil
}

// writeJSON encodes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}
