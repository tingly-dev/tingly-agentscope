package mcp

import (
	"regexp"
	"strings"
)

// KeywordExtractor extracts keywords from user input for server matching
type KeywordExtractor struct {
	stopWords map[string]bool
}

// NewKeywordExtractor creates a new keyword extractor with default stop words
func NewKeywordExtractor() *KeywordExtractor {
	return &KeywordExtractor{
		stopWords: map[string]bool{
			"the": true, "a": true, "an": true, "and": true, "or": true,
			"but": true, "in": true, "on": true, "at": true, "to": true,
			"for": true, "of": true, "with": true, "by": true, "from": true,
			"up": true, "about": true, "into": true, "through": true, "during": true,
			"before": true, "after": true, "above": true, "below": true, "between": true,
			"among": true, "is": true, "are": true, "was": true, "were": true,
			"be": true, "been": true, "being": true, "have": true, "has": true,
			"had": true, "do": true, "does": true, "did": true, "will": true,
			"would": true, "could": true, "should": true, "may": true, "might": true,
			"can": true, "need": true, "must": true, "shall": true, "get": true,
			"gets": true, "got": true, "use": true, "using": true, "used": true,
			"please": true, "help": true, "me": true, "my": true, "mine": true,
			"you": true, "your": true, "yours": true, "we": true, "our": true,
			"ours": true, "they": true, "their": true, "them": true, "it": true,
			"its": true, "this": true, "that": true, "these": true, "those": true,
			"i": true, "am": true, "he": true, "she": true, "his": true, "her": true,
			"how": true, "what": true, "when": true, "where": true, "why": true,
			"which": true, "who": true, "whom": true, "whose": true,
		},
	}
}

// Extract extracts keywords from input text
func (e *KeywordExtractor) Extract(input string) []string {
	// Normalize input
	input = strings.ToLower(input)
	input = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(input, " ")

	// Tokenize
	words := strings.Fields(input)

	// Filter stop words and short words
	var keywords []string
	seen := make(map[string]bool)
	for _, word := range words {
		if len(word) < 2 || e.stopWords[word] {
			continue
		}
		if !seen[word] {
			seen[word] = true
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// MatchScore calculates a match score between input and server keywords
// Returns a score between 0 and 1, where 1 is a perfect match
func (e *KeywordExtractor) MatchScore(input string, serverKeywords []string) float64 {
	if len(serverKeywords) == 0 {
		return 0
	}

	inputKeywords := e.Extract(input)
	if len(inputKeywords) == 0 {
		return 0
	}

	// Convert server keywords to a set
	serverSet := make(map[string]bool)
	for _, kw := range serverKeywords {
		serverSet[strings.ToLower(kw)] = true
	}

	// Count matches
	matches := 0
	for _, kw := range inputKeywords {
		if serverSet[kw] {
			matches++
		}
	}

	// Calculate score based on matches relative to input keywords
	// This favors servers that match a higher percentage of input keywords
	score := float64(matches) / float64(len(inputKeywords))

	return score
}

// ServerMatch represents a matching server with its score
type ServerMatch struct {
	ServerName string
	Score      float64
	Reason     string
}

// FindMatches finds servers that match the input above a threshold
func (e *KeywordExtractor) FindMatches(input string, servers map[string][]string, threshold float64) []ServerMatch {
	var matches []ServerMatch

	for serverName, keywords := range servers {
		score := e.MatchScore(input, keywords)
		if score >= threshold {
			matches = append(matches, ServerMatch{
				ServerName: serverName,
				Score:      score,
				Reason:     "keyword match",
			})
		}
	}

	// Sort by score descending
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].Score > matches[i].Score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	return matches
}
