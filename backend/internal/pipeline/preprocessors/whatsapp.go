package preprocessors

import (
	"regexp"
	"strings"
)

// WhatsAppPreprocessor parses WhatsApp chat exports (.txt format).
//
// Format examples:
//   [12/03/2024, 10:45:23] John Doe: Hey how are you?
//   12/03/2024, 10:45 - John Doe: Hey how are you?
//   3/22/24, 1:00 AM - John: message here
type WhatsAppPreprocessor struct {
	// Two common WhatsApp export formats
	bracketPattern *regexp.Regexp
	dashPattern    *regexp.Regexp
}

func NewWhatsAppPreprocessor() *WhatsAppPreprocessor {
	return &WhatsAppPreprocessor{
		bracketPattern: regexp.MustCompile(`^\[[\d/,: ]+\]\s+(.+?):\s+(.+)$`),
		dashPattern:    regexp.MustCompile(`^[\d/,: APM]+\s+-\s+(.+?):\s+(.+)$`),
	}
}

func (p *WhatsAppPreprocessor) Name() string { return "whatsapp" }

func (p *WhatsAppPreprocessor) CanHandle(sourceType, content string) bool {
	if sourceType == "pdf" {
		return false
	}
	lines := strings.Split(content, "\n")
	matches := 0
	for i, line := range lines {
		if i > 20 {
			break
		}
		if p.bracketPattern.MatchString(line) || p.dashPattern.MatchString(line) {
			matches++
		}
	}
	// Need at least 3 matching lines to be confident
	return matches >= 3
}

func (p *WhatsAppPreprocessor) Process(sourceType, content, speakerName string) *PreprocessResult {
	result := &PreprocessResult{
		Platform: "whatsapp",
	}

	speakerLower := strings.ToLower(strings.TrimSpace(speakerName))
	lines := strings.Split(content, "\n")

	speakersFound := map[string]int{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		speaker, message := p.parseLine(line)
		if speaker == "" {
			continue
		}

		speakersFound[strings.ToLower(speaker)]++

		// Match speaker
		if speakerLower == "" || strings.Contains(strings.ToLower(speaker), speakerLower) {
			result.TargetFound = true

			// Skip system messages
			if isSystemMessage(message) {
				continue
			}

			// Extract behavioral signals from messages
			p.extractBehavioral(message, result)

			// Only add as utterance if it's actual text (not just media/location)
			if isTextualContent(message) && !isLocationOnly(message) {
				// Strip location prefix before adding as utterance
				cleaned := stripLocationPrefix(message)
				if len(strings.TrimSpace(cleaned)) > 5 {
					result.Utterances = append(result.Utterances, Utterance{
						Text:    cleaned,
						Source:  "whatsapp",
						IsQuote: true,
					})
				}
			}
		}
	}

	// If no speaker specified, use all messages
	if speakerLower == "" {
		result.TargetFound = true
	}

	return result
}

func (p *WhatsAppPreprocessor) parseLine(line string) (speaker, message string) {
	if m := p.bracketPattern.FindStringSubmatch(line); len(m) == 3 {
		return m[1], m[2]
	}
	if m := p.dashPattern.FindStringSubmatch(line); len(m) == 3 {
		return m[1], m[2]
	}
	return "", ""
}

func isSystemMessage(msg string) bool {
	systemPhrases := []string{
		"messages and calls are end-to-end encrypted",
		"joined using this group's invite link",
		"added", "removed", "left", "changed the group",
		"changed this group's icon",
		"security code changed",
		"created group",
		"<media omitted>",
	}
	lower := strings.ToLower(msg)
	for _, phrase := range systemPhrases {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

func isTextualContent(msg string) bool {
	lower := strings.ToLower(msg)
	nonText := []string{
		"<media omitted>", "image omitted", "video omitted",
		"audio omitted", "document omitted", "sticker omitted",
		"location:", "live location shared",
		"this message was deleted",
	}
	for _, skip := range nonText {
		if strings.Contains(lower, skip) {
			return false
		}
	}
	return len(strings.TrimSpace(msg)) > 2
}

func (p *WhatsAppPreprocessor) extractBehavioral(msg string, result *PreprocessResult) {
	lower := strings.ToLower(msg)

	// Location signals
	if strings.Contains(lower, "location:") || strings.Contains(msg, "📍") {
		result.BehavioralSignals = append(result.BehavioralSignals, BehavioralSignal{
			Category: "location",
			Value:    extractAfter(msg, "📍"),
			Count:    1,
		})
	}

	// Food signals
	foodWords := []string{"eating", "food", "restaurant", "lunch", "dinner", "breakfast", "cafe", "coffee"}
	for _, fw := range foodWords {
		if strings.Contains(lower, fw) {
			result.BehavioralSignals = append(result.BehavioralSignals, BehavioralSignal{
				Category: "food",
				Value:    msg,
				Count:    1,
			})
			break
		}
	}
}

func isLocationOnly(msg string) bool {
	return strings.HasPrefix(strings.TrimSpace(msg), "📍") && len(strings.Fields(msg)) < 5
}

func stripLocationPrefix(msg string) string {
	if idx := strings.Index(msg, "📍"); idx != -1 {
		after := strings.TrimSpace(msg[idx+len("📍"):])
		// If location is inline with real text, keep only the text part
		parts := strings.SplitN(after, " ", 4)
		if len(parts) > 2 {
			return strings.Join(parts[2:], " ")
		}
		return ""
	}
	return msg
}

func extractAfter(s, prefix string) string {
	idx := strings.Index(s, prefix)
	if idx == -1 {
		return s
	}
	rest := strings.TrimSpace(s[idx+len(prefix):])
	if len(rest) > 50 {
		return rest[:50]
	}
	return rest
}
