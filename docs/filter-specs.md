# CCTXM — Filter Specifications

## Overview

The filter engine is the core value of cctxm. It reduces command output to only what's relevant, saving LLM context tokens and credits.

## Filter Pipeline

```
Raw Output → Classifier → Filter Strategy → Mode Limiter → Filtered Output
```

1. **Classifier**: Determines output type from the command string
2. **Filter Strategy**: Applies type-specific parsing and extraction
3. **Mode Limiter**: Applies strict/normal/verbose limits based on task context

## Output Classifiers

Detection is based on command pattern matching (checked in order):

| Pattern | Type |
|---|---|
| `docker logs *`, `docker-compose logs *`, `podman logs *` | `docker` |
| `npm test`, `npx jest`, `npx vitest`, `yarn test` | `test-jest` |
| `pytest`, `python -m pytest` | `test-pytest` |
| `python -m unittest`, `python -m nose2` | `test-python` |
| `go test` | `test-go` |
| `mvn test`, `mvn verify`, `mvnw test` | `test-maven` |
| `gradle test`, `gradlew test` | `test-gradle` |
| `dotnet test` | `test-dotnet` |
| `npx cypress`, `npx playwright` | `test-e2e` |
| `npm run build`, `go build`, `mvn package`, `gradle build` | `build` |
| `npm install`, `pip install`, `go mod download` | `install` |
| `*` | `generic` |

## Filter Strategies

### Docker Log Filter

**JSON structured logs** (detected by first line starting with `{`):
- Parse each line as JSON
- Filter by `level` field: keep `error`, `warn`, `fatal`, `panic`
- Keyword match on `msg`, `message`, `error` fields
- In normal mode: also keep `info` lines matching keywords
- Always keep first 3 lines (startup context) and last 3 lines (recent state)

**Plain text logs:**
- Regex match for error patterns:
  - `ERROR`, `WARN`, `FATAL`, `PANIC` (case-insensitive)
  - `Exception`, `Traceback`, `panic:`, `FAIL`
  - Stack trace patterns: lines starting with `\tat`, `  at `, `    `, `goroutine`
- Keyword match on all lines
- Group consecutive matching lines (don't split stack traces)

**Output format:**
```
[cctxm] Docker logs filtered: showing 23/1847 lines (strict mode, keywords: auth, token)
[cctxm] Full output: .cctxm/sessions/s_.../003-docker-logs.raw.log

<filtered lines here>
```

### Test Runner Filters

All test filters follow the same principle: **summary + failed test details only**.

#### Jest / Vitest
- Detect summary line: `Tests: N failed, N passed, N total`
- If all passed: return only summary line
- If failures: extract each `● <test name>` block until next `●` or end
- Include the `FAIL` file header for each failing test file

#### pytest
- Detect summary line: `N passed, N failed` or `N passed`
- If all passed: return only summary
- If failures: extract `FAILED` section and `short test summary info`
- Include captured output for failed tests only

#### Go test
- Detect `--- FAIL:` blocks
- If all passed: return `ok` lines only
- If failures: extract each `--- FAIL:` block with its output
- Include `FAIL` exit line

#### Maven / Gradle
- Detect `Tests run: N, Failures: N, Errors: N`
- If all passed: return summary only
- If failures: extract `<<< FAILURE!` blocks and stack traces

#### E2E (Cypress / Playwright)
- Detect `✓`/`✗` or `passed`/`failed` markers
- If all passed: return count only
- If failures: extract failed test blocks with error messages and screenshots paths

**Output format:**
```
[cctxm] Test results filtered: 50/52 passed, 2 FAILED (jest)
[cctxm] Full output: .cctxm/sessions/s_.../004-npm-test.raw.log

FAIL src/auth/tokenRefresh.test.ts
  ● TokenRefresh > should handle expired tokens
    Expected: "refreshed"
    Received: "expired"
    at Object.<anonymous> (src/auth/tokenRefresh.test.ts:42:5)

FAIL src/auth/tokenValidation.test.ts
  ● TokenValidation > should reject malformed tokens
    TypeError: Cannot read property 'exp' of undefined
    at validateToken (src/auth/tokenValidation.ts:15:22)
    at Object.<anonymous> (src/auth/tokenValidation.test.ts:28:5)
```

### Build Filter

- If exit code 0: `[cctxm] Build succeeded.`
- If exit code != 0: extract lines containing `error`, `warning`, `Error:`, `failed`
- Include N context lines around errors
- For TypeScript: extract full `TS####:` error blocks
- For Go: extract full `./file.go:line:col:` error blocks

### Install Filter

- If exit code 0: `[cctxm] Install completed. N packages added.` (parse from output)
- If exit code != 0: extract error/warning lines
- Always suppress progress bars, download indicators, and audit reports

### Generic Filter

Fallback for unrecognized commands.

- In strict mode: only lines matching error patterns + task keywords
- In normal mode: last N lines (configurable, default 200) + any error lines from earlier
- In verbose mode: full output up to max cap (default 1000 lines)
- Always include first 3 lines (command context) and last 10 lines (final state)

## Mode Limiter

After the strategy filter runs, the mode limiter applies final constraints:

| Mode | Max output lines | Context lines around matches |
|---|---|---|
| `strict` | 50 | 3 |
| `normal` | 200 | 10 |
| `verbose` | 1000 | all |

If output still exceeds the limit after filtering, it's truncated from the middle (keep start + end).

## Metadata Header

Every filtered output starts with a metadata line:

```
[cctxm] <type> filtered: <summary> (<mode> mode, keywords: <keywords>)
[cctxm] Full output: <path to raw log>
```

This tells both the user and Copilot that output was filtered and where to find the full version.
