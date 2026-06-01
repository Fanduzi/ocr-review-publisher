//go:build e2e

package gitlab_e2e

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Fanduzi/ocr-review-publisher/internal/lifecycle"
	"github.com/Fanduzi/ocr-review-publisher/internal/platform/gitlab"
	"github.com/Fanduzi/ocr-review-publisher/internal/review"
)

type e2eConfig struct {
	baseURL   string
	token     string
	projectID string
	mrIID     int
}

func loadGitLabE2EConfig(t *testing.T) e2eConfig {
	t.Helper()
	if os.Getenv("OCR_E2E_GITLAB") != "1" {
		t.Skip("skipping GitLab e2e: OCR_E2E_GITLAB != 1")
	}
	baseURL := os.Getenv("OCR_E2E_GITLAB_URL")
	token := os.Getenv("OCR_E2E_GITLAB_TOKEN")
	projectID := os.Getenv("OCR_E2E_GITLAB_PROJECT_ID")
	mrIIDStr := os.Getenv("OCR_E2E_GITLAB_MR_IID")
	if baseURL == "" || token == "" || projectID == "" || mrIIDStr == "" {
		t.Skip("skipping GitLab e2e: missing required env vars (OCR_E2E_GITLAB_URL, OCR_E2E_GITLAB_TOKEN, OCR_E2E_GITLAB_PROJECT_ID, OCR_E2E_GITLAB_MR_IID)")
	}
	mrIID, err := strconv.Atoi(mrIIDStr)
	if err != nil {
		t.Fatalf("invalid OCR_E2E_GITLAB_MR_IID %q: %v", mrIIDStr, err)
	}
	return e2eConfig{baseURL: baseURL, token: token, projectID: projectID, mrIID: mrIID}
}

func newE2EClient(t *testing.T, cfg e2eConfig) *gitlab.Client {
	t.Helper()
	return gitlab.NewClient(cfg.baseURL, cfg.token, nil)
}

func newE2EPublisher(t *testing.T, cfg e2eConfig, opts gitlab.PublishOptions) *gitlab.Publisher {
	t.Helper()
	client := newE2EClient(t, cfg)
	opts.Project = cfg.projectID
	opts.MergeRequestIID = cfg.mrIID
	return gitlab.NewPublisher(client, opts)
}

func listAllNotes(t *testing.T, client *gitlab.Client, cfg e2eConfig) []gitlab.Note {
	t.Helper()
	ctx := context.Background()
	discs, err := client.ListDiscussions(ctx, cfg.projectID, cfg.mrIID)
	if err != nil {
		t.Fatalf("list discussions: %v", err)
	}
	var notes []gitlab.Note
	for _, d := range discs {
		notes = append(notes, d.Notes...)
	}
	return notes
}

func findNotesWithMarker(notes []gitlab.Note, marker string) []gitlab.Note {
	var found []gitlab.Note
	for _, n := range notes {
		if !n.System && strings.Contains(n.Body, marker) {
			found = append(found, n)
		}
	}
	return found
}

func clearPublisherNotes(t *testing.T, pub *gitlab.Publisher) {
	t.Helper()
	ctx := context.Background()
	_, err := pub.ClearInline(ctx)
	if err != nil {
		t.Logf("clearPublisherNotes clear inline: %v", err)
	}
	_, err = pub.ClearSummary(ctx)
	if err != nil {
		t.Logf("clearPublisherNotes clear summary: %v", err)
	}
}

func findFirstAddedLine(changedFiles []gitlab.ChangedFile) (path string, line int, ok bool) {
	for _, f := range changedFiles {
		if f.DeletedFile || f.NewPath == "" || f.Diff == "" {
			continue
		}
		_, found := gitlab.SelectAddedLineInRange(f.Diff, 1, 10000)
		if found {
			for l := 1; l <= 10000; l++ {
				if _, ok := gitlab.SelectAddedLineInRange(f.Diff, l, l); ok {
					return f.NewPath, l, true
				}
			}
		}
	}
	return "", 0, false
}

func findFirstAddedLineByExtension(changedFiles []gitlab.ChangedFile, ext string) (path string, line int, ok bool) {
	for _, f := range changedFiles {
		if f.DeletedFile || f.NewPath == "" || f.Diff == "" {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(f.NewPath), strings.ToLower(ext)) {
			continue
		}
		for l := 1; l <= 10000; l++ {
			if _, ok := gitlab.SelectAddedLineInRange(f.Diff, l, l); ok {
				return f.NewPath, l, true
			}
		}
	}
	return "", 0, false
}

func assertNoteContains(t *testing.T, body, text, label string) {
	t.Helper()
	if !strings.Contains(body, text) {
		t.Errorf("missing %s (%q) in note body", label, text)
	}
}

func assertNoteNotContains(t *testing.T, body, text, label string) {
	t.Helper()
	if strings.Contains(body, text) {
		t.Errorf("note body must not contain %s (%q)", label, text)
	}
}

func assertFencedBlockContains(t *testing.T, body, fence, content string) {
	t.Helper()
	fenceStart := "```" + fence
	// Search all fence blocks of this language for the content.
	remaining := body
	for {
		idx := strings.Index(remaining, fenceStart)
		if idx < 0 {
			t.Errorf("expected fence %q containing %q in note body", fenceStart, content)
			return
		}
		blockStart := idx + len(fenceStart)
		blockEnd := strings.Index(remaining[blockStart:], "```")
		if blockEnd < 0 {
			t.Errorf("unclosed fence block %q", fenceStart)
			return
		}
		block := remaining[blockStart : blockStart+blockEnd]
		if strings.Contains(block, content) {
			return // found
		}
		remaining = remaining[blockStart+blockEnd+3:]
	}
}

func uniqueSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// --- E2E Tests ---

func TestGitLabE2E_CreateAndClearSummary(t *testing.T) {
	cfg := loadGitLabE2EConfig(t)
	pub := newE2EPublisher(t, cfg, gitlab.PublishOptions{NoInline: true})
	client := newE2EClient(t, cfg)
	ctx := context.Background()

	clearPublisherNotes(t, pub)
	t.Cleanup(func() { clearPublisherNotes(t, pub) })

	result, err := pub.Publish(ctx, review.Result{
		Findings: []review.Finding{
			{Path: "test.go", Content: "e2e summary test", StartLine: 1, EndLine: 1},
		},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if !result.SummaryCreated {
		t.Fatal("expected summary created")
	}

	notes := listAllNotes(t, client, cfg)
	marked := findNotesWithMarker(notes, lifecycle.SummaryMarker)
	if len(marked) == 0 {
		t.Fatal("no summary marker note found after publish")
	}
	if !strings.Contains(marked[0].Body, "OCR Review Publisher") {
		t.Fatalf("summary note missing header: %q", marked[0].Body[:min(100, len(marked[0].Body))])
	}

	clearResult, err := pub.ClearSummary(ctx)
	if err != nil {
		t.Fatalf("clear summary: %v", err)
	}
	if clearResult.SummaryDeleted < 1 {
		t.Fatalf("expected at least 1 summary deleted, got %d", clearResult.SummaryDeleted)
	}

	notes = listAllNotes(t, client, cfg)
	marked = findNotesWithMarker(notes, lifecycle.SummaryMarker)
	if len(marked) != 0 {
		t.Fatalf("summary marker notes still exist after clear: %d", len(marked))
	}
}

func TestGitLabE2E_CreateAndClearInline(t *testing.T) {
	cfg := loadGitLabE2EConfig(t)
	pub := newE2EPublisher(t, cfg, gitlab.PublishOptions{NoSummary: true})
	client := newE2EClient(t, cfg)
	ctx := context.Background()

	clearPublisherNotes(t, pub)
	t.Cleanup(func() { clearPublisherNotes(t, pub) })

	suffix := uniqueSuffix()

	files, err := client.ListChangedFiles(ctx, cfg.projectID, cfg.mrIID)
	if err != nil {
		t.Fatalf("list changed files: %v", err)
	}
	if len(files) == 0 {
		t.Skip("MR has no changed files; cannot test inline comments")
	}

	targetFile, targetLine, ok := findFirstAddedLine(files)
	if !ok {
		t.Skip("no suitable non-deleted file with a valid added line in MR diff")
	}

	result, err := pub.Publish(ctx, review.Result{
		Findings: []review.Finding{
			{Path: targetFile, Content: "e2e inline test " + suffix, StartLine: targetLine, EndLine: targetLine},
		},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if result.InlinePublished < 1 {
		if len(result.Warnings) > 0 {
			t.Skipf("inline publish skipped: %s", result.Warnings[0].Message)
		}
		t.Fatal("expected at least 1 inline published")
	}

	notes := listAllNotes(t, client, cfg)
	marked := findNotesWithMarker(notes, lifecycle.InlineMarker)
	found := false
	for _, n := range marked {
		if strings.Contains(n.Body, suffix) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("inline note with suffix %s not found", suffix)
	}

	clearResult, err := pub.ClearInline(ctx)
	if err != nil {
		t.Fatalf("clear inline: %v", err)
	}
	if clearResult.InlineDeleted < 1 {
		t.Fatalf("expected at least 1 inline deleted, got %d", clearResult.InlineDeleted)
	}

	notes = listAllNotes(t, client, cfg)
	marked = findNotesWithMarker(notes, lifecycle.InlineMarker)
	for _, n := range marked {
		if strings.Contains(n.Body, suffix) {
			t.Fatalf("inline note with suffix %s still exists after clear", suffix)
		}
	}
}

func TestGitLabE2E_ClearDoesNotDeleteUnmarkedNotes(t *testing.T) {
	cfg := loadGitLabE2EConfig(t)
	pub := newE2EPublisher(t, cfg, gitlab.PublishOptions{})
	client := newE2EClient(t, cfg)
	ctx := context.Background()

	unmarkedBody := "e2e unmarked note " + uniqueSuffix()
	note, err := client.CreateNote(ctx, cfg.projectID, cfg.mrIID, unmarkedBody)
	if err != nil {
		t.Fatalf("create unmarked note: %v", err)
	}
	t.Cleanup(func() {
		_ = client.DeleteNote(context.Background(), cfg.projectID, cfg.mrIID, note.ID)
	})

	_, err = pub.ClearInline(ctx)
	if err != nil {
		t.Fatalf("clear inline: %v", err)
	}
	_, err = pub.ClearSummary(ctx)
	if err != nil {
		t.Fatalf("clear summary: %v", err)
	}

	notes := listAllNotes(t, client, cfg)
	for _, n := range notes {
		if n.ID == note.ID {
			return
		}
	}
	t.Fatalf("unmarked note %d was deleted by clear operations", note.ID)
}

func TestGitLabE2E_PublishUpdatesSummaryNote(t *testing.T) {
	cfg := loadGitLabE2EConfig(t)
	pub := newE2EPublisher(t, cfg, gitlab.PublishOptions{NoInline: true})
	client := newE2EClient(t, cfg)
	ctx := context.Background()

	clearPublisherNotes(t, pub)
	t.Cleanup(func() { clearPublisherNotes(t, pub) })

	result1, err := pub.Publish(ctx, review.Result{
		Findings: []review.Finding{
			{Path: "test.go", Content: "e2e rerun test", StartLine: 1, EndLine: 1},
		},
	})
	if err != nil {
		t.Fatalf("publish 1: %v", err)
	}
	if !result1.SummaryCreated {
		t.Fatal("expected summary created on first publish")
	}

	result2, err := pub.Publish(ctx, review.Result{
		Findings: []review.Finding{
			{Path: "test.go", Content: "e2e rerun updated", StartLine: 1, EndLine: 1},
			{Path: "main.go", Content: "e2e second finding", StartLine: 2, EndLine: 2},
		},
	})
	if err != nil {
		t.Fatalf("publish 2: %v", err)
	}
	if !result2.SummaryUpdated {
		t.Fatal("expected summary updated (not created) on second publish")
	}

	notes := listAllNotes(t, client, cfg)
	marked := findNotesWithMarker(notes, lifecycle.SummaryMarker)
	if len(marked) != 1 {
		t.Fatalf("expected exactly 1 summary marker note, got %d", len(marked))
	}
}

func TestGitLabE2E_InlineCommentRenderingQuality(t *testing.T) {
	cfg := loadGitLabE2EConfig(t)
	pub := newE2EPublisher(t, cfg, gitlab.PublishOptions{NoSummary: true})
	client := newE2EClient(t, cfg)
	ctx := context.Background()

	clearPublisherNotes(t, pub)
	t.Cleanup(func() { clearPublisherNotes(t, pub) })

	suffix := uniqueSuffix()

	files, err := client.ListChangedFiles(ctx, cfg.projectID, cfg.mrIID)
	if err != nil {
		t.Fatalf("list changed files: %v", err)
	}
	if len(files) == 0 {
		t.Skip("MR has no changed files")
	}

	// Prefer .go files for language-aware fence testing.
	targetFile, targetLine, ok := findFirstAddedLineByExtension(files, ".go")
	isGoFile := ok
	if !ok {
		targetFile, targetLine, ok = findFirstAddedLine(files)
		if !ok {
			t.Skip("no suitable added line in MR diff")
		}
	}

	// Build a finding with all rendering features.
	finding := review.Finding{
		Path:           targetFile,
		Content:        "e2e rendering test " + suffix,
		StartLine:      targetLine,
		EndLine:        targetLine,
		ExistingCode:   "old code here",
		SuggestionCode: "fixed code here",
		Thinking:       "reasoning about the fix",
		Category:       "security",
		Severity:       "high",
	}

	result, err := pub.Publish(ctx, review.Result{Findings: []review.Finding{finding}})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if result.InlinePublished < 1 {
		if len(result.Warnings) > 0 {
			t.Skipf("inline publish skipped: %s", result.Warnings[0].Message)
		}
		t.Fatal("expected at least 1 inline published")
	}

	// Fetch note BEFORE cleanup to assert rendering quality.
	notes := listAllNotes(t, client, cfg)
	marked := findNotesWithMarker(notes, lifecycle.InlineMarker)
	var foundNote *gitlab.Note
	for i, n := range marked {
		if strings.Contains(n.Body, suffix) {
			foundNote = &marked[i]
			break
		}
	}
	if foundNote == nil {
		t.Fatalf("inline note with suffix %s not found", suffix)
	}

	body := foundNote.Body

	// Positive assertions.
	assertNoteContains(t, body, lifecycle.InlineMarker, "inline marker")
	assertNoteContains(t, body, "category-security", "category badge")
	assertNoteContains(t, body, "severity-high", "severity badge")
	assertNoteContains(t, body, "<details><summary>Review context</summary>", "details block")
	assertNoteContains(t, body, "old code here", "existing code content")
	assertNoteContains(t, body, "Reviewer notes:", "thinking label")
	assertNoteContains(t, body, "reasoning about the fix", "thinking content")
	assertNoteContains(t, body, "fixed code here", "suggested change content")

	// Existing code must be in a fenced block, not bare text.
	assertFencedBlockContains(t, body, "go", "old code here")

	// For .go files, fence should be ```go, not ```text.
	if isGoFile {
		assertNoteContains(t, body, "```go", "go language fence")
		assertNoteNotContains(t, body, "```text\nold code here", "existing code not in text fence")
	}

	// Suggested change should use language fence, not ```suggestion.
	assertNoteNotContains(t, body, "```suggestion", "suggestion fence")
	assertFencedBlockContains(t, body, "go", "fixed code here")

	// Negative assertions.
	assertNoteNotContains(t, body, "line_code can't be blank", "raw GitLab API error")
}

func TestGitLabE2E_PublishSkippedInlineAppearsInSummaryDiagnostics(t *testing.T) {
	cfg := loadGitLabE2EConfig(t)
	pub := newE2EPublisher(t, cfg, gitlab.PublishOptions{})
	client := newE2EClient(t, cfg)
	ctx := context.Background()

	clearPublisherNotes(t, pub)
	t.Cleanup(func() { clearPublisherNotes(t, pub) })

	// Publish a finding with line 0 (invalid) - should be skipped inline.
	result, err := pub.Publish(ctx, review.Result{
		Findings: []review.Finding{
			{Path: "test.go", Content: "skipped finding", StartLine: 0, EndLine: 0},
		},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	// No inline should be created.
	if result.InlinePublished > 0 {
		t.Errorf("expected 0 inline published for line 0, got %d", result.InlinePublished)
	}

	// Summary should exist.
	if !result.SummaryCreated && !result.SummaryUpdated {
		t.Fatal("expected summary created or updated")
	}

	// Fetch summary note BEFORE cleanup to assert content.
	notes := listAllNotes(t, client, cfg)
	marked := findNotesWithMarker(notes, lifecycle.SummaryMarker)
	if len(marked) == 0 {
		t.Fatal("no summary marker note found")
	}

	body := marked[0].Body

	// Positive assertions.
	assertNoteContains(t, body, lifecycle.SummaryMarker, "summary marker")
	assertNoteContains(t, body, "test.go", "skipped finding path")
	assertNoteContains(t, body, "<details><summary>Publish diagnostics", "diagnostics details block")

	// Negative assertions.
	assertNoteNotContains(t, body, "Publish warnings:", "old warnings wording")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
