package gitlab

// ChangedFile represents a file changed in a merge request.
type ChangedFile struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	Diff        string `json:"diff"`
	NewFile     bool   `json:"new_file"`
	DeletedFile bool   `json:"deleted_file"`
	RenamedFile bool   `json:"renamed_file"`
}

// DiffVersion represents a GitLab MR diff version.
type DiffVersion struct {
	ID             int    `json:"id"`
	BaseCommitSHA  string `json:"base_commit_sha"`
	StartCommitSHA string `json:"start_commit_sha"`
	HeadCommitSHA  string `json:"head_commit_sha"`
}

// Discussion represents a GitLab MR discussion thread.
type Discussion struct {
	ID    string `json:"id"`
	Notes []Note `json:"notes"`
}

// Note represents a single note (comment) in a GitLab MR discussion.
type Note struct {
	ID     int    `json:"id"`
	Body   string `json:"body"`
	System bool   `json:"system"`
}

// Position represents a text position for inline discussions.
type Position struct {
	PositionType string `json:"position_type"`
	BaseSHA      string `json:"base_sha"`
	StartSHA     string `json:"start_sha"`
	HeadSHA      string `json:"head_sha"`
	OldPath      string `json:"old_path"`
	NewPath      string `json:"new_path"`
	OldLine      int    `json:"old_line,omitempty"`
	NewLine      int    `json:"new_line,omitempty"`
}

// HTTPError represents a non-2xx response from the GitLab API.
type HTTPError struct {
	Method     string
	Path       string
	StatusCode int
	Body       string
}

func (e HTTPError) Error() string {
	return e.Method + " " + e.Path + " returned " + itoa(e.StatusCode) + ": " + e.Body
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
