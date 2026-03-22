// Package llm defines the LLM provider interface and implementations.
package llm

import "context"

// SynthesisInput is the structured data passed to the LLM.
type SynthesisInput struct {
	Name           string
	HumorScore     float64
	ToneScore      float64
	OpinionScore   float64
	EmotionScore   float64
	DirectScore    float64
	VocabScore     float64
	EngageScore    float64
	AvgSentenceLen float64
	TopInterests   []string
	SampleQuotes   []string
	CommonWords    []string
	Locations      []string
	Foods          []string
	Quirks         []string
}

// SynthesisOutput is the LLM response.
type SynthesisOutput struct {
	Summary      string
	VoiceGuide   string
	CoreTraits   []string
	SystemPrompt string
	Scores       map[string]float64 // LLM-calibrated scores — override stat scores
}

// Provider is the interface every LLM backend must implement.
type Provider interface {
	Name() string
	Available() bool
	Synthesize(ctx context.Context, input SynthesisInput) (*SynthesisOutput, error)
}
