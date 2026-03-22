package pipeline

import (
	"fmt"
	"sort"
	"strings"
)

// Distiller converts raw signals into a structured PersonaProfile
// using statistical analysis. The LLM layer in runner.go enriches
// the output further when Groq is available.
type Distiller struct{}

func NewDistiller() *Distiller { return &Distiller{} }

func (d *Distiller) Distill(name string, sigs *Signals) *PersonaProfile {
	profile := &PersonaProfile{
		Name:       name,
		Dimensions: make(map[string]Dimension),
	}

	stats := sigs.Stats
	total := max1(sigs.TotalSentences)

	// ── HUMOR ────────────────────────────────────────────────────
	// Signals: explicit markers, exclamations, emoji, short punchy sentences, caps
	humorScore := blended(
		scoreRatio(len(sigs.HumorEvidence), total, 0.15),  // explicit humor markers
		clamp(float64(stats.ExclamCount)/float64(total)*2), // exclamations
		clamp(float64(stats.EmojiCount)/float64(total)*3),  // emoji
		clamp(stats.CapsRatio*5),                           // CAPS intensity
	)
	profile.Dimensions["humor"] = Dimension{
		Name:        "Humor",
		Score:       humorScore,
		Description: describeHumor(humorScore),
		Evidence:    take(sigs.HumorEvidence, 3),
	}

	// ── TONE (casual ↔ formal) ────────────────────────────────────
	// High score = casual. Signals: short sentences, questions, first-person, emoji, contractions
	// Formal signals: long sentences, hedge words, low first-person
	casualScore := blended(
		clamp(1 - float64(stats.LongSentences)/float64(total)*3), // fewer long = more casual
		clamp(float64(stats.FirstPersonCount)/float64(total)*0.5),
		clamp(float64(stats.ShortSentences)/float64(total)*2),
		clamp(float64(stats.EmojiCount)/float64(total)*4),
		clamp(1-float64(stats.HedgeCount)/float64(total)*3),
	)
	profile.Dimensions["tone"] = Dimension{
		Name:        "Tone",
		Score:       casualScore,
		Description: describeTone(casualScore),
		Evidence:    []string{},
	}

	// ── OPINIONATED ───────────────────────────────────────────────
	// Signals: explicit opinion markers, direct words, low hedging, first-person
	opinionScore := blended(
		scoreRatio(len(sigs.OpinionEvidence), total, 0.2),
		clamp(float64(stats.DirectCount)/float64(total)*2),
		clamp(float64(stats.FirstPersonCount)/float64(total)*0.3),
		clamp(1-float64(stats.HedgeCount)/float64(total)*2),
	)
	profile.Dimensions["opinions"] = Dimension{
		Name:        "Opinionated",
		Score:       opinionScore,
		Description: describeOpinion(opinionScore),
		Evidence:    take(sigs.OpinionEvidence, 3),
	}

	// ── EMOTION ──────────────────────────────────────────────────
	// Signals: explicit emotion markers, exclamations, emoji, caps, sentence variance
	emotionScore := blended(
		scoreRatio(len(sigs.EmotionEvidence), total, 0.2),
		clamp(float64(stats.ExclamCount)/float64(total)*2.5),
		clamp(float64(stats.EmojiCount)/float64(total)*3),
		clamp(stats.CapsRatio*6),
		clamp(stats.LenVariance/15), // high variance = emotional rhythm
	)
	profile.Dimensions["emotion"] = Dimension{
		Name:        "Emotional expression",
		Score:       emotionScore,
		Description: describeEmotion(emotionScore),
		Evidence:    take(sigs.EmotionEvidence, 3),
	}

	// ── DIRECTNESS ────────────────────────────────────────────────
	// Signals: direct words, low hedging, short sentences, low negation
	directnessScore := blended(
		clamp(float64(stats.DirectCount)/float64(total)*2),
		clamp(1-float64(stats.HedgeCount)/float64(total)*3),
		clamp(float64(stats.ShortSentences)/float64(total)*1.5),
		clamp(1-float64(stats.NegationCount)/float64(total)*2),
	)
	profile.Dimensions["directness"] = Dimension{
		Name:        "Directness",
		Score:       directnessScore,
		Description: describeDirectness(directnessScore),
		Evidence:    []string{},
	}

	// ── VOCABULARY RICHNESS ───────────────────────────────────────
	// Signals: type-token ratio, avg word length, topic variety
	topicVariety := clamp(float64(len(sigs.TopicClusters)) / 6.0)
	richScore := blended(
		stats.TypeTokenRatio,
		clamp((stats.AvgWordLen-3)/4), // 3-7 char avg = richer
		topicVariety,
	)
	profile.Dimensions["vocabulary"] = Dimension{
		Name:        "Vocabulary richness",
		Score:       richScore,
		Description: describeVocabulary(richScore),
		Evidence:    []string{},
	}

	// ── ENGAGEMENT ────────────────────────────────────────────────
	// Signals: questions, exclamations, second-person ("you"), short responses
	youCount := strings.Count(strings.ToLower(sigs.AllText), " you ") +
		strings.Count(strings.ToLower(sigs.AllText), " your ")
	engagementScore := blended(
		clamp(float64(stats.QuestionCount)/float64(total)*3),
		clamp(float64(stats.ExclamCount)/float64(total)*1.5),
		clamp(float64(youCount)/float64(max1(sigs.TotalWords))*10),
	)
	profile.Dimensions["engagement"] = Dimension{
		Name:        "Engagement",
		Score:       engagementScore,
		Description: describeEngagement(engagementScore),
		Evidence:    []string{},
	}

	// ── VOCAB STYLE ───────────────────────────────────────────────
	formalScore := 1 - casualScore
	profile.VocabStyle = VocabStyle{
		Formality:     formalScore,
		Verbosity:     clamp(stats.AvgLen / 25),
		CommonWords:   topN(sigs.VocabFrequency, 10),
		AvoidWords:    []string{},
		SentenceStyle: sentenceStyle(stats.AvgLen),
	}

	// ── INTERESTS ────────────────────────────────────────────────
	profile.Interests = topTopics(sigs.TopicClusters, 5)

	// ── BELIEFS ──────────────────────────────────────────────────
	profile.CoreBeliefs = extractBeliefs(sigs.OpinionEvidence, 5)

	// ── SAMPLE PHRASES ───────────────────────────────────────────
	profile.SamplePhrases = take(sigs.AllSentences, 5)

	// ── QUIRKS ───────────────────────────────────────────────────
	profile.Quirks = detectQuirks(sigs)

	// ── BEHAVIORAL ───────────────────────────────────────────────
	profile.Locations = dedup(sigs.Locations, 5)
	profile.Foods = dedup(sigs.Foods, 5)
	for _, interest := range dedup(sigs.Interests, 10) {
		if !contains(profile.Interests, interest) {
			profile.Interests = append(profile.Interests, interest)
		}
	}

	// ── SUMMARY ──────────────────────────────────────────────────
	profile.Summary = buildSummary(name, profile)

	return profile
}

// blended averages multiple scores, ignoring zeros (no data shouldn't drag score down).
func blended(scores ...float64) float64 {
	sum := 0.0
	count := 0
	for _, s := range scores {
		sum += s
		count++
	}
	if count == 0 {
		return 0
	}
	return clamp(sum / float64(count))
}

// scoreRatio converts a hit count to a score, using a target ratio as the midpoint.
// targetRatio = the ratio of hits/sentences that maps to ~0.7 score
func scoreRatio(hits, sentences int, targetRatio float64) float64 {
	if sentences == 0 {
		return 0
	}
	ratio := float64(hits) / float64(sentences)
	return clamp(ratio / targetRatio * 0.7)
}

func describeHumor(score float64) string {
	switch {
	case score > 0.7:
		return "Frequently playful and humorous in communication"
	case score > 0.4:
		return "Occasionally uses humor and lightness"
	case score > 0.15:
		return "Some playful signals detected"
	default:
		return "Tone tends toward the serious — little humor detected"
	}
}

func describeTone(casualScore float64) string {
	switch {
	case casualScore > 0.75:
		return "Very casual and conversational"
	case casualScore > 0.5:
		return "Relaxed and approachable tone"
	case casualScore > 0.3:
		return "Balanced — professional yet accessible"
	default:
		return "Formal and structured in expression"
	}
}

func describeOpinion(score float64) string {
	switch {
	case score > 0.7:
		return "Highly opinionated — shares views openly and directly"
	case score > 0.4:
		return "Moderately opinionated — expresses views when relevant"
	case score > 0.15:
		return "Shares opinions occasionally but tends to be measured"
	default:
		return "Mostly descriptive — few strong opinion signals"
	}
}

func describeEmotion(score float64) string {
	switch {
	case score > 0.7:
		return "Highly emotionally expressive"
	case score > 0.4:
		return "Moderately expressive — emotions surface regularly"
	case score > 0.15:
		return "Emotionally present but measured"
	default:
		return "Emotionally reserved in expression"
	}
}

func describeDirectness(score float64) string {
	switch {
	case score > 0.7:
		return "Very direct — gets to the point, no hedging"
	case score > 0.4:
		return "Generally direct with occasional qualification"
	case score > 0.15:
		return "Balanced — direct when confident, hedges when uncertain"
	default:
		return "Tends to hedge and qualify statements"
	}
}

func describeVocabulary(score float64) string {
	switch {
	case score > 0.7:
		return "Rich and varied vocabulary"
	case score > 0.4:
		return "Moderate vocabulary variety"
	default:
		return "Tends toward simple, consistent word choice"
	}
}

func describeEngagement(score float64) string {
	switch {
	case score > 0.6:
		return "Highly engaging — asks questions, invites response"
	case score > 0.3:
		return "Moderately engaging communication style"
	default:
		return "More declarative than interactive"
	}
}

func sentenceStyle(avgLen float64) string {
	switch {
	case avgLen < 8:
		return "short"
	case avgLen < 15:
		return "medium-short"
	case avgLen < 22:
		return "medium"
	case avgLen < 30:
		return "long"
	default:
		return "very long"
	}
}

func topTopics(clusters map[string]int, n int) []string {
	type kv struct{ k string; v int }
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
	stats := sigs.Stats

	if stats.AvgLen < 7 {
		quirks = append(quirks, "Prefers very short, punchy sentences")
	} else if stats.AvgLen > 30 {
		quirks = append(quirks, "Tends toward long, complex sentence structures")
	}

	if stats.LenVariance > 10 {
		quirks = append(quirks, "Sentence length varies dramatically — expressive rhythm")
	}

	if stats.QuestionCount > sigs.TotalSentences/5 {
		quirks = append(quirks, "Frequently uses questions — invites dialogue")
	}

	if stats.EmojiCount > sigs.TotalSentences/3 {
		quirks = append(quirks, "Heavy emoji user")
	}

	if stats.CapsRatio > 0.05 {
		quirks = append(quirks, "Uses CAPS for emphasis")
	}

	if stats.ArabicCharRatio > 0.3 {
		quirks = append(quirks, "Communicates in Arabic")
	}

	if len(quirks) == 0 {
		quirks = append(quirks, "Balanced and consistent communication style")
	}

	return quirks
}

func buildSummary(name string, p *PersonaProfile) string {
	tone := p.Dimensions["tone"]
	humor := p.Dimensions["humor"]
	opinions := p.Dimensions["opinions"]
	directness := p.Dimensions["directness"]

	var parts []string

	toneDesc := "measured"
	if tone.Score > 0.65 {
		toneDesc = "casual"
	} else if tone.Score < 0.35 {
		toneDesc = "formal"
	}

	humorDesc := ""
	if humor.Score > 0.5 {
		humorDesc = " and humorous"
	}

	parts = append(parts, fmt.Sprintf("%s communicates in a %s%s style.", name, toneDesc, humorDesc))

	if opinions.Score > 0.5 {
		parts = append(parts, fmt.Sprintf("%s is opinionated and not afraid to share views directly.", name))
	}

	if directness.Score > 0.6 {
		parts = append(parts, "Gets to the point without hedging.")
	} else if directness.Score < 0.3 {
		parts = append(parts, "Tends to qualify and soften statements.")
	}

	if len(p.Interests) > 0 {
		n := min2(3, len(p.Interests))
		parts = append(parts, fmt.Sprintf("Key interests include %s.", strings.Join(p.Interests[:n], ", ")))
	}

	return strings.Join(parts, " ")
}

// updateDescription re-generates a description for a dimension after LLM score override.
func updateDescription(dimension string, score float64) string {
	switch dimension {
	case "humor":
		return describeHumor(score)
	case "tone":
		return describeTone(score)
	case "opinions":
		return describeOpinion(score)
	case "emotion":
		return describeEmotion(score)
	case "directness":
		return describeDirectness(score)
	case "vocabulary":
		return describeVocabulary(score)
	case "engagement":
		return describeEngagement(score)
	}
	return ""
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
