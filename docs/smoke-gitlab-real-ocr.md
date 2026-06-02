# Real OCR Smoke Gate

This document describes the real OCR smoke gate that validates the complete pipeline from OCR output to GitLab MR comments.

## Overview

The real OCR smoke gate (`make smoke-gitlab-real-ocr`) tests the full workflow:

1. Run actual OCR review on a fixture repository
2. Parse OCR output and publish to GitLab MR
3. Verify comment quality via GitLab API
4. Clean up publisher-owned comments

This is a **maintainer-only** smoke gate that requires:
- A local fixture repository with real code changes
- OCR binary with LLM credentials configured
- GitLab instance with API access
- Valid GitLab token and MR

## Quick Start

```bash
# Build the publisher binary
make build

# Run the smoke gate
OCR_SMOKE_REPO=~/path/to/fixture make smoke-gitlab-real-ocr
```

## Required Environment Variables

### GitLab Configuration

```bash
export OCR_E2E_GITLAB_URL=https://your-gitlab.example.com
export OCR_E2E_GITLAB_TOKEN=your-gitlab-token
export OCR_E2E_GITLAB_PROJECT_ID=123
export OCR_E2E_GITLAB_MR_IID=456
```

### Fixture Repository

```bash
export OCR_SMOKE_REPO=~/path/to/fixture-repo
```

The fixture repository should have:
- At least one commit on `main` branch
- Recent changes (on `HEAD` or a feature branch) for OCR to review
- Go files recommended for language-aware fence testing

## Optional Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OCR_SMOKE_OCR_BIN` | `ocr` | OCR binary path (falls back to npx) |
| `OCR_SMOKE_PUBLISHER_BIN` | `./ocr-review-publisher` | Publisher binary path |
| `OCR_SMOKE_FROM` | `main` | Base ref for OCR review |
| `OCR_SMOKE_TO` | `HEAD` | Head ref for OCR review |
| `OCR_SMOKE_CONCURRENCY` | `2` | Concurrency setting |
| `OCR_SMOKE_TIMEOUT` | `15` | Timeout in minutes |
| `OCR_SMOKE_CLEANUP` | `1` | Clean up comments after run |
| `OCR_SMOKE_KEEP_OUTPUT` | `0` | Keep output files in `.local/smoke/` |

## Local Environment File

For local development, create `env.gitlab.local` in the repository root:

```bash
cat > env.gitlab.local <<'EOF'
OCR_E2E_GITLAB=1
OCR_E2E_GITLAB_URL=https://your-gitlab.example.com
OCR_E2E_GITLAB_TOKEN=your-token
OCR_E2E_GITLAB_PROJECT_ID=123
OCR_E2E_GITLAB_MR_IID=456
EOF
```

This file is ignored by git and should never be committed.

When `env.gitlab.local` exists, `make smoke-gitlab-real-ocr` automatically sources it.

## Quality Assertions

The smoke gate verifies comment quality before cleanup:

### Required Assertions

1. **OCR marker comments exist** - At least 1 inline or summary comment with OCR markers
2. **Summary marker present** - At least 1 summary marker comment
3. **No raw API errors** - No `line_code can't be blank` errors
4. **No suggestion fences** - No ` ```suggestion ` fences (use language-aware fences)
5. **Language-aware fences** - Go files use ` ```go ` fences
6. **Code in fenced blocks** - Existing code in fenced code blocks
7. **Review context block** - `<details><summary>Review context</summary>` present
8. **No duplicate summaries** - Only 1 summary note (if multiple publish runs)

### Optional Assertions

- **Diagnostics block** - `<details><summary>Publish diagnostics` present if warnings exist

## Output Files

By default, output files are saved to a temporary directory and deleted on exit.

To keep output files for debugging:

```bash
OCR_SMOKE_KEEP_OUTPUT=1 OCR_SMOKE_REPO=~/path/to/fixture make smoke-gitlab-real-ocr
```

Output files will be saved to `.local/smoke/`:
- `ocr-output.json` - Raw OCR output
- `ocr-output-clean.json` - Cleaned OCR output (without summary line)
- `ocr-output-final.json` - Final OCR output
- `publish-output.txt` - Publisher output
- `discussions.json` - GitLab discussions API response
- `notes.json` - Extracted notes
- `inline-notes.json` - Inline marker notes
- `summary-notes.json` - Summary marker notes
- `verification-results.json` - Verification results

## Cleanup Behavior

By default, the smoke gate cleans up all publisher-owned comments after verification.

To skip cleanup (for debugging):

```bash
OCR_SMOKE_CLEANUP=0 OCR_SMOKE_REPO=~/path/to/fixture make smoke-gitlab-real-ocr
```

**Warning:** Skipping cleanup leaves comments on the MR. Manual cleanup may be needed.

## Troubleshooting

### OCR Binary Not Found

If OCR is not installed globally, the script will try `npx @alibaba-group/open-code-review`.

Install OCR globally:
```bash
npm install -g @alibaba-group/open-code-review
```

### Publisher Binary Not Found

Build the publisher first:
```bash
make build
```

### Fixture Repository Not Found

Ensure `OCR_SMOKE_REPO` points to a valid git repository:
```bash
ls -la ~/path/to/fixture/.git
```

### GitLab API Errors

Check your GitLab token and permissions:
```bash
curl -H "PRIVATE-TOKEN: $OCR_E2E_GITLAB_TOKEN" \
  "$OCR_E2E_GITLAB_URL/api/v4/projects/$OCR_E2E_GITLAB_PROJECT_ID"
```

### OCR Review Fails

Ensure LLM credentials are configured for OCR. See OCR documentation for supported providers.

## Comparison with `make test-e2e-gitlab`

| Feature | `make test-e2e-gitlab` | `make smoke-gitlab-real-ocr` |
|---------|------------------------|------------------------------|
| **Input** | Constructed `review.Result` | Real OCR output |
| **OCR Required** | No | Yes |
| **LLM Required** | No | Yes |
| **Fixture Repo** | No | Yes |
| **CI Default** | Yes (with env) | No |
| **Purpose** | Publisher correctness | Full pipeline validation |

## Safety Notes

- The smoke gate creates and deletes comments on the specified MR
- Do not run against production MRs unless you understand the impact
- The script cleans up after itself by default
- Output files may contain sensitive data (OCR findings, API responses)
- Local environment files contain secrets and should never be committed

## CI Integration

This smoke gate is **not** intended for CI by default. It requires:
- Real OCR binary with LLM credentials
- Fixture repository access
- GitLab API access with write permissions

For CI, use `make test-e2e-gitlab` with constructed test data.

## See Also

- [E2E GitLab Testing](e2e-gitlab.md) - Publisher e2e tests with constructed data
- [GitLab Integration](gitlab.md) - GitLab platform documentation
- [OCR Compatibility](ocr-compatibility.md) - OCR output format compatibility
