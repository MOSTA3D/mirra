package preprocessors

import (
	"encoding/json"
	"strings"
)

// InstagramPreprocessor parses Instagram data export JSON.
//
// Instagram exports multiple JSON files. We support the most common:
//   - posts_1.json: { "media": [{ "title": "caption", "creation_timestamp": ... }] }
//   - comments.json: { "comments_media_comments": [{ "string_map_data": { "Comment": { "value": "..." } } }] }
//   - liked_posts.json: signals only — no text content
//
// Instagram posts often don't have statement-like captions, so we rely heavily
// on behavioral signals (interests inferred from hashtags, location tags).
type InstagramPreprocessor struct{}

func NewInstagramPreprocessor() *InstagramPreprocessor { return &InstagramPreprocessor{} }

func (p *InstagramPreprocessor) Name() string { return "instagram" }

type instagramExport struct {
	// Posts format
	Media []instagramMedia `json:"media"`
	// Comments format
	CommentsMediaComments []instagramComment `json:"comments_media_comments"`
	// Liked posts (behavioral only)
	LikesMediaLikes []interface{} `json:"likes_media_likes"`
}

type instagramMedia struct {
	Title               string `json:"title"`
	CreationTimestamp   int64  `json:"creation_timestamp"`
}

type instagramComment struct {
	StringMapData map[string]struct {
		Value string `json:"value"`
	} `json:"string_map_data"`
}

func (p *InstagramPreprocessor) CanHandle(sourceType, content string) bool {
	if sourceType == "pdf" {
		return false
	}
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "{") {
		return false
	}
	var export instagramExport
	if err := json.Unmarshal([]byte(trimmed), &export); err != nil {
		return false
	}
	return len(export.Media) > 0 ||
		len(export.CommentsMediaComments) > 0 ||
		len(export.LikesMediaLikes) > 0
}

func (p *InstagramPreprocessor) Process(sourceType, content, speakerName string) *PreprocessResult {
	result := &PreprocessResult{Platform: "instagram", TargetFound: true}

	var export instagramExport
	if err := json.Unmarshal([]byte(content), &export); err != nil {
		return result
	}

	// Process posts — extract captions and hashtag signals
	for _, media := range export.Media {
		caption := strings.TrimSpace(media.Title)
		if caption == "" {
			continue
		}

		// Extract hashtags as behavioral signals
		words := strings.Fields(caption)
		for _, word := range words {
			if strings.HasPrefix(word, "#") {
				result.BehavioralSignals = append(result.BehavioralSignals, BehavioralSignal{
					Category: "interest",
					Value:    strings.ToLower(word),
					Count:    1,
				})
			}
			if strings.HasPrefix(word, "@") {
				result.BehavioralSignals = append(result.BehavioralSignals, BehavioralSignal{
					Category: "social",
					Value:    "tagged " + word,
					Count:    1,
				})
			}
		}

		// Clean caption (remove hashtags/mentions for linguistic analysis)
		cleaned := cleanCaption(caption)
		if len(cleaned) > 15 && !isPureHashtags(caption) {
			result.Utterances = append(result.Utterances, Utterance{
				Text:    cleaned,
				Source:  "instagram",
				IsQuote: true,
			})
		}
	}

	// Process comments — direct statements
	for _, comment := range export.CommentsMediaComments {
		if commentData, ok := comment.StringMapData["Comment"]; ok {
			text := strings.TrimSpace(commentData.Value)
			if len(text) > 5 {
				result.Utterances = append(result.Utterances, Utterance{
					Text:    text,
					Source:  "instagram_comment",
					IsQuote: true,
				})
			}
		}
	}

	// Liked posts = behavioral signal only (interest indicator)
	if len(export.LikesMediaLikes) > 0 {
		result.BehavioralSignals = append(result.BehavioralSignals, BehavioralSignal{
			Category: "activity",
			Value:    "active liker",
			Count:    len(export.LikesMediaLikes),
		})
	}

	return result
}

func cleanCaption(caption string) string {
	words := strings.Fields(caption)
	var kept []string
	for _, w := range words {
		if !strings.HasPrefix(w, "#") && !strings.HasPrefix(w, "@") {
			kept = append(kept, w)
		}
	}
	return strings.Join(kept, " ")
}

func isPureHashtags(caption string) bool {
	words := strings.Fields(caption)
	for _, w := range words {
		if !strings.HasPrefix(w, "#") && !strings.HasPrefix(w, "@") {
			return false
		}
	}
	return true
}
