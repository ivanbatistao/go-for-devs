package store

import "testing"

func TestMemoryStore_GetAndSet(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{{
		name:     "stores a value",
		key:      "name",
		value:    "Ivan",
		expected: "Ivan",
	},
		{
			name:     "stores an empty vale",
			key:      "empty",
			value:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewMemoryStore()

			store.Set(tt.key, tt.value)
			got, ok := store.Get(tt.key)

			if !ok {
				t.Fatalf("Get(%q) returned ok=false, want true", tt.key)
			}

			if got != tt.expected {
				t.Errorf("Get(%q) = (%q)) want %q",
					tt.key,
					got, tt.expected)
			}

		})
	}
}

func TestingMemoryStore_GetMissing(t *testing.T) {
	store := NewMemoryStore()

	got, ok := store.Get("missing")

	if ok {
		t.Error("Get(missing) ok=true, want false")
	}

	if got != "" {
		t.Errorf("Get(missing) = %q, want empty string", got)
	}
}
