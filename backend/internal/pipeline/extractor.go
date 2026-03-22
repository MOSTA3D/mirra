package pipeline

import (
	"math"
	"strings"
	"unicode"
)

// Signals is the raw extracted data from a set of chunks.
type Signals struct {
	HumorMarkers    []string
	OpinionMarkers  []string
	EmotionMarkers  []string
	ToneMarkers     []string
	TopicClusters   map[string]int
	VocabFrequency  map[string]int
	AvgSentenceLen  float64
	TotalWords      int
	TotalSentences  int
	AllSentences    []string
	// Behavioral signals collected from all sources (silently inform persona)
	Locations      []string
	Foods          []string
	Interests      []string
	SocialPatterns []string
	Activities     []string
	// Diagnostic — explains low scores to users
	Diagnostics SignalDiagnostics
}

// SignalDiagnostics explains why dimensions scored low.
type SignalDiagnostics struct {
	InsufficientContent bool   // fewer than 50 words
	NoFirstPerson       bool   // no "I" statements found — probably third-party content
	LikelyThirdPerson   bool   // text talks about the person, not from them
	SuggestedActions    []string
}

// Extractor pulls personality signals from ingested chunks.
type Extractor struct{}

func NewExtractor() *Extractor { return &Extractor{} }

// Extract analyzes a set of chunks and returns raw signals.
func (e *Extractor) Extract(chunks []*Chunk) *Signals {
	sigs := &Signals{
		TopicClusters:  make(map[string]int),
		VocabFrequency: make(map[string]int),
	}

	var totalSentenceLen int

	for _, chunk := range chunks {
		sigs.AllSentences = append(sigs.AllSentences, chunk.Sentences...)
		sigs.TotalWords += chunk.WordCount
		sigs.TotalSentences += len(chunk.Sentences)

		// Collect behavioral signals
		for _, sig := range chunk.BehavioralSignals {
			switch sig.Category {
			case "location":
				sigs.Locations = append(sigs.Locations, sig.Value)
			case "food":
				sigs.Foods = append(sigs.Foods, sig.Value)
			case "interest":
				sigs.Interests = append(sigs.Interests, sig.Value)
			case "social":
				sigs.SocialPatterns = append(sigs.SocialPatterns, sig.Value)
			case "activity":
				sigs.Activities = append(sigs.Activities, sig.Value)
			}
		}

		for _, sentence := range chunk.Sentences {
			totalSentenceLen += len(strings.Fields(sentence))
			lower := strings.ToLower(sentence)

			// Humor signals
			if containsAny(lower, humorWords) {
				sigs.HumorMarkers = append(sigs.HumorMarkers, sentence)
			}

			// Opinion signals
			if containsAny(lower, opinionWords) {
				sigs.OpinionMarkers = append(sigs.OpinionMarkers, sentence)
			}

			// Emotion signals
			if containsAny(lower, emotionWords) {
				sigs.EmotionMarkers = append(sigs.EmotionMarkers, sentence)
			}

			// Tone signals
			if containsAny(lower, formalWords) {
				sigs.ToneMarkers = append(sigs.ToneMarkers, sentence)
			}

			// Topic clustering
			for topic, keywords := range topicKeywords {
				if containsAny(lower, keywords) {
					sigs.TopicClusters[topic]++
				}
			}
		}

		// Vocabulary frequency
		for _, word := range strings.Fields(strings.ToLower(chunk.Content)) {
			word = cleanWord(word)
			if len(word) > 3 && !isStopWord(word) {
				sigs.VocabFrequency[word]++
			}
		}
	}

	if sigs.TotalSentences > 0 {
		sigs.AvgSentenceLen = float64(totalSentenceLen) / float64(sigs.TotalSentences)
	}

	// Build diagnostics
	sigs.Diagnostics = buildDiagnostics(sigs)

	return sigs
}

func buildDiagnostics(sigs *Signals) SignalDiagnostics {
	d := SignalDiagnostics{}

	if sigs.TotalWords < 50 {
		d.InsufficientContent = true
		d.SuggestedActions = append(d.SuggestedActions,
			"Add more content — at least a few paragraphs work best")
	}

	// Check for first-person signals
	firstPersonCount := len(sigs.OpinionMarkers) + len(sigs.EmotionMarkers)
	if firstPersonCount == 0 && sigs.TotalSentences > 5 {
		// Check if content is third-person
		d.NoFirstPerson = true
		d.SuggestedActions = append(d.SuggestedActions,
			"Add content written by the person directly (their posts, messages, or quotes)")
	}

	if len(sigs.HumorMarkers) == 0 && sigs.TotalSentences > 10 {
		d.SuggestedActions = append(d.SuggestedActions,
			"Humor score is 0 — add casual conversations or posts where they're being funny")
	}

	if len(sigs.OpinionMarkers) == 0 && sigs.TotalSentences > 10 {
		d.SuggestedActions = append(d.SuggestedActions,
			"Opinion score is 0 — add interviews, tweets, or texts where they share their views")
	}

	if len(sigs.EmotionMarkers) == 0 && sigs.TotalSentences > 10 {
		d.SuggestedActions = append(d.SuggestedActions,
			"Emotion score is 0 — add personal messages or posts where they express feelings")
	}

	return d
}

func cleanWord(w string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, w)
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// Clamp to [0,1]
func clamp(v float64) float64 {
	return math.Min(1.0, math.Max(0.0, v))
}

// --- Signal dictionaries ---

var humorWords = []string{
	// English
	"haha", "lol", "lmao", "rofl", "funny", "joke", "jokes", "laugh", "laughing",
	"hilarious", "humor", "humour", "sarcas", "ironic", "irony", "wit", "witty",
	"pun", "absurd", "ridiculous", "comedy", "😂", "🤣", "😄", "😆", "😅",
	"kidding", "just kidding", "jk", "playful", "teasing", "banter",
	// Arabic
	"هههه", "ههه", "هه", "طريف", "مضحك", "نكتة", "ضحك",
}

var opinionWords = []string{
	// English first-person
	"i think", "i believe", "i feel", "i know", "i'm sure", "i am sure",
	"in my opinion", "in my view", "from my perspective", "personally",
	"honestly", "frankly", "clearly", "obviously", "i agree", "i disagree",
	"you should", "we must", "i strongly", "i always", "i never",
	"i love", "i hate", "i prefer", "i find", "i consider", "i say",
	"my view", "my opinion", "my take", "my belief",
	// Third-person opinion markers (for articles/bios)
	"he believes", "she believes", "they believe",
	"he thinks", "she thinks", "he says", "she says",
	"according to", "stated that", "argued that", "claimed that",
	// Arabic
	"أعتقد", "أظن", "رأيي", "في رأيي", "أنا أفضل", "برأيي", "أرى",
}

var emotionWords = []string{
	// English
	"love", "hate", "fear", "angry", "anger", "happy", "happiness", "sad", "sadness",
	"excited", "excitement", "frustrated", "frustration", "passionate", "passion",
	"care", "caring", "miss", "proud", "pride", "anxious", "anxiety", "nervous",
	"joy", "pain", "suffer", "enjoy", "inspire", "hope", "dream", "believe",
	"amazing", "beautiful", "wonderful", "terrible", "awful", "incredible",
	"heartbroken", "thrilled", "devastated", "grateful", "thankful",
	"❤️", "💔", "😍", "😢", "😭", "🥹", "😤", "😡",
	// Arabic
	"أحب", "أكره", "سعيد", "حزين", "خائف", "متحمس", "ممتن", "أتمنى",
}

var formalWords = []string{
	"therefore", "furthermore", "consequently", "moreover", "nevertheless",
	"accordingly", "subsequently", "in conclusion", "regarding", "pertaining",
	"hereby", "pursuant", "notwithstanding", "aforementioned",
	"with respect to", "in accordance", "it should be noted",
}

var topicKeywords = map[string][]string{
	"technology":   {"technology", "software", "code", "computer", "ai", "tech", "digital", "internet", "app", "data"},
	"art":          {"art", "music", "film", "movie", "paint", "creative", "design", "aesthetic", "visual", "cinema"},
	"philosophy":   {"meaning", "truth", "philosophy", "existence", "conscious", "reality", "think", "wisdom", "moral"},
	"business":     {"business", "startup", "company", "market", "money", "invest", "product", "customer", "revenue"},
	"science":      {"science", "research", "study", "experiment", "theory", "data", "evidence", "discover"},
	"politics":     {"political", "government", "policy", "society", "freedom", "justice", "rights", "democracy"},
	"sports":       {"sport", "game", "play", "team", "win", "competition", "athlete", "training", "match"},
	"spirituality": {"god", "spirit", "soul", "faith", "meditation", "mindful", "universe", "energy", "divine"},
}

var stopWords = map[string]bool{
	"the": true, "and": true, "that": true, "this": true, "with": true,
	"have": true, "from": true, "they": true, "will": true, "been": true,
	"were": true, "said": true, "each": true, "which": true, "their": true,
	"time": true, "would": true, "there": true, "could": true, "other": true,
	"more": true, "when": true, "what": true, "some": true, "also": true,
	"into": true, "than": true, "then": true, "only": true, "just": true,
	"about": true, "very": true, "your": true, "know": true, "make": true,
}

func isStopWord(w string) bool { return stopWords[w] }
