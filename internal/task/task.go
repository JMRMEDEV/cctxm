package task

import (
	"strings"
)

// Debug keywords → strict mode
var debugKeywords = []string{
	"error", "fix", "bug", "issue", "debug", "fail", "crash",
	"broken", "wrong", "not working", "exception", "timeout",
	"500", "404",
}

// Explore keywords → normal mode
var exploreKeywords = []string{
	"how", "understand", "explain", "explore", "investigate",
	"check", "review", "what does", "look at",
}

// Common stop words to exclude from task keywords
var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "in": true, "on": true,
	"at": true, "to": true, "for": true, "of": true, "is": true,
	"it": true, "and": true, "or": true, "not": true, "with": true,
	"from": true, "that": true, "this": true, "be": true, "are": true,
	"was": true, "were": true, "been": true, "being": true, "have": true,
	"has": true, "had": true, "do": true, "does": true, "did": true,
	"will": true, "would": true, "could": true, "should": true,
	"may": true, "might": true, "can": true, "i": true, "we": true,
	"my": true, "our": true, "me": true, "us": true,
}

// DetectMode analyzes a task description and returns the filter mode.
func DetectMode(description string) string {
	lower := strings.ToLower(description)
	for _, kw := range debugKeywords {
		if strings.Contains(lower, kw) {
			return "strict"
		}
	}
	for _, kw := range exploreKeywords {
		if strings.Contains(lower, kw) {
			return "normal"
		}
	}
	return "normal"
}

// ExtractKeywords pulls meaningful words from a task description.
func ExtractKeywords(description string) []string {
	words := strings.Fields(strings.ToLower(description))
	seen := map[string]bool{}
	var keywords []string

	for _, w := range words {
		// Strip punctuation
		w = strings.Trim(w, ".,;:!?\"'()[]{}#")
		if len(w) < 2 || stopWords[w] || seen[w] {
			continue
		}
		// Skip mode-detection keywords (they're not useful for content filtering)
		isMode := false
		for _, dk := range debugKeywords {
			if w == dk || strings.HasPrefix(w, dk) {
				isMode = true
				break
			}
		}
		if isMode {
			continue
		}
		seen[w] = true
		keywords = append(keywords, w)
	}
	return keywords
}
