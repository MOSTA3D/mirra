package preprocessors

import (
	"encoding/json"
	"strings"
)

// TelegramPreprocessor parses Telegram JSON chat exports.
//
// Telegram export format (result.json):
//
//	{
//	  "messages": [
//	    { "id": 1, "type": "message", "from": "John", "text": "Hello" },
//	    { "id": 2, "type": "message", "from": "Jane", "text": [{"type":"plain","text":"Hi"}] }
//	  ]
//	}
type TelegramPreprocessor struct{}

func NewTelegramPreprocessor() *TelegramPreprocessor { return &TelegramPreprocessor{} }

func (p *TelegramPreprocessor) Name() string { return "telegram" }

type telegramExport struct {
	Messages []telegramMessage `json:"messages"`
}

type telegramMessage struct {
	Type      string      `json:"type"`
	From      string      `json:"from"`
	FromID    string      `json:"from_id"`
	Text      interface{} `json:"text"` // can be string or []interface{}
	MediaType string      `json:"media_type"`
	Location  *struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location_information"`
}

func (p *TelegramPreprocessor) CanHandle(sourceType, content string) bool {
	if sourceType == "pdf" {
		return false
	}
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "{") {
		return false
	}
	var export telegramExport
	if err := json.Unmarshal([]byte(content), &export); err != nil {
		return false
	}
	return len(export.Messages) > 0
}

func (p *TelegramPreprocessor) Process(sourceType, content, speakerName string) *PreprocessResult {
	result := &PreprocessResult{Platform: "telegram"}

	var export telegramExport
	if err := json.Unmarshal([]byte(content), &export); err != nil {
		return result
	}

	speakerLower := strings.ToLower(strings.TrimSpace(speakerName))

	for _, msg := range export.Messages {
		if msg.Type != "message" {
			continue
		}

		fromLower := strings.ToLower(msg.From)
		isTarget := speakerLower == "" || strings.Contains(fromLower, speakerLower)

		if !isTarget {
			continue
		}
		result.TargetFound = true

		// Location signal
		if msg.Location != nil {
			result.BehavioralSignals = append(result.BehavioralSignals, BehavioralSignal{
				Category: "location",
				Value:    "shared location",
				Count:    1,
			})
			continue
		}

		// Media only — behavioral signal
		if msg.MediaType != "" && msg.MediaType != "sticker" {
			result.BehavioralSignals = append(result.BehavioralSignals, BehavioralSignal{
				Category: "activity",
				Value:    "shared " + msg.MediaType,
				Count:    1,
			})
		}

		// Extract text
		text := extractTelegramText(msg.Text)
		if text != "" && isTextualContent(text) {
			result.Utterances = append(result.Utterances, Utterance{
				Text:    text,
				Source:  "telegram",
				IsQuote: true,
			})
		}
	}

	if speakerLower == "" {
		result.TargetFound = true
	}

	return result
}

func extractTelegramText(raw interface{}) string {
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case []interface{}:
		var parts []string
		for _, item := range v {
			switch t := item.(type) {
			case string:
				parts = append(parts, t)
			case map[string]interface{}:
				if text, ok := t["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.TrimSpace(strings.Join(parts, ""))
	}
	return ""
}
