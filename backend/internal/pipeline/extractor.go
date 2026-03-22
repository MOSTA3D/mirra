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
	Locations  []string
	Foods      []string
	Interests  []string
	SocialPatterns []string
	Activities []string
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

	return sigs
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
	"haha", "lol", "funny", "joke", "laugh", "hilarious", "humor", "sarcas",
	"ironic", "irony", "wit", "witty", "pun", "absurd", "ridiculous", "comedy",
}

var opinionWords = []string{
	"i think", "i believe", "in my opinion", "i feel", "i know", "i'm sure",
	"honestly", "frankly", "clearly", "obviously", "i agree", "i disagree",
	"you should", "we must", "i strongly", "i always", "i never", "i love", "i hate",
}

var emotionWords = []string{
	"love", "hate", "fear", "angry", "happy", "sad", "excited", "frustrated",
	"passionate", "care", "miss", "proud", "anxious", "nervous", "joy", "pain",
	"suffer", "enjoy", "inspire", "hope", "dream", "believe", "amazing", "beautiful",
}

var formalWords = []string{
	"therefore", "furthermore", "consequently", "moreover", "nevertheless",
	"accordingly", "subsequently", "in conclusion", "regarding", "pertaining",
	"hereby", "pursuant", "notwithstanding", "aforementioned",
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
