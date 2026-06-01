package gitlab

import "strings"

// AddedLines parses a unified diff and returns the set of new-file line numbers
// that were added (lines beginning with +, excluding file headers).
func AddedLines(diff string) map[int]struct{} {
	result := make(map[int]struct{})
	var newLine int
	inHunk := false

	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "@@") {
			// Parse hunk header: @@ -old,count +new,count @@
			newLine = parseHunkNewStart(line)
			inHunk = true
			continue
		}
		if !inHunk {
			continue
		}
		if strings.HasPrefix(line, "+++") {
			continue
		}
		if strings.HasPrefix(line, "---") {
			continue
		}
		if strings.HasPrefix(line, "+") {
			result[newLine] = struct{}{}
			newLine++
		} else if strings.HasPrefix(line, "-") {
			// Removed line: does not advance new line number.
		} else {
			// Context line (starts with space or is empty).
			newLine++
		}
	}
	return result
}

// SelectAddedLineInRange returns the first added line within [startLine, endLine].
// If endLine < startLine, treats as single-line range at startLine.
// Returns (0, false) if no anchor is available.
func SelectAddedLineInRange(diff string, startLine, endLine int) (int, bool) {
	if startLine <= 0 {
		return 0, false
	}
	if endLine < startLine {
		endLine = startLine
	}

	added := AddedLines(diff)
	if len(added) == 0 {
		return 0, false
	}

	for line := startLine; line <= endLine; line++ {
		if _, ok := added[line]; ok {
			return line, true
		}
	}
	return 0, false
}

// parseHunkNewStart extracts the start line number of the new file from a @@ header.
// Returns 0 if unparseable.
func parseHunkNewStart(header string) int {
	// Format: @@ -old_start[,old_count] +new_start[,new_count] @@
	idx := strings.Index(header, " +")
	if idx < 0 {
		return 0
	}
	rest := header[idx+2:]
	end := strings.Index(rest, " ")
	if end < 0 {
		end = len(rest)
	}
	numPart := rest[:end]
	// Remove optional comma and count.
	if comma := strings.Index(numPart, ","); comma >= 0 {
		numPart = numPart[:comma]
	}
	n := 0
	for _, c := range numPart {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return 0
		}
	}
	return n
}
