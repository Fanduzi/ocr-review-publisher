package ocroutput

import (
	"os"
	"strings"
	"testing"

	"github.com/Fanduzi/ocr-review-publisher/internal/review"
)

func TestParse_CleanJSON(t *testing.T) {
	data, err := os.ReadFile("../../testdata/ocr/basic.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Status: got %q, want %q", result.Status, "success")
	}
	if result.Message != "Review completed" {
		t.Errorf("Message: got %q, want %q", result.Message, "Review completed")
	}
	if len(result.Findings) != 2 {
		t.Fatalf("Findings count: got %d, want 2", len(result.Findings))
	}

	f0 := result.Findings[0]
	if f0.Path != "service/user.go" {
		t.Errorf("Finding[0].Path: got %q, want %q", f0.Path, "service/user.go")
	}
	if f0.Content != "Error return value of `fmt.Println` is not checked" {
		t.Errorf("Finding[0].Content: got %q", f0.Content)
	}
	if f0.ExistingCode != "fmt.Println(err)" {
		t.Errorf("Finding[0].ExistingCode: got %q", f0.ExistingCode)
	}
	if !strings.Contains(f0.SuggestionCode, "fmt.Errorf") {
		t.Errorf("Finding[0].SuggestionCode should contain 'fmt.Errorf', got %q", f0.SuggestionCode)
	}
	if f0.StartLine != 37 {
		t.Errorf("Finding[0].StartLine: got %d, want 37", f0.StartLine)
	}
	if f0.EndLine != 37 {
		t.Errorf("Finding[0].EndLine: got %d, want 37", f0.EndLine)
	}
	if f0.Thinking == "" {
		t.Errorf("Finding[0].Thinking should not be empty")
	}

	f1 := result.Findings[1]
	if f1.Path != "cmd/main.go" {
		t.Errorf("Finding[1].Path: got %q, want %q", f1.Path, "cmd/main.go")
	}
	if f1.StartLine != 12 {
		t.Errorf("Finding[1].StartLine: got %d, want 12", f1.StartLine)
	}
	if f1.EndLine != 20 {
		t.Errorf("Finding[1].EndLine: got %d, want 20", f1.EndLine)
	}
}

func TestParse_PrefixedAgentOutput(t *testing.T) {
	data, err := os.ReadFile("../../testdata/ocr/prefixed-agent-output.txt")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Status: got %q, want %q", result.Status, "success")
	}
	if result.Message != "1 finding" {
		t.Errorf("Message: got %q, want %q", result.Message, "1 finding")
	}
	if len(result.Findings) != 1 {
		t.Fatalf("Findings count: got %d, want 1", len(result.Findings))
	}
	if result.Findings[0].Path != "main.go" {
		t.Errorf("Path: got %q, want %q", result.Findings[0].Path, "main.go")
	}
}

func TestParse_EmptyComments(t *testing.T) {
	data, err := os.ReadFile("../../testdata/ocr/empty-comments.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if result.Status != "success" {
		t.Errorf("Status: got %q, want %q", result.Status, "success")
	}
	if result.Message != "No findings" {
		t.Errorf("Message: got %q, want %q", result.Message, "No findings")
	}
	if result.Findings == nil {
		t.Errorf("Findings should not be nil, want empty slice")
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings count: got %d, want 0", len(result.Findings))
	}
}

func TestParse_WithWarnings(t *testing.T) {
	data, err := os.ReadFile("../../testdata/ocr/with-warnings.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if result.Status != "partial" {
		t.Errorf("Status: got %q, want %q", result.Status, "partial")
	}
	if len(result.Findings) != 1 {
		t.Fatalf("Findings count: got %d, want 1", len(result.Findings))
	}
	if len(result.Warnings) != 2 {
		t.Fatalf("Warnings count: got %d, want 2", len(result.Warnings))
	}

	w0 := result.Warnings[0]
	if w0.Type != "subtask_failed" {
		t.Errorf("Warning[0].Type: got %q, want %q", w0.Type, "subtask_failed")
	}
	if w0.Path != "b.go" {
		t.Errorf("Warning[0].Path: got %q, want %q", w0.Path, "b.go")
	}

	w1 := result.Warnings[1]
	if w1.Type != "timeout" {
		t.Errorf("Warning[1].Type: got %q, want %q", w1.Type, "timeout")
	}
	if w1.Metadata == nil {
		t.Fatalf("Warning[1].Metadata should not be nil")
	}
	if w1.Metadata["extra_field"] != "preserved" {
		t.Errorf("Warning[1].Metadata[extra_field]: got %v, want %q", w1.Metadata["extra_field"], "preserved")
	}
}

func TestParse_FutureFields(t *testing.T) {
	data, err := os.ReadFile("../../testdata/ocr/future-fields.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(result.Findings) != 1 {
		t.Fatalf("Findings count: got %d, want 1", len(result.Findings))
	}

	f := result.Findings[0]
	if f.Category != "security" {
		t.Errorf("Category: got %q, want %q", f.Category, "security")
	}
	if f.Severity != "high" {
		t.Errorf("Severity: got %q, want %q", f.Severity, "high")
	}
	if f.Metadata == nil {
		t.Fatalf("Metadata should not be nil")
	}
	if f.Metadata["confidence"] != 0.95 {
		t.Errorf("Metadata[confidence]: got %v, want 0.95", f.Metadata["confidence"])
	}
	if f.Metadata["rule_id"] != "sql-injection-001" {
		t.Errorf("Metadata[rule_id]: got %v, want %q", f.Metadata["rule_id"], "sql-injection-001")
	}
	if f.Metadata["extra_tag"] != "future" {
		t.Errorf("Metadata[extra_tag]: got %v, want %q", f.Metadata["extra_tag"], "future")
	}
}

func TestParse_TopLevelUnknownFieldsMetadata(t *testing.T) {
	input := []byte(`{
		"status": "success",
		"comments": [],
		"run_id": "abc-123",
		"model": "test-model"
	}`)

	result, err := Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if result.Metadata == nil {
		t.Fatalf("Metadata should not be nil")
	}
	if result.Metadata["run_id"] != "abc-123" {
		t.Errorf("Metadata[run_id]: got %v, want %q", result.Metadata["run_id"], "abc-123")
	}
	if result.Metadata["model"] != "test-model" {
		t.Errorf("Metadata[model]: got %v, want %q", result.Metadata["model"], "test-model")
	}
}

func TestParse_MalformedJSONReturnsHelpfulError(t *testing.T) {
	input := []byte(`{"status": "success", "comments": [}`)

	_, err := Parse(input)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "json") && !strings.Contains(errMsg, "parse") && !strings.Contains(errMsg, "JSON") {
		t.Errorf("error should mention json/parse, got: %s", errMsg)
	}
	if len(errMsg) > 500 {
		t.Errorf("error message too long (%d chars), should be concise", len(errMsg))
	}
}

func TestParse_EmptyInputReturnsHelpfulError(t *testing.T) {
	_, err := Parse([]byte{})
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty, got: %s", err.Error())
	}
}

func TestParse_NoJSONObjectReturnsHelpfulError(t *testing.T) {
	_, err := Parse([]byte("this is just text with no JSON"))
	if err == nil {
		t.Fatal("expected error for no JSON object, got nil")
	}
	if !strings.Contains(err.Error(), "json") && !strings.Contains(err.Error(), "JSON") && !strings.Contains(err.Error(), "object") {
		t.Errorf("error should mention json/object, got: %s", err.Error())
	}
}

func TestParseReader(t *testing.T) {
	input := strings.NewReader(`{
		"status": "success",
		"message": "from reader",
		"comments": [{"path": "r.go", "content": "test", "start_line": 1, "end_line": 1}]
	}`)

	result, err := ParseReader(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if result.Message != "from reader" {
		t.Errorf("Message: got %q, want %q", result.Message, "from reader")
	}
	if len(result.Findings) != 1 {
		t.Fatalf("Findings count: got %d, want 1", len(result.Findings))
	}
	if result.Findings[0].Path != "r.go" {
		t.Errorf("Path: got %q, want %q", result.Findings[0].Path, "r.go")
	}
}

func TestParseFile(t *testing.T) {
	result, err := ParseFile("../../testdata/ocr/basic.json")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Status: got %q, want %q", result.Status, "success")
	}
	if len(result.Findings) != 2 {
		t.Fatalf("Findings count: got %d, want 2", len(result.Findings))
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile("nonexistent.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent.json") {
		t.Errorf("error should contain file path, got: %s", err.Error())
	}
}

// Helper to convert OCR JSON comment into review.Finding for comparison.
func findingPtr(f review.Finding) *review.Finding { return &f }
