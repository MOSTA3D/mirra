// Package llm defines the LLM provider interface and implementations.
// The pipeline uses this for persona synthesis when an API key is configured.
// Falls back to rule-based distillation if no provider is available.
package llm

import "context"

// SynthesisInput is the structured data passed to the LLM.
// We send signals, not raw text — keeps tokens low and output focused.
type SynthesisInput struct {
	Name           string
	HumorScore     float64
	ToneScore      float64  // 0=casual, 1=formal
	OpinionScore   float64
	EmotionScore   float64
	AvgSentenceLen float64
	TopInterests   []string
	SampleQuotes   []string // up to 8 representative quotes
	CommonWords    []string
	Locations      []string
	Foods          []string
	Quirks         []string
}

// SynthesisOutput is the LLM response.
type SynthesisOutput struct {
	Summary       string // 2-3 sentence persona summary
	VoiceGuide    string // how they actually talk — the key deliverable
	CoreTraits    []string
	SystemPrompt  string // complete system prompt for the persona
}

// Provider is the interface every LLM backend must implement.
type Provider interface {
	Name() string
	Available() bool
	Synthesize(ctx context.Context, input SynthesisInput) (*SynthesisOutput, error)
}
