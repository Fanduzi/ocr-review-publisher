# OCR Output Compatibility

`ocr-review-publisher` depends on the machine-readable output from `ocr review --format json --audience agent`. That output is an external contract owned by [Open Code Review](https://github.com/alibaba/open-code-review), so this project must verify it continuously.

## Compatibility Policy

The parser should support:

- the minimum OCR version documented by this project;
- the latest OCR version verified by manual live capture (`workflow_dispatch`); and
- forward-compatible optional fields such as `category`, `severity`, or `confidence`.

The parser should be strict about malformed JSON, but tolerant of harmless wrapper text before the JSON object because some OCR modes may print a summary line before the structured payload.

## Fixture Strategy

Keep captured OCR outputs under `testdata/ocr/` or `testdata/fixtures/`.

Each fixture should record:

- OCR version;
- command used;
- whether the output came from a live LLM run or a sanitized sample;
- expected parser result shape.

Fixtures must not contain secrets, private repository names, private URLs, or local filesystem paths.

## Local Compatibility Flow

Run compatibility tests locally:

```bash
make test-compat
```

This runs `go test ./internal/compat -count=1` which validates all checked-in fixtures parse correctly and have the expected shape. No network or LLM credentials required.

For regenerating fixtures from a live OCR run:

```bash
scripts/capture-ocr-output.sh --ocr-version latest --output testdata/ocr/latest.json
```

The capture script creates a temporary sample repository with deterministic changes, installs the requested OCR version, runs `ocr review --format json --audience agent`, and writes the output to the specified path. LLM credentials must be configured for OCR to generate reviews.

## Checked-In Fixtures

Fixtures are stored under `testdata/ocr/`:

- `basic.json` - Standard OCR output with findings, existing code, suggestions, thinking
- `prefixed-agent-output.txt` - OCR output with non-JSON text before the JSON object
- `empty-comments.json` - OCR output with empty comments array
- `with-warnings.json` - OCR output with warnings
- `future-fields.json` - OCR output with category, severity, confidence, and unknown future fields
- `ocr-v1.1.13-live.json` - Sanitized real OCR output from a live run against OCR v1.1.13

Fixtures must not contain local paths, tokens, private URLs, or real GitLab info.

## GitHub Actions

The project uses two levels of CI:

- **Pull request CI** (`.github/workflows/ci.yml`): runs `make check` which includes `make test-compat`, unit tests, vet, build, and format checks.
- **OCR compatibility CI** (`.github/workflows/ocr-compatibility.yml`):
  - **Scheduled** (weekly): runs checked-in fixture compatibility tests only. No LLM credentials required.
  - **Manual** (`workflow_dispatch`): runs fixture tests and live latest OCR capture. Requires LLM secrets. Fails explicitly if secrets are missing.

Scheduled compatibility CI does not require GitLab tokens or platform access. Live capture is manual-only and requires GitHub Actions secrets to be configured.

### Required GitHub Actions Secrets and Variables

Live capture requires these secrets (Settings > Secrets and variables > Actions):

| Name | Type | Description |
|------|------|-------------|
| `OCR_LLM_URL` | secret | LLM API endpoint URL |
| `OCR_LLM_TOKEN` | secret | LLM API token |
| `OCR_LLM_MODEL` | secret | LLM model name |
| `OCR_USE_ANTHROPIC` | variable | Optional. Set to `true` to use Anthropic protocol |

These names match the Open Code Review LLM resolver contract (verified against `internal/llm/resolver.go`). The resolver also supports Claude Code env vars (`ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, `ANTHROPIC_MODEL`) as a fallback.

### Live Capture Direct Validation

When live capture runs, the workflow directly parses `/tmp/ocr-latest.json` using `TestCapturedOCROutputParses`. This verifies the captured output is valid OCR JSON, not just that checked-in fixtures still parse.

## Release Requirement

Before a public release:

- `make test-compat` must pass.
- The latest OCR compatibility workflow must be green or have a documented known incompatibility.
- Release notes must state the OCR versions verified for the release.
