# CCTXM — Copilot Context Manager

Intelligent output filtering middleware for LLM-assisted development. Reduces wasted context tokens and credits when using GitHub Copilot (or any LLM coding agent) by filtering terminal output, smart-reading files, and managing session context.

## Problem

VS Code + Copilot pipes raw, unfiltered terminal output to the model. Running `npm test` with 52 tests? Copilot reads all of it. Docker logs with 2000 lines? All of it. This burns context and credits on irrelevant noise.

## Solution

`cctxm` sits between Copilot and the terminal. It captures full output to temp files and returns only what's relevant — errors, failures, keyword matches — based on your current task.

## Install

```bash
# From source
go install github.com/jmrmedev/cctxm@latest

# Or download binary
curl -sSL https://raw.githubusercontent.com/jmrmedev/cctxm/main/install.sh | sh

# Or Homebrew (macOS/Linux)
brew install jmrmedev/tap/cctxm

# Or Scoop (Windows)
scoop bucket add jmrmedev https://github.com/jmrmedev/scoop-bucket
scoop install cctxm
```

## Quick Start

```bash
# Initialize in your workspace
cctxm init

# Map your projects
cctxm config set projects.api ./services/api
cctxm config set projects.ui ./frontend

# Start a session
cctxm session start "fix auth token refresh bug"

# Run commands (Copilot sees filtered output)
cctxm exec --project api -- docker logs auth-svc
cctxm exec --project api -- npm test

# Smart file reading
cctxm read ./docs/large-api-spec.md
```

## What It Filters

| Command | All Pass | Failures |
|---|---|---|
| `npm test` / `jest` / `vitest` | "52 passed" | Summary + failed test details only |
| `pytest` | "10 passed" | FAILURES section + summary |
| `go test` | "ok" lines | `--- FAIL` blocks only |
| `docker logs` | Errors + keyword matches | Errors + keyword matches |
| `npm run build` | "Build succeeded." | Error lines only |
| `npm install` | "added 150 packages" | Error lines only |

## How It Works

1. **`cctxm init`** injects instructions into `.github/copilot/` telling Copilot to use `cctxm exec` and `cctxm read`
2. **`cctxm session start`** extracts keywords from your task description and auto-detects filter aggressiveness
3. **`cctxm exec`** runs commands, captures full output to session logs, returns filtered output
4. **`cctxm read`** returns small files in full, extracts relevant sections from large files

## Filter Modes

Automatically detected from your task description:

- **strict** — triggered by: fix, bug, error, debug, crash, fail → only errors + keyword matches
- **normal** — default, or triggered by: explore, review, investigate → errors + context
- **verbose** — manual override → full output (capped)

## Project Routing

Solves the "Copilot runs commands in the wrong directory" problem:

```yaml
# .cctxm/config.yaml
projects:
  api: ./services/api
  ui: ./frontend
  org: ./project-org
```

```bash
cctxm exec --project api -- npm test    # Runs in ./services/api
cctxm exec --project ui -- npm run build # Runs in ./frontend
```

## Custom Rules

Add your coding standards and conventions to `.cctxm/rules/`:

```bash
echo "# Always use TypeScript strict mode" > .cctxm/rules/standards.md
cctxm rules inject  # Re-injects into .github/copilot/
```

## Session Management

Sessions track your work context — task, keywords, filter mode, and all command logs:

```bash
cctxm session start "fix auth refresh"  # New session
cctxm session list                       # List all sessions
cctxm session restore <id>              # Resume previous work
cctxm session logs                       # View command history
```

Full raw output is always available in `.cctxm/sessions/` for debugging.

## License

MIT

## Example: Before vs After

### Without cctxm — Copilot sees everything (2000+ tokens wasted):

```
$ npm test

> api@1.0.0 test
> jest --verbose

 PASS src/utils/format.test.ts
  ✓ formats date correctly (3 ms)
  ✓ formats currency (1 ms)
  ✓ handles null input (1 ms)
 PASS src/utils/validate.test.ts
  ✓ validates email (2 ms)
  ✓ validates phone (1 ms)
  ... (48 more passing tests)
 FAIL src/auth/tokenRefresh.test.ts
  ● TokenRefresh › should handle expired tokens
    expect(received).toBe(expected)
    Expected: "refreshed"
    Received: "expired"
      at Object.<anonymous> (src/auth/tokenRefresh.test.ts:42:5)

Tests: 1 failed, 51 passed, 52 total
Time:  4.2s
```

### With cctxm — Copilot sees only what matters (~100 tokens):

```
$ cctxm exec --project api -- npm test

[cctxm] Filtered: showing 8/94 lines (strict mode, keywords: auth, token, refresh)
[cctxm] Test results: 51/52 passed, 1 FAILED (jest)

Tests: 1 failed, 51 passed, 52 total

FAIL src/auth/tokenRefresh.test.ts
  ● TokenRefresh › should handle expired tokens
    expect(received).toBe(expected)
    Expected: "refreshed"
    Received: "expired"
      at Object.<anonymous> (src/auth/tokenRefresh.test.ts:42:5)

[cctxm] Full output: .cctxm/sessions/s_20260503_2252/003-npm-test.raw.log
```

### Docker logs — 1847 lines → 15 relevant lines:

```
$ cctxm exec --project api -- docker logs auth-svc

[cctxm] Filtered: showing 15/1847 lines (strict mode, keywords: auth, token, refresh)
[cctxm] Docker logs (JSON)

{"level":"error","msg":"token refresh failed: invalid grant","service":"auth","timestamp":"2026-05-03T22:15:00Z"}
{"level":"error","msg":"upstream timeout on /oauth/token","service":"auth","timestamp":"2026-05-03T22:15:01Z"}
{"level":"warn","msg":"retry limit reached for token refresh","service":"auth","timestamp":"2026-05-03T22:15:05Z"}

[cctxm] Full output: .cctxm/sessions/s_20260503_2252/004-docker-logs.raw.log
```
