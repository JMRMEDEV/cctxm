package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HookInfo describes a detected pre-commit hook entry.
type HookInfo struct {
	Command    string
	Expensive  bool // true if it's a test runner (jest, vitest, pytest, etc.)
	Patchable  bool // true if we can wrap it with SKIP_TESTS
	SourceFile string
}

// HookAnalysis is the result of scanning a workspace for pre-commit hooks.
type HookAnalysis struct {
	Hooks      []HookInfo
	HookFile   string // path to the hook file (if patchable)
	Patchable  bool   // true if the hook file is simple enough to patch
	CommitConv string // detected commit convention (e.g., "conventional")
}

var expensivePatterns = []string{
	"jest", "vitest", "npm test", "npm run test", "npx jest", "npx vitest",
	"yarn test", "pnpm test", "pytest", "python -m pytest", "go test",
	"mvn test", "gradle test", "dotnet test", "npx cypress", "npx playwright",
}

var cheapPatterns = []string{
	"lint-staged", "eslint", "prettier", "commitlint", "tsc --noEmit",
	"stylelint", "markdownlint",
}

// DetectHooks scans the workspace for pre-commit hook configuration.
func DetectHooks(workspaceRoot string) *HookAnalysis {
	analysis := &HookAnalysis{}

	// Try husky v5+ (.husky/pre-commit)
	huskyPath := filepath.Join(workspaceRoot, ".husky", "pre-commit")
	if data, err := os.ReadFile(huskyPath); err == nil {
		analysis.HookFile = huskyPath
		parseHookScript(string(data), huskyPath, analysis)
	}

	// Try husky v4 / legacy (.huskyrc, package.json husky field)
	if len(analysis.Hooks) == 0 {
		parsePackageJSONHooks(workspaceRoot, analysis)
	}

	// Detect commit convention from commitlint config or package.json
	detectCommitConvention(workspaceRoot, analysis)

	// Determine if patchable (simple line-per-command)
	analysis.Patchable = analysis.HookFile != "" && isSimpleHook(analysis.Hooks)

	return analysis
}

// PatchHookFile wraps expensive commands with SKIP_TESTS guard.
func PatchHookFile(analysis *HookAnalysis) error {
	if !analysis.Patchable || analysis.HookFile == "" {
		return nil
	}

	data, err := os.ReadFile(analysis.HookFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var patched []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "#!/") {
			patched = append(patched, line)
			continue
		}
		// Skip already-patched lines
		if strings.Contains(trimmed, "SKIP_TESTS") {
			patched = append(patched, line)
			continue
		}
		if isExpensiveCommand(trimmed) {
			patched = append(patched, fmt.Sprintf(`[ -z "$SKIP_TESTS" ] && %s`, trimmed))
		} else {
			patched = append(patched, line)
		}
	}

	return os.WriteFile(analysis.HookFile, []byte(strings.Join(patched, "\n")), 0755)
}

// GenerateHookRule produces the instruction text for Copilot.
func GenerateHookRule(analysis *HookAnalysis) string {
	if len(analysis.Hooks) == 0 {
		return ""
	}

	hasExpensive := false
	for _, h := range analysis.Hooks {
		if h.Expensive {
			hasExpensive = true
			break
		}
	}
	if !hasExpensive {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n## Pre-commit Hook Strategy\n\n")
	b.WriteString("This project's pre-commit hooks run:\n")
	for _, h := range analysis.Hooks {
		marker := "KEEP"
		if h.Expensive {
			marker = "SKIP (run separately via cctxm)"
		}
		b.WriteString(fmt.Sprintf("- `%s` ← %s\n", h.Command, marker))
	}

	b.WriteString("\n**Workflow for committing:**\n")

	// List expensive commands to run first
	b.WriteString("1. Run tests through cctxm first:\n")
	for _, h := range analysis.Hooks {
		if h.Expensive {
			b.WriteString(fmt.Sprintf("   ```\n   cctxm exec -- %s\n   ```\n", h.Command))
		}
	}

	if analysis.Patchable {
		b.WriteString("2. Commit with SKIP_TESTS to bypass test hooks (lint/format hooks still run):\n")
		b.WriteString("   ```\n   SKIP_TESTS=1 git commit -m \"<message>\"\n   ```\n")
	} else {
		b.WriteString("2. Commit with --no-verify (hooks cannot be partially skipped):\n")
		b.WriteString("   ```\n   git commit --no-verify -m \"<message>\"\n   ```\n")
	}

	// Add commit convention if detected
	if analysis.CommitConv == "conventional" {
		b.WriteString("\n**Commit message format (Conventional Commits):**\n")
		b.WriteString("```\n<type>(<scope>): <description>\n```\n")
		b.WriteString("Allowed types: `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `ci`, `style`, `perf`, `build`, `revert`\n")
	}

	b.WriteString("\nNever run bare `git commit` — unfiltered test output wastes context.\n")

	return b.String()
}

func parseHookScript(content, source string, analysis *HookAnalysis) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "#!/") {
			continue
		}
		// Skip existing SKIP_TESTS guards
		if strings.Contains(trimmed, "SKIP_TESTS") {
			// Extract the command after &&
			if idx := strings.Index(trimmed, "&&"); idx >= 0 {
				trimmed = strings.TrimSpace(trimmed[idx+2:])
			}
		}
		hook := HookInfo{
			Command:    trimmed,
			Expensive:  isExpensiveCommand(trimmed),
			SourceFile: source,
		}
		analysis.Hooks = append(analysis.Hooks, hook)
	}
}

func parsePackageJSONHooks(workspaceRoot string, analysis *HookAnalysis) {
	pkgPath := filepath.Join(workspaceRoot, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	// Check husky field (v4 style)
	if husky, ok := pkg["husky"].(map[string]interface{}); ok {
		if hooks, ok := husky["hooks"].(map[string]interface{}); ok {
			if precommit, ok := hooks["pre-commit"].(string); ok {
				for _, cmd := range strings.Split(precommit, "&&") {
					cmd = strings.TrimSpace(cmd)
					if cmd != "" {
						analysis.Hooks = append(analysis.Hooks, HookInfo{
							Command:    cmd,
							Expensive:  isExpensiveCommand(cmd),
							SourceFile: pkgPath,
						})
					}
				}
			}
		}
	}

	// Check scripts for precommit/pre-commit
	if scripts, ok := pkg["scripts"].(map[string]interface{}); ok {
		for _, key := range []string{"precommit", "pre-commit"} {
			if cmd, ok := scripts[key].(string); ok {
				for _, part := range strings.Split(cmd, "&&") {
					part = strings.TrimSpace(part)
					if part != "" {
						analysis.Hooks = append(analysis.Hooks, HookInfo{
							Command:    part,
							Expensive:  isExpensiveCommand(part),
							SourceFile: pkgPath,
						})
					}
				}
			}
		}
	}
}

func detectCommitConvention(workspaceRoot string, analysis *HookAnalysis) {
	// Check for commitlint config files
	conventionalIndicators := []string{
		"commitlint.config.js",
		"commitlint.config.cjs",
		"commitlint.config.ts",
		".commitlintrc",
		".commitlintrc.json",
		".commitlintrc.yml",
		".commitlintrc.yaml",
	}
	for _, f := range conventionalIndicators {
		if _, err := os.Stat(filepath.Join(workspaceRoot, f)); err == nil {
			analysis.CommitConv = "conventional"
			return
		}
	}

	// Check package.json for commitlint config
	pkgPath := filepath.Join(workspaceRoot, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return
	}
	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}
	if _, ok := pkg["commitlint"]; ok {
		analysis.CommitConv = "conventional"
	}
}

func isExpensiveCommand(cmd string) bool {
	lower := strings.ToLower(cmd)
	for _, p := range expensivePatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func isCheapCommand(cmd string) bool {
	lower := strings.ToLower(cmd)
	for _, p := range cheapPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func isSimpleHook(hooks []HookInfo) bool {
	// A hook is "simple" if all entries are single commands (no pipes, no complex logic)
	for _, h := range hooks {
		cmd := h.Command
		if strings.Contains(cmd, "|") || strings.Contains(cmd, ";") ||
			strings.Contains(cmd, "if ") || strings.Contains(cmd, "for ") {
			return false
		}
	}
	return len(hooks) > 0
}
