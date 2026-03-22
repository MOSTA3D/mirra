package pipeline

import (
	"math"
	"strings"
	"unicode"
)

// Signals is the raw extracted data from a set of chunks.
type Signals struct {
	// Linguistic content
	AllSentences   []string
	AllText        string
	TotalWords     int
	TotalSentences int
	AvgSentenceLen float64

	// Statistical metrics (used in fallback scoring)
	Stats SentenceStats

	// Explicit markers (evidence quotes)
	HumorEvidence    []string
	OpinionEvidence  []string
	EmotionEvidence  []string

	// Topic clusters
	TopicClusters  map[string]int
	VocabFrequency map[string]int

	// Behavioral signals
	Locations      []string
	Foods          []string
	Interests      []string
	SocialPatterns []string
	Activities     []string

	// Diagnostics
	Diagnostics SignalDiagnostics
}

// SentenceStats holds rich statistical features extracted from text.
type SentenceStats struct {
	// Sentence structure
	AvgLen          float64
	LenVariance     float64 // high = expressive, low = monotone
	ShortSentences  int     // < 8 words
	LongSentences   int     // > 25 words
	QuestionCount   int
	ExclamCount     int

	// Vocabulary
	UniqueWords     int
	TotalWordTokens int
	TypeTokenRatio  float64 // uniqueWords / totalWords — richness
	AvgWordLen      float64

	// Patterns
	FirstPersonCount  int // I, me, my, mine, myself
	NegationCount     int // not, never, no, don't, doesn't
	HedgeCount        int // maybe, perhaps, probably, might, could
	DirectCount       int // definitely, absolutely, clearly, must, will
	EmojiCount        int
	CapsRatio         float64 // proportion of CAPS words — intensity

	// Language detection (rough)
	ArabicCharRatio float64
}

// SignalDiagnostics explains why dimensions scored low.
type SignalDiagnostics struct {
	InsufficientContent bool
	NoFirstPerson       bool
	LikelyThirdPerson   bool
	SuggestedActions    []string
}

// Extractor pulls rich statistical signals from ingested chunks.
type Extractor struct{}

func NewExtractor() *Extractor { return &Extractor{} }

// Extract analyzes chunks and returns rich signals for scoring.
func (e *Extractor) Extract(chunks []*Chunk) *Signals {
	sigs := &Signals{
		TopicClusters:  make(map[string]int),
		VocabFrequency: make(map[string]int),
	}

	var allWords []string
	var sentenceLens []float64
	var totalSentenceLen int

	for _, chunk := range chunks {
		sigs.AllSentences = append(sigs.AllSentences, chunk.Sentences...)
		sigs.TotalWords += chunk.WordCount
		sigs.TotalSentences += len(chunk.Sentences)
		sigs.AllText += " " + chunk.Content

		// Behavioral signals
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
			words := strings.Fields(sentence)
			sentenceLens = append(sentenceLens, float64(len(words)))
			totalSentenceLen += len(words)
			lower := strings.ToLower(sentence)

			// Evidence collection for explicit markers
			if containsAny(lower, humorMarkers) {
				sigs.HumorEvidence = append(sigs.HumorEvidence, sentence)
			}
			if containsAny(lower, opinionMarkers) {
				sigs.OpinionEvidence = append(sigs.OpinionEvidence, sentence)
			}
			if containsAny(lower, emotionMarkers) {
				sigs.EmotionEvidence = append(sigs.EmotionEvidence, sentence)
			}

			// Topic clustering
			for topic, keywords := range topicKeywords {
				if containsAny(lower, keywords) {
					sigs.TopicClusters[topic]++
				}
			}
		}

		// Vocabulary
		for _, word := range strings.Fields(strings.ToLower(chunk.Content)) {
			cleaned := cleanWord(word)
			if len(cleaned) > 2 && !isStopWord(cleaned) {
				sigs.VocabFrequency[cleaned]++
				allWords = append(allWords, cleaned)
			}
		}
	}

	if sigs.TotalSentences > 0 {
		sigs.AvgSentenceLen = float64(totalSentenceLen) / float64(sigs.TotalSentences)
	}

	// Compute rich stats
	sigs.Stats = computeStats(sigs.AllText, sigs.AllSentences, sentenceLens, allWords)
	sigs.Diagnostics = buildDiagnostics(sigs)

	return sigs
}

// computeStats extracts rich statistical features from the full text.
func computeStats(fullText string, sentences []string, sentenceLens []float64, words []string) SentenceStats {
	stats := SentenceStats{}
	if len(sentences) == 0 {
		return stats
	}

	// Sentence length stats
	stats.AvgLen = mean(sentenceLens)
	stats.LenVariance = variance(sentenceLens, stats.AvgLen)
	for _, l := range sentenceLens {
		if l < 8 {
			stats.ShortSentences++
		}
		if l > 25 {
			stats.LongSentences++
		}
	}

	// Punctuation patterns
	for _, s := range sentences {
		trimmed := strings.TrimSpace(s)
		if strings.HasSuffix(trimmed, "?") {
			stats.QuestionCount++
		}
		if strings.HasSuffix(trimmed, "!") {
			stats.ExclamCount++
		}
	}

	// Vocabulary richness
	uniqueWords := map[string]bool{}
	var totalWordLen float64
	var capsCount int

	for _, w := range words {
		uniqueWords[w] = true
		totalWordLen += float64(len(w))
	}
	stats.UniqueWords = len(uniqueWords)
	stats.TotalWordTokens = len(words)
	if len(words) > 0 {
		stats.TypeTokenRatio = clamp(float64(stats.UniqueWords) / float64(stats.TotalWordTokens) * 1.5)
		stats.AvgWordLen = totalWordLen / float64(len(words))
	}

	// Count CAPS words in full text (intensity signal)
	rawWords := strings.Fields(fullText)
	for _, w := range rawWords {
		if len(w) > 2 && w == strings.ToUpper(w) && isAlpha(w) {
			capsCount++
		}
	}
	if len(rawWords) > 0 {
		stats.CapsRatio = float64(capsCount) / float64(len(rawWords))
	}

	// Linguistic patterns
	lower := strings.ToLower(fullText)

	firstPersonWords := []string{" i ", " me ", " my ", " mine ", " myself ", "^i ", "^me "}
	for _, fp := range firstPersonWords {
		stats.FirstPersonCount += strings.Count(lower, fp)
	}

	for _, neg := range negationWords {
		stats.NegationCount += strings.Count(lower, neg)
	}
	for _, hedge := range hedgeWords {
		stats.HedgeCount += strings.Count(lower, hedge)
	}
	for _, direct := range directnessWords {
		stats.DirectCount += strings.Count(lower, direct)
	}

	// Emoji count
	for _, r := range fullText {
		if r > 0x1F300 {
			stats.EmojiCount++
		}
	}

	// Arabic character ratio
	arabicCount := 0
	totalChars := 0
	for _, r := range fullText {
		if !unicode.IsSpace(r) {
			totalChars++
			if r >= 0x0600 && r <= 0x06FF {
				arabicCount++
			}
		}
	}
	if totalChars > 0 {
		stats.ArabicCharRatio = float64(arabicCount) / float64(totalChars)
	}

	return stats
}

func buildDiagnostics(sigs *Signals) SignalDiagnostics {
	d := SignalDiagnostics{}

	if sigs.TotalWords < 50 {
		d.InsufficientContent = true
		d.SuggestedActions = append(d.SuggestedActions,
			"Add more content — at least a few paragraphs give better results")
	}

	if sigs.Stats.FirstPersonCount < 2 && sigs.TotalSentences > 5 {
		d.NoFirstPerson = true
		d.SuggestedActions = append(d.SuggestedActions,
			"Add content written by the person directly — their messages, posts, or interview quotes")
	}

	if len(sigs.HumorEvidence) == 0 && sigs.TotalSentences > 8 {
		d.SuggestedActions = append(d.SuggestedActions,
			"Humor score needs more data — add casual chats, tweets, or moments where they're being playful")
	}

	if len(sigs.OpinionEvidence) == 0 && sigs.TotalSentences > 8 {
		d.SuggestedActions = append(d.SuggestedActions,
			"Opinion score needs more data — add interviews or posts where they share views on topics")
	}

	if len(sigs.EmotionEvidence) == 0 && sigs.TotalSentences > 8 {
		d.SuggestedActions = append(d.SuggestedActions,
			"Emotion score needs more data — personal messages or expressive posts work best")
	}

	return d
}

// --- Statistical helpers ---

func mean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

func variance(vals []float64, mean float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		diff := v - mean
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(vals)))
}

func clamp(v float64) float64 {
	return math.Min(1.0, math.Max(0.0, v))
}

func isAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
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

// --- Signal dictionaries (evidence collection only — not used for scoring) ---

var humorMarkers = []string{
	"haha", "hehe", "lol", "lmao", "😂", "🤣", "😄", "funny", "joke", "laugh",
	"hilarious", "sarcas", "ironic", "wit", "witty", "absurd", "comedy", "kidding",
	"jk", "بضحك", "هههه", "ههه", "مضحك", "نكتة",
}

var opinionMarkers = []string{
	"i think", "i believe", "i feel", "i know", "i'm sure", "i am sure",
	"in my opinion", "personally", "honestly", "frankly", "clearly",
	"i agree", "i disagree", "i prefer", "i find", "my view", "my take",
	"he believes", "she believes", "he thinks", "she thinks", "according to",
	"stated that", "argued", "أعتقد", "أظن", "رأيي", "برأيي",
}

var emotionMarkers = []string{
	"love", "hate", "fear", "angry", "happy", "sad", "excited", "frustrated",
	"passionate", "care", "miss", "proud", "anxious", "joy", "pain",
	"amazing", "beautiful", "terrible", "incredible", "heartbroken", "grateful",
	"❤️", "💔", "😍", "😢", "😭", "🥹", "😤",
	"أحب", "أكره", "سعيد", "حزين", "ممتن",
}

var negationWords = []string{" not ", " never ", " no ", " don't ", " doesn't ", " won't ", " can't ", " isn't "}
var hedgeWords = []string{" maybe ", " perhaps ", " probably ", " might ", " could ", " possibly ", " sort of ", " kind of "}
var directnessWords = []string{" definitely ", " absolutely ", " always ", " must ", " will ", " clearly ", " exactly ", " certainly "}

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

func topN(freq map[string]int, n int) []string {
	type kv struct{ k string; v int }
	var sorted []kv
	for k, v := range freq {
		sorted = append(sorted, kv{k, v})
	}
	// Simple sort
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].v > sorted[i].v {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	var result []string
	for i, kv := range sorted {
		if i >= n {
			break
		}
		result = append(result, kv.k)
	}
	return result
}
