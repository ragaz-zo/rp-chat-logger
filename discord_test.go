package main

import (
	"testing"
)

func TestSplitMessage(t *testing.T) {
	tests := []struct {
		name          string
		base          string
		message       string
		expectedParts []string
		size          int
	}{
		{
			name:    "short message fits in one chunk",
			base:    "John: ",
			message: "This is a simple test.",
			expectedParts: []string{
				"John: This is a simple test.",
			},
			size: 100,
		},
		{
			name:    "message split into two parts",
			base:    "John: ",
			message: "This is a longer test message that will probably get split into two parts.",
			expectedParts: []string{
				"John: This is a longer test message that will probably get ...",
				"John: ... split into two parts.",
			},
			size: 65,
		},
		{
			name:    "message split into four parts",
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
			name:          "empty message produces no chunks",
			base:          "John: ",
			message:       "",
			expectedParts: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitMessage(tt.base, tt.message, tt.size-len(tt.base))
			if len(result) != len(tt.expectedParts) {
				t.Fatalf("expected %d parts but got %d parts",
					len(tt.expectedParts), len(result))
			}
			for i, part := range result {
				if part != tt.expectedParts[i] {
					t.Errorf("chunk %d: expected %q, got %q", i, tt.expectedParts[i], part)
				}
			}
		})
	}
}

func TestExtractChunk(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		maxLength         int
		expectedChunk     string
		expectedRemainder string
	}{
		{
			name:              "no space within max length",
			input:             "This is a test",
			maxLength:         5,
			expectedChunk:     "...",
			expectedRemainder: "... This is a test",
		},
		{
			name:              "split at word boundary",
			input:             "This is a test",
			maxLength:         10,
			expectedChunk:     "This ...",
			expectedRemainder: "... is a test",
		},
		{
			name:              "message fits within max length",
			input:             "This is a test",
			maxLength:         15,
			expectedChunk:     "This is a test",
			expectedRemainder: "",
		},
		{
			name:              "max length exceeds message",
			input:             "This is a test",
			maxLength:         30,
			expectedChunk:     "This is a test",
			expectedRemainder: "",
		},
		{
			name:              "split long words at boundary",
			input:             "Testing very long words",
			maxLength:         20,
			expectedChunk:     "Testing very ...",
			expectedRemainder: "... long words",
		},
		{
			name:              "split near 50 chars",
			input:             "Testing very long words. This is a test to see how far it can go.",
			maxLength:         50,
			expectedChunk:     "Testing very long words. This is a test to see ...",
			expectedRemainder: "... how far it can go.",
		},
		{
			name:              "split extended sentence near 50 chars",
			input:             "Testing very long words. This is a test to see how far it can go. This is even longer, so that I can test different scenarios.",
			maxLength:         50,
			expectedChunk:     "Testing very long words. This is a test to see ...",
			expectedRemainder: "... how far it can go. This is even longer, so that I can test different scenarios.",
		},
		{
			name:              "split extended sentence near 100 chars",
			input:             "Testing very long words. This is a test to see how far it can go. This is even longer, so that I can test different scenarios.",
			maxLength:         100,
			expectedChunk:     "Testing very long words. This is a test to see how far it can go. This is even longer, so that I ...",
			expectedRemainder: "... can test different scenarios.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunk, remainder := extractChunk(tt.input, tt.maxLength)
			if chunk != tt.expectedChunk {
				t.Errorf("chunk: expected %q, got %q", tt.expectedChunk, chunk)
			}
			if remainder != tt.expectedRemainder {
				t.Errorf("remainder: expected %q, got %q", tt.expectedRemainder, remainder)
			}
		})
	}
}
