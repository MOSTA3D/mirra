package preprocessors

import "strings"

// PlainTextPreprocessor handles raw text — the fallback strategy.
// Treats the entire content as the target person's own words.
type PlainTextPreprocessor struct{}

func NewPlainTextPreprocessor() *PlainTextPreprocessor { return &PlainTextPreprocessor{} }

func (p *PlainTextPreprocessor) Name() string { return "plain_text" }

func (p *PlainTextPreprocessor) CanHandle(sourceType, content string) bool {
	// Always returns false — this is used only as fallback via registry
	return false
}

func (p *PlainTextPreprocessor) Process(sourceType, content, speakerName string) *PreprocessResult {
	result := &PreprocessResult{
		Platform:    "plain_text",
		TargetFound: true,
	}

	// Split into paragraphs, treat each as an utterance
	paragraphs := strings.Split(content, "\n")
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if len(para) > 10 {
			result.Utterances = append(result.Utterances, Utterance{
				Text:    para,
				Source:  "plain_text",
				IsQuote: true,
			})
		}
	}

	return result
}
