# OCR Review Publisher

`ocr-review-publisher` is a platform publishing layer for Open Code Review output.

Open Code Review generates review findings. This project consumes its machine-readable output and publishes those findings as high-quality GitLab merge request comments.

Version 1 is intentionally narrow:

- OCR is the only supported review producer.
- GitLab is the only supported publishing platform.
- The project focuses on comment rendering, safe inline anchors, summary updates, marker-scoped clear operations, and CI-friendly execution.

Project documentation:

- [Quality Gates](docs/quality-gates.md)
- [OCR Compatibility](docs/ocr-compatibility.md)
- [Release Process](docs/release.md)
- [GitLab E2E Testing](docs/e2e-gitlab.md)
- [Contributing](CONTRIBUTING.md)

This repository is not an OCR fork and does not replace OCR's review engine.

## Quick Start

```bash
# Build
make build

# Run all quality checks
make check

# Show help
./ocr-review-publisher --help
./ocr-review-publisher help
./ocr-review-publisher version
```

## Development

```bash
# Format code
make fmt

# Run tests
make test

# Run go vet
make vet

# Build binary
make build

# Run all checks (fmt + test + vet + build + compat)
make check

# Run OCR output compatibility tests
make test-compat

# Run GitLab e2e tests (opt-in, requires env vars)
make test-e2e-gitlab
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development workflow, TDD requirements, and quality gates.

## Quality Gates

This project enforces quality gates to ensure rendered comments are readable and safe to publish:

- Unit tests for all changed behavior.
- Golden Markdown tests for renderer output.
- OCR output compatibility tests against captured fixtures.
- GitLab e2e tests for publishing changes (opt-in).
- No tokens, local paths, or secrets in tracked files.

See [docs/quality-gates.md](docs/quality-gates.md) for the full checklist.

## OCR Compatibility

The parser depends on `ocr review --format json --audience agent` output. This is an external contract owned by Open Code Review. The project runs compatibility tests against captured OCR output fixtures. See [docs/ocr-compatibility.md](docs/ocr-compatibility.md) for details.
