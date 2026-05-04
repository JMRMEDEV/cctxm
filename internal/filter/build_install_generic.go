package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// --- Build Filter ---

type BuildFilter struct{}

func (f *BuildFilter) Apply(output string, mode Mode, keywords []string) *Result {
	if mode == ModeVerbose {
		lines := strings.Split(output, "\n")
		limited := LimitLines(lines, mode)
		return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "Build (verbose)"}
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// Check for errors
	hasError := false
	for _, line := range lines {
		if IsErrorLine(line) {
			hasError = true
			break
		}
	}

	if !hasError {
		return &Result{Output: "Build succeeded.", Lines: 1, Summary: "Build succeeded"}
	}

	extracted := ExtractWithContext(lines, mode, func(line string) bool {
		return IsErrorLine(line) || IsStackTraceLine(line) || isTSError(line) || isGoError(line)
	})

	limited := LimitLines(extracted, mode)
	return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "Build: errors detected"}
}

var tsErrorRe = regexp.MustCompile(`TS\d{4}:`)
var goErrorRe = regexp.MustCompile(`\.go:\d+:\d+:`)

func isTSError(line string) bool  { return tsErrorRe.MatchString(line) }
func isGoError(line string) bool   { return goErrorRe.MatchString(line) }

// --- Install Filter ---

type InstallFilter struct{}

var npmAddedRe = regexp.MustCompile(`added (\d+) packages?`)
var pipInstalledRe = regexp.MustCompile(`Successfully installed`)

func (f *InstallFilter) Apply(output string, mode Mode, keywords []string) *Result {
	if mode == ModeVerbose {
		lines := strings.Split(output, "\n")
		limited := LimitLines(lines, mode)
		return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "Install (verbose)"}
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	hasError := false
	for _, line := range lines {
		if IsErrorLine(line) {
			hasError = true
			break
		}
	}

	if !hasError {
		summary := "Install completed."
		for _, line := range lines {
			if m := npmAddedRe.FindString(line); m != "" {
				summary = fmt.Sprintf("Install completed. %s.", m)
			}
			if pipInstalledRe.MatchString(line) {
				summary = "Install completed. " + strings.TrimSpace(line)
			}
		}
		return &Result{Output: summary, Lines: 1, Summary: summary}
	}

	extracted := ExtractWithContext(lines, mode, func(line string) bool {
		return IsErrorLine(line)
	})

	limited := LimitLines(extracted, mode)
	return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "Install: errors detected"}
}

// --- Generic Filter ---

type GenericFilter struct{}

func (f *GenericFilter) Apply(output string, mode Mode, keywords []string) *Result {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if mode == ModeVerbose {
		limited := LimitLines(lines, mode)
		return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "Generic (verbose)"}
	}

	extracted := ExtractWithContext(lines, mode, func(line string) bool {
		return IsErrorLine(line) || IsStackTraceLine(line) || MatchesKeyword(line, keywords)
	})

	if len(extracted) == 0 {
		// No matches — return tail
		max := maxLines(mode)
		if len(lines) <= max {
			return &Result{Output: strings.Join(lines, "\n"), Lines: len(lines), Summary: "Generic (full)"}
		}
		tail := lines[len(lines)-max:]
		return &Result{Output: strings.Join(tail, "\n"), Lines: len(tail), Summary: "Generic (tail)"}
	}

	limited := LimitLines(extracted, mode)
	return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "Generic (filtered)"}
}
