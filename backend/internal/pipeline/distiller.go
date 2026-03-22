package pipeline

import (
	"fmt"
	"sort"
	"strings"
)

// Distiller converts raw signals into a structured PersonaProfile.
type Distiller struct{}

func NewDistiller() *Distiller { return &Distiller{} }

// Distill takes extracted signals and a persona name, returns a full PersonaProfile.
func (d *Distiller) Distill(name string, sigs *Signals) *PersonaProfile {
	profile := &PersonaProfile{
		Name:       name,
		Dimensions: make(map[string]Dimension),
	}

	// --- Humor dimension ---
	humorScore := clamp(float64(len(sigs.HumorMarkers)) / float64(max1(sigs.TotalSentences)) * 8)
	humorEvidence := take(sigs.HumorMarkers, 3)
	profile.Dimensions["humor"] = Dimension{
		Name:        "Humor",
		Score:       humorScore,
		Description: describeHumor(humorScore),
		Evidence:    humorEvidence,
	}

	// --- Tone dimension ---
	formalScore := clamp(float64(len(sigs.ToneMarkers)) / float64(max1(sigs.TotalSentences)) * 10)
	profile.Dimensions["tone"] = Dimension{
		Name:        "Tone",
		Score:       1 - formalScore, // invert: low formal = casual tone score high
		Description: describeTone(formalScore),
		Evidence:    take(sigs.ToneMarkers, 3),
	}

	// --- Opinions dimension ---
	opinionScore := clamp(float64(len(sigs.OpinionMarkers)) / float64(max1(sigs.TotalSentences)) * 5)
	profile.Dimensions["opinions"] = Dimension{
		Name:        "Opinionated",
		Score:       opinionScore,
		Description: describeOpinion(opinionScore),
		Evidence:    take(sigs.OpinionMarkers, 3),
	}

	// --- Emotion dimension ---
	emotionScore := clamp(float64(len(sigs.EmotionMarkers)) / float64(max1(sigs.TotalSentences)) * 6)
	profile.Dimensions["emotion"] = Dimension{
		Name:        "Emotional expression",
		Score:       emotionScore,
		Description: describeEmotion(emotionScore),
		Evidence:    take(sigs.EmotionMarkers, 3),
	}

	// --- Vocab style ---
	profile.VocabStyle = VocabStyle{
		Formality:     formalScore,
		Verbosity:     clamp(sigs.AvgSentenceLen / 30),
		CommonWords:   topN(sigs.VocabFrequency, 10),
		AvoidWords:    []string{},
		SentenceStyle: sentenceStyle(sigs.AvgSentenceLen),
	}

	// --- Interests from topic clusters ---
	profile.Interests = topTopics(sigs.TopicClusters, 5)

	// --- Core beliefs from opinion markers ---
	profile.CoreBeliefs = extractBeliefs(sigs.OpinionMarkers, 5)

	// --- Sample phrases ---
	profile.SamplePhrases = take(sigs.AllSentences, 5)

	// --- Quirks ---
	profile.Quirks = detectQuirks(sigs)

	// --- Behavioral data (silent signals → enrich interests + quirks) ---
	// Deduplicate and merge locations/foods into profile
	if len(sigs.Locations) > 0 {
		profile.Locations = dedup(sigs.Locations, 5)
	}
	if len(sigs.Foods) > 0 {
		profile.Foods = dedup(sigs.Foods, 5)
	}
	// Merge behavioral interests into topic-based interests
	for _, interest := range dedup(sigs.Interests, 10) {
		if !contains(profile.Interests, interest) {
			profile.Interests = append(profile.Interests, interest)
		}
	}

	// --- Summary ---
	profile.Summary = buildSummary(name, profile)

	return profile
}

func describeHumor(score float64) string {
	switch {
	case score > 0.7:
		return "Frequently uses humor, wit, and levity in communication"
	case score > 0.4:
		return "Occasionally incorporates humor and lighthearted remarks"
	case score > 0.1:
		return "Rarely uses humor; tends toward straightforward communication"
	default:
		return "Very little evidence of humor in available sources"
	}
}

func describeTone(formalScore float64) string {
	switch {
	case formalScore > 0.6:
		return "Highly formal and structured in communication"
	case formalScore > 0.3:
		return "Balanced tone — professional yet approachable"
	default:
		return "Casual and conversational in style"
	}
}

func describeOpinion(score float64) string {
	switch {
	case score > 0.7:
		return "Strongly opinionated, frequently shares views and beliefs"
	case score > 0.4:
		return "Moderately opinionated; expresses views when relevant"
	default:
		return "Tends to be measured in expressing opinions"
	}
}

func describeEmotion(score float64) string {
	switch {
	case score > 0.7:
		return "Highly emotionally expressive; wears feelings openly"
	case score > 0.4:
		return "Moderately expressive; emotional context surfaces regularly"
	default:
		return "Emotionally reserved; keeps feelings beneath the surface"
	}
}

func sentenceStyle(avgLen float64) string {
	switch {
	case avgLen < 10:
		return "short"
	case avgLen < 20:
		return "medium"
	case avgLen < 30:
		return "long"
	default:
		return "very long"
	}
}

func topN(freq map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	var sorted []kv
	for k, v := range freq {
		sorted = append(sorted, kv{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })

	var result []string
	for i, kv := range sorted {
		if i >= n {
			break
		}
		result = append(result, kv.k)
	}
	return result
}

func topTopics(clusters map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	var sorted []kv
	for k, v := range clusters {
		if v > 0 {
			sorted = append(sorted, kv{k, v})
		}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].v > sorted[j].v })

	var result []string
	for i, kv := range sorted {
		if i >= n {
			break
		}
		result = append(result, kv.k)
	}
	return result
}

func extractBeliefs(opinions []string, n int) []string {
	var beliefs []string
	for _, s := range opinions {
		lower := strings.ToLower(s)
		if strings.Contains(lower, "i believe") ||
			strings.Contains(lower, "i think") ||
			strings.Contains(lower, "we must") ||
			strings.Contains(lower, "you should") {
			beliefs = append(beliefs, s)
			if len(beliefs) >= n {
				break
			}
		}
	}
	return beliefs
}

func detectQuirks(sigs *Signals) []string {
	var quirks []string

	if sigs.AvgSentenceLen < 8 {
		quirks = append(quirks, "Prefers very short, punchy sentences")
	} else if sigs.AvgSentenceLen > 30 {
		quirks = append(quirks, "Tends toward long, complex sentence structures")
	}

	if len(sigs.HumorMarkers) > len(sigs.OpinionMarkers) {
		quirks = append(quirks, "Uses humor more than direct opinion statements")
	}

	if len(sigs.EmotionMarkers) > sigs.TotalSentences/3 {
		quirks = append(quirks, "Emotional language is pervasive throughout")
	}

	if len(quirks) == 0 {
		quirks = append(quirks, "Communication style is balanced and adaptive")
	}

	return quirks
}

func buildSummary(name string, p *PersonaProfile) string {
	humor := p.Dimensions["humor"]
	tone := p.Dimensions["tone"]
	opinions := p.Dimensions["opinions"]

	parts := []string{fmt.Sprintf("%s communicates in a", name)}

	if tone.Score > 0.6 {
		parts = append(parts, "casual,")
	} else {
		parts = append(parts, "measured,")
	}

	if humor.Score > 0.5 {
		parts = append(parts, "humorous")
	} else {
		parts = append(parts, "direct")
	}

	parts = append(parts, "style.")

	if opinions.Score > 0.5 {
		parts = append(parts, fmt.Sprintf("%s is highly opinionated and not afraid to share views.", name))
	}

	if len(p.Interests) > 0 {
		parts = append(parts, fmt.Sprintf("Key interests include %s.", strings.Join(p.Interests[:min2(3, len(p.Interests))], ", ")))
	}

	return strings.Join(parts, " ")
}

func dedup(s []string, max int) []string {
	seen := map[string]bool{}
	var result []string
	for _, v := range s {
		if v != "" && !seen[v] {
			seen[v] = true
			result = append(result, v)
			if len(result) >= max {
				break
			}
		}
	}
	return result
}

func contains(s []string, v string) bool {
	for _, item := range s {
		if item == v {
			return true
		}
	}
	return false
}

func take(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}
