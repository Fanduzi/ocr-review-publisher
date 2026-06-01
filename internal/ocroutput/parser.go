package ocroutput

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Fanduzi/ocr-review-publisher/internal/review"
)

// rawResult mirrors the OCR JSON shape for unmarshalling.
type rawResult struct {
	Status   string          `json:"status"`
	Message  string          `json:"message"`
	Comments json.RawMessage `json:"comments"`
	Warnings json.RawMessage `json:"warnings"`
}

// rawComment mirrors an OCR comment entry.
type rawComment struct {
	Path           string         `json:"path"`
	Content        string         `json:"content"`
	ExistingCode   string         `json:"existing_code"`
	SuggestionCode string         `json:"suggestion_code"`
	StartLine      int            `json:"start_line"`
	EndLine        int            `json:"end_line"`
	Thinking       string         `json:"thinking"`
	Category       string         `json:"category"`
	Severity       string         `json:"severity"`
	Extra          map[string]any `json:"-"`
}

// rawWarning mirrors an OCR warning entry.
type rawWarning struct {
	Type    string         `json:"type"`
	Path    string         `json:"path"`
	Message string         `json:"message"`
	Extra   map[string]any `json:"-"`
}

func (r *rawComment) UnmarshalJSON(data []byte) error {
	type Alias rawComment
	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*r = rawComment(a)

	var extra map[string]any
	if err := json.Unmarshal(data, &extra); err != nil {
		return err
	}
	delete(extra, "path")
	delete(extra, "content")
	delete(extra, "existing_code")
	delete(extra, "suggestion_code")
	delete(extra, "start_line")
	delete(extra, "end_line")
	delete(extra, "thinking")
	delete(extra, "category")
	delete(extra, "severity")
	if len(extra) > 0 {
		r.Extra = extra
	}
	return nil
}

func (w *rawWarning) UnmarshalJSON(data []byte) error {
	type Alias rawWarning
	var a Alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*w = rawWarning(a)

	var extra map[string]any
	if err := json.Unmarshal(data, &extra); err != nil {
		return err
	}
	delete(extra, "type")
	delete(extra, "path")
	delete(extra, "message")
	if len(extra) > 0 {
		w.Extra = extra
	}
	return nil
}

// Parse parses OCR review output from a byte slice.
func Parse(data []byte) (*review.Result, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("ocroutput: empty input")
	}

	start := bytes.IndexByte(data, '{')
	if start < 0 {
		return nil, fmt.Errorf("ocroutput: no JSON object found in input")
	}

	var raw rawResult
	if err := json.Unmarshal(data[start:], &raw); err != nil {
		return nil, fmt.Errorf("ocroutput: failed to parse JSON: %w", err)
	}

	result := &review.Result{
		Status:  raw.Status,
		Message: raw.Message,
	}

	// Parse top-level unknown fields.
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data[start:], &top); err == nil {
		delete(top, "status")
		delete(top, "message")
		delete(top, "comments")
		delete(top, "warnings")
		if len(top) > 0 {
			meta := make(map[string]any, len(top))
			for k, v := range top {
				var val any
				if err := json.Unmarshal(v, &val); err == nil {
					meta[k] = val
				}
			}
			result.Metadata = meta
		}
	}

	// Parse comments into findings.
	if raw.Comments != nil {
		var comments []rawComment
		if err := json.Unmarshal(raw.Comments, &comments); err != nil {
			return nil, fmt.Errorf("ocroutput: failed to parse comments: %w", err)
		}
		findings := make([]review.Finding, len(comments))
		for i, c := range comments {
			f := review.Finding{
				Path:           c.Path,
				Content:        c.Content,
				ExistingCode:   c.ExistingCode,
				SuggestionCode: c.SuggestionCode,
				StartLine:      c.StartLine,
				EndLine:        c.EndLine,
				Thinking:       c.Thinking,
				Category:       c.Category,
				Severity:       c.Severity,
			}
			if len(c.Extra) > 0 {
				f.Metadata = c.Extra
			}
			findings[i] = f
		}
		result.Findings = findings
	} else {
		result.Findings = []review.Finding{}
	}

	// Parse warnings.
	if raw.Warnings != nil {
		var warnings []rawWarning
		if err := json.Unmarshal(raw.Warnings, &warnings); err != nil {
			return nil, fmt.Errorf("ocroutput: failed to parse warnings: %w", err)
		}
		out := make([]review.Warning, len(warnings))
		for i, w := range warnings {
			ww := review.Warning{
				Type:    w.Type,
				Path:    w.Path,
				Message: w.Message,
			}
			if len(w.Extra) > 0 {
				ww.Metadata = w.Extra
			}
			out[i] = ww
		}
		result.Warnings = out
	}

	return result, nil
}

// ParseReader parses OCR review output from an io.Reader.
func ParseReader(r io.Reader) (*review.Result, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("ocroutput: failed to read input: %w", err)
	}
	return Parse(data)
}

// ParseFile parses OCR review output from a file path.
func ParseFile(path string) (*review.Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ocroutput: failed to read file %s: %w", path, err)
	}
	return Parse(data)
}
