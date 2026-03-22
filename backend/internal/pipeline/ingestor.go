package pipeline

import (
	"strings"
	"unicode"
)

// Chunk represents a normalized piece of text from any source.
type Chunk struct {
	SourceType string
	Content    string
	Sentences  []string
	WordCount  int
}

// Ingestor normalizes raw source content into clean text chunks.
type Ingestor struct{}

func NewIngestor() *Ingestor { return &Ingestor{} }

// Ingest takes raw source content and returns a clean, normalized Chunk.
func (i *Ingestor) Ingest(sourceType, content string) *Chunk {
	cleaned := cleanText(content)
	sentences := splitSentences(cleaned)
	words := strings.Fields(cleaned)

	return &Chunk{
		SourceType: sourceType,
		Content:    cleaned,
		Sentences:  sentences,
		WordCount:  len(words),
	}
}

// cleanText removes excessive whitespace and normalizes unicode.
func cleanText(s string) string {
	// Normalize whitespace
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

// splitSentences splits text into individual sentences.
func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			// Check if next char is space or end of string
			if i+1 >= len(text) || text[i+1] == ' ' || text[i+1] == '\n' {
				s := strings.TrimSpace(current.String())
				if len(s) > 10 { // filter noise
					sentences = append(sentences, s)
				}
				current.Reset()
			}
		}
	}

	// Remaining text without ending punctuation
	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		sentences = append(sentences, remaining)
	}

	return sentences
}
