package preprocessors

import (
	"encoding/json"
	"regexp"
	"strings"
)

// TwitterPreprocessor parses Twitter/X data archive exports.
//
// Twitter export format (tweet.js):
//
//	window.YTD.tweets.part0 = [
//	  { "tweet": { "full_text": "Hello world", "created_at": "...", "entities": {...} } }
//	]
//
// Also handles plain JSON array of tweets.
type TwitterPreprocessor struct {
	jsVarPattern *regexp.Regexp
}

func NewTwitterPreprocessor() *TwitterPreprocessor {
	return &TwitterPreprocessor{
		jsVarPattern: regexp.MustCompile(`(?s)window\.YTD\.\w+\.part\d+\s*=\s*(\[.+\])`),
	}
}

func (p *TwitterPreprocessor) Name() string { return "twitter" }

type twitterTweetWrapper struct {
	Tweet twitterTweet `json:"tweet"`
}

type twitterTweet struct {
	FullText  string `json:"full_text"`
	CreatedAt string `json:"created_at"`
	Entities  struct {
		Hashtags []struct {
			Text string `json:"text"`
		} `json:"hashtags"`
		URLs []struct {
			ExpandedURL string `json:"expanded_url"`
		} `json:"urls"`
	} `json:"entities"`
	RetweetedStatus *struct {
		FullText string `json:"full_text"`
	} `json:"retweeted_status"`
}

func (p *TwitterPreprocessor) CanHandle(sourceType, content string) bool {
	if sourceType == "pdf" {
		return false
	}
	trimmed := strings.TrimSpace(content)
	// Match Twitter JS export
	if p.jsVarPattern.MatchString(trimmed) {
		return true
	}
	// Match JSON array with tweet objects
	if strings.HasPrefix(trimmed, "[") {
		var tweets []twitterTweetWrapper
		if err := json.Unmarshal([]byte(trimmed), &tweets); err == nil && len(tweets) > 0 {
			return tweets[0].Tweet.FullText != ""
		}
	}
	return false
}

func (p *TwitterPreprocessor) Process(sourceType, content, speakerName string) *PreprocessResult {
	result := &PreprocessResult{Platform: "twitter", TargetFound: true}

	jsonContent := content
	// Strip JS variable wrapper
	if matches := p.jsVarPattern.FindStringSubmatch(content); len(matches) == 2 {
		jsonContent = matches[1]
	}

	var tweets []twitterTweetWrapper
	if err := json.Unmarshal([]byte(jsonContent), &tweets); err != nil {
		return result
	}

	topicCount := map[string]int{}

	for _, wrapper := range tweets {
		tweet := wrapper.Tweet

		// Skip retweets — not their words
		if tweet.RetweetedStatus != nil {
			continue
		}

		text := strings.TrimSpace(tweet.FullText)
		if text == "" {
			continue
		}

		// Skip pure @mentions or replies with no original content
		if strings.HasPrefix(text, "@") && len(strings.Fields(text)) < 4 {
			result.BehavioralSignals = append(result.BehavioralSignals, BehavioralSignal{
				Category: "social",
				Value:    "frequent replier",
				Count:    1,
			})
			continue
		}

		// Hashtags → interest signals (behavioral)
		for _, ht := range tweet.Entities.Hashtags {
			topicCount[strings.ToLower(ht.Text)]++
		}

		// Clean tweet text (remove URLs, @mentions for analysis)
		cleaned := cleanTweetText(text)
		if len(cleaned) > 10 {
			result.Utterances = append(result.Utterances, Utterance{
				Text:    cleaned,
				Source:  "twitter",
				IsQuote: true,
			})
		}
	}

	// Convert top hashtags to behavioral signals
	for topic, count := range topicCount {
		result.BehavioralSignals = append(result.BehavioralSignals, BehavioralSignal{
			Category: "interest",
			Value:    "#" + topic,
			Count:    count,
		})
	}

	return result
}

func cleanTweetText(text string) string {
	// Remove URLs
	urlPattern := regexp.MustCompile(`https?://\S+`)
	text = urlPattern.ReplaceAllString(text, "")
	// Remove @mentions at start of replies but keep mid-sentence ones
	text = strings.TrimSpace(text)
	return text
}
