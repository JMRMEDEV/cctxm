package filter

import (
	"strings"
	"testing"
)

// --- Classifier tests ---

func TestClassifyDocker(t *testing.T) {
	cases := []struct{ cmd string; want OutputType }{
		{"docker logs my-app", TypeDocker},
		{"docker-compose logs", TypeDocker},
		{"docker compose logs -f app", TypeDocker},
		{"podman logs container", TypeDocker},
	}
	for _, c := range cases {
		if got := Classify(c.cmd); got != c.want {
			t.Errorf("Classify(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}

func TestClassifyTestRunners(t *testing.T) {
	cases := []struct{ cmd string; want OutputType }{
		{"npm test", TypeTestJest},
		{"npx jest --coverage", TypeTestJest},
		{"npx vitest run", TypeTestJest},
		{"pytest tests/", TypeTestPytest},
		{"python -m pytest -v", TypeTestPytest},
		{"go test ./...", TypeTestGo},
		{"mvn test", TypeTestMaven},
		{"gradle test", TypeTestGradle},
		{"./gradlew test", TypeTestGradle},
		{"dotnet test", TypeTestDotnet},
		{"npx cypress run", TypeTestE2E},
		{"npx playwright test", TypeTestE2E},
	}
	for _, c := range cases {
		if got := Classify(c.cmd); got != c.want {
			t.Errorf("Classify(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}

func TestClassifyBuildInstall(t *testing.T) {
	cases := []struct{ cmd string; want OutputType }{
		{"npm run build", TypeBuild},
		{"go build ./...", TypeBuild},
		{"npm install", TypeInstall},
		{"pip install -r requirements.txt", TypeInstall},
	}
	for _, c := range cases {
		if got := Classify(c.cmd); got != c.want {
			t.Errorf("Classify(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}

func TestClassifyGeneric(t *testing.T) {
	if got := Classify("ls -la"); got != TypeGeneric {
		t.Errorf("expected TypeGeneric, got %v", got)
	}
}

// --- Mode tests ---

func TestParseMode(t *testing.T) {
	if ParseMode("strict") != ModeStrict { t.Error("strict") }
	if ParseMode("verbose") != ModeVerbose { t.Error("verbose") }
	if ParseMode("normal") != ModeNormal { t.Error("normal") }
	if ParseMode("anything") != ModeNormal { t.Error("default") }
}

// --- Helper tests ---

func TestMatchesKeyword(t *testing.T) {
	if !MatchesKeyword("auth token expired", []string{"auth", "token"}) {
		t.Error("should match")
	}
	if MatchesKeyword("hello world", []string{"auth"}) {
		t.Error("should not match")
	}
}

func TestIsErrorLine(t *testing.T) {
	if !IsErrorLine("ERROR: something failed") { t.Error("ERROR") }
	if !IsErrorLine("Warning: deprecated") { t.Error("Warning") }
	if !IsErrorLine("java.lang.NullPointerException") { t.Error("Exception") }
	if IsErrorLine("all good here") { t.Error("false positive") }
}

func TestIsStackTraceLine(t *testing.T) {
	if !IsStackTraceLine("    at Object.<anonymous> (file.js:10:5)") { t.Error("js stack") }
	if !IsStackTraceLine("\tat com.example.Main.run(Main.java:42)") { t.Error("java stack") }
	if IsStackTraceLine("normal log line") { t.Error("false positive") }
}

// --- Docker filter tests ---

func TestDockerFilterJSON(t *testing.T) {
	output := `{"level":"info","msg":"server started","port":8080}
{"level":"info","msg":"handling request","path":"/api"}
{"level":"error","msg":"auth token expired","user":"john"}
{"level":"info","msg":"request complete","status":200}
{"level":"warn","msg":"rate limit approaching","service":"auth"}`

	f := &DockerFilter{}
	result := f.Apply(output, ModeStrict, []string{"auth"})

	if !strings.Contains(result.Output, "auth token expired") {
		t.Error("should contain error about auth")
	}
	if !strings.Contains(result.Output, "rate limit") {
		t.Error("should contain warning")
	}
}

func TestDockerFilterPlain(t *testing.T) {
	output := `2024-01-01 INFO Starting server
2024-01-01 INFO Listening on :8080
2024-01-01 INFO Request received
2024-01-01 ERROR Connection refused to database
2024-01-01 INFO Retrying...
2024-01-01 WARN Slow query detected`

	f := &DockerFilter{}
	result := f.Apply(output, ModeStrict, nil)

	if !strings.Contains(result.Output, "ERROR") {
		t.Error("should contain ERROR line")
	}
	if !strings.Contains(result.Output, "WARN") {
		t.Error("should contain WARN line")
	}
}

// --- Jest filter tests ---

func TestJestFilterAllPass(t *testing.T) {
	output := `PASS src/utils.test.ts
PASS src/auth.test.ts
Tests: 52 passed, 52 total`

	f := &JestFilter{}
	result := f.Apply(output, ModeNormal, nil)

	if !strings.Contains(result.Output, "52 passed") {
		t.Error("should contain summary")
	}
	if result.Lines != 1 {
		t.Errorf("expected 1 line for all-pass, got %d", result.Lines)
	}
}

func TestJestFilterWithFailures(t *testing.T) {
	output := `PASS src/utils.test.ts
FAIL src/auth/tokenRefresh.test.ts
  ● TokenRefresh > should handle expired tokens
    Expected: "refreshed"
    Received: "expired"
    at Object.<anonymous> (src/auth/tokenRefresh.test.ts:42:5)

PASS src/other.test.ts
Tests: 1 failed, 51 passed, 52 total`

	f := &JestFilter{}
	result := f.Apply(output, ModeStrict, nil)

	if !strings.Contains(result.Output, "FAIL") {
		t.Error("should contain FAIL block")
	}
	if !strings.Contains(result.Output, "tokenRefresh") {
		t.Error("should contain failing test name")
	}
	if !strings.Contains(result.Output, "1 failed") {
		t.Error("should contain summary")
	}
}

// --- pytest filter tests ---

func TestPytestFilterAllPass(t *testing.T) {
	output := `tests/test_auth.py::test_login PASSED
tests/test_auth.py::test_logout PASSED
========================= 2 passed in 0.5s =========================`

	f := &PytestFilter{}
	result := f.Apply(output, ModeNormal, nil)

	if !strings.Contains(result.Output, "2 passed") {
		t.Error("should contain summary")
	}
	if result.Lines != 1 {
		t.Errorf("expected 1 line, got %d", result.Lines)
	}
}

func TestPytestFilterWithFailures(t *testing.T) {
	output := `tests/test_auth.py::test_login PASSED
tests/test_auth.py::test_refresh FAILED
================================ FAILURES ================================
___________________________ test_refresh ___________________________
    def test_refresh():
>       assert refresh_token() == "new_token"
E       AssertionError: assert None == "new_token"
========================= short test summary info =========================
FAILED tests/test_auth.py::test_refresh
========================= 1 failed, 1 passed in 0.5s =========================`

	f := &PytestFilter{}
	result := f.Apply(output, ModeStrict, nil)

	if !strings.Contains(result.Output, "FAILURES") {
		t.Error("should contain FAILURES section")
	}
	if !strings.Contains(result.Output, "test_refresh") {
		t.Error("should contain failing test")
	}
}

// --- Go test filter tests ---

func TestGoTestFilterAllPass(t *testing.T) {
	output := `ok  	github.com/example/pkg1	0.005s
ok  	github.com/example/pkg2	0.012s`

	f := &GoTestFilter{}
	result := f.Apply(output, ModeNormal, nil)

	if !strings.Contains(result.Summary, "2 packages") {
		t.Errorf("expected 2 packages in summary, got '%s'", result.Summary)
	}
}

func TestGoTestFilterWithFailures(t *testing.T) {
	output := `--- FAIL: TestAuth (0.00s)
    auth_test.go:15: expected "valid", got "invalid"
FAIL	github.com/example/auth	0.005s
ok  	github.com/example/utils	0.003s`

	f := &GoTestFilter{}
	result := f.Apply(output, ModeStrict, nil)

	if !strings.Contains(result.Output, "FAIL: TestAuth") {
		t.Error("should contain failing test")
	}
}

// --- Build filter tests ---

func TestBuildFilterSuccess(t *testing.T) {
	output := `Compiling main.go
Linking...
Done.`

	f := &BuildFilter{}
	result := f.Apply(output, ModeNormal, nil)

	if result.Output != "Build succeeded." {
		t.Errorf("expected 'Build succeeded.', got '%s'", result.Output)
	}
}

func TestBuildFilterWithErrors(t *testing.T) {
	output := `src/main.ts:15:3 - error TS2304: Cannot find name 'foo'.
src/main.ts:20:1 - error TS2322: Type 'string' is not assignable.`

	f := &BuildFilter{}
	result := f.Apply(output, ModeStrict, nil)

	if !strings.Contains(result.Output, "TS2304") {
		t.Error("should contain TS error")
	}
}

// --- Install filter tests ---

func TestInstallFilterSuccess(t *testing.T) {
	output := `npm warn deprecated package@1.0
added 150 packages in 5s`

	f := &InstallFilter{}
	result := f.Apply(output, ModeNormal, nil)

	if !strings.Contains(result.Output, "added 150 packages") {
		t.Errorf("expected package count, got '%s'", result.Output)
	}
}

// --- Generic filter tests ---

func TestGenericFilterWithKeywords(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "normal log line"
	}
	lines[50] = "ERROR: auth token refresh failed"
	output := strings.Join(lines, "\n")

	f := &GenericFilter{}
	result := f.Apply(output, ModeStrict, []string{"auth"})

	if !strings.Contains(result.Output, "auth token refresh") {
		t.Error("should contain matching line")
	}
	if result.Lines >= 100 {
		t.Error("should be filtered down")
	}
}

func TestGenericFilterNoMatches(t *testing.T) {
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = "normal log line"
	}
	output := strings.Join(lines, "\n")

	f := &GenericFilter{}
	result := f.Apply(output, ModeNormal, []string{"nonexistent"})

	// Should return full output since it's small
	if result.Lines != 10 {
		t.Errorf("expected 10 lines (full), got %d", result.Lines)
	}
}

// --- Apply integration test ---

func TestApplyEndToEnd(t *testing.T) {
	output := `PASS src/utils.test.ts
Tests: 10 passed, 10 total`

	result := Apply("npm test", output, ModeNormal, nil)

	if !strings.Contains(result.Output, "[cctxm]") {
		t.Error("should have cctxm header")
	}
	if !strings.Contains(result.Output, "10 passed") {
		t.Error("should contain summary")
	}
}
