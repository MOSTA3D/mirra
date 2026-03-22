package preprocessors

// Registry holds all available preprocessors and picks the right one automatically.
// This is the context in the Strategy pattern.
type Registry struct {
	strategies []Preprocessor
	fallback   Preprocessor
}

// NewRegistry creates the registry with all built-in strategies registered.
func NewRegistry() *Registry {
	r := &Registry{}

	// Order matters — more specific strategies first
	r.Register(NewWhatsAppPreprocessor())
	r.Register(NewTelegramPreprocessor())
	r.Register(NewTwitterPreprocessor())
	r.Register(NewInstagramPreprocessor())
	r.Register(NewArticlePreprocessor())

	// Plain text is always the fallback
	r.fallback = NewPlainTextPreprocessor()

	return r
}

// Register adds a preprocessor strategy to the registry.
func (r *Registry) Register(p Preprocessor) {
	r.strategies = append(r.strategies, p)
}

// Resolve picks the best preprocessor for the given source.
// Falls back to plain text if nothing matches.
func (r *Registry) Resolve(sourceType, content string) Preprocessor {
	for _, strategy := range r.strategies {
		if strategy.CanHandle(sourceType, content) {
			return strategy
		}
	}
	return r.fallback
}

// Guidance returns user-facing instructions for how to export data from a platform.
// Shown in the frontend "Add Source" step.
func Guidance(platform string) string {
	guides := map[string]string{
		"whatsapp":  "Export: Open the chat → ⋮ Menu → More → Export Chat → Without Media. Paste the .txt content here.",
		"telegram":  "Export: Desktop app → Chat → ⋮ → Export Chat History → JSON format. Paste the JSON content here.",
		"twitter":   "Export: Settings → Your Account → Download Archive → tweets.js. Paste the file content here.",
		"instagram": "Export: Settings → Your Activity → Download Your Information → JSON → Posts. Paste the content here.",
	}
	if g, ok := guides[platform]; ok {
		return g
	}
	return "Paste any text, URLs, or content related to this person."
}
