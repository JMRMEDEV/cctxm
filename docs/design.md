# CCTXM — Design Decisions

## Language: Go

**Decision:** Go as the implementation language.

**Rationale:**
- Single binary distribution — no runtime dependencies
- Native cross-compilation (`GOOS`/`GOARCH`)
- Fast startup time (critical since every Copilot command invocation spawns cctxm)
- Strong stdlib for process execution, file I/O, and JSON/YAML parsing
- Cobra CLI framework is mature and well-documented

**Alternatives considered:**
- Node.js/TypeScript: Requires Node runtime on target machine. Slower startup.
- Rust: Faster but slower development velocity. Overkill for I/O-bound CLI tool.
- Python: Requires Python runtime. Packaging is painful cross-platform.

## Configuration: YAML

**Decision:** YAML as the primary config format.

**Rationale:**
- Supports comments (critical for annotating project mappings and rules)
- Cleaner readability for nested structures
- Standard in DevOps tooling (Docker Compose, K8s, GitHub Actions)
- Go has mature YAML parsing (`gopkg.in/yaml.v3`)

JSON is accepted as an alternative (auto-detected by file extension).

## Session Management: Hybrid Explicit + Implicit

**Decision:** Sessions are explicitly created via `cctxm session start` but implicitly tracked via `state.json` active session.

**Rationale:**
- Copilot has no native session ID we can hook into
- Explicit creation via injected instructions is reliable (Copilot follows system instructions well)
- `state.json` active session acts as fallback when `--session` flag is omitted
- Daily default session prevents data loss if session management is skipped entirely

## Output Filtering: Heuristic, Not LLM

**Decision:** All filtering is done via pattern matching, parsing, and heuristics. No LLM calls within cctxm.

**Rationale:**
- The whole point is to reduce LLM token usage — calling an LLM to filter would be counterproductive
- Test runner output is structured and parseable
- Docker logs (especially JSON format) are structured
- Error patterns (stack traces, ERROR/WARN levels) are well-defined
- Keyword matching from task context covers the remaining cases
- Keeps the tool fast and free of API dependencies

## Filter Mode Detection: Keyword-Based

**Decision:** Filter aggressiveness is determined by keywords in the task description.

**Keyword → Mode mapping:**

| Keywords | Mode | Behavior |
|---|---|---|
| `error`, `fix`, `bug`, `issue`, `debug`, `fail`, `crash`, `broken`, `wrong`, `exception`, `timeout`, `500`, `404` | `strict` | Errors + warnings + keyword matches only |
| `how`, `understand`, `explain`, `explore`, `investigate`, `check`, `review`, `what does`, `look at` | `normal` | Strict + surrounding context lines |
| `run`, `build`, `deploy`, `start`, `compile`, `install`, `setup`, `init` | `normal` | Summary-focused |
| (no task set) | `normal` | Default behavior |

Mode can always be overridden per-command with `--strict`, `--normal`, `--verbose` flags.

## File Read Threshold: 5KB Default

**Decision:** Files ≤ 5KB are returned in full. Larger files are filtered by keyword.

**Exceptions:**
- `.md` files: Always returned in full regardless of size (Jira tasks, specs, READMEs are high-value context)
- Configurable per extension in `config.yaml`

**Rationale:**
- 5KB ≈ ~1500 tokens. Small enough to not waste context, large enough to cover most config files, small source files, and task descriptions.
- Markdown files are typically written for human consumption and are information-dense. Filtering them loses structure and meaning.

## Rules Injection: File Replacement

**Decision:** Override `.github/copilot/` by replacing its contents, not by intercepting file reads.

**Rationale:**
- Intercepting `cat`/file reads would require wrapping every possible file access command — fragile
- Direct file replacement is simple and reliable
- Backup + restore mechanism makes it reversible
- `.git/info/exclude` prevents accidental commits
- Copilot reads these files at chat start, so replacement is sufficient

## Project Routing: Config-Based Mapping

**Decision:** Project names map to directories via `config.yaml`, not auto-discovery.

**Rationale:**
- Explicit mapping is unambiguous
- Workspace layouts vary wildly — auto-discovery would need heuristics for monorepos, polyrepos, nested projects
- Config is written once and rarely changes
- Supports aliases (e.g., `input` → `./services/input-service`)

## Command Classification: Command Pattern Matching

**Decision:** Output type is detected from the command string, not from output content.

**Rationale:**
- Faster — no need to buffer and analyze output before choosing a filter
- More reliable — `npm test` is always a test command regardless of output format
- Allows streaming — filter can process output line-by-line as it arrives
- Fallback to generic filter for unrecognized commands

**Pattern matching examples:**
```
docker logs *          → docker filter
docker-compose logs *  → docker filter
npm test *             → test filter (jest)
npx jest *             → test filter (jest)
pytest *               → test filter (pytest)
python -m pytest *     → test filter (pytest)
go test *              → test filter (go)
mvn test *             → test filter (maven)
gradle test *          → test filter (gradle)
npm run build *        → build filter
go build *             → build filter
*                      → generic filter
```
