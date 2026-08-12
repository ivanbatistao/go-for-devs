package main

import "testing"

func TestGreeting(t *testing.T) {
	tests := []struct {
		name     string
		age      int
		expected string
	}{
		{
			name:     "Alice",
			age:      30,
			expected: "Hello, Alice. I see your age is 30\n",
		},
		{
			name:     "Bob",
			age:      25,
			expected: "Hello, Bob. I see your age is 25\n",
		},
		{
			name:     "",
			age:      20,
			expected: "Hello, . I see your age is 0\n",
		},
		{
			name:     "John",
			age:      27,
			expected: "Hello, John. I see your age is -1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := greeting(tt.name, tt.age)

			if got != tt.expected {
				t.Errorf("greeting(%q, %d) = %q, want %q",
					tt.name,
					tt.age,
					got,
					tt.expected,
				)
			}
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := greeting(tt.name, tt.age)

			if got != tt.expected {
				t.Errorf("greeting(%q, %d) = %q; want %q",
					tt.name, tt.age, got, tt.expected)
			}
		})
	}
}
