// Package preprocessors implements the Strategy pattern for handling different source formats.
// Each preprocessor knows how to parse one specific format and extract two things:
//   - Utterances: text that can be attributed to the target person's own voice
//   - BehavioralSignals: non-verbal data (locations, likes, activity patterns)
//
// Adding a new platform = implement Preprocessor interface in a new file. That's it.
package preprocessors

// Utterance is a piece of text that can be attributed to the target person.
type Utterance struct {
	Text      string
	Source    string // e.g. "whatsapp", "twitter"
	IsQuote   bool   // true = directly their words; false = attributed/paraphrased
	Timestamp string // optional
}

// BehavioralSignal is a non-verbal data point that informs the persona silently.
// It never appears verbatim in the export — only influences the profile.
type BehavioralSignal struct {
	Category string // location | food | interest | activity | social | schedule
	Value    string // e.g. "Dubai Marina", "sushi", "football"
	Count    int    // frequency/weight
}

// PreprocessResult is the output of any preprocessor.
type PreprocessResult struct {
	Utterances        []Utterance
	BehavioralSignals []BehavioralSignal
	Platform          string
	TargetFound       bool // false = speaker not found in source (for chat formats)
}

// Preprocessor is the strategy interface. Each platform implements this.
type Preprocessor interface {
	// Name returns the platform identifier.
	Name() string

	// CanHandle reports whether this preprocessor can handle the given source.
	// Used by the registry for auto-detection.
	CanHandle(sourceType, content string) bool

	// Process parses the source and returns utterances + behavioral signals.
	// speakerName is the target person's display name (empty = not specified).
	Process(sourceType, content, speakerName string) *PreprocessResult
}
