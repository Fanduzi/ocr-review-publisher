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

## Running E2E Tests

```bash
OCR_E2E_GITLAB=1 go test -tags=e2e ./internal/e2e/gitlab -count=1 -v
```

Or using the Makefile target:

```bash
make test-e2e-gitlab
```

When `OCR_E2E_GITLAB` is not set to `1`, all e2e tests skip cleanly.

## What The Tests Cover

1. **Create and clear summary** - Publish a summary note, verify it exists, clear it, verify it's gone.
2. **Create and clear inline** - Publish an inline comment on an added line, verify it exists, clear it, verify it's gone.
3. **Unmarked notes preserved** - Create an unmarked note, run clear operations, verify unmarked note survives.
4. **Summary update (no duplicate)** - Publish twice, verify exactly one summary marker note exists.
5. **Inline rendering quality** - Publish a finding with all features (existing code, suggestion, thinking, category, severity), verify rendered Markdown quality.
6. **Skipped inline in summary diagnostics** - Publish a finding with invalid line, verify it appears in summary diagnostics.

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
