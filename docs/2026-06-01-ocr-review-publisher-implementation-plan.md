# OCR Review Publisher Implementation Plan

> Agent workers must implement this plan task by task. Use TDD for behavior changes. Do not skip rendering and real GitLab verification gates.

## Goal

Create a public open-source CLI that consumes Open Code Review machine-readable output and publishes high-quality GitLab MR comments with marker-scoped lifecycle management.

Version 1 delivers GitLab publishing only. GitHub, CI action packaging, slash commands, and server mode are deferred.

## Non-Negotiable Lessons From The Previous Prototype

The first implementation must prove comment quality before it is considered complete.

Every task that changes rendering or publishing must include at least one of:

- Golden Markdown test.
- Real OCR output fixture test.
- GitLab `httptest` request/response test.
- Opt-in real GitLab e2e/smoke check.

Do not accept "API call succeeded" as proof that the feature is good. The rendered comment must be readable and must not contain raw platform errors or confusing Markdown.

## Proposed Repository Layout

```text
cmd/ocr-review-publisher/
internal/ocroutput/
internal/review/
internal/render/
internal/platform/gitlab/
internal/lifecycle/
internal/e2e/gitlab/
internal/compat/
testdata/ocr/
testdata/render/
testdata/fixtures/
docs/
.github/workflows/
```

## Task 0: Project Scaffold and Contribution Baseline

Files:

- `go.mod`
- `cmd/ocr-review-publisher/main.go`
- `README.md`
- `CONTRIBUTING.md`
- `.gitignore`

Steps:

- [ ] Choose final repository name.
- [ ] Initialize Go module.
- [ ] Add minimal CLI with `--help`.
- [ ] Add `make test`, `make build`, `make vet`, `make check`.
- [ ] Add README scope statement: OCR output publisher, not OCR fork.
- [ ] Add contribution rules: no AI attribution, TDD for publisher/rendering changes, e2e opt-in.

Verification:

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `go build ./...`
- [ ] `make check`

## Task 1: Internal Review Model

Files:

- `internal/review/types.go`
- `internal/review/types_test.go`

Behavior:

- Define normalized `Result`, `Finding`, `Warning`, and `PublishReport`.
- Fields include path, content, existing code, suggestion code, start/end lines, thinking/context, optional category, optional severity, and unknown metadata if useful.
- Validate only structural constraints; do not reject findings just because line numbers are missing.

Tests:

- [ ] Missing optional fields remain valid.
- [ ] Zero line numbers are preserved for summary fallback.
- [ ] Optional category/severity round-trip through JSON.

## Task 2: OCR Output Parser

Files:

- `internal/ocroutput/parser.go`
- `internal/ocroutput/parser_test.go`
- `testdata/ocr/basic.json`
- `testdata/ocr/prefixed-agent-output.txt`

Behavior:

- Parse OCR JSON from file, stdin, or byte slice.
- Tolerate leading non-JSON text before the first `{`.
- Decode `comments`, `warnings`, and `message`.
- Map OCR fields into the internal review model.
- Preserve future `category` and `severity` fields when present.
- Return actionable parse errors without dumping secrets.

Tests:

- [ ] Parses clean JSON.
- [ ] Parses OCR output with a leading human line.
- [ ] Empty comments produce an empty result, not an error.
- [ ] Malformed JSON returns a clear error.
- [ ] Future category/severity fields are preserved.

## Task 3: Markdown Renderer With Golden Tests

Files:

- `internal/render/markdown.go`
- `internal/render/language.go`
- `internal/render/markers.go`
- `internal/render/markdown_test.go`
- `testdata/render/*.golden.md`

Behavior:

- Render inline comments.
- Render summary comments.
- Render diagnostics details.
- Render category/severity badges only when fields are present.
- Use language-aware code fences.
- Put existing code and reviewer notes in `<details>`.
- Use ordinary language fences for suggested changes.
- Add stable ownership markers.

Golden cases:

- [ ] Minimal inline comment.
- [ ] Inline comment with Go existing code.
- [ ] Inline comment with suggested change.
- [ ] Inline comment with category/severity badges.
- [ ] Summary with zero findings.
- [ ] Summary with published/skipped diagnostics.
- [ ] Unknown extension falls back to `text`.

Quality assertions:

- [ ] No ` ```suggestion ` in version 1 output.
- [ ] Known file extensions use the expected fence language.
- [ ] Existing code is never rendered as bare text.
- [ ] Diagnostics are folded under `<details>`.

## Task 4: GitLab Client

Files:

- `internal/platform/gitlab/client.go`
- `internal/platform/gitlab/types.go`
- `internal/platform/gitlab/mr.go`
- `internal/platform/gitlab/client_test.go`

Behavior:

- Support base URL and token.
- Escape namespace project paths correctly.
- Fetch MR versions.
- Fetch changed files with `/diffs` and 404 fallback to `/changes?access_raw_diffs=true`.
- List discussions with pagination using exact `X-Next-Page`.
- Create inline discussions.
- Create/update/delete MR notes.

Tests:

- [ ] Token header is sent.
- [ ] Project namespace path is escaped.
- [ ] `/diffs` success.
- [ ] `/diffs` 404 fallback to `/changes`.
- [ ] Discussions fetch all pages and follow non-sequential `X-Next-Page`.
- [ ] HTTP errors include method/path/status.

## Task 5: Diff Anchor Selection

Files:

- `internal/platform/gitlab/diff.go`
- `internal/platform/gitlab/diff_test.go`

Behavior:

- Parse unified diffs enough to identify added new-file lines.
- Select first added line inside a finding range.
- Treat `EndLine < StartLine` as a single-line range.
- Return no anchor for zero/negative start lines.
- Return no anchor when a valid diff has no added line in range.
- Keep fallback behavior explicit and tested when diff is unavailable.

Tests:

- [ ] Start line is added.
- [ ] Start line is context, later line in range is added.
- [ ] No added line in range.
- [ ] Zero start line.
- [ ] Deleted file.
- [ ] Multiple hunks.
- [ ] Unparseable diff behavior.

## Task 6: GitLab Publisher Lifecycle

Files:

- `internal/platform/gitlab/publisher.go`
- `internal/lifecycle/markers.go`
- `internal/platform/gitlab/publisher_test.go`

Behavior:

- Publish inline comments for safe anchors.
- Skip unsafe inline comments and include them in summary diagnostics.
- Create summary when none exists.
- Update existing publisher summary by marker.
- Clear publisher inline comments by marker.
- Clear publisher summary comments by marker.
- Never delete unmarked comments.
- Continue after individual inline failures.
- Return a structured publish report.

Tests:

- [ ] Creates inline discussion with rendered marker.
- [ ] Skips missing line and reports diagnostic.
- [ ] Skips range with no added line.
- [ ] Creates summary.
- [ ] Updates existing summary instead of duplicating.
- [ ] Clear inline deletes only inline marker notes.
- [ ] Clear summary deletes only summary marker notes.
- [ ] Delete failures continue and are reported.

## Task 7: CLI Commands

Files:

- `cmd/ocr-review-publisher/main.go`
- `cmd/ocr-review-publisher/publish.go`
- `cmd/ocr-review-publisher/clear.go`
- `cmd/ocr-review-publisher/flags.go`
- `cmd/ocr-review-publisher/*_test.go`

Commands:

```bash
ocr-review-publisher publish --platform gitlab --input result.json
ocr-review-publisher clear --platform gitlab --scope inline
ocr-review-publisher clear --platform gitlab --scope summary
ocr-review-publisher render --input result.json
```

GitLab env inference:

- `GITLAB_TOKEN` or `OCR_GITLAB_TOKEN`
- `CI_SERVER_URL`
- `CI_PROJECT_ID`
- `CI_MERGE_REQUEST_IID`

Behavior:

- `publish` reads OCR output, renders comments, and publishes.
- `clear` only clears publisher-owned comments.
- `render` prints Markdown without platform API calls.
- `--dry-run` renders and reports without publishing.
- JSON output mode prints only machine-readable reports.

Tests:

- [ ] Flags override env.
- [ ] Missing token/project/MR produces clear errors.
- [ ] `--dry-run` does not call GitLab.
- [ ] JSON output mode is stdout-clean.
- [ ] Human output mode summarizes counts.

## Task 8: Real GitLab E2E and Smoke Script

Files:

- `internal/e2e/gitlab/gitlab_e2e_test.go`
- `.local/gitlab-smoke.sh` (ignored, not committed if repo policy prefers local-only)
- `docs/e2e-gitlab.md`

Behavior:

- E2E runs only with explicit env:
  - `OCR_E2E_GITLAB=1`
  - `OCR_E2E_GITLAB_URL`
  - `OCR_E2E_GITLAB_TOKEN`
  - `OCR_E2E_GITLAB_PROJECT_ID`
  - `OCR_E2E_GITLAB_MR_IID`
- Publish summary, update summary, clear summary.
- Publish inline comments, clear inline comments.
- Verify unmarked comments survive.
- Verify rendered Markdown quality from fetched GitLab notes.

Smoke assertions:

- [ ] No raw `suggestion` fence.
- [ ] No bare existing code.
- [ ] Known Go comments contain ` ```go `.
- [ ] No `line_code can't be blank`.
- [ ] No duplicate summary after repeated publish.
- [ ] Diagnostics are inside `<details>`.
- [ ] All created notes are cleaned up.

## Task 9: OCR Output Compatibility CI

Files:

- `internal/compat/ocr_output_test.go`
- `testdata/fixtures/ocr-sample-repo/`
- `scripts/capture-ocr-output.sh`
- `.github/workflows/ci.yml`
- `.github/workflows/ocr-compatibility.yml`
- `docs/ocr-compatibility.md`

Behavior:

- Keep checked-in fixtures for known OCR output shapes.
- Provide a script that installs a requested OCR version and captures `ocr review --format json --audience agent` output from a small fixture repository.
- CI must verify that captured output parses successfully and produces the expected normalized finding/result structure.
- A scheduled GitHub Action must install the latest published OCR package and run the compatibility parser test.
- Compatibility CI should not require platform tokens or GitLab access.
- If no LLM credentials are available, the compatibility job should still validate parser fixtures; a separate opt-in job may run live OCR generation when credentials are configured.

Tests:

- [ ] Parser accepts output captured from the pinned minimum supported OCR version.
- [ ] Parser accepts output captured from the current latest OCR version.
- [ ] Parser rejects malformed output with a useful error.
- [ ] Scheduled workflow can run without publishing comments.

## Task 10: GitHub Actions And Local Make Gates

Files:

- `Makefile`
- `.github/workflows/ci.yml`
- `.github/workflows/ocr-compatibility.yml`
- `.github/workflows/release.yml`
- `docs/release.md`

Behavior:

- `make check` is the local equivalent of pull request CI.
- Pull request CI runs `make check`.
- Scheduled compatibility CI checks the latest OCR output contract.
- Release workflow runs the complete gate set before publishing artifacts.
- Release docs describe which checks are automatic and which require local GitLab e2e/smoke credentials.

Required local targets:

- [ ] `make fmt`
- [ ] `make test`
- [ ] `make vet`
- [ ] `make build`
- [ ] `make check`
- [ ] `make test-compat`
- [ ] `make test-e2e-gitlab`

Required GitHub Actions:

- [ ] PR CI: fmt check, unit tests, vet, build, parser fixtures, golden renderer tests.
- [ ] Scheduled OCR compatibility: install latest OCR, capture/validate output where credentials allow; always run fixture compatibility.
- [ ] Release readiness: verify clean tree, run checks, build artifacts, require documented GitLab smoke result.

## Task 11: Documentation

Files:

- `README.md`
- `README.zh-CN.md`
- `docs/gitlab.md`
- `docs/ci.md`
- `docs/output-contract.md`
- `docs/ocr-compatibility.md`
- `docs/release.md`

Content:

- Scope and non-goals.
- OCR command examples.
- Publisher command examples.
- GitLab CI example.
- Token permissions.
- Marker ownership model.
- Comment rendering examples.
- Troubleshooting inline anchor failures.
- Output compatibility policy.
- OCR compatibility matrix and scheduled test policy.
- Release process and required gates.
- English and Chinese README coverage with matching scope, commands, limitations, and release notes.
- README badge presentation following the local `readme-badges` skill.

Verification:

- [ ] Commands in docs match CLI flags.
- [ ] Docs state GitLab-only v1 scope.
- [ ] Docs do not promise category/severity unless OCR output includes those fields.
- [ ] `README.md` and `README.zh-CN.md` are both updated for public release.
- [ ] README badges follow the `readme-badges` skill.

## Task 12: Release Readiness

Required gates:

- [ ] `go fmt ./...`
- [ ] `go vet ./...`
- [ ] `go test ./... -count=1`
- [ ] `go test -race ./...`
- [ ] `go build ./...`
- [ ] `make check`
- [ ] `make test-compat`
- [ ] GitLab e2e passes or is explicitly skipped with env explanation.
- [ ] Local smoke script passes against a real GitLab test MR before public release.
- [ ] Latest OCR compatibility job passes or the release notes state the known incompatibility.
- [ ] README includes known limitations.
- [ ] `README.md` and `README.zh-CN.md` both describe the current release scope.
- [ ] README badges follow the local `readme-badges` skill.
- [ ] No tokens or local paths committed.
- [ ] No AI attribution in commits.

## First Public Release Scope

The first release is acceptable when:

- It can publish OCR output to GitLab MR inline comments and summary.
- It can update/clear its own comments.
- It has golden rendering tests.
- It has real GitLab e2e coverage.
- It has one documented GitLab CI integration path.

Anything beyond that should wait for later releases.
