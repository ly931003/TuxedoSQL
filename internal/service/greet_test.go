package service

import "testing"

func TestGreetService_Greet(t *testing.T) {
	svc := &GreetService{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "regular name",
			input:    "World",
			expected: "Hello World!",
		},
		{
			name:     "empty name",
			input:    "",
			expected: "Hello !",
		},
		{
			name:     "chinese name",
			input:    "世界",
			expected: "Hello 世界!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.Greet(tt.input)
			if result != tt.expected {
				t.Errorf("Greet(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
