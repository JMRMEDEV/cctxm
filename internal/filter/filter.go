package filter

import (
	"fmt"
	"strings"
)

// Mode represents the filter aggressiveness level.
type Mode int

const (
	ModeStrict  Mode = iota // errors + keyword matches only
	ModeNormal              // strict + surrounding context
	ModeVerbose             // full output, capped
)

func ParseMode(s string) Mode {
	switch strings.ToLower(s) {
	case "strict":
		return ModeStrict
	case "verbose":
		return ModeVerbose
	default:
		return ModeNormal
	}
}

func (m Mode) String() string {
	switch m {
	case ModeStrict:
		return "strict"
	case ModeVerbose:
		return "verbose"
	default:
		return "normal"
	}
}

// OutputType classifies what kind of output we're dealing with.
type OutputType int

const (
	TypeGeneric OutputType = iota
	TypeDocker
	TypeTestJest
	TypeTestPytest
	TypeTestGo
	TypeTestMaven
	TypeTestGradle
	TypeTestDotnet
	TypeTestE2E
	TypeBuild
	TypeInstall
)

// Classify determines the output type from the command string.
func Classify(command string) OutputType {
	cmd := strings.ToLower(command)

	// Docker
	if matchAny(cmd, "docker logs", "docker-compose logs", "docker compose logs", "podman logs") {
		return TypeDocker
	}

	// Test runners
	if matchAny(cmd, "npx jest", "npx vitest", "yarn test", "yarn jest", "yarn vitest") {
		return TypeTestJest
	}
	if matchAny(cmd, "npm test", "npm run test") {
		return TypeTestJest
	}
	if matchAny(cmd, "pytest", "python -m pytest") {
		return TypeTestPytest
	}
	if matchAny(cmd, "python -m unittest", "python -m nose") {
		return TypeTestPytest
	}
	if strings.Contains(cmd, "go test") {
		return TypeTestGo
	}
	if matchAny(cmd, "mvn test", "mvn verify", "mvnw test", "./mvnw test") {
		return TypeTestMaven
	}
	if matchAny(cmd, "gradle test", "gradlew test", "./gradlew test") {
		return TypeTestGradle
	}
	if strings.Contains(cmd, "dotnet test") {
		return TypeTestDotnet
	}
	if matchAny(cmd, "npx cypress", "npx playwright", "cypress run", "playwright test") {
		return TypeTestE2E
	}

	// Build
	if matchAny(cmd, "npm run build", "go build", "mvn package", "gradle build",
		"gradlew build", "./gradlew build", "dotnet build", "cargo build", "make build") {
		return TypeBuild
	}

	// Install
	if matchAny(cmd, "npm install", "npm ci", "yarn install", "pip install",
		"go mod download", "go mod tidy", "dotnet restore", "cargo fetch") {
		return TypeInstall
	}

	return TypeGeneric
}

func matchAny(cmd string, patterns ...string) bool {
	for _, p := range patterns {
		if strings.HasPrefix(cmd, p) || strings.Contains(cmd, " "+p) {
			return true
		}
	}
	return false
}

// Result holds filtered output.
type Result struct {
	Output  string
	Summary string
	Lines   int // lines in filtered output
	Total   int // total lines in raw output
}

// Filter is the interface all filters implement.
type Filter interface {
	Apply(output string, mode Mode, keywords []string) *Result
}

// Apply runs the appropriate filter for a command.
func Apply(command, output string, mode Mode, keywords []string) *Result {
	typ := Classify(command)
	var f Filter

	switch typ {
	case TypeDocker:
		f = &DockerFilter{}
	case TypeTestJest:
		f = &JestFilter{}
	case TypeTestPytest:
		f = &PytestFilter{}
	case TypeTestGo:
		f = &GoTestFilter{}
	case TypeTestMaven:
		f = &MavenFilter{}
	case TypeTestGradle:
		f = &GradleFilter{}
	case TypeTestDotnet:
		f = &DotnetFilter{}
	case TypeTestE2E:
		f = &E2EFilter{}
	case TypeBuild:
		f = &BuildFilter{}
	case TypeInstall:
		f = &InstallFilter{}
	default:
		f = &GenericFilter{}
	}

	result := f.Apply(output, mode, keywords)
	result.Total = len(strings.Split(strings.TrimRight(output, "\n"), "\n"))

	header := fmt.Sprintf("[cctxm] Filtered: showing %d/%d lines (%s mode",
		result.Lines, result.Total, mode)
	if len(keywords) > 0 {
		header += fmt.Sprintf(", keywords: %s", strings.Join(keywords, ", "))
	}
	header += ")"
	if result.Summary != "" {
		header += "\n[cctxm] " + result.Summary
	}

	result.Output = header + "\n\n" + result.Output
	return result
}

// --- Shared helpers ---

// LimitLines applies mode-based line limits with middle truncation.
func LimitLines(lines []string, mode Mode) []string {
	max := maxLines(mode)
	if len(lines) <= max {
		return lines
	}
	half := max / 2
	top := lines[:half]
	bottom := lines[len(lines)-half:]
	truncated := append(top, fmt.Sprintf("\n... [%d lines truncated] ...\n", len(lines)-max))
	return append(truncated, bottom...)
}

func maxLines(mode Mode) int {
	switch mode {
	case ModeStrict:
		return 50
	case ModeVerbose:
		return 1000
	default:
		return 200
	}
}

func contextLines(mode Mode) int {
	switch mode {
	case ModeStrict:
		return 3
	case ModeVerbose:
		return 999999
	default:
		return 10
	}
}

// MatchesKeyword checks if a line contains any of the keywords (case-insensitive).
func MatchesKeyword(line string, keywords []string) bool {
	lower := strings.ToLower(line)
	for _, kw := range keywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// IsErrorLine checks if a line looks like an error/warning.
func IsErrorLine(line string) bool {
	lower := strings.ToLower(line)
	patterns := []string{
		"error", "err:", "fatal", "panic", "fail",
		"warn", "warning", "exception", "traceback",
		"stack trace", "caused by",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// IsStackTraceLine checks if a line looks like part of a stack trace.
func IsStackTraceLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "at ") ||
		strings.HasPrefix(trimmed, "at\t") ||
		strings.HasPrefix(line, "\tat ") ||
		strings.HasPrefix(line, "    at ") ||
		strings.HasPrefix(trimmed, "goroutine ") ||
		strings.HasPrefix(trimmed, "File \"") ||
		(len(trimmed) > 0 && trimmed[0] == '/' && strings.Contains(trimmed, ".go:"))
}

// ExtractWithContext returns lines matching a predicate plus surrounding context.
func ExtractWithContext(lines []string, mode Mode, match func(string) bool) []string {
	ctx := contextLines(mode)
	marked := make([]bool, len(lines))

	for i, line := range lines {
		if match(line) {
			start := i - ctx
			if start < 0 {
				start = 0
			}
			end := i + ctx + 1
			if end > len(lines) {
				end = len(lines)
			}
			for j := start; j < end; j++ {
				marked[j] = true
			}
		}
	}

	var result []string
	inBlock := false
	for i, line := range lines {
		if marked[i] {
			if !inBlock && len(result) > 0 {
				result = append(result, "---")
			}
			result = append(result, line)
			inBlock = true
		} else {
			inBlock = false
		}
	}
	return result
}
