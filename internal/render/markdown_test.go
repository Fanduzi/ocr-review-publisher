package render

import (
	"os"
	"strings"
	"testing"

	"github.com/Fanduzi/ocr-review-publisher/internal/review"
)

func loadGolden(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("../../testdata/render/" + name)
	if err != nil {
		t.Fatalf("load golden %s: %v", name, err)
	}
	return string(data)
}

// --- Golden tests ---

func TestInlineComment_Minimal(t *testing.T) {
	f := review.Finding{
		Path:      "cmd/main.go",
		Content:   "Missing error check on return value.",
		StartLine: 10,
		EndLine:   10,
	}
	got := InlineComment(f, InlineOptions{IncludeMarker: true})
	want := loadGolden(t, "inline_minimal.golden.md")
	if got != want {
		t.Errorf("mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestInlineComment_GoExistingCode(t *testing.T) {
	f := review.Finding{
		Path:         "service/user.go",
		Content:      "Error return value of `fmt.Println` is not checked.",
		ExistingCode: "fmt.Println(err)",
		StartLine:    37,
		EndLine:      37,
		Thinking:     "The error from fmt.Println is silently discarded.",
	}
	got := InlineComment(f, InlineOptions{IncludeMarker: true})
	want := loadGolden(t, "inline_go_existing_code.golden.md")
	if got != want {
		t.Errorf("mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestInlineComment_SuggestedChange(t *testing.T) {
	f := review.Finding{
		Path:           "service/user.go",
		Content:        "Error return value of `fmt.Println` is not checked.",
		SuggestionCode: "if err != nil {\n\treturn fmt.Errorf(\"log error: %w\", err)\n}",
		StartLine:      37,
		EndLine:        37,
	}
	got := InlineComment(f, InlineOptions{IncludeMarker: true})
	want := loadGolden(t, "inline_suggested_change.golden.md")
	if got != want {
		t.Errorf("mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestInlineComment_Badges(t *testing.T) {
	f := review.Finding{
		Path:      "handler.go",
		Content:   "SQL injection risk in user query.",
		Category:  "security",
		Severity:  "high",
		StartLine: 42,
		EndLine:   42,
	}
	got := InlineComment(f, InlineOptions{IncludeMarker: true})
	want := loadGolden(t, "inline_badges.golden.md")
	if got != want {
		t.Errorf("mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSummary_ZeroFindings(t *testing.T) {
	result := review.Result{
		Findings: []review.Finding{},
	}
	got := Summary(result, nil, SummaryOptions{IncludeMarker: true})
	want := loadGolden(t, "summary_zero_findings.golden.md")
	if got != want {
		t.Errorf("mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestSummary_WithDiagnostics(t *testing.T) {
	result := review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "finding 1", StartLine: 1, EndLine: 1},
			{Path: "b.go", Content: "finding 2", StartLine: 5, EndLine: 5},
		},
	}
	diags := []Diagnostic{
		{Type: "anchor", Path: "c.go", Message: "no safe anchor found"},
	}
	got := Summary(result, diags, SummaryOptions{IncludeMarker: true})
	want := loadGolden(t, "summary_with_diagnostics.golden.md")
	if got != want {
		t.Errorf("mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestInlineComment_UnknownExtension(t *testing.T) {
	f := review.Finding{
		Path:           "config.xyz",
		Content:        "Configuration value should be quoted.",
		SuggestionCode: "\"correct_value\"",
		StartLine:      3,
		EndLine:        3,
	}
	got := InlineComment(f, InlineOptions{IncludeMarker: true})
	want := loadGolden(t, "inline_unknown_extension.golden.md")
	if got != want {
		t.Errorf("mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// --- Non-golden assertions ---

func TestFenceLanguage_CaseInsensitive(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"main.GO", "go"},
		{"app.Js", "javascript"},
		{"app.JSX", "javascript"},
		{"index.Ts", "typescript"},
		{"index.TSX", "typescript"},
		{"main.Py", "python"},
		{"Main.Java", "java"},
		{"lib.Rs", "rust"},
		{"config.JSON", "json"},
		{"config.Yaml", "yaml"},
		{"config.YML", "yaml"},
		{"readme.Md", "markdown"},
		{"unknown.xyz", "text"},
		{"noext", "text"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := FenceLanguage(tt.path)
			if got != tt.want {
				t.Errorf("FenceLanguage(%q): got %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestInlineComment_DoesNotUseSuggestionFence(t *testing.T) {
	f := review.Finding{
		Path:           "main.go",
		Content:        "test",
		SuggestionCode: "fix",
		StartLine:      1,
		EndLine:        1,
	}
	got := InlineComment(f, InlineOptions{IncludeMarker: true})
	if strings.Contains(got, "```suggestion") {
		t.Errorf("rendered comment must not contain ```suggestion fence")
	}
}

func TestInlineComment_ExistingCodeNeverBareText(t *testing.T) {
	f := review.Finding{
		Path:         "main.go",
		Content:      "test",
		ExistingCode: "var x = 1",
		StartLine:    1,
		EndLine:      1,
	}
	got := InlineComment(f, InlineOptions{IncludeMarker: true})
	// ExistingCode should be inside a fenced block, not bare text.
	if strings.Contains(got, "var x = 1") && !strings.Contains(got, "```go") {
		t.Errorf("existing code must be inside a fenced code block")
	}
}

func TestSummary_DiagnosticsFolded(t *testing.T) {
	result := review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "f", StartLine: 1, EndLine: 1},
		},
	}
	diags := []Diagnostic{
		{Type: "anchor", Path: "x.go", Message: "no anchor"},
	}
	got := Summary(result, diags, SummaryOptions{IncludeMarker: true})
	if !strings.Contains(got, "<details>") {
		t.Errorf("diagnostics should be folded in <details>")
	}
	if !strings.Contains(got, "Publish diagnostics") {
		t.Errorf("diagnostics should have summary label")
	}
}

func TestMarkers_AreStable(t *testing.T) {
	if InlineMarker != "<!-- ocr-review-publisher:inline -->" {
		t.Errorf("InlineMarker: got %q, want %q", InlineMarker, "<!-- ocr-review-publisher:inline -->")
	}
	if SummaryMarker != "<!-- ocr-review-publisher:summary -->" {
		t.Errorf("SummaryMarker: got %q, want %q", SummaryMarker, "<!-- ocr-review-publisher:summary -->")
	}
}

func TestInlineComment_MarkerOmittedWhenFalse(t *testing.T) {
	f := review.Finding{
		Path:      "main.go",
		Content:   "test",
		StartLine: 1,
		EndLine:   1,
	}
	got := InlineComment(f, InlineOptions{IncludeMarker: false})
	if strings.Contains(got, InlineMarker) {
		t.Errorf("marker should be omitted when IncludeMarker is false")
	}
}

func TestSummary_MarkerOmittedWhenFalse(t *testing.T) {
	result := review.Result{Findings: []review.Finding{}}
	got := Summary(result, nil, SummaryOptions{IncludeMarker: false})
	if strings.Contains(got, SummaryMarker) {
		t.Errorf("marker should be omitted when IncludeMarker is false")
	}
}

func TestInlineComment_NoLineShowsOnlyPath(t *testing.T) {
	f := review.Finding{
		Path:    "README.md",
		Content: "general observation",
	}
	got := InlineComment(f, InlineOptions{IncludeMarker: true})
	if !strings.Contains(got, "`README.md`") {
		t.Errorf("should show path without line numbers, got:\n%s", got)
	}
	if strings.Contains(got, ":0") {
		t.Errorf("should not show zero line numbers, got:\n%s", got)
	}
}
