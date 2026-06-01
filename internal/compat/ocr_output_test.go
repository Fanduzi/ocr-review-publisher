package compat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Fanduzi/ocr-review-publisher/internal/ocroutput"
)

// fixtureDir is the directory containing OCR output fixtures.
var fixtureDir = filepath.Join("..", "..", "testdata", "ocr")

// TestCompatibilityFixturesParse walks all fixture files and verifies they parse successfully.
func TestCompatibilityFixturesParse(t *testing.T) {
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("read fixture dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(fixtureDir, name)

		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			result, err := ocroutput.Parse(data)
			if err != nil {
				t.Fatalf("parse fixture %s: %v", name, err)
			}
			if result == nil {
				t.Fatalf("parse returned nil result for %s", name)
			}
		})
	}
}

// TestCompatibilityFixturesHaveExpectedShape verifies parsed results have the expected structure.
func TestCompatibilityFixturesHaveExpectedShape(t *testing.T) {
	t.Run("basic.json yields status and comments", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(fixtureDir, "basic.json"))
		if err != nil {
			t.Fatal(err)
		}
		result, err := ocroutput.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status == "" {
			t.Error("expected non-empty status")
		}
		if len(result.Findings) == 0 {
			t.Error("expected at least one finding")
		}
		if result.Findings[0].Path == "" {
			t.Error("expected finding to have a path")
		}
		if result.Findings[0].Content == "" {
			t.Error("expected finding to have content")
		}
	})

	t.Run("prefixed output parses", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(fixtureDir, "prefixed-agent-output.txt"))
		if err != nil {
			t.Fatal(err)
		}
		result, err := ocroutput.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if result.Status != "success" {
			t.Errorf("expected status success, got %q", result.Status)
		}
		if len(result.Findings) != 1 {
			t.Errorf("expected 1 finding, got %d", len(result.Findings))
		}
	})

	t.Run("empty-comments.json yields empty findings", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(fixtureDir, "empty-comments.json"))
		if err != nil {
			t.Fatal(err)
		}
		result, err := ocroutput.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if result.Findings == nil {
			t.Error("expected non-nil empty findings slice")
		}
		if len(result.Findings) != 0 {
			t.Errorf("expected 0 findings, got %d", len(result.Findings))
		}
		if result.Message == "" {
			t.Error("expected message to be preserved")
		}
	})

	t.Run("with-warnings.json yields warnings", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(fixtureDir, "with-warnings.json"))
		if err != nil {
			t.Fatal(err)
		}
		result, err := ocroutput.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Warnings) == 0 {
			t.Error("expected at least one warning")
		}
		if result.Warnings[0].Type == "" {
			t.Error("expected warning to have a type")
		}
	})
}

// TestCompatibilityFixturesDoNotContainSecrets scans fixtures for forbidden patterns.
func TestCompatibilityFixturesDoNotContainSecrets(t *testing.T) {
	forbidden := []string{
		"localhost:8929",
		"/Users/fan",
		"PRIVATE-TOKEN",
		"Authorization: Bearer",
		"fm2Z",
	}

	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("read fixture dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(fixtureDir, name)

		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		content := string(data)
		for _, pattern := range forbidden {
			if strings.Contains(content, pattern) {
				t.Errorf("fixture %s contains forbidden pattern %q", name, pattern)
			}
		}
	}
}

// TestFutureFieldsFixtureStillParses verifies category/severity mapping and unknown future field metadata.
func TestFutureFieldsFixtureStillParses(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(fixtureDir, "future-fields.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := ocroutput.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}

	f := result.Findings[0]
	if f.Category != "security" {
		t.Errorf("expected category security, got %q", f.Category)
	}
	if f.Severity != "high" {
		t.Errorf("expected severity high, got %q", f.Severity)
	}
	if f.Metadata == nil {
		t.Fatal("expected metadata for future fields")
	}
	if f.Metadata["confidence"] != 0.95 {
		t.Errorf("expected confidence 0.95, got %v", f.Metadata["confidence"])
	}
	if f.Metadata["rule_id"] != "sql-injection-001" {
		t.Errorf("expected rule_id sql-injection-001, got %v", f.Metadata["rule_id"])
	}
}

// TestMalformedCompatibilityFixtureRejected verifies malformed JSON is rejected.
func TestMalformedCompatibilityFixtureRejected(t *testing.T) {
	_, err := ocroutput.Parse([]byte(`{"status": "success", "comments": [}`))
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if !strings.Contains(err.Error(), "json") && !strings.Contains(err.Error(), "JSON") && !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention json/parse, got: %s", err.Error())
	}
}
