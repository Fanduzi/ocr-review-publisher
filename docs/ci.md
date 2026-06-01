# CI Integration

This document covers CI/CD integration for `ocr-review-publisher`.

## GitLab CI

### Basic Example

```yaml
review:
  stage: review
  image: golang:1.22
  script:
    # Install OCR
    - npm install -g @alibaba-group/open-code-review

    # Install publisher
    - go build -o ocr-review-publisher ./cmd/ocr-review-publisher

    # Generate OCR output
    - ocr review --from origin/main --to HEAD --format json --audience agent > ocr-result.json

    # Publish to GitLab
    - ./ocr-review-publisher publish
        --platform gitlab
        --input ocr-result.json
        --format text
  variables:
    GITLAB_TOKEN: ${CI_JOB_TOKEN}
    CI_PROJECT_ID: ${CI_PROJECT_ID}
    CI_MERGE_REQUEST_IID: ${CI_MERGE_REQUEST_IID}
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
```

### With Clear Before Publish

```yaml
review:
  stage: review
  script:
    - ./ocr-review-publisher clear --platform gitlab --scope all || true
    - ./ocr-review-publisher publish --platform gitlab --input ocr-result.json
```

### Dry Run

Use `--dry-run` to preview without publishing:

```yaml
review-dry-run:
  stage: review
  script:
    - ./ocr-review-publisher publish --platform gitlab --input ocr-result.json --dry-run
```

### JSON Output

Use `--format json` for machine-readable output:

```yaml
review:
  stage: review
  script:
    - ./ocr-review-publisher publish --platform gitlab --input ocr-result.json --format json > publish-report.json
  artifacts:
    paths:
      - publish-report.json
```

## Required Environment Variables

| Variable | Description | Source |
|----------|-------------|--------|
| `GITLAB_TOKEN` | GitLab API token | CI/CD variable or `CI_JOB_TOKEN` |
| `CI_PROJECT_ID` | Project ID | Built-in GitLab CI variable |
| `CI_MERGE_REQUEST_IID` | MR IID | Built-in GitLab CI variable |
| `CI_SERVER_URL` | GitLab URL | Built-in GitLab CI variable |

## GitHub Actions

### PR CI

The project uses GitHub Actions for its own CI:

```yaml
# .github/workflows/ci.yml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: make check
```

### OCR Compatibility

Weekly scheduled check for OCR output compatibility:

```yaml
# .github/workflows/ocr-compatibility.yml
name: OCR Compatibility
on:
  schedule:
    - cron: '23 3 * * 1'
  workflow_dispatch:
```

This runs fixture compatibility tests without secrets. Optional live capture when LLM credentials are configured.

## What CI Should Fail On

### Current Behavior

- Parser compatibility failures
- Build/test/vet failures
- Format check failures

### Future Behavior (When Implemented)

- Rendered comment quality failures
- Severity/category gate failures (when OCR provides these fields)

## Output Modes

### Text Mode (Default)

Human-readable output:

```
Published: 3 inline, skipped 1, failed 0
Summary: created
```

### JSON Mode

Machine-readable output:

```json
{
  "inline_published": 3,
  "inline_skipped": 1,
  "inline_failed": 0,
  "summary_created": true,
  "summary_updated": false
}
```

JSON mode prints only JSON to stdout. Diagnostics and errors go to stderr.
