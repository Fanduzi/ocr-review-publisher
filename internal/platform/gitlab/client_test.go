package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient_DefaultBaseURL(t *testing.T) {
	c := NewClient("", "tok", nil)
	if c.baseURL != "https://gitlab.com" {
		t.Errorf("expected default baseURL https://gitlab.com, got %s", c.baseURL)
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("https://gitlab.example.com/", "tok", nil)
	if strings.HasSuffix(c.baseURL, "/") {
		t.Errorf("expected trailing slash trimmed, got %s", c.baseURL)
	}
}

func TestClient_SendsPrivateTokenHeader(t *testing.T) {
	var gotToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("PRIVATE-TOKEN")
		json.NewEncoder(w).Encode([]any{})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "my-secret-token", nil)
	_, err := client.ListChangedFiles(context.Background(), "proj", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotToken != "my-secret-token" {
		t.Errorf("expected PRIVATE-TOKEN my-secret-token, got %s", gotToken)
	}
}

func TestProjectEscaping_NamespacePath(t *testing.T) {
	var capturedRawPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRawPath = r.URL.RawPath
		json.NewEncoder(w).Encode([]any{})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	_, err := client.ListChangedFiles(context.Background(), "group/sub/project", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedRawPath == "" {
		t.Fatal("expected non-empty raw path")
	}
	if !strings.Contains(capturedRawPath, "group%2Fsub%2Fproject") {
		t.Errorf("project path not escaped in raw path, got: %s", capturedRawPath)
	}
}

func TestHTTPError_IncludesMethodPathStatusBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"403 Forbidden"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	_, err := client.ListChangedFiles(context.Background(), "proj", 1)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "403") {
		t.Errorf("expected error to contain 403, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "GET") {
		t.Errorf("expected error to contain GET method, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "/api/v4/") {
		t.Errorf("expected error to contain API path, got: %s", errMsg)
	}
}

func TestListChangedFiles_DiffsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/merge_requests/42/diffs") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]map[string]string{
			{"old_path": "a.go", "new_path": "a.go", "diff": "@@ -1 +1 @@\n-old\n+new"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	files, err := client.ListChangedFiles(context.Background(), "proj", 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].NewPath != "a.go" {
		t.Errorf("expected NewPath a.go, got %s", files[0].NewPath)
	}
	if files[0].Diff == "" {
		t.Error("expected non-empty diff")
	}
}

func TestListChangedFiles_FallsBackToChangesOn404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/diffs") {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"404 Not Found"}`))
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/changes") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"changes": []map[string]string{
				{"old_path": "b.go", "new_path": "b.go", "diff": "@@ -1 +1 @@\n-x\n+y"},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	files, err := client.ListChangedFiles(context.Background(), "proj", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].NewPath != "b.go" {
		t.Errorf("expected NewPath b.go, got %s", files[0].NewPath)
	}
}

func TestListChangedFiles_FallbackSendsAccessRawDiffs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/diffs") {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"404"}`))
			return
		}
		if r.URL.Query().Get("access_raw_diffs") != "true" {
			t.Errorf("missing access_raw_diffs query param: %s", r.URL.RawQuery)
		}
		json.NewEncoder(w).Encode(map[string]any{"changes": []any{}})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	_, err := client.ListChangedFiles(context.Background(), "proj", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListDiffVersions_DecodesVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/versions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode([]map[string]any{
			{"id": 1, "base_commit_sha": "abc", "start_commit_sha": "def", "head_commit_sha": "ghi"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	versions, err := client.ListDiffVersions(context.Background(), "proj", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}
	if versions[0].BaseCommitSHA != "abc" {
		t.Errorf("expected BaseCommitSHA abc, got %s", versions[0].BaseCommitSHA)
	}
}

func TestListDiscussions_DecodesDiscussionAndNotes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]Discussion{
			{ID: "disc-1", Notes: []Note{{ID: 1, Body: "note-1", System: false}}},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	discs, err := client.ListDiscussions(context.Background(), "proj", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(discs) != 1 {
		t.Fatalf("expected 1 discussion, got %d", len(discs))
	}
	if discs[0].ID != "disc-1" {
		t.Errorf("expected ID disc-1, got %s", discs[0].ID)
	}
	if len(discs[0].Notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(discs[0].Notes))
	}
	if discs[0].Notes[0].Body != "note-1" {
		t.Errorf("expected note body note-1, got %s", discs[0].Notes[0].Body)
	}
}

func TestListDiscussions_FetchesMultiplePages(t *testing.T) {
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RawQuery)
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("X-Next-Page", "2")
			json.NewEncoder(w).Encode([]Discussion{{ID: "d1"}})
		case "2":
			w.Header().Set("X-Next-Page", "")
			json.NewEncoder(w).Encode([]Discussion{{ID: "d2"}})
		default:
			t.Errorf("unexpected page: %s", r.URL.Query().Get("page"))
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	discs, err := client.ListDiscussions(context.Background(), "proj", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(discs) != 2 {
		t.Fatalf("expected 2 discussions, got %d", len(discs))
	}
	if discs[0].ID != "d1" || discs[1].ID != "d2" {
		t.Errorf("expected [d1, d2], got [%s, %s]", discs[0].ID, discs[1].ID)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(requests))
	}
	if !strings.Contains(requests[0], "per_page=100") {
		t.Errorf("expected per_page=100 in first request, got %s", requests[0])
	}
	if !strings.Contains(requests[1], "page=2") {
		t.Errorf("expected page=2 in second request, got %s", requests[1])
	}
}

func TestListDiscussions_FollowsNonSequentialNextPage(t *testing.T) {
	var requestedPages []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPages = append(requestedPages, r.URL.Query().Get("page"))
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("X-Next-Page", "3")
			json.NewEncoder(w).Encode([]Discussion{{ID: "d1"}})
		case "3":
			w.Header().Set("X-Next-Page", "")
			json.NewEncoder(w).Encode([]Discussion{{ID: "d3"}})
		default:
			t.Errorf("unexpected page: %s", r.URL.Query().Get("page"))
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	discs, err := client.ListDiscussions(context.Background(), "proj", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(discs) != 2 {
		t.Fatalf("expected 2 discussions, got %d", len(discs))
	}
	if discs[0].ID != "d1" || discs[1].ID != "d3" {
		t.Errorf("expected [d1, d3], got [%s, %s]", discs[0].ID, discs[1].ID)
	}
	if len(requestedPages) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(requestedPages))
	}
	if requestedPages[1] != "3" {
		t.Errorf("expected second request page=3, got page=%s", requestedPages[1])
	}
}

func TestListDiscussions_SecondPageErrorReturned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("X-Next-Page", "2")
			json.NewEncoder(w).Encode([]Discussion{{ID: "d1"}})
		case "2":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"500 Internal Server Error"}`))
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	_, err := client.ListDiscussions(context.Background(), "proj", 1)
	if err == nil {
		t.Fatal("expected error from second page, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to contain 500, got: %v", err)
	}
}

func TestCreateNote_SendsBodyToNotes(t *testing.T) {
	var capturedBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/notes") {
			t.Errorf("expected path ending in /notes, got %s", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&capturedBody)
		json.NewEncoder(w).Encode(map[string]any{"id": 99, "body": capturedBody["body"]})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	note, err := client.CreateNote(context.Background(), "proj", 1, "hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if note.Body != "hello world" {
		t.Errorf("expected body hello world, got %s", note.Body)
	}
	if capturedBody["body"] != "hello world" {
		t.Errorf("expected captured body hello world, got %s", capturedBody["body"])
	}
}

func TestUpdateNote_SendsPUTToNoteID(t *testing.T) {
	var capturedMethod string
	var capturedBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		json.NewDecoder(r.Body).Decode(&capturedBody)
		if !strings.HasSuffix(r.URL.Path, "/notes/42") {
			t.Errorf("expected path ending in /notes/42, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	err := client.UpdateNote(context.Background(), "proj", 1, 42, "updated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedMethod != http.MethodPut {
		t.Errorf("expected PUT, got %s", capturedMethod)
	}
	if capturedBody["body"] != "updated" {
		t.Errorf("expected body updated, got %s", capturedBody["body"])
	}
}

func TestDeleteNote_SendsDELETEToNoteID(t *testing.T) {
	var capturedMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		if !strings.HasSuffix(r.URL.Path, "/notes/55") {
			t.Errorf("expected path ending in /notes/55, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	err := client.DeleteNote(context.Background(), "proj", 1, 55)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", capturedMethod)
	}
}

func TestCreateInlineDiscussion_SendsBodyAndPosition(t *testing.T) {
	var capturedPayload map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/discussions") {
			t.Errorf("expected path ending in /discussions, got %s", r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&capturedPayload)
		json.NewEncoder(w).Encode(map[string]any{"id": "disc-new"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "tok", nil)
	pos := Position{
		PositionType: "text",
		BaseSHA:      "base",
		StartSHA:     "start",
		HeadSHA:      "head",
		OldPath:      "old.go",
		NewPath:      "new.go",
		NewLine:      10,
	}
	_, err := client.CreateInlineDiscussion(context.Background(), "proj", 1, "inline comment", pos)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedPayload["body"] != "inline comment" {
		t.Errorf("expected body 'inline comment', got %v", capturedPayload["body"])
	}
	posMap, ok := capturedPayload["position"].(map[string]any)
	if !ok {
		t.Fatalf("expected position to be a map, got %T", capturedPayload["position"])
	}
	for _, key := range []string{"position_type", "base_sha", "start_sha", "head_sha", "old_path", "new_path", "new_line"} {
		if _, exists := posMap[key]; !exists {
			t.Errorf("position missing key %s", key)
		}
	}
}
