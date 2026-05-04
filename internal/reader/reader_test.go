package reader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadSmallFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "small.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	out, err := Read(path, nil, DefaultThreshold, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hello world") {
		t.Error("should contain full content")
	}
	if !strings.Contains(out, "full") {
		t.Error("should say full read")
	}
}

func TestReadMarkdownAlwaysFull(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.md")
	// Create a file larger than threshold
	content := strings.Repeat("line of markdown content\n", 500)
	os.WriteFile(path, []byte(content), 0644)

	out, err := Read(path, nil, 100, false, "") // threshold=100 bytes
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "full") {
		t.Error("markdown should always be full read")
	}
}

func TestReadLargeFileNoKeywords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.log")
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d: some content", i)
	}
	os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)

	out, err := Read(path, nil, 100, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "filtered") {
		t.Error("should be filtered")
	}
	if !strings.Contains(out, "lines omitted") {
		t.Error("should indicate omitted lines")
	}
}

func TestReadLargeFileWithKeywords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.log")
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d: normal content", i)
	}
	lines[100] = "ERROR: auth token refresh failed"
	os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)

	out, err := Read(path, []string{"auth", "token"}, 100, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "auth token refresh") {
		t.Error("should contain matching line")
	}
	if !strings.Contains(out, "filtered") {
		t.Error("should be filtered")
	}
}

func TestReadLargeFileWithSearchTerms(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.log")
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d: normal content", i)
	}
	lines[50] = "database connection timeout"
	os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)

	out, err := Read(path, nil, 100, false, "database timeout")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "database connection timeout") {
		t.Error("should contain matching line")
	}
}

func TestReadForceFullRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.log")
	content := strings.Repeat("x\n", 1000)
	os.WriteFile(path, []byte(content), 0644)

	out, err := Read(path, nil, 100, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "full") {
		t.Error("force full should say full")
	}
}

func TestReadFileNotFound(t *testing.T) {
	_, err := Read("/nonexistent/file.txt", nil, 0, false, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadDirectory(t *testing.T) {
	_, err := Read(t.TempDir(), nil, 0, false, "")
	if err == nil {
		t.Fatal("expected error for directory")
	}
}

func TestReadNoKeywordMatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.log")
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d: normal content", i)
	}
	os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)

	out, err := Read(path, []string{"nonexistent_keyword"}, 100, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "no keyword matches") {
		t.Error("should indicate no matches found")
	}
}

func TestReadYAMLAlwaysFull(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := strings.Repeat("key: value\n", 500)
	os.WriteFile(path, []byte(content), 0644)

	out, err := Read(path, nil, 100, false, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "full") {
		t.Error("yaml should always be full read")
	}
}
