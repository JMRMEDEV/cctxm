package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const copilotDir = ".github/copilot"
const instructionsFile = "instructions.md"

var coreInstructions = `# Copilot Instructions (managed by cctxm)

## Command Execution
Always use ` + "`cctxm exec`" + ` to run terminal commands.
` + "```" + `
cctxm exec -- <command>
cctxm exec --project <name> -- <command>
` + "```" + `

## File Reading
Always use ` + "`cctxm read`" + ` to read files.
` + "```" + `
cctxm read <file>
cctxm read --full <file>
cctxm read --search "terms" <file>
` + "```" + `

## Session Management
At the start of every chat, run:
` + "```" + `
cctxm session start "<brief task description>"
` + "```" + `
If continuing previous work, run:
` + "```" + `
cctxm session list
cctxm session restore <id>
` + "```" + `

## Task Context
To update the task focus:
` + "```" + `
cctxm task set "<description>"
` + "```" + `
`

// Inject backs up the original .github/copilot/ and replaces it with cctxm instructions.
func Inject(workspaceRoot, cctxmDir string, ruleGlobs []string) error {
	src := filepath.Join(workspaceRoot, copilotDir)
	backupDir := filepath.Join(cctxmDir, "overridden", "copilot")

	// Backup original if it exists and hasn't been backed up
	if _, err := os.Stat(src); err == nil {
		if _, err := os.Stat(backupDir); os.IsNotExist(err) {
			if err := copyDir(src, backupDir); err != nil {
				return fmt.Errorf("failed to backup %s: %w", copilotDir, err)
			}
		}
	}

	// Create .github/copilot/
	if err := os.MkdirAll(src, 0755); err != nil {
		return err
	}

	// Build instructions content
	var content strings.Builder
	content.WriteString(coreInstructions)

	// Detect and append hook rules
	analysis := DetectHooks(workspaceRoot)
	if hookRule := GenerateHookRule(analysis); hookRule != "" {
		content.WriteString("\n---\n")
		content.WriteString(hookRule)

		// Patch hook file if possible
		if analysis.Patchable {
			PatchHookFile(analysis)
		}
	}

	// Append user rules
	for _, glob := range ruleGlobs {
		pattern := glob
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(workspaceRoot, pattern)
		}
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			content.WriteString("\n---\n\n")
			content.WriteString(fmt.Sprintf("<!-- From: %s -->\n", filepath.Base(match)))
			content.Write(data)
			content.WriteString("\n")
		}
	}

	// Write instructions
	instrPath := filepath.Join(src, instructionsFile)
	if err := os.WriteFile(instrPath, []byte(content.String()), 0644); err != nil {
		return err
	}

	// Add to .git/info/exclude
	addGitExclude(workspaceRoot)

	return nil
}

// Restore puts back the original .github/copilot/ from backup.
func Restore(workspaceRoot, cctxmDir string) error {
	src := filepath.Join(workspaceRoot, copilotDir)
	backupDir := filepath.Join(cctxmDir, "overridden", "copilot")

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		// No backup — just remove the injected files
		os.RemoveAll(src)
		return nil
	}

	os.RemoveAll(src)
	return copyDir(backupDir, src)
}

// Show returns what would be injected without writing anything.
func Show(workspaceRoot string, ruleGlobs []string) string {
	var content strings.Builder
	content.WriteString(coreInstructions)

	// Include hook rules in preview
	analysis := DetectHooks(workspaceRoot)
	if hookRule := GenerateHookRule(analysis); hookRule != "" {
		content.WriteString("\n---\n")
		content.WriteString(hookRule)
	}

	for _, glob := range ruleGlobs {
		pattern := glob
		if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(workspaceRoot, pattern)
		}
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			content.WriteString("\n---\n\n")
			content.WriteString(fmt.Sprintf("<!-- From: %s -->\n", filepath.Base(match)))
			content.Write(data)
			content.WriteString("\n")
		}
	}
	return content.String()
}

func addGitExclude(workspaceRoot string) {
	excludePath := filepath.Join(workspaceRoot, ".git", "info", "exclude")
	entry := ".github/copilot/"

	if data, err := os.ReadFile(excludePath); err == nil {
		if strings.Contains(string(data), entry) {
			return
		}
	}

	os.MkdirAll(filepath.Dir(excludePath), 0755)
	f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString("\n" + entry + "\n")
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}
