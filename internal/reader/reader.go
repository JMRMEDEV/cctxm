package reader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const DefaultThreshold = 5120 // 5KB

// AlwaysFullExtensions are read in full regardless of size.
var AlwaysFullExtensions = map[string]bool{
	".md": true, ".yaml": true, ".yml": true, ".toml": true,
	".json": true, ".env.example": true,
}

// Read returns file content, filtered if large and not exempt.
func Read(path string, keywords []string, threshold int, forceFullRead bool, searchTerms string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("file not found: %s", path)
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	content := string(data)

	if forceFullRead {
		return withHeader(path, content, len(data), false), nil
	}

	ext := filepath.Ext(path)
	if AlwaysFullExtensions[ext] {
		return withHeader(path, content, len(data), false), nil
	}

	if threshold <= 0 {
		threshold = DefaultThreshold
	}

	if len(data) <= threshold {
		return withHeader(path, content, len(data), false), nil
	}

	// Large file — extract relevant sections
	terms := keywords
	if searchTerms != "" {
		terms = append(terms, strings.Fields(searchTerms)...)
	}

	if len(terms) == 0 {
		// No keywords — return head + tail
		lines := strings.Split(content, "\n")
		head := take(lines, 50)
		tail := takeLast(lines, 20)
		out := strings.Join(head, "\n") +
			fmt.Sprintf("\n\n... [%d lines omitted] ...\n\n", len(lines)-70) +
			strings.Join(tail, "\n")
		return withHeader(path, out, len(data), true), nil
	}

	extracted := extractSections(content, terms)
	return withHeader(path, extracted, len(data), true), nil
}

func extractSections(content string, terms []string) string {
	lines := strings.Split(content, "\n")
	contextRadius := 10
	marked := make([]bool, len(lines))

	for i, line := range lines {
		lower := strings.ToLower(line)
		for _, term := range terms {
			if strings.Contains(lower, strings.ToLower(term)) {
				start := i - contextRadius
				if start < 0 {
					start = 0
				}
				end := i + contextRadius + 1
				if end > len(lines) {
					end = len(lines)
				}
				for j := start; j < end; j++ {
					marked[j] = true
				}
				break
			}
		}
	}

	var sections []string
	inBlock := false
	for i, line := range lines {
		if marked[i] {
			if !inBlock && len(sections) > 0 {
				sections = append(sections, "\n... [lines omitted] ...\n")
			}
			sections = append(sections, line)
			inBlock = true
		} else {
			inBlock = false
		}
	}

	if len(sections) == 0 {
		head := take(lines, 30)
		return strings.Join(head, "\n") + fmt.Sprintf("\n\n... [no keyword matches in %d lines] ...", len(lines))
	}
	return strings.Join(sections, "\n")
}

func withHeader(path, content string, size int, filtered bool) string {
	status := "full"
	if filtered {
		status = "filtered"
	}
	return fmt.Sprintf("[cctxm] Reading %s (%d bytes, %s)\n\n%s", path, size, status, content)
}

func take(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	return lines[:n]
}

func takeLast(lines []string, n int) []string {
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}
