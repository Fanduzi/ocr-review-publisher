# Output Contract

This document describes the OCR output format accepted by `ocr-review-publisher`.

## Accepted Format

The parser accepts output from:

```bash
ocr review --format json --audience agent
```

## JSON Structure

```json
{
  "status": "success",
  "message": "Review completed",
  "comments": [
    {
      "path": "service/user.go",
      "content": "Error return value is not checked",
      "existing_code": "fmt.Println(err)",
      "suggestion_code": "if err != nil {\n\treturn err\n}",
      "start_line": 37,
      "end_line": 37,
      "thinking": "The error is silently discarded."
    }
  ],
  "warnings": [
    {
      "type": "subtask_failed",
      "path": "service/user.go",
      "message": "Could not analyze file"
    }
  ]
}
```

## Fields

### Top-Level

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `status` | string | No | Review status (e.g., "success", "partial") |
| `message` | string | No | Human-readable status message |
| `comments` | array | No | Array of finding objects |
| `warnings` | array | No | Array of warning objects |

If `comments` is missing or null, the parser returns an empty findings slice.

### Comment (Finding)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | File path relative to repository root |
| `content` | string | Yes | Finding description |
| `existing_code` | string | No | Code snippet from the file |
| `suggestion_code` | string | No | Suggested replacement code |
| `start_line` | int | No | Start line number (0 if unknown) |
| `end_line` | int | No | End line number (0 if unknown) |
| `thinking` | string | No | Reviewer reasoning/context |
| `category` | string | No | Finding category (e.g., "security", "performance") |
| `severity` | string | No | Finding severity (e.g., "high", "medium", "low") |

### Warning

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Warning type (e.g., "subtask_failed", "timeout") |
| `path` | string | No | Associated file path |
| `message` | string | Yes | Warning description |

## Prefixed Output

OCR may print human-readable text before the JSON object:

```
Review completed in 12.3s using model claude-sonnet-4-20250514
Found 1 finding across 1 file.
{
  "status": "success",
  "comments": [...]
}
```

The parser handles this by finding the first `{` character and parsing from there.

## Optional Fields

### Forward Compatibility

Unknown top-level fields are preserved in `Result.Metadata`.

Unknown comment fields are preserved in `Finding.Metadata`.

Unknown warning fields are preserved in `Warning.Metadata`.

### Category and Severity

When OCR output includes `category` and `severity` fields, they are mapped to strong typed fields in the internal model. When absent, they default to empty strings.

### Confidence

When OCR output includes a `confidence` field, it is preserved in `Finding.Metadata["confidence"]`.

## Line Numbers

- Missing line numbers default to 0, not failure
- Zero line numbers are valid (used for summary fallback)
- Negative line numbers are preserved as-is

## Error Handling

### Empty Input

Returns error: "empty input"

### No JSON Object

Returns error: "no JSON object found in input"

### Malformed JSON

Returns error with context: "failed to parse JSON: ..."

Error messages are concise and do not dump large input excerpts.

## Compatibility Policy

The parser should support:

- The minimum OCR version documented by this project
- The latest OCR version verified by scheduled CI
- Forward-compatible optional fields

The parser should be strict about malformed JSON but tolerant of harmless wrapper text before the JSON object.

## Fixtures

Compatibility fixtures are stored under `testdata/ocr/`:

- `basic.json` - Standard output with findings
- `prefixed-agent-output.txt` - Output with leading text
- `empty-comments.json` - Output with empty comments
- `with-warnings.json` - Output with warnings
- `future-fields.json` - Output with category, severity, and unknown fields

Fixtures must not contain secrets, private repository names, private URLs, or local filesystem paths.
