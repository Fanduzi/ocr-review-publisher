# OCR Review Publisher

[![CI](https://github.com/Fanduzi/ocr-review-publisher/actions/workflows/ci.yml/badge.svg)](https://github.com/Fanduzi/ocr-review-publisher/actions/workflows/ci.yml)
![Go Version](https://img.shields.io/badge/go-1.26.1-00ADD8?logo=go)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

[![English](https://img.shields.io/badge/docs-English-blue)](README.md) [![简体中文](https://img.shields.io/badge/docs-简体中文-yellow)](README.zh-CN.md)

[![Quality Gates](https://img.shields.io/badge/Quality_Gates-informational)](docs/quality-gates.md) [![OCR Compatibility](https://img.shields.io/badge/OCR_Compatibility-informational)](docs/ocr-compatibility.md) [![Release Process](https://img.shields.io/badge/Release_Process-success)](docs/release.md) [![GitLab E2E](https://img.shields.io/badge/GitLab_E2E-informational)](docs/e2e-gitlab.md) [![Contributing](https://img.shields.io/badge/Contributing-important)](CONTRIBUTING.md)

`ocr-review-publisher` is a platform publishing layer for [Open Code Review](https://github.com/alibaba/open-code-review) output.

Open Code Review generates review findings. This project consumes its machine-readable output and publishes those findings as GitLab merge request comments.

## Background

This project started after an attempt to add platform publishing directly to Open Code Review ([PR #15](https://github.com/alibaba/open-code-review/pull/15)). The OCR maintainers preferred keeping OCR focused on review generation — a lightweight CLI with few dependencies — and leaving comment publishing to external scripts or CI actions ([PR #11](https://github.com/alibaba/open-code-review/pull/11)).

`ocr-review-publisher` follows that boundary: OCR produces findings; this tool publishes them to GitLab with marker-based ownership, summary lifecycle management, and CI smoke gates.

## Current Scope

Version 1 is intentionally narrow:

- OCR is the only supported review producer.
- GitLab is the only supported publishing platform.
- The project focuses on comment rendering, safe inline anchors, summary updates, marker-scoped clear operations, and CI-friendly execution.

**Not currently supported:**

- GitHub PR publishing
- Webhook/server mode
- Slash command triggers
- One-click platform suggestions
- Severity/category gating (unless OCR output provides these fields)

## Installation

### Download from GitHub Releases

Pre-built binaries are available for macOS and Linux (amd64/arm64):

```bash
# macOS arm64 (Apple Silicon)
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.3/ocr-review-publisher_0.1.3_darwin_arm64.tar.gz
tar xzf ocr-review-publisher_0.1.3_darwin_arm64.tar.gz

# Linux amd64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.3/ocr-review-publisher_0.1.3_linux_amd64.tar.gz
tar xzf ocr-review-publisher_0.1.3_linux_amd64.tar.gz
```

Verify checksums:

```bash
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.3/ocr-review-publisher_0.1.3_checksums.txt
sha256sum -c ocr-review-publisher_0.1.3_checksums.txt
```

See all releases: [GitHub Releases](https://github.com/Fanduzi/ocr-review-publisher/releases)

### Build from Source

```bash
git clone https://github.com/Fanduzi/ocr-review-publisher.git
cd ocr-review-publisher
make build
```

The binary is created at `./ocr-review-publisher`.

### Verify

```bash
./ocr-review-publisher version
./ocr-review-publisher --help
```

## Quick Start

### 1. Generate OCR Output

```bash
ocr review --from origin/main --to HEAD --format json --audience agent > ocr-result.json
```

### 2. Render Locally (Preview)

```bash
./ocr-review-publisher render --input ocr-result.json
```

### 3. Publish to GitLab

```bash
./ocr-review-publisher publish \
  --platform gitlab \
  --input ocr-result.json \
  --gitlab-base-url https://gitlab.example.com \
  --project-id group/project \
  --mr 123
```

### 4. Clear Publisher Comments

```bash
./ocr-review-publisher clear \
  --platform gitlab \
  --scope all \
  --gitlab-base-url https://gitlab.example.com \
  --project-id group/project \
  --mr 123
```

## Commands

### `render`

Render OCR findings as Markdown without publishing. Useful for debugging comment quality.

```bash
./ocr-review-publisher render --input ocr-result.json
./ocr-review-publisher render --input ocr-result.json --format json
./ocr-review-publisher render --input - < ocr-result.json
```

### `publish`

Publish OCR findings to a GitLab MR.

```bash
./ocr-review-publisher publish \
  --platform gitlab \
  --input ocr-result.json \
  --gitlab-base-url https://gitlab.example.com \
  --project-id group/project \
  --mr 123 \
  --token $GITLAB_TOKEN
```

Options:

- `--dry-run` - Render without publishing
- `--no-inline` - Skip inline comments
- `--no-summary` - Skip summary comment
- `--clear-existing` - Clear publisher comments first
- `--format text|json` - Output format

### `clear`

Clear publisher-owned comments from a GitLab MR.

```bash
./ocr-review-publisher clear --platform gitlab --scope inline ...
./ocr-review-publisher clear --platform gitlab --scope summary ...
./ocr-review-publisher clear --platform gitlab --scope all ...
```

### `version`

Print version information.

```bash
./ocr-review-publisher version
```

## GitLab Configuration

### Environment Variables

The CLI infers GitLab configuration from environment variables:

| Flag | Env Var | Description |
|------|---------|-------------|
| `--token` | `GITLAB_TOKEN` or `OCR_GITLAB_TOKEN` | GitLab API token |
| `--gitlab-base-url` | `OCR_GITLAB_BASE_URL` or `CI_SERVER_URL` | GitLab base URL (default: `https://gitlab.com`) |
| `--project-id` | `CI_PROJECT_ID` | Project ID or namespace path |
| `--mr` | `CI_MERGE_REQUEST_IID` | Merge request IID |

Flags override environment variables.

### Required Permissions

The GitLab token needs:

- `api` scope for creating/updating/deleting notes
- `read_repository` scope for reading MR diffs

### GitLab 13.12 Compatibility

The publisher supports GitLab 13.12 self-hosted instances:

- Uses `/diffs` endpoint with fallback to `/changes?access_raw_diffs=true`
- Paginates discussions with `per_page=100` and `X-Next-Page` headers

See [docs/gitlab.md](docs/gitlab.md) for detailed GitLab usage.

## Safety Model

All publisher-owned comments use stable markers:

```markdown
<!-- ocr-review-publisher:inline -->
<!-- ocr-review-publisher:summary -->
```

Clear operations only delete notes containing these markers. User comments and other bot comments are never touched.

## Development

```bash
make fmt              # Format code
make test             # Run tests
make vet              # Run go vet
make build            # Build binary
make check            # Run all checks (fmt + test + vet + build + compat)
make test-compat      # Run OCR output compatibility tests
make test-e2e-gitlab  # Run GitLab e2e tests (opt-in, requires env vars)
make release-readiness # Run strict pre-release gates
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development workflow and TDD requirements.

## Documentation

- [Quality Gates](docs/quality-gates.md) - Definition of done and release blockers
- [OCR Compatibility](docs/ocr-compatibility.md) - Parser compatibility policy and fixtures
- [GitLab Usage](docs/gitlab.md) - Detailed GitLab configuration and troubleshooting
- [CI Integration](docs/ci.md) - GitLab CI usage and project GitHub Actions workflows
- [Output Contract](docs/output-contract.md) - Accepted OCR output format
- [GitLab E2E Testing](docs/e2e-gitlab.md) - Opt-in real GitLab tests
- [Release Process](docs/release.md) - Release gates and workflow

## Limitations

- Only supports Open Code Review output (not other review engines)
- Only supports GitLab MR publishing (not GitHub PRs)
- Does not improve OCR's review intelligence; only publishes its findings
- Category/severity badges appear only when OCR output provides those fields
- No webhook/server mode
- No one-click platform suggestions
