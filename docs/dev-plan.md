# CCTXM — Development Plan

## Phase 1: Foundation

### 1.1 Project Scaffolding
- [ ] Initialize Go module (`go mod init github.com/<owner>/cctxm`)
- [ ] Set up cobra CLI skeleton with root command
- [ ] Set up project directory structure (`cmd/`, `internal/`)
- [ ] Add Makefile with build targets for linux, darwin, windows
- [ ] Add `.goreleaser.yml` for release builds

### 1.2 Configuration
- [ ] Define config struct and YAML schema
- [ ] Implement config loader with defaults
- [ ] Implement `cctxm config show`
- [ ] Implement `cctxm config edit`
- [ ] Implement `cctxm config set`

### 1.3 Project Router
- [ ] Implement project name → directory resolution from config
- [ ] Handle `--project` flag
- [ ] Handle missing/invalid project names with clear errors
- [ ] Cross-platform path resolution

## Phase 2: Session Management

### 2.1 Session Lifecycle
- [ ] Implement `cctxm session start` — create session directory + meta.json
- [ ] Implement `cctxm session list` — list sessions with metadata
- [ ] Implement `cctxm session show` — show active session details
- [ ] Implement `cctxm session restore` — reactivate session + reload task
- [ ] Implement `cctxm session switch` — alias for restore
- [ ] Implement `cctxm session clean` — remove old sessions

### 2.2 State Tracking
- [ ] Implement `state.json` read/write for active session
- [ ] Implement session resolution order (flag → state → daily default)
- [ ] Implement command logging within sessions (commands.json)

## Phase 3: Command Execution

### 3.1 Executor
- [ ] Implement `cctxm exec` — run command in resolved cwd
- [ ] Capture stdout + stderr to raw log file
- [ ] Stream output in real-time (don't buffer entire output before filtering)
- [ ] Record command metadata (exit code, duration, timestamp)
- [ ] Cross-platform shell execution (`sh -c` vs `cmd /C`)

### 3.2 Init Command
- [ ] Implement `cctxm init` — create `.cctxm/` structure
- [ ] Generate default config.yaml
- [ ] Create rules directory with example rule file

## Phase 4: Filter Engine

### 4.1 Core Framework
- [ ] Implement command classifier (command string → output type)
- [ ] Implement filter interface and mode limiter
- [ ] Implement metadata header generation
- [ ] Implement keyword extraction from task description

### 4.2 Test Runner Filters
- [ ] Jest / Vitest filter
- [ ] pytest filter
- [ ] Go test filter
- [ ] Maven / Gradle filter
- [ ] dotnet test filter
- [ ] Cypress / Playwright (E2E) filter

### 4.3 Docker Log Filter
- [ ] JSON structured log parser + filter
- [ ] Plain text log filter (regex-based)
- [ ] Error pattern detection (stack traces, exceptions)

### 4.4 Build & Install Filters
- [ ] Build output filter (success summary / error extraction)
- [ ] Install output filter (success summary / error extraction)

### 4.5 Generic Filter
- [ ] Line-based error/keyword matching
- [ ] Head + tail preservation
- [ ] Middle truncation for oversized output

## Phase 5: Smart File Reading

### 5.1 File Reader
- [ ] Implement `cctxm read` command
- [ ] Size-based decision (full vs filtered)
- [ ] Extension-based overrides (`.md` always full)
- [ ] Keyword-based section extraction for large files
- [ ] `--full` and `--search` flag overrides

## Phase 6: Task Context

### 6.1 Task Management
- [ ] Implement `cctxm task set` — store task + extract keywords
- [ ] Implement `cctxm task show` — display task, keywords, mode
- [ ] Implement `cctxm task clear`
- [ ] Implement `cctxm task mode` — manual mode override
- [ ] Keyword → filter mode detection logic

## Phase 7: Rules Injection

### 7.1 Copilot Override
- [ ] Implement `cctxm rules inject` — backup + replace `.github/copilot/`
- [ ] Generate instructions.md with cctxm usage instructions
- [ ] Append user rules from `.cctxm/rules/*.md`
- [ ] Add `.github/copilot/` to `.git/info/exclude`
- [ ] Implement `cctxm rules restore` — restore from backup
- [ ] Implement `cctxm rules show` — preview injected content

### 7.2 Init Integration
- [ ] `cctxm init` triggers rules injection by default
- [ ] `--skip-rules` flag to opt out

## Phase 8: Polish & Distribution

### 8.1 Error Handling & UX
- [ ] Consistent error messages across all commands
- [ ] Helpful suggestions on common mistakes
- [ ] `--help` text for all commands
- [ ] Shell completion (bash, zsh, fish, powershell)

### 8.2 Testing
- [ ] Unit tests for each filter strategy
- [ ] Unit tests for config parsing
- [ ] Unit tests for session management
- [ ] Integration tests for exec pipeline
- [ ] Cross-platform CI (GitHub Actions: linux, macos, windows)

### 8.3 Distribution
- [ ] GoReleaser config for multi-platform binaries
- [ ] GitHub Releases
- [ ] Homebrew formula (macOS/Linux)
- [ ] Scoop manifest (Windows)
- [ ] Installation script (`curl | sh` style)

## Dependency Order

```
Phase 1 (Foundation)
  └→ Phase 2 (Sessions)
  └→ Phase 3 (Execution)
       └→ Phase 4 (Filters) — depends on executor for output capture
       └→ Phase 5 (File Reader) — independent of executor
  └→ Phase 6 (Task Context) — can be built in parallel with Phase 4
  └→ Phase 7 (Rules) — independent, can be built anytime after Phase 1
Phase 8 (Polish) — after all features are functional
```

Phases 4, 5, 6, and 7 can be developed in parallel once Phase 3 is complete.
