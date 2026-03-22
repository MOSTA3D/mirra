package preprocessors

import (
	"regexp"
	"strings"
)

// ArticlePreprocessor handles news articles, Wikipedia pages, and similar third-party content.
//
// Strategy: extract sentences that appear to be direct quotes from the target person.
// Third-person narrative is treated as background context, not the person's voice.
// Only sentences with attribution markers become utterances.
type ArticlePreprocessor struct {
	quotePattern *regexp.Regexp
}

func NewArticlePreprocessor() *ArticlePreprocessor {
	return &ArticlePreprocessor{
		// Matches patterns like: said "...", told reporters "...", according to X, "...", said X
		quotePattern: regexp.MustCompile(`(?i)(?:said|told|stated|wrote|tweeted|posted|explained|added|noted|remarked|commented)[,\s]+"([^"]{10,300})"`),
	}
}

func (p *ArticlePreprocessor) Name() string { return "article" }

func (p *ArticlePreprocessor) CanHandle(sourceType, content string) bool {
	if sourceType == "pdf" || sourceType == "url" {
		return true // Articles come as URLs or pasted text
	}
	// For plain text, check if it looks like an article (third-person narrative)
	lower := strings.ToLower(content)
	thirdPersonSignals := []string{
		" he ", " she ", " they ", "according to", "was born", "is known for",
		"has been", "the actor", "the singer", "the politician",
	}
	matches := 0
	for _, sig := range thirdPersonSignals {
		if strings.Contains(lower, sig) {
			matches++
		}
	}
	return matches >= 3
}

func (p *ArticlePreprocessor) Process(sourceType, content, speakerName string) *PreprocessResult {
	result := &PreprocessResult{Platform: "article", TargetFound: true}

	// Extract direct quotes
	matches := p.quotePattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			quote := strings.TrimSpace(match[1])
			if len(quote) > 10 {
				result.Utterances = append(result.Utterances, Utterance{
					Text:    quote,
					Source:  "article_quote",
					IsQuote: true,
				})
			}
		}
	}

	// Extract behavioral signals from third-person narrative
	p.extractBehavioralFromArticle(content, result)

	return result
}

func (p *ArticlePreprocessor) extractBehavioralFromArticle(content string, result *PreprocessResult) {
	lower := strings.ToLower(content)

	// Interest signals from "known for", "famous for", "loves", "passionate about"
	interestPatterns := []struct {
		trigger string
		extract bool
	}{
		{"known for", true},
		{"famous for", true},
		{"passionate about", true},
		{"love of", true},
		{"interest in", true},
		{"fan of", true},
	}

	sentences := strings.Split(content, ".")
	for _, sentence := range sentences {
		sentLower := strings.ToLower(sentence)
		for _, pat := range interestPatterns {
			if strings.Contains(sentLower, pat.trigger) {
				// Extract just the subject after the trigger phrase
				idx := strings.Index(sentLower, pat.trigger)
				fragment := strings.TrimSpace(sentence[idx+len(pat.trigger):])
				// Clean punctuation and cap at 40 chars
				end := strings.IndexAny(fragment, ".,;:()")
				if end == -1 || end > 40 {
					end = min(40, len(fragment))
				}
				value := strings.TrimSpace(fragment[:end])
				if len(value) > 3 {
					result.BehavioralSignals = append(result.BehavioralSignals, BehavioralSignal{
						Category: "interest",
						Value:    value,
						Count:    1,
					})
				}
				break
			}
		}
	}

	// Location signals
	locationTriggers := []string{"based in", "lives in", "born in", "grew up in", "from "}
	for _, trigger := range locationTriggers {
		idx := strings.Index(lower, trigger)
		if idx != -1 {
			fragment := content[idx+len(trigger):]
			if len(fragment) > 5 {
				end := strings.IndexAny(fragment, ".,;")
				if end == -1 || end > 50 {
					end = 30
				}
				result.BehavioralSignals = append(result.BehavioralSignals, BehavioralSignal{
					Category: "location",
					Value:    strings.TrimSpace(fragment[:end]),
					Count:    1,
				})
			}
		}
	}
}
