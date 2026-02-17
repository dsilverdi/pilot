package browser

import (
	"strings"
	"testing"
)

func TestCleanText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single line",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "multiple lines with empty",
			input:    "Line 1\n\nLine 2\n\n\nLine 3",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "whitespace only lines",
			input:    "Line 1\n   \n\t\nLine 2",
			expected: "Line 1\nLine 2",
		},
		{
			name:     "leading and trailing whitespace",
			input:    "  Hello  \n  World  ",
			expected: "Hello\nWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanText(tt.input)
			if result != tt.expected {
				t.Errorf("cleanText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSearchResultStruct(t *testing.T) {
	result := SearchResult{
		Title:       "Test Title",
		URL:         "https://example.com",
		Description: "Test description",
	}

	if result.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", result.Title, "Test Title")
	}
	if result.URL != "https://example.com" {
		t.Errorf("URL = %q, want %q", result.URL, "https://example.com")
	}
	if result.Description != "Test description" {
		t.Errorf("Description = %q, want %q", result.Description, "Test description")
	}
}

func TestPageContentStruct(t *testing.T) {
	content := PageContent{
		Title:   "Page Title",
		URL:     "https://example.com/page",
		Content: "Page content here",
	}

	if content.Title != "Page Title" {
		t.Errorf("Title = %q, want %q", content.Title, "Page Title")
	}
	if content.URL != "https://example.com/page" {
		t.Errorf("URL = %q, want %q", content.URL, "https://example.com/page")
	}
	if content.Content != "Page content here" {
		t.Errorf("Content = %q, want %q", content.Content, "Page content here")
	}
}

func TestCleanTextPreservesContent(t *testing.T) {
	// Test that cleanText doesn't modify actual content
	input := "This is a paragraph with multiple words.\nThis is another paragraph."
	result := cleanText(input)

	if !strings.Contains(result, "This is a paragraph") {
		t.Error("cleanText should preserve paragraph content")
	}
	if !strings.Contains(result, "another paragraph") {
		t.Error("cleanText should preserve second paragraph")
	}
}

func TestCleanTextHandlesSpecialCharacters(t *testing.T) {
	input := "Hello\r\nWorld\r\n"
	result := cleanText(input)

	// Should handle different line endings
	if !strings.Contains(result, "Hello") || !strings.Contains(result, "World") {
		t.Error("cleanText should handle CRLF line endings")
	}
}
