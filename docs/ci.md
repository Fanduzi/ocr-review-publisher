# CI Integration

This document covers CI/CD integration for `ocr-review-publisher`.

## GitLab CI

Use the publisher in your GitLab CI pipelines to publish [Open Code Review (OCR)](https://github.com/alibaba/open-code-review) findings to merge requests.

### Basic Example (Release Binary)

The example below uses a `node:22-bookworm` image (for the OCR CLI), downloads a pre-built `ocr-review-publisher` binary from GitHub Releases, and publishes review findings to the MR. No source clone or build step required.

```yaml
stages:
  - review

image: node:22-bookworm

variables:
  # Full clone so `origin/main` is available for diff comparison.
  GIT_DEPTH: "0"
  OCR_PUBLISHER_VERSION: "v0.1.3"

review:
  stage: review
  before_script:
    # Install OCR CLI.
    - npm install -g @alibaba-group/open-code-review
    # Install ocr-review-publisher from GitHub Releases.
    - PLATFORM=$(uname -s | tr '[:upper:]' '[:lower:]')_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
    - VERSION_NUM=${OCR_PUBLISHER_VERSION#v}
    - TARBALL="ocr-review-publisher_${VERSION_NUM}_${PLATFORM}.tar.gz"
    - curl -fsSL -o /tmp/${TARBALL} "https://github.com/Fanduzi/ocr-review-publisher/releases/download/${OCR_PUBLISHER_VERSION}/${TARBALL}"
    - curl -fsSL -o /tmp/checksums.txt "https://github.com/Fanduzi/ocr-review-publisher/releases/download/${OCR_PUBLISHER_VERSION}/ocr-review-publisher_${VERSION_NUM}_checksums.txt"
    - cd /tmp && sha256sum -c --ignore-missing checksums.txt && cd "$CI_PROJECT_DIR"
    - tar xzf /tmp/${TARBALL} -C /usr/local/bin ocr-review-publisher
    - ocr-review-publisher version
  script:
    - |
      ocr review --from origin/main --to HEAD --format json --audience agent > ocr-result.json
    - |
      ocr-review-publisher publish \
        --platform gitlab \
        --input ocr-result.json \
        --format text
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
```

> **Notes:**
> - The tarball filename must match the checksums file entries. The `TARBALL` variable preserves the original filename so `sha256sum -c` can verify it.
> - `--ignore-missing` is needed because the checksums file lists all platforms but you only download one.
> - The publisher binary is installed to `/usr/local/bin` and is available system-wide. Pin `OCR_PUBLISHER_VERSION` for reproducible builds; bump it when upgrading. See [GitHub Releases](https://github.com/Fanduzi/ocr-review-publisher/releases) for available versions.

### Basic Example (Build from Source)

If you prefer building from source (for development or custom forks):

```yaml
  before_script:
    # ... install OCR CLI as above ...
    # Build publisher from source.
    - git clone --depth=1 https://github.com/Fanduzi/ocr-review-publisher.git /tmp/ocr-publisher
    - cd /tmp/ocr-publisher && go build -o /usr/local/bin/ocr-review-publisher ./cmd/ocr-review-publisher
    - cd "$CI_PROJECT_DIR"
```

> **Token note:** Store a Personal Access Token, Project Access Token, or Group Access Token with `api` scope as `GITLAB_TOKEN` in CI/CD variables. The built-in `CI_JOB_TOKEN` has limited permissions and may not support creating merge request discussions. The publisher reads `GITLAB_TOKEN` from the environment automatically.

### With Clear Before Publish

```yaml
review:
  stage: review
  script:
    - |
      ocr-review-publisher clear --platform gitlab --scope all || true
    - |
      ocr review --from origin/main --to HEAD --format json --audience agent > ocr-result.json
    - |
      ocr-review-publisher publish \
        --platform gitlab \
        --input ocr-result.json
```

### Dry Run

Use `--dry-run` to preview without publishing:

```yaml
review-dry-run:
  stage: review
  script:
    - |
      ./ocr-review-publisher publish \
        --platform gitlab \
        --input ocr-result.json \
        --dry-run
```

### JSON Output

Use `--format json` for machine-readable output:

```yaml
review:
  stage: review
  script:
    - |
      ./ocr-review-publisher publish \
        --platform gitlab \
        --input ocr-result.json \
        --format json > publish-report.json
  artifacts:
    paths:
      - publish-report.json
```

## Required Environment Variables

### GitLab Variables (Built-in)

These are set automatically by GitLab CI in merge request pipelines:

| Variable | Description | Source |
|----------|-------------|--------|
| `CI_PROJECT_ID` | Project ID | Built-in GitLab CI variable |
| `CI_MERGE_REQUEST_IID` | MR IID | Built-in GitLab CI variable |
| `CI_SERVER_URL` | GitLab URL | Built-in GitLab CI variable |

### CI/CD Variables to Configure

Set these in **Settings > CI/CD > Variables**:

| Variable | Description | Masked |
|----------|-------------|--------|
| `GITLAB_TOKEN` | GitLab API token with `api` scope | Yes |

### OCR LLM Variables

The OCR CLI requires LLM credentials to perform code review. Set these as CI/CD variables:

| Variable | Description | Masked |
|----------|-------------|--------|
| `OCR_LLM_URL` | LLM API endpoint URL | No |
| `OCR_LLM_TOKEN` | LLM API token | Yes |
| `OCR_LLM_MODEL` | LLM model name | No |

## GitHub Actions (This Repository)

This section documents the GitHub Actions workflows used by the `ocr-review-publisher` project itself. These workflows are for CI/CD of the publisher tool, not for publishing to GitHub PRs (which is not supported).

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
          go-version: '1.26.1'
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
