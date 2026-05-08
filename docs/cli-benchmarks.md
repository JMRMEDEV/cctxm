# CCTXM — Copilot CLI Benchmarks

> Tested: 2026-05-08 | Copilot CLI v1.0.43 | Model: gpt-5-mini | cctxm dev build

## Setup

- Test workspace: `~/repos/cctxm-tests`
- Instructions injected at `.github/copilot-instructions.md`
- Active session with task: "fix auth token refresh bug" (strict mode, keywords: auth, token, refresh)
- Copilot CLI invoked with: `copilot -p "<prompt>" --allow-all-tools --allow-all --model gpt-5-mini`

## Results

### Test 1: Jest Test Filtering

**Prompt:** `"Run the tests using cctxm exec and tell me what's failing"`

| Metric | Value |
|---|---|
| Raw output lines | 155 |
| Filtered lines shown to Copilot | 16 |
| Reduction | 90% |
| Copilot used `cctxm exec` | ✅ |
| Correctly identified failures | ✅ (4/4 failing tests named) |
| Referenced raw log path | ✅ |
| Premium requests used | 1 |
| Time | 16s |

**Copilot output:**
> Test run result: 4 failed, 151 passed (155 total).
> Failing tests (tests/auth.test.js):
> - Auth Token Refresh › should refresh expired token
> - Auth Token Refresh › should handle invalid token format
> - Auth Token Refresh › should validate token expiry
> - Auth Token Validation › should reject malformed token

---

### Test 2: Docker JSON Log Filtering

**Prompt:** `"Check the docker logs for auth errors by running: cctxm exec -- bash scripts/docker-json-logs.sh"`

| Metric | Value |
|---|---|
| Raw output lines | 500 |
| Filtered lines shown to Copilot | 58 |
| Reduction | 88% |
| Copilot used `cctxm exec` | ✅ |
| Correctly identified errors | ✅ (token timeout + expiry warnings) |
| Referenced raw log path | ✅ |
| Premium requests used | 1 |
| Time | 17s |

**Copilot output:**
> Found auth-related entries: multiple "Token expiry approaching" warnings and repeated errors: "Failed to refresh auth token: connection timeout" (trace_ids abc-50, abc-450, abc-500).

---

### Test 3: Smart File Reading (2416-line file)

**Prompt:** `"Read the large auth service file using cctxm read and find the token refresh logic"`

| Metric | Value |
|---|---|
| File size | 2416 lines (61KB) |
| Lines shown to Copilot | ~30 (keyword-filtered section) |
| Reduction | 99% |
| Copilot used `cctxm read` | ✅ |
| Found target code | ✅ (refreshToken method extracted) |
| Premium requests used | 1 |
| Time | 73s |

**Copilot output:**
> Found in src/large-auth-service.ts — the token refresh logic:
> ```
> async refreshToken(userId: string): Promise<Token> {
>   const existing = await this.tokenStore.get(userId);
>   if (!existing || existing.exp < Date.now()) {
>     throw new TokenExpiredError('Token expired for user: ' + userId);
>   }
>   const newToken = await this.issuer.refresh(existing);
>   await this.tokenStore.set(userId, newToken);
>   return newToken;
> }
> ```

---

### Test 4: Task Mode Switching (Verbose vs Strict)

**Prompt:** `"Switch task mode to verbose, run tests with --project, then switch to strict and run again"`

| Metric | Verbose | Strict |
|---|---|---|
| Lines shown to Copilot | 87 | 15 |
| Reduction vs raw (155 lines) | 44% | 90% |
| `--project` flag worked | ✅ | ✅ |
| `task mode` switch worked | ✅ | ✅ |
| Premium requests used | 1 | — |
| Time | 81s | — |

**Copilot output:**
> Verbose run lines: 87. Strict run lines: 15. Verbose output had 72 more lines. Verbose mode shows full failure traces and context; strict mode shows a concise summary of failures.

---

### Test 5: Docker Plain Text Log Filter (No Task Keywords)

**Prompt:** `"Clear task, then run docker plain logs script"`

| Metric | Value |
|---|---|
| Raw output lines | 532 |
| Filtered lines shown to Copilot | 201 |
| Reduction | 62% |
| Filter mode | normal (no keywords, default fallback) |
| Premium requests used | 1 |
| Time | 38s |

Note: With no task keywords, the generic filter uses normal mode (200-line cap). With keywords set, the same output was filtered to 51 lines (90% reduction).

---

### Test 6: Session Management (Create, List, Restore, Clean)

**Prompt:** `"List sessions, restore the old one, show details"`

| Action | Result |
|---|---|
| `session start` | ✅ Created with auto-detected keywords + mode |
| `session list` | ✅ Showed all sessions with metadata |
| `session restore` | ✅ Switched active session, task context reloaded |
| `session clean --all` | ✅ Removed 2 old sessions, preserved active |
| Commands logged | ✅ All `cctxm exec` calls recorded in session |
| Premium requests used | 1 per prompt |

---

### Test 7: Rules Restore + Re-inject

**Prompt:** `"Restore rules, check if file is gone, then inject again"`

| Action | Result |
|---|---|
| `rules restore` | ✅ Removed `.github/copilot/instructions.md` |
| File check after restore | FILE_MISSING (confirmed) |
| `rules inject` | ✅ Re-created instructions with hook rules |
| Premium requests used | 1 |
| Time | 50s |

---

## Summary

| Scenario | Raw Lines | Filtered Lines | Reduction | Copilot Accuracy |
|---|---|---|---|---|
| Jest (155 tests) | 155 | 16 | 90% | 100% (4/4 failures) |
| Docker JSON logs | 500 | 58 | 88% | 100% (all errors found) |
| Large file read | 2416 | ~30 | 99% | 100% (target code found) |
| Jest (verbose mode) | 155 | 87 | 44% | 100% |
| Jest (strict mode) | 155 | 15 | 90% | 100% |
| Docker plain (no keywords) | 532 | 201 | 62% | 100% |

## Observations

1. **Instruction adherence**: Copilot CLI consistently used `cctxm exec` and `cctxm read` as instructed via `.github/copilot-instructions.md`.
2. **Context savings**: 62–99% reduction depending on mode and keywords. Strict mode with keywords yields the best savings (~90%).
3. **Accuracy preserved**: Despite heavy filtering, Copilot correctly identified all failures, errors, and target code in every test.
4. **Session continuity**: All commands logged to the active cctxm session with raw logs preserved for reference. Session restore correctly reloads task context and filter mode.
5. **Mode switching**: Changing between strict/normal/verbose mid-session works correctly and immediately affects filter output.
6. **Project routing**: `--project` flag correctly resolves directory from config mappings.
7. **Keywords matter**: With task keywords set (strict mode), Docker logs went from 532→51 lines. Without keywords (normal mode), same output was 532→201 lines. Setting a task description is critical for maximum context savings.

## Notes

- Copilot CLI reads `.github/copilot-instructions.md` automatically (no extra config needed).
- The `--allow-all-tools --allow-all` flags are needed for non-interactive mode to avoid permission prompts.
- Classic PATs (`ghp_`) are not supported by Copilot CLI; use fine-grained PATs or `gh auth login`.
- Model used was `gpt-5-mini` (included in free tier at time of testing).
