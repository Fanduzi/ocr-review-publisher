# Contributing to OCR Review Publisher

## Code of Conduct

Be respectful, constructive, and focused on the project's goals.

## Development Workflow

### TDD for Parser, Renderer, and Publisher Changes

All changes to parser (`internal/ocroutput/`), renderer (`internal/render/`), and GitLab publisher (`internal/platform/gitlab/`) must follow test-driven development:

1. Write a failing test that defines the expected behavior.
2. Run the test and confirm it fails for the expected reason.
3. Write the minimal implementation to pass the test.
4. Run the test and confirm it passes.
5. Refactor while keeping tests green.

### No AI Attribution

Do not add AI co-authorship lines, generated-by annotations, or any AI attribution to commit messages. The commit author is the human contributor.

### Quality Gates

Before submitting a pull request, run:

```bash
make check
```

This runs `go fmt`, `go test`, `go vet`, `go build`, and `make test-compat`.

For changes that affect rendered comment quality, include golden Markdown test fixtures and verify against the rendered comment checklist in `docs/quality-gates.md`.

### GitLab E2E Requirement

Changes to GitLab publishing behavior require opt-in e2e tests. Set the following environment variables to enable:

```bash
export OCR_E2E_GITLAB=1
export OCR_E2E_GITLAB_URL=https://gitlab.example.com
export OCR_E2E_GITLAB_TOKEN=your-token
export OCR_E2E_GITLAB_PROJECT_ID=123
export OCR_E2E_GITLAB_MR_IID=456
make test-e2e-gitlab
```

E2e tests are not required for local development but are required before merging publishing changes.

### OCR Output Compatibility

The parser depends on `ocr review --format json --audience agent` output. This is an external contract owned by Open Code Review.

- Parser changes must pass `make test-compat`.
- New OCR output fixtures should be captured from real OCR releases.
- Before a release, the verified OCR version range must be documented.

## Pull Request Checklist

- [ ] `make check` passes locally.
- [ ] New behavior has tests (TDD for parser/renderer/publisher).
- [ ] Rendered comment quality verified if rendering changed.
- [ ] No tokens, local paths, or secrets in tracked files.
- [ ] No AI attribution in commits.
- [ ] GitLab e2e ran if publishing behavior changed (or explicitly skipped with env explanation).
