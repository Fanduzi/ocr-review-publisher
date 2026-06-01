package review

import (
	"encoding/json"
	"testing"
)

func TestFinding_JSONRoundTrip(t *testing.T) {
	original := Finding{
		Path:           "cmd/main.go",
		Content:        "missing error check",
		ExistingCode:   "fmt.Println(err)",
		SuggestionCode: "if err != nil {\n\treturn err\n}",
		StartLine:      10,
		EndLine:        12,
		Thinking:       "unchecked error returned by fmt.Println",
		Category:       "error-handling",
		Severity:       "warning",
		Metadata:       map[string]any{"source": "ocr"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Finding
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Path != original.Path {
		t.Errorf("Path: got %q, want %q", decoded.Path, original.Path)
	}
	if decoded.Content != original.Content {
		t.Errorf("Content: got %q, want %q", decoded.Content, original.Content)
	}
	if decoded.ExistingCode != original.ExistingCode {
		t.Errorf("ExistingCode: got %q, want %q", decoded.ExistingCode, original.ExistingCode)
	}
	if decoded.SuggestionCode != original.SuggestionCode {
		t.Errorf("SuggestionCode: got %q, want %q", decoded.SuggestionCode, original.SuggestionCode)
	}
	if decoded.StartLine != original.StartLine {
		t.Errorf("StartLine: got %d, want %d", decoded.StartLine, original.StartLine)
	}
	if decoded.EndLine != original.EndLine {
		t.Errorf("EndLine: got %d, want %d", decoded.EndLine, original.EndLine)
	}
	if decoded.Thinking != original.Thinking {
		t.Errorf("Thinking: got %q, want %q", decoded.Thinking, original.Thinking)
	}
	if decoded.Category != original.Category {
		t.Errorf("Category: got %q, want %q", decoded.Category, original.Category)
	}
	if decoded.Severity != original.Severity {
		t.Errorf("Severity: got %q, want %q", decoded.Severity, original.Severity)
	}
	if decoded.Metadata["source"] != "ocr" {
		t.Errorf("Metadata[source]: got %v, want %q", decoded.Metadata["source"], "ocr")
	}
}

func TestFinding_ZeroLineNumbersRemainValid(t *testing.T) {
	original := Finding{
		Path:      "README.md",
		Content:   "general observation",
		StartLine: 0,
		EndLine:   0,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Finding
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.StartLine != 0 {
		t.Errorf("StartLine: got %d, want 0", decoded.StartLine)
	}
	if decoded.EndLine != 0 {
		t.Errorf("EndLine: got %d, want 0", decoded.EndLine)
	}
	if decoded.Path != original.Path {
		t.Errorf("Path: got %q, want %q", decoded.Path, original.Path)
	}
	if decoded.Content != original.Content {
		t.Errorf("Content: got %q, want %q", decoded.Content, original.Content)
	}
}

func TestFinding_OptionalFieldsOmittedWhenEmpty(t *testing.T) {
	f := Finding{
		Path:    "main.go",
		Content: "a finding",
	}

	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	jsonStr := string(data)
	for _, field := range []string{
		"existing_code",
		"suggestion_code",
		"thinking",
		"category",
		"severity",
		"metadata",
	} {
		if contains(jsonStr, `"`+field+`"`) {
			t.Errorf("empty field %q should be omitted from JSON, got: %s", field, jsonStr)
		}
	}
}

func TestResult_JSONRoundTrip(t *testing.T) {
	original := Result{
		Status:  "completed",
		Message: "review finished",
		Findings: []Finding{
			{Path: "a.go", Content: "finding 1", StartLine: 5, EndLine: 5},
			{Path: "b.go", Content: "finding 2", StartLine: 0, EndLine: 0},
		},
		Warnings: []Warning{
			{Type: "parse", Path: "c.go", Message: "could not parse"},
		},
		Metadata: map[string]any{"version": "1.0"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Result
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Status != original.Status {
		t.Errorf("Status: got %q, want %q", decoded.Status, original.Status)
	}
	if decoded.Message != original.Message {
		t.Errorf("Message: got %q, want %q", decoded.Message, original.Message)
	}
	if len(decoded.Findings) != 2 {
		t.Fatalf("Findings count: got %d, want 2", len(decoded.Findings))
	}
	if decoded.Findings[0].Path != "a.go" {
		t.Errorf("Findings[0].Path: got %q, want %q", decoded.Findings[0].Path, "a.go")
	}
	if decoded.Findings[1].StartLine != 0 {
		t.Errorf("Findings[1].StartLine: got %d, want 0", decoded.Findings[1].StartLine)
	}
	if len(decoded.Warnings) != 1 {
		t.Fatalf("Warnings count: got %d, want 1", len(decoded.Warnings))
	}
	if decoded.Warnings[0].Type != "parse" {
		t.Errorf("Warnings[0].Type: got %q, want %q", decoded.Warnings[0].Type, "parse")
	}
	if decoded.Metadata["version"] != "1.0" {
		t.Errorf("Metadata[version]: got %v, want %q", decoded.Metadata["version"], "1.0")
	}
}

func TestWarning_JSONRoundTrip(t *testing.T) {
	original := Warning{
		Type:     "coverage",
		Path:     "pkg/util.go",
		Message:  "no test coverage",
		Metadata: map[string]any{"confidence": 0.9},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Warning
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("Type: got %q, want %q", decoded.Type, original.Type)
	}
	if decoded.Path != original.Path {
		t.Errorf("Path: got %q, want %q", decoded.Path, original.Path)
	}
	if decoded.Message != original.Message {
		t.Errorf("Message: got %q, want %q", decoded.Message, original.Message)
	}
	if decoded.Metadata["confidence"] != 0.9 {
		t.Errorf("Metadata[confidence]: got %v, want %v", decoded.Metadata["confidence"], 0.9)
	}
}

func TestPublishReport_JSONRoundTrip(t *testing.T) {
	original := PublishReport{
		InlinePublished: 5,
		InlineSkipped:   2,
		InlineFailed:    1,
		SummaryCreated:  true,
		SummaryUpdated:  false,
		InlineDeleted:   3,
		SummaryDeleted:  1,
		Warnings: []Warning{
			{Type: "anchor", Path: "x.go", Message: "no safe anchor"},
		},
		Metadata: map[string]any{"duration_ms": 420},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded PublishReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.InlinePublished != 5 {
		t.Errorf("InlinePublished: got %d, want 5", decoded.InlinePublished)
	}
	if decoded.InlineSkipped != 2 {
		t.Errorf("InlineSkipped: got %d, want 2", decoded.InlineSkipped)
	}
	if decoded.InlineFailed != 1 {
		t.Errorf("InlineFailed: got %d, want 1", decoded.InlineFailed)
	}
	if !decoded.SummaryCreated {
		t.Errorf("SummaryCreated: got false, want true")
	}
	if decoded.SummaryUpdated {
		t.Errorf("SummaryUpdated: got true, want false")
	}
	if decoded.InlineDeleted != 3 {
		t.Errorf("InlineDeleted: got %d, want 3", decoded.InlineDeleted)
	}
	if decoded.SummaryDeleted != 1 {
		t.Errorf("SummaryDeleted: got %d, want 1", decoded.SummaryDeleted)
	}
	if len(decoded.Warnings) != 1 {
		t.Fatalf("Warnings count: got %d, want 1", len(decoded.Warnings))
	}
	if decoded.Warnings[0].Type != "anchor" {
		t.Errorf("Warnings[0].Type: got %q, want %q", decoded.Warnings[0].Type, "anchor")
	}
	if decoded.Metadata["duration_ms"] != float64(420) {
		t.Errorf("Metadata[duration_ms]: got %v, want 420", decoded.Metadata["duration_ms"])
	}
}

func TestFinding_LineRange(t *testing.T) {
	tests := []struct {
		name      string
		startLine int
		endLine   int
		want      string
	}{
		{"single line", 10, 10, "10"},
		{"range", 10, 12, "10-12"},
		{"missing lines", 0, 0, ""},
		{"end before start", 12, 10, "12"},
		{"start only", 5, 0, "5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Finding{StartLine: tt.startLine, EndLine: tt.endLine}
			got := f.LineRange()
			if got != tt.want {
				t.Errorf("LineRange(): got %q, want %q", got, tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
