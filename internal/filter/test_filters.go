package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// --- Jest / Vitest ---

type JestFilter struct{}

var jestSummaryRe = regexp.MustCompile(`Tests:\s+(.*)`)
var jestFailBlockRe = regexp.MustCompile(`(?i)^(\s*●\s|FAIL\s)`)

func (f *JestFilter) Apply(output string, mode Mode, keywords []string) *Result {
	if mode == ModeVerbose {
		lines := strings.Split(output, "\n")
		limited := LimitLines(lines, mode)
		return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "Jest (verbose)"}
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	summary := ""
	for _, l := range lines {
		if m := jestSummaryRe.FindString(l); m != "" {
			summary = strings.TrimSpace(m)
		}
	}

	// Check if there are failures
	if !strings.Contains(output, "FAIL") || (summary != "" && !strings.Contains(summary, "failed")) {
		if summary == "" {
			summary = "All tests passed"
		}
		return &Result{Output: summary, Lines: 1, Summary: summary}
	}

	// Extract FAIL blocks
	var result []string
	inFailBlock := false
	for _, line := range lines {
		if jestFailBlockRe.MatchString(line) || strings.HasPrefix(line, "FAIL ") {
			inFailBlock = true
		}
		if inFailBlock {
			result = append(result, line)
			if line == "" && len(result) > 1 {
				inFailBlock = false
			}
		}
	}
	if summary != "" {
		result = append([]string{summary, ""}, result...)
	}

	limited := LimitLines(result, mode)
	return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: summary}
}

// --- pytest ---

type PytestFilter struct{}

var pytestSummaryRe = regexp.MustCompile(`=+\s*([\d]+ passed|[\d]+ failed|FAILURES)`)

func (f *PytestFilter) Apply(output string, mode Mode, keywords []string) *Result {
	if mode == ModeVerbose {
		lines := strings.Split(output, "\n")
		limited := LimitLines(lines, mode)
		return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "pytest (verbose)"}
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// Find summary line (last line with = signs)
	summary := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "passed") || strings.Contains(lines[i], "failed") || strings.Contains(lines[i], "error") {
			if strings.Contains(lines[i], "=") {
				summary = strings.TrimSpace(lines[i])
				break
			}
		}
	}

	if !strings.Contains(output, "FAILED") && !strings.Contains(output, "FAILURES") {
		if summary == "" {
			summary = "All tests passed"
		}
		return &Result{Output: summary, Lines: 1, Summary: summary}
	}

	// Extract FAILURES section and short test summary
	var result []string
	inSection := false
	for _, line := range lines {
		if strings.Contains(line, "FAILURES") || strings.Contains(line, "short test summary") {
			inSection = true
		}
		if inSection {
			result = append(result, line)
		}
	}
	if summary != "" {
		result = append(result, "", summary)
	}

	limited := LimitLines(result, mode)
	return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: summary}
}

// --- Go test ---

type GoTestFilter struct{}

func (f *GoTestFilter) Apply(output string, mode Mode, keywords []string) *Result {
	if mode == ModeVerbose {
		lines := strings.Split(output, "\n")
		limited := LimitLines(lines, mode)
		return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "go test (verbose)"}
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	hasFail := false
	var okLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "--- FAIL") || strings.HasPrefix(line, "FAIL") {
			hasFail = true
		}
		if strings.HasPrefix(line, "ok ") {
			okLines = append(okLines, line)
		}
	}

	if !hasFail {
		summary := fmt.Sprintf("All tests passed (%d packages)", len(okLines))
		out := strings.Join(okLines, "\n")
		return &Result{Output: out, Lines: len(okLines), Summary: summary}
	}

	// Extract --- FAIL blocks
	var result []string
	inFail := false
	for _, line := range lines {
		if strings.HasPrefix(line, "--- FAIL") {
			inFail = true
		}
		if inFail {
			result = append(result, line)
		}
		if strings.HasPrefix(line, "FAIL\t") || strings.HasPrefix(line, "ok \t") {
			if inFail {
				inFail = false
			}
			result = append(result, line)
		}
	}

	limited := LimitLines(result, mode)
	return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "go test: failures detected"}
}

// --- Maven ---

type MavenFilter struct{}

func (f *MavenFilter) Apply(output string, mode Mode, keywords []string) *Result {
	if mode == ModeVerbose {
		lines := strings.Split(output, "\n")
		limited := LimitLines(lines, mode)
		return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "Maven (verbose)"}
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	summary := ""
	for _, line := range lines {
		if strings.Contains(line, "Tests run:") {
			summary = strings.TrimSpace(line)
		}
	}

	if !strings.Contains(output, "FAILURE") && !strings.Contains(output, "<<< FAIL") {
		if summary == "" {
			summary = "All tests passed"
		}
		return &Result{Output: summary, Lines: 1, Summary: summary}
	}

	var result []string
	inFail := false
	for _, line := range lines {
		if strings.Contains(line, "<<< FAILURE!") || strings.Contains(line, "<<< ERROR!") {
			inFail = true
		}
		if inFail {
			result = append(result, line)
			if line == "" {
				inFail = false
			}
		}
		if strings.Contains(line, "Tests run:") {
			result = append(result, line)
		}
	}

	limited := LimitLines(result, mode)
	return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: summary}
}

// --- Gradle ---

type GradleFilter struct{}

func (f *GradleFilter) Apply(output string, mode Mode, keywords []string) *Result {
	if mode == ModeVerbose {
		lines := strings.Split(output, "\n")
		limited := LimitLines(lines, mode)
		return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "Gradle (verbose)"}
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	if !strings.Contains(output, "FAILED") && !strings.Contains(output, "failures") {
		summary := "All tests passed"
		for _, line := range lines {
			if strings.Contains(line, "BUILD SUCCESSFUL") {
				summary = strings.TrimSpace(line)
			}
		}
		return &Result{Output: summary, Lines: 1, Summary: summary}
	}

	extracted := ExtractWithContext(lines, mode, func(line string) bool {
		return strings.Contains(line, "FAILED") || strings.Contains(line, "> Task") || IsErrorLine(line)
	})

	limited := LimitLines(extracted, mode)
	return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "Gradle: failures detected"}
}

// --- dotnet test ---

type DotnetFilter struct{}

func (f *DotnetFilter) Apply(output string, mode Mode, keywords []string) *Result {
	if mode == ModeVerbose {
		lines := strings.Split(output, "\n")
		limited := LimitLines(lines, mode)
		return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "dotnet test (verbose)"}
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	summary := ""
	for _, line := range lines {
		if strings.Contains(line, "Passed!") || strings.Contains(line, "Failed!") || strings.Contains(line, "Total tests:") {
			summary = strings.TrimSpace(line)
		}
	}

	if !strings.Contains(output, "Failed") {
		if summary == "" {
			summary = "All tests passed"
		}
		return &Result{Output: summary, Lines: 1, Summary: summary}
	}

	extracted := ExtractWithContext(lines, mode, func(line string) bool {
		lower := strings.ToLower(line)
		return strings.Contains(lower, "failed") || strings.Contains(lower, "error") || IsStackTraceLine(line)
	})

	limited := LimitLines(extracted, mode)
	return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: summary}
}

// --- E2E (Cypress / Playwright) ---

type E2EFilter struct{}

func (f *E2EFilter) Apply(output string, mode Mode, keywords []string) *Result {
	if mode == ModeVerbose {
		lines := strings.Split(output, "\n")
		limited := LimitLines(lines, mode)
		return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "E2E (verbose)"}
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	hasFail := false
	for _, line := range lines {
		if strings.Contains(line, "✗") || strings.Contains(line, "failing") ||
			strings.Contains(line, "failed") || strings.Contains(line, "✕") {
			hasFail = true
			break
		}
	}

	if !hasFail {
		summary := "All E2E tests passed"
		for _, line := range lines {
			if strings.Contains(line, "passing") || strings.Contains(line, "passed") {
				summary = strings.TrimSpace(line)
			}
		}
		return &Result{Output: summary, Lines: 1, Summary: summary}
	}

	extracted := ExtractWithContext(lines, mode, func(line string) bool {
		return strings.Contains(line, "✗") || strings.Contains(line, "✕") ||
			strings.Contains(line, "failing") || strings.Contains(line, "AssertionError") ||
			strings.Contains(line, "Error:") || IsStackTraceLine(line)
	})

	limited := LimitLines(extracted, mode)
	return &Result{Output: strings.Join(limited, "\n"), Lines: len(limited), Summary: "E2E: failures detected"}
}
