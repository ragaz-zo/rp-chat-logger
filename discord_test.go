package main

import (
	"testing"
)

func TestSplitMessage(t *testing.T) {
	tests := []struct {
		base          string
		message       string
		expectedParts []string
		size          int
	}{
		{
			base:    "John: ",
			message: "This is a simple test.",
			expectedParts: []string{
				"John: This is a simple test.",
			},
			size: 100,
		},
		{
			base:    "John: ",
			message: "This is a longer test message that will probably get split into two parts.",
			expectedParts: []string{
				"John: This is a longer test message that will probably get ...",
				"John: ... split into two parts.",
			},
			size: 65,
		},
		{
			base:    "John: ",
			message: "A really long test sentence. A really long test sentence. A really long test sentence. A really long test sentence. A really long test sentence.",
			expectedParts: []string{
				"John: A really long test sentence. A really ...",
				"John: ... long test sentence. A really long ...",
				"John: ... test sentence. A really long test ...",
				"John: ... sentence. A really long test sentence.",
			},
			size: 50,
		},
		{
			base:          "John: ",
			message:       "",
			expectedParts: []string{},
		},
	}

	for _, tt := range tests {
		result := splitMessage(tt.base, tt.message, tt.size-len(tt.base))
		if len(result) != len(tt.expectedParts) {
			t.Errorf("For base '%s' and message '%s', expected %d parts but got %d parts.",
				tt.base, tt.message, len(tt.expectedParts), len(result))
			continue
		}
		for i, part := range result {
			if part != tt.expectedParts[i] {
				t.Errorf("Expected chunk '%s', but got '%s'", tt.expectedParts[i], part)
			}
		}
	}
}

func TestExtractChunk(t *testing.T) {
	tests := []struct {
		input             string
		maxLength         int
		expectedChunk     string
		expectedRemainder string
	}{
		{
			input:             "This is a test",
			maxLength:         5,
			expectedChunk:     "...",
			expectedRemainder: "... This is a test",
		},
		{
			input:             "This is a test",
			maxLength:         10,
			expectedChunk:     "This ...",
			expectedRemainder: "... is a test",
		},
		{
			input:             "This is a test",
			maxLength:         15,
			expectedChunk:     "This is a test",
			expectedRemainder: "",
		},
		{
			input:             "This is a test",
			maxLength:         30,
			expectedChunk:     "This is a test",
			expectedRemainder: "",
		},
		{
			input:             "Testing very long words",
			maxLength:         20,
			expectedChunk:     "Testing very ...",
			expectedRemainder: "... long words",
		},
		{
			input:             "Testing very long words. This is a test to see how far it can go.",
			maxLength:         50,
			expectedChunk:     "Testing very long words. This is a test to see ...",
			expectedRemainder: "... how far it can go.",
		},
		{
			input:             "Testing very long words. This is a test to see how far it can go. This is even longer, so that I can test different scenarios.",
			maxLength:         50,
			expectedChunk:     "Testing very long words. This is a test to see ...",
			expectedRemainder: "... how far it can go. This is even longer, so that I can test different scenarios.",
		},
		{
			input:             "Testing very long words. This is a test to see how far it can go. This is even longer, so that I can test different scenarios.",
			maxLength:         100,
			expectedChunk:     "Testing very long words. This is a test to see how far it can go. This is even longer, so that I ...",
			expectedRemainder: "... can test different scenarios.",
		},
	}

	for _, test := range tests {
		chunk, remainder := extractChunk(test.input, test.maxLength)
		if chunk != test.expectedChunk {
			t.Errorf("For input '%s' with maxLength %d, expected chunk '%s', but got '%s'", test.input, test.maxLength, test.expectedChunk, chunk)
		}
		if remainder != test.expectedRemainder {
			t.Errorf("For input '%s' with maxLength %d, expected remainder '%s', but got '%s'", test.input, test.maxLength, test.expectedRemainder, remainder)
		}
	}
}
