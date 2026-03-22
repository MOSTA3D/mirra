package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	groqBaseURL = "https://api.groq.com/openai/v1/chat/completions"
	groqModel   = "llama-3.1-8b-instant"
	groqTimeout = 30 * time.Second
)

// GroqProvider implements Provider using Groq's API (OpenAI-compatible).
type GroqProvider struct {
	apiKey string
	client *http.Client
}

func NewGroqProvider(apiKey string) *GroqProvider {
	return &GroqProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: groqTimeout},
	}
}

func (g *GroqProvider) Name() string      { return "groq/llama-3.1-8b-instant" }
func (g *GroqProvider) Available() bool   { return g.apiKey != "" }

// Synthesize sends structured signals to Groq and returns a rich persona profile.
func (g *GroqProvider) Synthesize(ctx context.Context, input SynthesisInput) (*SynthesisOutput, error) {
	prompt := buildPrompt(input)

	body, err := json.Marshal(map[string]interface{}{
		"model": groqModel,
		"messages": []map[string]string{
			{"role": "system", "content": systemInstruction},
			{"role": "user", "content": prompt},
		},
		"temperature":     0.7,
		"max_tokens":      1200,
		"response_format": map[string]string{"type": "json_object"},
	})
	if err != nil {
		return nil, fmt.Errorf("groq: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", groqBaseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("groq: request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("groq: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("groq: read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("groq: status %d: %s", resp.StatusCode, raw)
	}

	// Parse OpenAI-compatible response
	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		return nil, fmt.Errorf("groq: parse response: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("groq: empty response")
	}

	content := apiResp.Choices[0].Message.Content
	return parseOutput(content, input.Name)
}

// buildPrompt constructs a tight, token-efficient prompt from structured signals.
func buildPrompt(in SynthesisInput) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Build a persona profile for: %s\n\n", in.Name))

	b.WriteString("## Extracted signals (from their actual content):\n")
	b.WriteString(fmt.Sprintf("- Humor level: %.0f%%\n", in.HumorScore*100))
	b.WriteString(fmt.Sprintf("- Tone: %.0f%% formal (0=very casual, 100=very formal)\n", in.ToneScore*100))
	b.WriteString(fmt.Sprintf("- Opinionated: %.0f%%\n", in.OpinionScore*100))
	b.WriteString(fmt.Sprintf("- Emotional expression: %.0f%%\n", in.EmotionScore*100))
	b.WriteString(fmt.Sprintf("- Avg sentence length: %.0f words (%s)\n", in.AvgSentenceLen, sentenceStyleLabel(in.AvgSentenceLen)))

	if len(in.TopInterests) > 0 {
		b.WriteString(fmt.Sprintf("- Key interests/topics: %s\n", strings.Join(in.TopInterests, ", ")))
	}
	if len(in.CommonWords) > 0 {
		b.WriteString(fmt.Sprintf("- Characteristic vocabulary: %s\n", strings.Join(in.CommonWords[:min5(len(in.CommonWords))], ", ")))
	}
	if len(in.Locations) > 0 {
		b.WriteString(fmt.Sprintf("- Places mentioned: %s\n", strings.Join(in.Locations, ", ")))
	}
	if len(in.Foods) > 0 {
		b.WriteString(fmt.Sprintf("- Food/dining signals: %s\n", strings.Join(in.Foods, ", ")))
	}

	if len(in.SampleQuotes) > 0 {
		b.WriteString("\n## Sample quotes (their actual words):\n")
		for i, q := range in.SampleQuotes {
			if i >= 8 {
				break
			}
			b.WriteString(fmt.Sprintf("- \"%s\"\n", q))
		}
	}

	b.WriteString(`
Respond with a JSON object with these exact keys:
{
  "summary": "2-3 sentences describing who this person is and how they communicate",
  "voice_guide": "3-5 sentences describing HOW they talk — their rhythm, phrasing, mannerisms. Be specific and vivid, not generic.",
  "core_traits": ["trait1", "trait2", "trait3", "trait4"],
  "system_prompt": "Complete system prompt (300-400 words) for an AI to roleplay as this person. Include personality, communication style, what to avoid, and example phrases."
}`)

	return b.String()
}

func parseOutput(content, name string) (*SynthesisOutput, error) {
	var parsed struct {
		Summary      string   `json:"summary"`
		VoiceGuide   string   `json:"voice_guide"`
		CoreTraits   []string `json:"core_traits"`
		SystemPrompt string   `json:"system_prompt"`
	}

	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		// If JSON parse fails, use content as summary fallback
		return &SynthesisOutput{
			Summary:      content,
			VoiceGuide:   "",
			CoreTraits:   []string{},
			SystemPrompt: fmt.Sprintf("You are roleplaying as %s. %s", name, content),
		}, nil
	}

	return &SynthesisOutput{
		Summary:      parsed.Summary,
		VoiceGuide:   parsed.VoiceGuide,
		CoreTraits:   parsed.CoreTraits,
		SystemPrompt: parsed.SystemPrompt,
	}, nil
}

func sentenceStyleLabel(avg float64) string {
	switch {
	case avg < 8:
		return "very short"
	case avg < 15:
		return "short"
	case avg < 22:
		return "medium"
	case avg < 30:
		return "long"
	default:
		return "very long"
	}
}

func min5(n int) int {
	if n < 5 {
		return n
	}
	return 5
}

const systemInstruction = `You are a persona analyst. Your job is to synthesize structured behavioral signals into a vivid, accurate persona profile.

Rules:
- Be specific and grounded in the actual signals provided
- Avoid generic descriptions like "communicates clearly" — say HOW they actually communicate
- The voice_guide should feel like a writer's note on how to write this character
- The system_prompt should be immediately usable — paste it into any AI and get the persona
- Always include the disclaimer that this is AI-generated and not the real person
- Respond only with valid JSON`
