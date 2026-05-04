# CCTXM — Architecture

## Overview

CCTXM (Copilot Context Manager) is a CLI middleware that sits between VS Code Copilot and the terminal. It intercepts command execution and file reads, filters output intelligently, and injects context/rules — reducing wasted LLM context and credits.

## System Diagram

```
┌──────────────────────────────────────────────────────────┐
│  VS Code + GitHub Copilot                                │
│  (reads injected instructions from .github/copilot/)     │
└────────────────────┬─────────────────────────────────────┘
                     │ runs: cctxm exec --project X -- <cmd>
                     │       cctxm read <file>
                     ▼
┌──────────────────────────────────────────────────────────┐
│  CCTXM CLI (single Go binary)                           │
│                                                          │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────┐  │
│  │ Command      │  │ Session      │  │ Rules          │  │
│  │ Router       │  │ Manager      │  │ Injector       │  │
│  │              │  │              │  │                │  │
│  │ Resolves cwd │  │ Tracks state │  │ Manages        │  │
│  │ per project  │  │ per session  │  │ .github/       │  │
│  │ mapping      │  │              │  │ copilot/ files │  │
│  └──────┬──────┘  └──────┬───────┘  └────────────────┘  │
│         │                │                               │
│  ┌──────▼──────────────────────────────────────────────┐ │
│  │ Executor                                            │ │
│  │                                                     │ │
│  │ 1. Runs command in resolved cwd                     │ │
│  │ 2. Captures full output → .cctxm/sessions/raw.log   │ │
│  │ 3. Pipes output through Filter Engine               │ │
│  │ 4. Returns filtered output → stdout (to Copilot)   │ │
│  └──────┬──────────────────────────────────────────────┘ │
│         │                                                │
│  ┌──────▼──────────────────────────────────────────────┐ │
│  │ Filter Engine                                       │ │
│  │                                                     │ │
│  │ ┌────────────┐ ┌────────────┐ ┌──────────────────┐ │ │
│  │ │ Test Runner │ │ Docker Log │ │ Generic/Build    │ │ │
│  │ │ Filter     │ │ Filter     │ │ Filter           │ │ │
│  │ └────────────┘ └────────────┘ └──────────────────┘ │ │
│  │                                                     │ │
│  │ Filter mode driven by task keywords:                │ │
│  │   strict  → errors + keyword matches only           │ │
│  │   normal  → strict + surrounding context            │ │
│  │   verbose → full output (still capped)              │ │
│  └─────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

## Core Components

### 1. CLI Layer (`cmd/`)

Built with [cobra](https://github.com/spf13/cobra). Entry point for all commands.

Commands:
- `cctxm init` — Initialize workspace
- `cctxm exec` — Execute and filter commands
- `cctxm read` — Smart file reading
- `cctxm task` — Manage task context
- `cctxm session` — Manage sessions
- `cctxm rules` — Manage rule injection
- `cctxm config` — Edit configuration

### 2. Config Manager (`internal/config/`)

Parses `.cctxm/config.yaml` at workspace root. Supports:
- Project-to-directory mappings
- Filter thresholds and defaults
- Rule file paths
- Override targets

Falls back to sensible defaults if no config exists.

### 3. Command Router (`internal/router/`)

Resolves the correct working directory for command execution.

Input: `--project input-service` + command
Process: Looks up `input-service` in config → `./services/input-service`
Output: Absolute path to execute the command in

If no `--project` flag, uses the directory where `cctxm` was invoked.

### 4. Executor (`internal/executor/`)

Runs shell commands and captures output.

- Spawns the command as a subprocess
- Streams stdout/stderr to a raw log file in the session directory
- Simultaneously pipes output through the filter engine
- Returns filtered output to stdout
- Records command metadata (exit code, duration, timestamp) in session state

### 5. Session Manager (`internal/session/`)

Manages session lifecycle and state.

State file: `.cctxm/state.json`
```json
{
  "active_session": "s_20260503_2252",
  "workspace": "/home/user/my-workspace"
}
```

Session directory: `.cctxm/sessions/<session-id>/`
```
meta.json           — Task, keywords, filter mode, created timestamp
commands.json       — Ordered list of commands run with metadata
NNN-<label>.raw.log — Full command output
NNN-<label>.filtered.log — Filtered output sent to Copilot
```

Session resolution order:
1. `--session <id>` flag (explicit)
2. Active session from `state.json` (implicit)
3. Auto-created daily default session (fallback)

### 6. Filter Engine (`internal/filter/`)

The core value of cctxm. Classifies output and applies the appropriate filter.

**Output classification** — Detected from the command being run:
| Command pattern | Classifier |
|---|---|
| `docker logs`, `docker-compose logs` | docker |
| `npm test`, `pytest`, `go test`, `mvn test`, etc. | test |
| `npm run build`, `go build`, `mvn package`, etc. | build |
| `npm install`, `pip install`, etc. | install |
| Everything else | generic |

**Filter modes** — Driven by task keywords:
- `strict` — Only errors, warnings, stack traces, and keyword matches
- `normal` — Strict output + N lines of surrounding context (default N=10)
- `verbose` — Full output, capped at configurable max lines

**Keyword detection** from task description:
- Debug keywords → strict: `error`, `fix`, `bug`, `issue`, `debug`, `fail`, `crash`, `broken`, `wrong`, `exception`, `timeout`
- Explore keywords → normal: `how`, `understand`, `explain`, `explore`, `investigate`, `check`, `review`
- Build/run keywords → normal (summary only): `run`, `build`, `deploy`, `start`, `compile`, `install`

### 7. File Reader (`internal/reader/`)

Smart file reading for `cctxm read`.

- Files ≤ threshold (default 5KB): return full content
- Files > threshold: extract sections matching task keywords
- Markdown files (`.md`): always return full content (Jira tasks, specs, etc.)
- Configurable per file extension

### 8. Rules Injector (`internal/rules/`)

Manages `.github/copilot/` directory.

`cctxm init` / `cctxm rules inject`:
1. Backs up original `.github/copilot/` → `.cctxm/overridden/`
2. Generates `.github/copilot/instructions.md` containing:
   - Core cctxm usage instructions (always use `cctxm exec`, `cctxm read`, etc.)
   - Session management instructions
   - User-defined rules from `.cctxm/rules/*.md`
3. Adds `.github/copilot/` to `.git/info/exclude`

`cctxm rules restore`:
1. Restores original `.github/copilot/` from backup

## Data Flow Example

```
User → Copilot: "fix the auth token refresh bug in input-service"

Copilot runs: cctxm session start "fix auth token refresh bug in input-service"
  → Creates session s_20260503_2252
  → Detects keywords: ["fix", "bug"] → strict mode
  → Task keywords: ["auth", "token", "refresh", "input-service"]
  → Sets as active session

Copilot runs: cctxm exec --project input-service -- docker logs auth-svc
  → Router: cwd = /workspace/services/input-service
  → Executor: runs `docker logs auth-svc`, writes to 001-docker-logs.raw.log
  → Classifier: command matches "docker logs" → docker filter
  → Filter (strict mode):
      - Scans for ERROR/WARN level entries
      - Scans for lines containing "auth", "token", "refresh"
      - Extracts matching lines + 3 lines context
      - Writes to 001-docker-logs.filtered.log
  → Returns ~15 relevant lines instead of 2000+

Copilot runs: cctxm exec --project input-service -- npm test
  → Router: cwd = /workspace/services/input-service
  → Executor: runs `npm test`, writes to 002-npm-test.raw.log
  → Classifier: command matches "npm test" → test filter
  → Filter (strict mode):
      - Parses Jest output
      - 50/52 passed, 2 failed
      - Extracts: summary + 2 failed test details
      - Writes to 002-npm-test.filtered.log
  → Returns: "50/52 passed. 2 FAILED:" + failure details
```

## Cross-Platform Considerations

- Go compiles to native binaries for linux/amd64, darwin/amd64, darwin/arm64, windows/amd64
- Shell command execution uses `os/exec` with platform-appropriate shell (`sh -c` on Unix, `cmd /C` on Windows)
- File paths use `filepath.Join` for OS-correct separators
- Temp/session files use `os.TempDir()` fallback if `.cctxm/` is not writable
- No dependency on bash, python, or any runtime
