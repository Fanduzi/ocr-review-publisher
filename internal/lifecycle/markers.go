// Package lifecycle manages marker ownership for publisher comments.
package lifecycle

import "github.com/Fanduzi/ocr-review-publisher/internal/render"

// InlineMarker identifies publisher-owned inline comments.
const InlineMarker = render.InlineMarker

// SummaryMarker identifies publisher-owned summary comments.
const SummaryMarker = render.SummaryMarker
