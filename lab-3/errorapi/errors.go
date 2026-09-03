package errorapi

import (
	"errors"
)

var (
	ErrNotFound  = errors.New("resource not found")
	ErrForbidden = errors.New("operation forbidden")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func Process(resource string) error {
	switch resource {
	case "":
		return &ValidationError{
			Field:   "resource",
			Message: "resource field missing",
		}
	case "missing":
		return ErrNotFound
	case "forbidden":
		return ErrForbidden
	default:
		return nil
	}
}
