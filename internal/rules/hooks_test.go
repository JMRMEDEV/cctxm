package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectHooksHusky(t *testing.T) {
	root := t.TempDir()
	huskyDir := filepath.Join(root, ".husky")
	os.MkdirAll(huskyDir, 0755)
	os.WriteFile(filepath.Join(huskyDir, "pre-commit"), []byte("#!/bin/sh\nnpx lint-staged\nnpm test\n"), 0755)

	analysis := DetectHooks(root)

	if len(analysis.Hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(analysis.Hooks))
	}
	if analysis.Hooks[0].Command != "npx lint-staged" || analysis.Hooks[0].Expensive {
		t.Error("first hook should be lint-staged (cheap)")
	}
	if analysis.Hooks[1].Command != "npm test" || !analysis.Hooks[1].Expensive {
		t.Error("second hook should be npm test (expensive)")
	}
	if !analysis.Patchable {
		t.Error("simple hook should be patchable")
	}
}

func TestDetectHooksPackageJSON(t *testing.T) {
	root := t.TempDir()
	pkg := `{"husky":{"hooks":{"pre-commit":"lint-staged && jest"}}}`
	os.WriteFile(filepath.Join(root, "package.json"), []byte(pkg), 0644)

	analysis := DetectHooks(root)

	if len(analysis.Hooks) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(analysis.Hooks))
	}
	if !analysis.Hooks[1].Expensive {
		t.Error("jest should be expensive")
	}
}

func TestDetectHooksScripts(t *testing.T) {
	root := t.TempDir()
	pkg := `{"scripts":{"precommit":"jest --bail"}}`
	os.WriteFile(filepath.Join(root, "package.json"), []byte(pkg), 0644)

	analysis := DetectHooks(root)

	if len(analysis.Hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(analysis.Hooks))
	}
	if !analysis.Hooks[0].Expensive {
		t.Error("jest should be expensive")
	}
}

func TestDetectCommitConvention(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".husky"), 0755)
	os.WriteFile(filepath.Join(root, ".husky", "pre-commit"), []byte("npm test\n"), 0755)
	os.WriteFile(filepath.Join(root, "commitlint.config.js"), []byte("module.exports = {}"), 0644)

	analysis := DetectHooks(root)

	if analysis.CommitConv != "conventional" {
		t.Error("should detect conventional commits")
	}
}

func TestGenerateHookRuleWithExpensive(t *testing.T) {
	analysis := &HookAnalysis{
		Hooks: []HookInfo{
			{Command: "npx lint-staged", Expensive: false},
			{Command: "npm test", Expensive: true},
		},
		Patchable:  true,
		CommitConv: "conventional",
	}

	rule := GenerateHookRule(analysis)

	if !strings.Contains(rule, "SKIP_TESTS=1 git commit") {
		t.Error("should suggest SKIP_TESTS for patchable hooks")
	}
	if !strings.Contains(rule, "cctxm exec -- npm test") {
		t.Error("should suggest running tests via cctxm")
	}
	if !strings.Contains(rule, "Conventional Commits") {
		t.Error("should include commit format")
	}
	if !strings.Contains(rule, "KEEP") {
		t.Error("should mark cheap hooks as KEEP")
	}
}

func TestGenerateHookRuleNonPatchable(t *testing.T) {
	analysis := &HookAnalysis{
		Hooks: []HookInfo{
			{Command: "npm test | tee log.txt", Expensive: true},
		},
		Patchable: false,
	}

	rule := GenerateHookRule(analysis)

	if !strings.Contains(rule, "--no-verify") {
		t.Error("should suggest --no-verify for non-patchable hooks")
	}
}

func TestGenerateHookRuleNoExpensive(t *testing.T) {
	analysis := &HookAnalysis{
		Hooks: []HookInfo{
			{Command: "npx lint-staged", Expensive: false},
		},
	}

	rule := GenerateHookRule(analysis)

	if rule != "" {
		t.Error("should return empty string when no expensive hooks")
	}
}

func TestPatchHookFile(t *testing.T) {
	root := t.TempDir()
	hookPath := filepath.Join(root, "pre-commit")
	os.WriteFile(hookPath, []byte("#!/bin/sh\nnpx lint-staged\nnpm test\n"), 0755)

	analysis := &HookAnalysis{
		Hooks: []HookInfo{
			{Command: "npx lint-staged", Expensive: false},
			{Command: "npm test", Expensive: true},
		},
		HookFile:  hookPath,
		Patchable: true,
	}

	err := PatchHookFile(analysis)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(hookPath)
	content := string(data)

	if !strings.Contains(content, `[ -z "$SKIP_TESTS" ] && npm test`) {
		t.Error("should wrap expensive command with SKIP_TESTS guard")
	}
	if strings.Contains(content, `[ -z "$SKIP_TESTS" ] && npx lint-staged`) {
		t.Error("should NOT wrap cheap commands")
	}
}

func TestPatchHookFileIdempotent(t *testing.T) {
	root := t.TempDir()
	hookPath := filepath.Join(root, "pre-commit")
	// Already patched content
	os.WriteFile(hookPath, []byte("#!/bin/sh\nnpx lint-staged\n[ -z \"$SKIP_TESTS\" ] && npm test\n"), 0755)

	analysis := &HookAnalysis{
		Hooks: []HookInfo{
			{Command: "npx lint-staged", Expensive: false},
			{Command: "npm test", Expensive: true},
		},
		HookFile:  hookPath,
		Patchable: true,
	}

	PatchHookFile(analysis)

	data, _ := os.ReadFile(hookPath)
	// Should not double-wrap
	count := strings.Count(string(data), "SKIP_TESTS")
	if count != 1 {
		t.Errorf("should not double-wrap, found %d occurrences", count)
	}
}

func TestInjectWithHooks(t *testing.T) {
	root, cctxmDir := setupWorkspace(t)

	// Create husky hook
	huskyDir := filepath.Join(root, ".husky")
	os.MkdirAll(huskyDir, 0755)
	os.WriteFile(filepath.Join(huskyDir, "pre-commit"), []byte("#!/bin/sh\nnpx lint-staged\nnpm test\n"), 0755)

	err := Inject(root, cctxmDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Check instructions contain hook rule
	instrPath := filepath.Join(root, ".github", "copilot", "instructions.md")
	data, _ := os.ReadFile(instrPath)
	content := string(data)

	if !strings.Contains(content, "Pre-commit Hook Strategy") {
		t.Error("should contain hook strategy in instructions")
	}
	if !strings.Contains(content, "cctxm exec -- npm test") {
		t.Error("should instruct to run tests via cctxm")
	}

	// Check hook was patched
	hookData, _ := os.ReadFile(filepath.Join(huskyDir, "pre-commit"))
	if !strings.Contains(string(hookData), "SKIP_TESTS") {
		t.Error("hook file should be patched")
	}
}

func TestIsSimpleHookWithPipes(t *testing.T) {
	hooks := []HookInfo{{Command: "npm test | tee output.log"}}
	if isSimpleHook(hooks) {
		t.Error("hook with pipe should not be simple")
	}
}

func TestDetectHooksNone(t *testing.T) {
	root := t.TempDir()
	analysis := DetectHooks(root)
	if len(analysis.Hooks) != 0 {
		t.Error("should detect no hooks in empty dir")
	}
}
