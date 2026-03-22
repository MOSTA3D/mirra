package pipeline

// PersonaProfile is the structured output of the distillation pipeline.
// It captures the key dimensions that define a person's personality and communication style.
type PersonaProfile struct {
	Name        string             `json:"name"`
	Summary     string             `json:"summary"`
	Dimensions  map[string]Dimension `json:"dimensions"`
	VocabStyle  VocabStyle         `json:"vocabStyle"`
	CoreBeliefs []string           `json:"coreBeliefs"`
	Interests   []string           `json:"interests"`
	Quirks      []string           `json:"quirks"`
	SamplePhrases []string         `json:"samplePhrases"`
}

// Dimension represents a single personality/communication dimension with a score and evidence.
type Dimension struct {
	Name        string   `json:"name"`
	Score       float64  `json:"score"`       // 0.0–1.0 confidence
	Description string   `json:"description"`
	Evidence    []string `json:"evidence"`    // quotes/signals from sources
}

// VocabStyle captures the linguistic fingerprint of the person.
type VocabStyle struct {
	Formality    float64  `json:"formality"`    // 0=casual, 1=formal
	Verbosity    float64  `json:"verbosity"`    // 0=terse, 1=verbose
	CommonWords  []string `json:"commonWords"`
	AvoidWords   []string `json:"avoidWords"`
	SentenceStyle string  `json:"sentenceStyle"` // short/medium/long/mixed
}
