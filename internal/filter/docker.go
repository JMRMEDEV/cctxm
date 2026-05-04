package filter

import (
	"encoding/json"
	"strings"
)

type DockerFilter struct{}

func (f *DockerFilter) Apply(output string, mode Mode, keywords []string) *Result {
	if mode == ModeVerbose {
		lines := strings.Split(output, "\n")
		limited := LimitLines(lines, mode)
		return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "Docker logs (verbose)"}
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) == 0 {
		return &Result{Output: "(empty)", Lines: 0, Summary: "Docker logs: empty"}
	}

	// Detect JSON logs
	if isJSON(lines[0]) {
		return f.filterJSON(lines, mode, keywords)
	}
	return f.filterPlain(lines, mode, keywords)
}

func (f *DockerFilter) filterJSON(lines []string, mode Mode, keywords []string) *Result {
	var matched []string
	for _, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		level, _ := entry["level"].(string)
		level = strings.ToLower(level)

		isErr := level == "error" || level == "fatal" || level == "panic" || level == "warn" || level == "warning"
		isKw := MatchesKeyword(line, keywords)

		if isErr || isKw || (mode == ModeNormal && level == "info" && isKw) {
			matched = append(matched, line)
		}
	}

	limited := LimitLines(matched, mode)
	return &Result{
		Output:  strings.Join(limited, "\n"),
		Lines:   len(limited),
		Summary: "Docker logs (JSON)",
	}
}

func (f *DockerFilter) filterPlain(lines []string, mode Mode, keywords []string) *Result {
	extracted := ExtractWithContext(lines, mode, func(line string) bool {
		return IsErrorLine(line) || IsStackTraceLine(line) || MatchesKeyword(line, keywords)
	})

	limited := LimitLines(extracted, mode)
	return &Result{
		Output:  strings.Join(limited, "\n"),
		Lines:   len(limited),
		Summary: "Docker logs (plain text)",
	}
}

func isJSON(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")
}
