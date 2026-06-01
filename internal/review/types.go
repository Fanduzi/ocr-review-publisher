package review

import "fmt"

// Result holds the parsed output from a review producer.
type Result struct {
	Status   string         `json:"status,omitempty"`
	Message  string         `json:"message,omitempty"`
	Findings []Finding      `json:"findings,omitempty"`
	Warnings []Warning      `json:"warnings,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Finding represents a single review finding.
type Finding struct {
	Path           string         `json:"path"`
	Content        string         `json:"content"`
	ExistingCode   string         `json:"existing_code,omitempty"`
	SuggestionCode string         `json:"suggestion_code,omitempty"`
	StartLine      int            `json:"start_line,omitempty"`
	EndLine        int            `json:"end_line,omitempty"`
	Thinking       string         `json:"thinking,omitempty"`
	Category       string         `json:"category,omitempty"`
	Severity       string         `json:"severity,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// Warning represents a non-fatal warning encountered during review or publishing.
type Warning struct {
	Type     string         `json:"type"`
	Path     string         `json:"path,omitempty"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// PublishReport summarizes the outcome of a publish operation.
type PublishReport struct {
	InlinePublished int            `json:"inline_published"`
	InlineSkipped   int            `json:"inline_skipped"`
	InlineFailed    int            `json:"inline_failed"`
	SummaryCreated  bool           `json:"summary_created"`
	SummaryUpdated  bool           `json:"summary_updated"`
	InlineDeleted   int            `json:"inline_deleted"`
	SummaryDeleted  int            `json:"summary_deleted"`
	Warnings        []Warning      `json:"warnings,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// LineRange returns a human-readable line range string.
// Returns empty string when both lines are zero (summary fallback).
// Returns single number when start equals end.
// Returns "start-end" for ranges.
func (f Finding) LineRange() string {
	if f.StartLine == 0 && f.EndLine == 0 {
		return ""
	}
	if f.EndLine == 0 || f.StartLine == f.EndLine {
		return fmt.Sprintf("%d", f.StartLine)
	}
	if f.EndLine < f.StartLine {
		return fmt.Sprintf("%d", f.StartLine)
	}
	return fmt.Sprintf("%d-%d", f.StartLine, f.EndLine)
}
