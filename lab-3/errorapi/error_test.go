package errorapi

import (
	"errors"
	"testing"
)

func TestProcessSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		expected error
	}{
		{
			name:     "resource not found",
			resource: "missing",
			expected: ErrNotFound,
		},
		{
			name:     "operation not permitted",
			resource: "forbidden",
			expected: ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Process(tt.resource)

			if !errors.Is(err, tt.expected) {
				t.Errorf(
					"Process(%q) error = %v, want %v",
					tt.resource,
					err,
					tt.expected,
				)
			}
		})
	}
}

func TestProcessCustomErrors(t *testing.T) {
	t.Helper()
	err := Process("")

	var validationErr *ValidationError

	if !errors.As(err, &validationErr) {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if validationErr.Field != "resource" {
		t.Errorf(
			"Field = %q, want %q",
			validationErr.Field,
			"resource",
		)
	}

	if validationErr.Message != "resource field missing" {
		t.Errorf(
			"Message = %q, want %q",
			validationErr.Message,
			"resource field missing",
		)
	}
}

func TestProcessSuccess(t *testing.T) {
	err := Process("existing")

	if err != nil {
		t.Errorf("Process() error = %v, want nil", err)
	}
}
