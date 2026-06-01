package gitlab

import "testing"

func TestAddedLines_SingleHunk(t *testing.T) {
	diff := `@@ -1,4 +1,5 @@
 context
-old
+new1
+new2
 context
`
	got := AddedLines(diff)
	if len(got) != 2 {
		t.Fatalf("expected 2 added lines, got %d", len(got))
	}
	if _, ok := got[2]; !ok {
		t.Errorf("expected added line 2")
	}
	if _, ok := got[3]; !ok {
		t.Errorf("expected added line 3")
	}
}

func TestAddedLines_MultipleHunks(t *testing.T) {
	diff := `@@ -1,3 +1,4 @@
 context
+added1
 context
@@ -10,3 +11,4 @@
 context
+added2
 context
`
	got := AddedLines(diff)
	if len(got) != 2 {
		t.Fatalf("expected 2 added lines, got %d", len(got))
	}
	if _, ok := got[2]; !ok {
		t.Errorf("expected added line 2 from first hunk")
	}
	if _, ok := got[12]; !ok {
		t.Errorf("expected added line 12 from second hunk")
	}
}

func TestAddedLines_IgnoresFileHeaders(t *testing.T) {
	diff := `--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
 context
+added
 context
`
	got := AddedLines(diff)
	if len(got) != 1 {
		t.Fatalf("expected 1 added line, got %d", len(got))
	}
	if _, ok := got[2]; !ok {
		t.Errorf("expected added line 2, got keys: %v", mapKeys(got))
	}
}

func TestAddedLines_NewFile(t *testing.T) {
	diff := `@@ -0,0 +1,3 @@
+line1
+line2
+line3
`
	got := AddedLines(diff)
	if len(got) != 3 {
		t.Fatalf("expected 3 added lines, got %d", len(got))
	}
	for _, line := range []int{1, 2, 3} {
		if _, ok := got[line]; !ok {
			t.Errorf("expected added line %d", line)
		}
	}
}

func TestAddedLines_DeletedFile(t *testing.T) {
	diff := `@@ -1,3 +0,0 @@
-old1
-old2
-old3
`
	got := AddedLines(diff)
	if len(got) != 0 {
		t.Errorf("expected 0 added lines for deleted file, got %d", len(got))
	}
}

func TestSelectAddedLineInRange_StartLineAdded(t *testing.T) {
	diff := `@@ -1,3 +1,4 @@
 context
+added
 context
`
	line, ok := SelectAddedLineInRange(diff, 2, 2)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if line != 2 {
		t.Errorf("expected line 2, got %d", line)
	}
}

func TestSelectAddedLineInRange_ContextStartAddedLaterInRange(t *testing.T) {
	diff := `@@ -1,5 +1,6 @@
 context
 context
+added
 context
 context
`
	line, ok := SelectAddedLineInRange(diff, 2, 4)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if line != 3 {
		t.Errorf("expected line 3, got %d", line)
	}
}

func TestSelectAddedLineInRange_NoAddedLineInRange(t *testing.T) {
	diff := `@@ -1,5 +1,5 @@
 context
+added
 context
 context
 context
`
	line, ok := SelectAddedLineInRange(diff, 3, 5)
	if ok {
		t.Errorf("expected ok=false, got line %d", line)
	}
}

func TestSelectAddedLineInRange_EndBeforeStartTreatsSingleLine(t *testing.T) {
	diff := `@@ -1,3 +1,4 @@
 context
+added
 context
`
	line, ok := SelectAddedLineInRange(diff, 2, 1)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if line != 2 {
		t.Errorf("expected line 2, got %d", line)
	}
}

func TestSelectAddedLineInRange_StartLineZero(t *testing.T) {
	diff := `@@ -1,3 +1,4 @@
+added
 context
`
	_, ok := SelectAddedLineInRange(diff, 0, 5)
	if ok {
		t.Errorf("expected ok=false for start line 0")
	}
}

func TestSelectAddedLineInRange_EmptyDiff(t *testing.T) {
	_, ok := SelectAddedLineInRange("", 1, 5)
	if ok {
		t.Errorf("expected ok=false for empty diff")
	}
}

func TestSelectAddedLineInRange_NoHunkHeaders(t *testing.T) {
	diff := `just some text without hunk headers`
	_, ok := SelectAddedLineInRange(diff, 1, 5)
	if ok {
		t.Errorf("expected ok=false for diff without hunk headers")
	}
}

func TestSelectAddedLineInRange_AddedLineAtRangeEnd(t *testing.T) {
	diff := `@@ -1,5 +1,6 @@
 context
 context
 context
 context
+added
`
	line, ok := SelectAddedLineInRange(diff, 1, 5)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if line != 5 {
		t.Errorf("expected line 5, got %d", line)
	}
}

func TestSelectAddedLineInRange_AddedLineBeforeRangeIgnored(t *testing.T) {
	diff := `@@ -1,5 +1,6 @@
+added
 context
 context
 context
 context
`
	_, ok := SelectAddedLineInRange(diff, 2, 5)
	if ok {
		t.Errorf("expected ok=false, added line 1 is before range [2,5]")
	}
}

func mapKeys(m map[int]struct{}) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
