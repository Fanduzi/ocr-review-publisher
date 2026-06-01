package gitlab

import (
	"context"
	"fmt"
	"strings"

	"github.com/Fanduzi/ocr-review-publisher/internal/lifecycle"
	"github.com/Fanduzi/ocr-review-publisher/internal/render"
	"github.com/Fanduzi/ocr-review-publisher/internal/review"
)

// Publisher manages publishing review findings to a GitLab MR.
type Publisher struct {
	client *Client
	opts   PublishOptions
}

// PublishOptions configures a publisher.
type PublishOptions struct {
	Project         string
	MergeRequestIID int
	NoInline        bool
	NoSummary       bool
	ClearExisting   bool
}

// NewPublisher creates a GitLab MR publisher.
func NewPublisher(client *Client, opts PublishOptions) *Publisher {
	return &Publisher{client: client, opts: opts}
}

// Publish publishes review findings to the GitLab MR.
func (p *Publisher) Publish(ctx context.Context, result review.Result) (*review.PublishReport, error) {
	report := &review.PublishReport{}

	// Step 1: Clear existing if requested.
	if p.opts.ClearExisting {
		clearReport, err := p.clearByMarkers(ctx, lifecycle.InlineMarker)
		if err != nil {
			return report, fmt.Errorf("clear inline: %w", err)
		}
		report.InlineDeleted = clearReport.InlineDeleted
		report.Warnings = append(report.Warnings, clearReport.Warnings...)

		clearReport, err = p.clearByMarkers(ctx, lifecycle.SummaryMarker)
		if err != nil {
			return report, fmt.Errorf("clear summary: %w", err)
		}
		report.SummaryDeleted = clearReport.SummaryDeleted
		report.Warnings = append(report.Warnings, clearReport.Warnings...)
	}

	// Step 2: Load diff version for inline positions.
	var diffVersion *DiffVersion
	if !p.opts.NoInline && len(result.Findings) > 0 {
		versions, err := p.client.ListDiffVersions(ctx, p.opts.Project, p.opts.MergeRequestIID)
		if err != nil {
			return report, fmt.Errorf("list diff versions: %w", err)
		}
		if len(versions) > 0 {
			diffVersion = &versions[0]
		}
	}

	// Step 3: Load changed files for position validation.
	fileDiffs := make(map[string]string)
	if !p.opts.NoInline && diffVersion != nil && len(result.Findings) > 0 {
		changedFiles, err := p.client.ListChangedFiles(ctx, p.opts.Project, p.opts.MergeRequestIID)
		if err == nil {
			for _, f := range changedFiles {
				fileDiffs[f.NewPath] = f.Diff
			}
		}
	}

	// Step 4: Create inline discussions.
	if !p.opts.NoInline && diffVersion == nil && len(result.Findings) > 0 {
		for _, f := range result.Findings {
			report.InlineFailed++
			report.Warnings = append(report.Warnings, review.Warning{
				Type:    "inline_failed",
				Path:    f.Path,
				Message: fmt.Sprintf("line %d: no GitLab diff version available", f.StartLine),
			})
		}
	}
	if !p.opts.NoInline && diffVersion != nil {
		for _, f := range result.Findings {
			if f.StartLine <= 0 {
				report.InlineSkipped++
				report.Warnings = append(report.Warnings, review.Warning{
					Type:    "inline_skipped",
					Path:    f.Path,
					Message: fmt.Sprintf("line %d: start line must be positive", f.StartLine),
				})
				continue
			}

			diff := fileDiffs[f.Path]
			selectedLine, ok := SelectAddedLineInRange(diff, f.StartLine, f.EndLine)
			if !ok {
				report.InlineSkipped++
				report.Warnings = append(report.Warnings, review.Warning{
					Type:    "inline_skipped",
					Path:    f.Path,
					Message: fmt.Sprintf("line %d-%d: no added line in range", f.StartLine, f.EndLine),
				})
				continue
			}

			pos := Position{
				PositionType: "text",
				BaseSHA:      diffVersion.BaseCommitSHA,
				StartSHA:     diffVersion.StartCommitSHA,
				HeadSHA:      diffVersion.HeadCommitSHA,
				NewPath:      f.Path,
				NewLine:      selectedLine,
			}
			body := render.InlineComment(f, render.InlineOptions{IncludeMarker: true})
			_, err := p.client.CreateInlineDiscussion(ctx, p.opts.Project, p.opts.MergeRequestIID, body, pos)
			if err != nil {
				report.InlineFailed++
				report.Warnings = append(report.Warnings, review.Warning{
					Type:    "inline_failed",
					Path:    f.Path,
					Message: fmt.Sprintf("line %d: %v", f.StartLine, err),
				})
				continue
			}
			report.InlinePublished++
		}
	}

	// Step 5: Create or update summary note.
	if !p.opts.NoSummary {
		var diagnostics []render.Diagnostic
		for _, w := range report.Warnings {
			diagnostics = append(diagnostics, render.Diagnostic{
				Type:    w.Type,
				Path:    w.Path,
				Message: w.Message,
			})
		}
		summaryBody := render.Summary(result, diagnostics, render.SummaryOptions{IncludeMarker: true})

		existingNote, err := p.findExistingSummaryNote(ctx)
		if err != nil {
			return report, fmt.Errorf("find summary note: %w", err)
		}

		if existingNote != nil {
			if err := p.client.UpdateNote(ctx, p.opts.Project, p.opts.MergeRequestIID, existingNote.ID, summaryBody); err != nil {
				return report, fmt.Errorf("update summary note: %w", err)
			}
			report.SummaryUpdated = true
		} else {
			if _, err := p.client.CreateNote(ctx, p.opts.Project, p.opts.MergeRequestIID, summaryBody); err != nil {
				return report, fmt.Errorf("create summary note: %w", err)
			}
			report.SummaryCreated = true
		}
	}

	return report, nil
}

// ClearInline deletes all inline marker notes from the MR.
func (p *Publisher) ClearInline(ctx context.Context) (*review.PublishReport, error) {
	return p.clearByMarkers(ctx, lifecycle.InlineMarker)
}

// ClearSummary deletes all summary marker notes from the MR.
func (p *Publisher) ClearSummary(ctx context.Context) (*review.PublishReport, error) {
	return p.clearByMarkers(ctx, lifecycle.SummaryMarker)
}

func (p *Publisher) clearByMarkers(ctx context.Context, markers ...string) (*review.PublishReport, error) {
	discs, err := p.client.ListDiscussions(ctx, p.opts.Project, p.opts.MergeRequestIID)
	if err != nil {
		return nil, fmt.Errorf("list discussions: %w", err)
	}

	report := &review.PublishReport{}
	var deleteErrors []string

	for _, disc := range discs {
		for _, note := range disc.Notes {
			if note.System {
				continue
			}
			if !containsAnyMarker(note.Body, markers) {
				continue
			}
			if err := p.client.DeleteNote(ctx, p.opts.Project, p.opts.MergeRequestIID, note.ID); err != nil {
				deleteErrors = append(deleteErrors, fmt.Sprintf("note %d: %v", note.ID, err))
				continue
			}
			if strings.Contains(note.Body, lifecycle.InlineMarker) {
				report.InlineDeleted++
			}
			if strings.Contains(note.Body, lifecycle.SummaryMarker) {
				report.SummaryDeleted++
			}
		}
	}

	if len(deleteErrors) > 0 {
		return report, fmt.Errorf("failed to delete %d note(s): %s", len(deleteErrors), strings.Join(deleteErrors, "; "))
	}
	return report, nil
}

func (p *Publisher) findExistingSummaryNote(ctx context.Context) (*Note, error) {
	discs, err := p.client.ListDiscussions(ctx, p.opts.Project, p.opts.MergeRequestIID)
	if err != nil {
		return nil, err
	}
	for _, disc := range discs {
		for _, note := range disc.Notes {
			if note.System {
				continue
			}
			if strings.Contains(note.Body, lifecycle.SummaryMarker) {
				return &note, nil
			}
		}
	}
	return nil, nil
}

func containsAnyMarker(body string, markers []string) bool {
	for _, m := range markers {
		if strings.Contains(body, m) {
			return true
		}
	}
	return false
}
