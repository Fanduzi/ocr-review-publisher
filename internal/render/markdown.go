package render

import (
	"fmt"
	"strings"

	"github.com/Fanduzi/ocr-review-publisher/internal/review"
)

// InlineOptions controls inline comment rendering.
type InlineOptions struct {
	IncludeMarker bool
}

// SummaryOptions controls summary comment rendering.
type SummaryOptions struct {
	IncludeMarker bool
}

// Diagnostic represents a publishing issue to include in summary.
type Diagnostic struct {
	Type    string
	Path    string
	Message string
}

// InlineComment renders a single finding as a Markdown inline comment.
func InlineComment(f review.Finding, opts InlineOptions) string {
	var b strings.Builder

	b.WriteString("**OCR Review Publisher**\n")

	// Badges
	catBadge := categoryBadge(f.Category)
	sevBadge := severityBadge(f.Severity)
	if catBadge != "" || sevBadge != "" {
		b.WriteString("\n")
		if catBadge != "" {
			b.WriteString(catBadge)
		}
		if catBadge != "" && sevBadge != "" {
			b.WriteString(" ")
		}
		if sevBadge != "" {
			b.WriteString(sevBadge)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(strings.TrimSpace(f.Content))
	b.WriteString("\n")

	// Location
	b.WriteString("\n")
	if f.StartLine > 0 {
		b.WriteString(fmt.Sprintf("\U0001f4cd `%s:%s`\n", f.Path, f.LineRange()))
	} else {
		b.WriteString(fmt.Sprintf("\U0001f4cd `%s`\n", f.Path))
	}

	// Review context (existing code + thinking)
	if f.ExistingCode != "" || f.Thinking != "" {
		b.WriteString("\n<details><summary>Review context</summary>\n")
		if f.ExistingCode != "" {
			lang := FenceLanguage(f.Path)
			b.WriteString("\nExisting code:\n\n")
			b.WriteString("```" + lang + "\n")
			b.WriteString(strings.TrimRight(f.ExistingCode, "\n"))
			b.WriteString("\n```\n")
		}
		if f.Thinking != "" {
			b.WriteString("\nReviewer notes:\n\n")
			b.WriteString(strings.TrimSpace(f.Thinking))
			b.WriteString("\n")
		}
		b.WriteString("\n</details>\n")
	}

	// Suggested change
	if f.SuggestionCode != "" {
		lang := FenceLanguage(f.Path)
		b.WriteString("\nSuggested change:\n\n")
		b.WriteString("```" + lang + "\n")
		b.WriteString(strings.TrimRight(f.SuggestionCode, "\n"))
		b.WriteString("\n```\n")
	}

	if opts.IncludeMarker {
		b.WriteString("\n")
		b.WriteString(InlineMarker)
		b.WriteString("\n")
	}

	return b.String()
}

// Summary renders a summary comment for a review result.
func Summary(result review.Result, diagnostics []Diagnostic, opts SummaryOptions) string {
	var b strings.Builder

	b.WriteString("**OCR Review Publisher**\n")

	if len(result.Findings) == 0 {
		b.WriteString("\nNo findings generated. Looks good to me.\n")
	} else {
		b.WriteString(fmt.Sprintf("\n%d finding(s) reviewed.\n\n", len(result.Findings)))
		for _, f := range result.Findings {
			if f.StartLine > 0 {
				b.WriteString(fmt.Sprintf("- `%s:%s`\n", f.Path, f.LineRange()))
			} else {
				b.WriteString(fmt.Sprintf("- `%s`\n", f.Path))
			}
		}
	}

	if len(diagnostics) > 0 {
		b.WriteString(fmt.Sprintf("\n<details><summary>Publish diagnostics (%d issue", len(diagnostics)))
		if len(diagnostics) != 1 {
			b.WriteString("s")
		}
		b.WriteString(")</summary>\n\n")
		for _, d := range diagnostics {
			b.WriteString(fmt.Sprintf("- **%s**: `%s` — %s\n", d.Type, d.Path, d.Message))
		}
		b.WriteString("\n</details>\n")
	}

	if opts.IncludeMarker {
		b.WriteString("\n")
		b.WriteString(SummaryMarker)
		b.WriteString("\n")
	}

	return b.String()
}

func categoryBadge(category string) string {
	if category == "" {
		return ""
	}
	color := "blue"
	switch strings.ToLower(category) {
	case "security":
		color = "red"
	case "performance":
		color = "orange"
	case "bug":
		color = "red"
	}
	return fmt.Sprintf("![category](https://img.shields.io/badge/category-%s-%s)", category, color)
}

func severityBadge(severity string) string {
	if severity == "" {
		return ""
	}
	color := "lightgrey"
	switch strings.ToLower(severity) {
	case "critical":
		color = "red"
	case "high":
		color = "orange"
	case "medium":
		color = "yellow"
	case "low":
		color = "blue"
	case "info":
		color = "lightgrey"
	}
	return fmt.Sprintf("![severity](https://img.shields.io/badge/severity-%s-%s)", severity, color)
}
