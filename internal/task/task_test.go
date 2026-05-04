package task

import (
	"testing"
)

func TestDetectModeStrict(t *testing.T) {
	cases := []string{
		"fix auth token refresh",
		"debug the login flow",
		"there's a bug in the payment service",
		"error when submitting form",
		"crash on startup",
		"failing unit tests",
		"timeout on API call",
	}
	for _, c := range cases {
		if got := DetectMode(c); got != "strict" {
			t.Errorf("DetectMode(%q) = %q, want strict", c, got)
		}
	}
}

func TestDetectModeNormal(t *testing.T) {
	cases := []string{
		"how does the auth module work",
		"review the payment service",
		"explore the codebase",
		"investigate slow queries",
		"add pagination to users endpoint",
		"implement new feature",
	}
	for _, c := range cases {
		if got := DetectMode(c); got != "normal" {
			t.Errorf("DetectMode(%q) = %q, want normal", c, got)
		}
	}
}

func TestExtractKeywords(t *testing.T) {
	kw := ExtractKeywords("fix auth token refresh failing in input-service")
	// Should contain: auth, token, refresh, input-service
	// Should NOT contain: fix, failing (debug keywords), in (stop word)
	found := map[string]bool{}
	for _, k := range kw {
		found[k] = true
	}

	expected := []string{"auth", "token", "refresh", "input-service"}
	for _, e := range expected {
		if !found[e] {
			t.Errorf("expected keyword '%s', got %v", e, kw)
		}
	}

	notExpected := []string{"fix", "in", "failing"}
	for _, ne := range notExpected {
		if found[ne] {
			t.Errorf("did not expect keyword '%s' in %v", ne, kw)
		}
	}
}

func TestExtractKeywordsDedup(t *testing.T) {
	kw := ExtractKeywords("auth auth auth token token")
	if len(kw) != 2 {
		t.Errorf("expected 2 unique keywords, got %d: %v", len(kw), kw)
	}
}

func TestExtractKeywordsEmpty(t *testing.T) {
	kw := ExtractKeywords("")
	if len(kw) != 0 {
		t.Errorf("expected 0 keywords, got %v", kw)
	}
}

func TestExtractKeywordsStopWordsOnly(t *testing.T) {
	kw := ExtractKeywords("the a an in on at to for")
	if len(kw) != 0 {
		t.Errorf("expected 0 keywords, got %v", kw)
	}
}
