# GitLab E2E Testing

This project includes opt-in end-to-end tests that run against a real GitLab instance.

## Required Environment Variables

```bash
export OCR_E2E_GITLAB=1
export OCR_E2E_GITLAB_URL=https://gitlab.example.com
export OCR_E2E_GITLAB_TOKEN=your-token
export OCR_E2E_GITLAB_PROJECT_ID=123
export OCR_E2E_GITLAB_MR_IID=456
```

All five variables are required when `OCR_E2E_GITLAB=1`.

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

This file is ignored by git (`env*.local` in `.gitignore`) and should never be committed. It contains local secrets.

When `env.gitlab.local` exists, `make test-e2e-gitlab` automatically sources it before running tests.

## Running E2E Tests

```bash
# With env file (auto-sourced if present)
make test-e2e-gitlab

# With explicit env vars
OCR_E2E_GITLAB=1 go test -tags=e2e ./internal/e2e/gitlab -count=1 -v
```

When `OCR_E2E_GITLAB` is not set to `1` and no `env.gitlab.local` exists, all e2e tests skip cleanly.

## What The Tests Cover

1. **Create and clear summary** - Publish a summary note, verify it exists, clear it, verify it's gone.
2. **Create and clear inline** - Publish an inline comment on an added line, verify it exists, clear it, verify it's gone.
3. **Unmarked notes preserved** - Create an unmarked note, run clear operations, verify unmarked note survives.
4. **Summary update (no duplicate)** - Publish twice, verify exactly one summary marker note exists.
5. **Inline rendering quality** - Publish a finding with all features (existing code, suggestion, thinking, category, severity), verify rendered Markdown quality before cleanup.
6. **Skipped inline in summary diagnostics** - Publish a finding with invalid line, verify it appears in summary diagnostics before cleanup.

## Comment Quality Assertions

E2E tests verify rendered comment quality by fetching GitLab notes **before cleanup**:

- Inline marker present
- Category and severity badges rendered
- Existing code in language-aware fenced block (e.g., ` ```go ` for `.go` files)
- Suggested change in language fence, not ` ```suggestion `
- Review context in `<details>` block
- Thinking/reviewer notes present
- No raw GitLab API errors (`line_code can't be blank`)
- Summary diagnostics in `<details>` block with correct wording

## Safety Behavior

- Tests clean up after themselves using `t.Cleanup`.
- Tests clear publisher-owned notes before and after each run.
- Tests do not delete unmarked user or bot comments.
- Tests are safe to rerun on the same MR.

## Smoke Script

A local smoke script can be used for manual verification:

```bash
# Copy the example
cp scripts/gitlab-smoke.example.sh .local/gitlab-smoke.sh

# Edit .local/gitlab-smoke.sh with your local config
# Then run:
chmod +x .local/gitlab-smoke.sh
.local/gitlab-smoke.sh publish
.local/gitlab-smoke.sh check
.local/gitlab-smoke.sh cleanup
```

## Warning

Do not run e2e tests against a production MR unless you understand the impact. Tests will create and delete comments on the specified MR.
