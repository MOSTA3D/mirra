package pipeline

import (
	"strings"
	"unicode"

	"github.com/mirra-ai/mirra/backend/internal/pipeline/preprocessors"
)

// Chunk represents normalized content ready for extraction.
type Chunk struct {
	SourceType        string
	Content           string
	Sentences         []string
	WordCount         int
	BehavioralSignals []preprocessors.BehavioralSignal
	Platform          string
}

// Ingestor normalizes raw source content using the appropriate preprocessor strategy.
type Ingestor struct {
	registry *preprocessors.Registry
}

func NewIngestor() *Ingestor {
	return &Ingestor{registry: preprocessors.NewRegistry()}
}

// Ingest takes a source and returns a normalized Chunk.
// speakerName is used for chat formats to isolate the target person's messages.
func (i *Ingestor) Ingest(sourceType, content, speakerName string) *Chunk {
	// Pick the right strategy
	strategy := i.registry.Resolve(sourceType, content)
	result := strategy.Process(sourceType, content, speakerName)

	// Combine all utterances into content for linguistic analysis
	var texts []string
	for _, u := range result.Utterances {
		if u.Text != "" {
			texts = append(texts, u.Text)
		}
	}
	combined := strings.Join(texts, " ")

	cleaned := cleanText(combined)
	sentences := splitSentences(cleaned)
	words := strings.Fields(cleaned)

	return &Chunk{
		SourceType:        sourceType,
		Content:           cleaned,
		Sentences:         sentences,
		WordCount:         len(words),
		BehavioralSignals: result.BehavioralSignals,
		Platform:          result.Platform,
	}
}

func cleanText(s string) string {
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

func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			if i+1 >= len(text) || text[i+1] == ' ' || text[i+1] == '\n' {
				s := strings.TrimSpace(current.String())
				if len(s) > 10 {
					sentences = append(sentences, s)
				}
				current.Reset()
			}
		}
	}

	if remaining := strings.TrimSpace(current.String()); remaining != "" {
		sentences = append(sentences, remaining)
	}

	return sentences
}
