# ocr-review-publisher v0.1.0 Release Notes

## Overview

First public release of `ocr-review-publisher`, a CLI tool that publishes [Open Code Review (OCR)](https://github.com/alibaba/open-code-review) findings as high-quality GitLab merge request comments.

## Highlights

- Publishes OCR review output as inline discussions and summary comments on GitLab MRs
- Stable marker-based ownership: publisher manages only its own comments
- Language-aware code fences for rendered findings
- Reproducible CI smoke gate for ongoing quality validation
- Cross-platform release assets (macOS and Linux, amd64 and arm64)

## What's New

### GitLab MR Publishing

- Parse OCR `--format json --audience agent` output from file or stdin
- Create inline discussions for findings with safe diff anchors
- Create and update a managed summary comment with finding counts and diagnostics
- Clear publisher-owned comments without affecting user or other bot comments
- Continue publishing when individual inline comments fail

### Markdown Rendering

- Language-aware fenced code blocks (Go, JavaScript, TypeScript, Python, Java, Rust, JSON, YAML, Markdown)
- Collapsed details blocks for review context and publish diagnostics
- Category and severity badges when OCR output includes those fields
- Stable ownership markers for lifecycle management

### CI Integration

- GitLab CI env inference (`CI_SERVER_URL`, `CI_PROJECT_ID`, `CI_MERGE_REQUEST_IID`)
- `--dry-run` mode for previewing without publishing
- `--format json` for machine-readable publish reports
- `--fail-on-publish-error` for strict CI pipelines

### Quality Gates

- Golden Markdown tests for all comment types
- Real OCR fixture tests against representative review output
- GitLab API shape tests with `httptest`
- Diff anchor selection tests for edge cases
- Opt-in GitLab 13.12 e2e tests
- Real OCR local smoke gate
- Reproducible GitLab CI smoke gate

## Installation

Download from GitHub Releases:

```bash
# Linux amd64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.0/ocr-review-publisher_0.1.0_linux_amd64.tar.gz
tar xzf ocr-review-publisher_0.1.0_linux_amd64.tar.gz

# macOS arm64 (Apple Silicon)
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.0/ocr-review-publisher_0.1.0_darwin_arm64.tar.gz
tar xzf ocr-review-publisher_0.1.0_darwin_arm64.tar.gz

# macOS amd64 (Intel)
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.0/ocr-review-publisher_0.1.0_darwin_amd64.tar.gz
tar xzf ocr-review-publisher_0.1.0_darwin_amd64.tar.gz
```

Verify checksums:

```bash
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.0/ocr-review-publisher_0.1.0_checksums.txt
sha256sum -c ocr-review-publisher_0.1.0_checksums.txt
```

Or build from source:

```bash
git clone https://github.com/Fanduzi/ocr-review-publisher.git
cd ocr-review-publisher
make build
```

## Compatibility

- Verified OCR version: v1.1.9
- Supported platform: GitLab (including 13.12 self-hosted)
- Supported operating systems: darwin, linux
- Supported architectures: amd64, arm64

## Verified Quality Gates

- `make check` (fmt, test, vet, build, test-compat): passed
- `make test-e2e-gitlab`: passed (GitLab 13.12)
- `make smoke-gitlab-real-ocr`: passed (real OCR v1.1.9 against fixture repository)
- `make smoke-gitlab-ci`: passed (reproducible CI smoke gate)
- `make release-readiness`: passed
- `make release-snapshot`: passed (darwin/linux amd64/arm64 archives with checksums)

## Known Limitations

- GitLab only: no GitHub PR publishing in this release
- No webhook, server, or slash-command mode
- No Homebrew, npm, or Docker packages (single binary distribution only)
- Category and severity badges render only when OCR output includes those fields
- True one-click GitLab suggestions are not enabled; suggestions render as normal fenced code blocks
- Requires OCR CLI installed separately to generate review input

## Migration Notes

- First release; no migration required
- Publisher uses stable markers (`<!-- ocr-review-publisher:inline -->`, `<!-- ocr-review-publisher:summary -->`) for comment ownership
