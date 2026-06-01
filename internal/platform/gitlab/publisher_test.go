package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Fanduzi/ocr-review-publisher/internal/lifecycle"
	"github.com/Fanduzi/ocr-review-publisher/internal/review"
)

// --- Publisher tests ---

func TestPublish_CreatesInlineDiscussionWithRenderedMarker(t *testing.T) {
	var capturedBody string
	var capturedPos Position
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{
				{ID: 1, BaseCommitSHA: "base1", StartCommitSHA: "start1", HeadCommitSHA: "head1"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/diffs"):
			json.NewEncoder(w).Encode([]ChangedFile{
				{NewPath: "main.go", Diff: "@@ -1,3 +1,4 @@\n context\n+added\n rest"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
			var req struct {
				Body     string   `json:"body"`
				Position Position `json:"position"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			capturedBody = req.Body
			capturedPos = req.Position
			json.NewEncoder(w).Encode(Discussion{ID: "disc-new", Notes: []Note{{ID: 100, Body: req.Body}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Note{ID: 200, Body: req.Body})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "main.go", Content: "fix this", StartLine: 2, EndLine: 2},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InlinePublished != 1 {
		t.Errorf("expected 1 inline published, got %d", result.InlinePublished)
	}
	if !strings.Contains(capturedBody, lifecycle.InlineMarker) {
		t.Errorf("inline body missing marker: %q", capturedBody)
	}
	if capturedPos.NewPath != "main.go" {
		t.Errorf("expected new_path main.go, got %s", capturedPos.NewPath)
	}
	if capturedPos.NewLine != 2 {
		t.Errorf("expected new_line 2, got %d", capturedPos.NewLine)
	}
	if capturedPos.BaseSHA != "base1" {
		t.Errorf("expected base_sha from diff version, got %s", capturedPos.BaseSHA)
	}
}

func TestPublish_ContextStartRangeAnchorsToFirstAddedLine(t *testing.T) {
	var capturedPos Position
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{
				{BaseCommitSHA: "b", StartCommitSHA: "s", HeadCommitSHA: "h"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/diffs"):
			json.NewEncoder(w).Encode([]ChangedFile{
				{NewPath: "a.go", Diff: "@@ -1,5 +1,6 @@\n context\n context\n+added\n context\n context"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
			var req struct {
				Body     string   `json:"body"`
				Position Position `json:"position"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			capturedPos = req.Position
			json.NewEncoder(w).Encode(Discussion{ID: "d1", Notes: []Note{{ID: 1, Body: req.Body}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Note{ID: 200, Body: req.Body})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "issue", StartLine: 1, EndLine: 5},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InlinePublished != 1 {
		t.Errorf("expected 1 inline published, got %d", result.InlinePublished)
	}
	if capturedPos.NewLine != 3 {
		t.Errorf("expected new_line 3 (first added in range), got %d", capturedPos.NewLine)
	}
}

func TestPublish_SkipsInlineWhenNoAddedLineInRange(t *testing.T) {
	var inlineCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{
				{BaseCommitSHA: "b", StartCommitSHA: "s", HeadCommitSHA: "h"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/diffs"):
			json.NewEncoder(w).Encode([]ChangedFile{
				{NewPath: "a.go", Diff: "@@ -1,3 +1,3 @@\n line1\n line2\n line3"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
			inlineCalled = true
			json.NewEncoder(w).Encode(Discussion{ID: "d1", Notes: []Note{{ID: 1}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Note{ID: 200, Body: req.Body})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "issue", StartLine: 1, EndLine: 3},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inlineCalled {
		t.Error("should NOT create inline discussion when no added line in range")
	}
	if result.InlineSkipped != 1 {
		t.Errorf("expected 1 inline skipped, got %d", result.InlineSkipped)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Type != "inline_skipped" {
		t.Errorf("expected warning type inline_skipped, got %s", result.Warnings[0].Type)
	}
}

func TestPublish_SkipsInlineWhenPathMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{
				{BaseCommitSHA: "b", StartCommitSHA: "s", HeadCommitSHA: "h"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/diffs"):
			json.NewEncoder(w).Encode([]ChangedFile{})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
			t.Error("should NOT create inline discussion for missing path")
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Note{ID: 200, Body: req.Body})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "missing.go", Content: "issue", StartLine: 1, EndLine: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InlineSkipped != 1 {
		t.Errorf("expected 1 inline skipped, got %d", result.InlineSkipped)
	}
}

func TestPublish_NoDiffVersionsSkipsInlineAndCreatesSummary(t *testing.T) {
	var inlineCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
			inlineCalled = true
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Note{ID: 200, Body: req.Body})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "fix", StartLine: 1, EndLine: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inlineCalled {
		t.Error("should NOT create inline when no diff versions")
	}
	if result.InlineFailed != 1 {
		t.Errorf("expected 1 inline failed, got %d", result.InlineFailed)
	}
	if !result.SummaryCreated {
		t.Error("expected summary created")
	}
}

func TestPublish_ContinuesWhenInlineFails(t *testing.T) {
	var mu sync.Mutex
	inlineAttempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{
				{BaseCommitSHA: "b", StartCommitSHA: "s", HeadCommitSHA: "h"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/diffs"):
			json.NewEncoder(w).Encode([]ChangedFile{
				{NewPath: "a.go", Diff: "@@ -1,1 +1,2 @@\n+added"},
				{NewPath: "b.go", Diff: "@@ -1,1 +1,2 @@\n+added"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
			mu.Lock()
			inlineAttempts++
			n := inlineAttempts
			mu.Unlock()
			if n == 1 {
				w.WriteHeader(http.StatusUnprocessableEntity)
				json.NewEncoder(w).Encode(map[string]string{"message": "cannot create"})
				return
			}
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Discussion{ID: "d", Notes: []Note{{ID: 1, Body: req.Body}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Note{ID: 200, Body: req.Body})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "first", StartLine: 1, EndLine: 1},
			{Path: "b.go", Content: "second", StartLine: 1, EndLine: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InlinePublished != 1 {
		t.Errorf("expected 1 inline published, got %d", result.InlinePublished)
	}
	if result.InlineFailed != 1 {
		t.Errorf("expected 1 inline failed, got %d", result.InlineFailed)
	}
}

func TestPublish_CreatesSummaryWhenNoneExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Note{ID: 200, Body: req.Body})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "fix", StartLine: 1, EndLine: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.SummaryCreated {
		t.Error("expected summary created")
	}
	if result.SummaryUpdated {
		t.Error("expected summary not updated")
	}
}

func TestPublish_UpdatesExistingSummary(t *testing.T) {
	var updatedNoteID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{
				{ID: "d1", Notes: []Note{{ID: 50, Body: "old\n\n" + lifecycle.SummaryMarker, System: false}}},
			})
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/notes/"):
			parts := strings.Split(r.URL.Path, "/")
			updatedNoteID = parts[len(parts)-1]
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Note{ID: 50, Body: req.Body})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "fix", StartLine: 1, EndLine: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SummaryCreated {
		t.Error("expected summary not created")
	}
	if !result.SummaryUpdated {
		t.Error("expected summary updated")
	}
	if updatedNoteID != "50" {
		t.Errorf("expected update to note 50, got %s", updatedNoteID)
	}
}

func TestPublish_ClearExistingBeforePublishing(t *testing.T) {
	inlineBody := "old inline\n\n" + lifecycle.InlineMarker
	summaryBody := "old summary\n\n" + lifecycle.SummaryMarker

	var mu sync.Mutex
	var deleteIDs []string
	listCallCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{
				{BaseCommitSHA: "b", StartCommitSHA: "s", HeadCommitSHA: "h"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/diffs"):
			json.NewEncoder(w).Encode([]ChangedFile{
				{NewPath: "a.go", Diff: "@@ -1,1 +1,2 @@\n+added"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			mu.Lock()
			listCallCount++
			call := listCallCount
			mu.Unlock()
			if call <= 2 {
				json.NewEncoder(w).Encode([]Discussion{
					{ID: "d1", Notes: []Note{{ID: 10, Body: inlineBody}}},
					{ID: "d2", Notes: []Note{{ID: 20, Body: summaryBody}}},
				})
			} else {
				json.NewEncoder(w).Encode([]Discussion{})
			}
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Discussion{ID: "new", Notes: []Note{{ID: 100, Body: req.Body}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Note{ID: 200, Body: req.Body})
		case r.Method == http.MethodDelete:
			parts := strings.Split(r.URL.Path, "/")
			mu.Lock()
			deleteIDs = append(deleteIDs, parts[len(parts)-1])
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1, ClearExisting: true})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "fix", StartLine: 1, EndLine: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deleteIDs) != 2 {
		t.Errorf("expected 2 deletes, got %d: %v", len(deleteIDs), deleteIDs)
	}
	if result.InlineDeleted != 1 {
		t.Errorf("expected 1 inline deleted, got %d", result.InlineDeleted)
	}
	if result.SummaryDeleted != 1 {
		t.Errorf("expected 1 summary deleted, got %d", result.SummaryDeleted)
	}
	if result.InlinePublished != 1 {
		t.Errorf("expected 1 inline published after clear, got %d", result.InlinePublished)
	}
}

func TestPublish_RespectsNoInline(t *testing.T) {
	var inlineBodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			inlineBodies = append(inlineBodies, req.Body)
			json.NewEncoder(w).Encode(Discussion{ID: "d", Notes: []Note{{ID: 1, Body: req.Body}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Note{ID: 200, Body: req.Body})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1, NoInline: true})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "fix", StartLine: 1, EndLine: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InlinePublished != 0 {
		t.Errorf("expected 0 inline published, got %d", result.InlinePublished)
	}
	if len(inlineBodies) != 0 {
		t.Errorf("expected no inline discussions, got %d", len(inlineBodies))
	}
}

func TestPublish_RespectsNoSummary(t *testing.T) {
	var summaryBodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{
				{BaseCommitSHA: "b", StartCommitSHA: "s", HeadCommitSHA: "h"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/diffs"):
			json.NewEncoder(w).Encode([]ChangedFile{
				{NewPath: "a.go", Diff: "@@ -1,1 +1,2 @@\n+added"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			json.NewEncoder(w).Encode(Discussion{ID: "d", Notes: []Note{{ID: 1, Body: req.Body}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			summaryBodies = append(summaryBodies, req.Body)
			json.NewEncoder(w).Encode(Note{ID: 200, Body: req.Body})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1, NoSummary: true})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "fix", StartLine: 1, EndLine: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SummaryCreated || result.SummaryUpdated {
		t.Error("expected no summary created/updated")
	}
	if len(summaryBodies) != 0 {
		t.Errorf("expected no summary bodies, got %d", len(summaryBodies))
	}
}

func TestClearInline_DeletesOnlyInlineMarkerNotes(t *testing.T) {
	inlineBody := "some comment\n\n" + lifecycle.InlineMarker
	summaryBody := "summary\n\n" + lifecycle.SummaryMarker
	userBody := "user comment without marker"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/discussions") && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]Discussion{
				{ID: "d1", Notes: []Note{{ID: 10, Body: inlineBody, System: false}}},
				{ID: "d2", Notes: []Note{{ID: 20, Body: summaryBody, System: false}}},
				{ID: "d3", Notes: []Note{{ID: 30, Body: userBody, System: false}}},
			})
			return
		}
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.ClearInline(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InlineDeleted != 1 {
		t.Errorf("expected 1 inline deleted, got %d", result.InlineDeleted)
	}
	if result.SummaryDeleted != 0 {
		t.Errorf("expected 0 summary deleted, got %d", result.SummaryDeleted)
	}
}

func TestClearSummary_DeletesOnlySummaryMarkerNotes(t *testing.T) {
	inlineBody := "some comment\n\n" + lifecycle.InlineMarker
	summaryBody := "summary\n\n" + lifecycle.SummaryMarker
	userBody := "user comment"

	var mu sync.Mutex
	var deletedIDs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/discussions") && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]Discussion{
				{ID: "d1", Notes: []Note{{ID: 10, Body: inlineBody, System: false}}},
				{ID: "d2", Notes: []Note{{ID: 20, Body: summaryBody, System: false}}},
				{ID: "d3", Notes: []Note{{ID: 30, Body: userBody, System: false}}},
			})
			return
		}
		if r.Method == http.MethodDelete {
			parts := strings.Split(r.URL.Path, "/")
			mu.Lock()
			deletedIDs = append(deletedIDs, parts[len(parts)-1])
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.ClearSummary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SummaryDeleted != 1 {
		t.Errorf("expected 1 summary deleted, got %d", result.SummaryDeleted)
	}
	if result.InlineDeleted != 0 {
		t.Errorf("expected 0 inline deleted, got %d", result.InlineDeleted)
	}
	if len(deletedIDs) != 1 || deletedIDs[0] != "20" {
		t.Errorf("expected only note 20 deleted, got %v", deletedIDs)
	}
}

func TestClear_ContinuesOnDeleteFailure(t *testing.T) {
	inlineBody1 := "comment 1\n\n" + lifecycle.InlineMarker
	inlineBody2 := "comment 2\n\n" + lifecycle.InlineMarker

	var mu sync.Mutex
	deletedIDs := make(map[int]bool)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/discussions") && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]Discussion{
				{ID: "d1", Notes: []Note{
					{ID: 10, Body: inlineBody1, System: false},
					{ID: 20, Body: inlineBody2, System: false},
				}},
			})
			return
		}
		if r.Method == http.MethodDelete {
			parts := strings.Split(r.URL.Path, "/")
			noteIDStr := parts[len(parts)-1]
			var noteID int
			for _, c := range noteIDStr {
				noteID = noteID*10 + int(c-'0')
			}
			mu.Lock()
			deletedIDs[noteID] = true
			mu.Unlock()
			if noteID == 10 {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"message": "internal error"})
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.ClearInline(context.Background())
	if err == nil {
		t.Fatal("expected error when delete fails")
	}
	if !strings.Contains(err.Error(), "1") {
		t.Errorf("expected error to mention failure count, got: %v", err)
	}
	if result.InlineDeleted != 1 {
		t.Errorf("expected 1 successful delete, got %d", result.InlineDeleted)
	}
	if len(deletedIDs) != 2 {
		t.Errorf("expected 2 delete attempts, got %d", len(deletedIDs))
	}
}

func TestClear_DoesNotDeleteUnmarkedNotes(t *testing.T) {
	userBody := "user comment without any marker"

	var deleteCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/discussions") && r.Method == http.MethodGet {
			json.NewEncoder(w).Encode([]Discussion{
				{ID: "d1", Notes: []Note{{ID: 10, Body: userBody, System: false}}},
			})
			return
		}
		if r.Method == http.MethodDelete {
			deleteCount++
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})

	result, err := pub.ClearInline(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleteCount != 0 {
		t.Errorf("expected 0 deletes for unmarked notes, got %d", deleteCount)
	}
	if result.InlineDeleted != 0 {
		t.Errorf("expected 0 inline deleted, got %d", result.InlineDeleted)
	}

	result, err = pub.ClearSummary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deleteCount != 0 {
		t.Errorf("expected 0 deletes for unmarked notes, got %d", deleteCount)
	}
	if result.SummaryDeleted != 0 {
		t.Errorf("expected 0 summary deleted, got %d", result.SummaryDeleted)
	}
}

func TestPublish_SummaryFailureIsFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{
				{BaseCommitSHA: "b", StartCommitSHA: "s", HeadCommitSHA: "h"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode(Discussion{ID: "d", Notes: []Note{{ID: 1, Body: "ok"}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "internal error"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	_, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "fix", StartLine: 1, EndLine: 1},
		},
	})
	if err == nil {
		t.Fatal("expected fatal error for summary failure")
	}
	if !strings.Contains(err.Error(), "summary") {
		t.Errorf("expected error context 'summary', got: %v", err)
	}
}

func TestPublish_AllFindingsRemainInSummaryWhenInlineSkipped(t *testing.T) {
	var summaryBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/versions"):
			json.NewEncoder(w).Encode([]DiffVersion{
				{BaseCommitSHA: "b", StartCommitSHA: "s", HeadCommitSHA: "h"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/diffs"):
			json.NewEncoder(w).Encode([]ChangedFile{})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/discussions"):
			json.NewEncoder(w).Encode([]Discussion{})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/notes"):
			var req struct{ Body string }
			json.NewDecoder(r.Body).Decode(&req)
			summaryBody = req.Body
			json.NewEncoder(w).Encode(Note{ID: 200, Body: req.Body})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pub := NewPublisher(client, PublishOptions{Project: "proj", MergeRequestIID: 1})
	result, err := pub.Publish(context.Background(), review.Result{
		Findings: []review.Finding{
			{Path: "a.go", Content: "skipped finding", StartLine: 1, EndLine: 1},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InlineSkipped != 1 {
		t.Errorf("expected 1 inline skipped, got %d", result.InlineSkipped)
	}
	if !strings.Contains(summaryBody, "a.go") {
		t.Errorf("summary should include skipped finding path, got: %s", summaryBody)
	}
	if !strings.Contains(summaryBody, lifecycle.SummaryMarker) {
		t.Errorf("summary should contain summary marker, got: %s", summaryBody)
	}
}
