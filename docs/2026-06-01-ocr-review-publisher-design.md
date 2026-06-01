# OCR Review Publisher Design

## Purpose

Build a public open-source integration layer for Open Code Review (OCR). The project consumes `ocr review` machine-readable output and publishes high-quality review comments to code hosting platforms.

This project does not fork OCR, replace OCR's review engine, or require changes in the upstream OCR core. OCR remains responsible for generating findings. This project owns platform publishing, comment lifecycle, rendering quality, CI wiring, and later webhook/server triggers.

Working name: `ocr-review-publisher`.

## Product Boundary

Version 1 supports only OCR as the review producer.

In scope:

- Parse OCR JSON/agent output from a file or stdin.
- Publish OCR findings to GitLab merge requests.
- Create inline comments when a safe platform anchor exists.
- Create or update one managed summary comment.
- Clear/update only publisher-owned comments through stable markers.
- Provide CI-friendly CLI commands and documentation.
- Test real comment rendering against GitLab, not just API request shape.

Out of scope for version 1:

- Reimplementing OCR review logic.
- Supporting non-OCR review engines.
- GitHub PR publishing.
- Slash command triggers.
- Webhook/server mode.
- Web UI, database, history dashboard, or SaaS behavior.
- Automatic severity/category inference before OCR exposes stable fields.

## User Experience

OCR runs first and writes a result file:

```bash
ocr review --from origin/main --to HEAD --format json --audience agent > ocr-result.json
```

The publisher consumes that output:

```bash
ocr-review-publisher publish \
  --platform gitlab \
  --input ocr-result.json \
  --gitlab-base-url https://gitlab.example.com \
  --project-id group/project \
  --mr 123
```

Clear operations do not run OCR:

```bash
ocr-review-publisher clear \
  --platform gitlab \
  --scope inline \
  --project-id group/project \
  --mr 123
```

CI jobs can pipe directly:

```bash
ocr review --format json --audience agent ... \
  | ocr-review-publisher publish --platform gitlab --input -
```

## Architecture

The project is organized around stable boundaries:

- `ocroutput`: Parse and normalize OCR output. This is the only layer coupled to OCR's JSON shape.
- `review`: Internal normalized finding/result model used by renderers and publishers.
- `render`: Markdown rendering for inline comments, summary comments, diagnostics, and managed markers.
- `platform/gitlab`: GitLab REST client and MR publisher.
- `lifecycle`: Marker ownership, clear/update behavior, deduplication, and summary replacement.
- `cmd`: CLI parsing, env inference, command orchestration, exit codes.
- `e2e/gitlab`: Opt-in tests against real GitLab.

The platform layer never calls OCR. The parser never calls GitLab. Renderers are pure functions with golden tests.

## Output Parsing Contract

The parser accepts OCR `--format json --audience agent` output. OCR currently prints a human-readable first line before JSON in some modes, so version 1 should parse robustly:

- Trim leading non-JSON text until the first `{`.
- Reject malformed JSON with a useful error and a small redacted excerpt.
- Preserve unknown fields for forward compatibility when practical.
- Normalize missing line numbers to zero, not failure.
- Normalize missing `comments` to an empty list.

If OCR later adds `category`, `severity`, or `confidence`, the parser should pass those through as optional fields.

The contract must be verified against real OCR binaries, not inferred from documentation or example PRs. The project should keep fixtures for known OCR versions and run a compatibility job against the latest published OCR package on a schedule. If OCR output changes, the parser should fail in CI before a user discovers the breakage in a GitLab publish job.

## Comment Quality Contract

Comment quality is a first-class acceptance criterion. Published comments must be readable without knowing OCR internals.

Inline comments include:

- Product badge/header.
- Finding content.
- Location in `path:start-end` form.
- Optional category/severity badges when available.
- Optional suggested change as a normal language-aware fenced code block.
- Optional review context in a collapsed details block.
- Existing code in a language-aware fenced code block.
- Stable ownership marker.

Summary comments include:

- Product badge/header.
- Total findings count.
- Inline published / inline skipped counts.
- Findings list with path and line range.
- Collapsed diagnostics for skipped/failed publishing.
- Stable ownership marker.

Do not use GitLab-specific `suggestion` fences in version 1. We previously saw confusing rendering when suggestion range metadata was not exact. Use ordinary language fences until one-click suggestions have a dedicated design and tests.

## Language-Aware Rendering

Code fences use file extension mapping:

- `.go` -> `go`
- `.js`, `.jsx` -> `javascript`
- `.ts`, `.tsx` -> `typescript`
- `.py` -> `python`
- `.java` -> `java`
- `.rs` -> `rust`
- `.json` -> `json`
- `.yaml`, `.yml` -> `yaml`
- `.md` -> `markdown`
- unknown -> `text`

The extension match is case-insensitive.

## GitLab Publishing

Version 1 targets GitLab because the current need is MR publishing and local validation exists.

Required behavior:

- Support self-hosted GitLab base URLs.
- Support project ID or namespace path.
- Support GitLab 13.12.
- Create inline discussions only for safe anchors.
- Create/update one summary note.
- Clear publisher-owned inline and summary notes.
- Preserve user comments and other bot comments.
- Continue publishing when one inline comment fails.
- Report skipped/failed inline comments in diagnostics.

GitLab 13.12 compatibility:

- Use MR versions for diff refs.
- Fetch changed files through `/diffs` when available.
- Fallback to `/changes?access_raw_diffs=true` when `/diffs` is unavailable.
- Paginate discussions with `per_page=100` and exact `X-Next-Page`.

## Inline Anchor Selection

GitLab rejects invalid inline positions. The publisher must avoid known-bad requests:

- If a finding has no path or no positive line range, skip inline and include it in summary.
- If the finding range overlaps added lines, anchor to the first added line in the range.
- If no added line exists in the range, skip inline and include a friendly diagnostic.
- If the diff is unavailable or unparseable, use a conservative fallback only when the platform API can still validate safely; otherwise skip inline.

All findings remain represented in the summary, even when inline publishing is skipped.

## Marker Ownership

All publisher-owned content uses stable markers:

```markdown
<!-- ocr-review-publisher:inline -->
<!-- ocr-review-publisher:summary -->
```

Clear/update operations may delete or replace only notes containing these markers. They must not delete:

- User comments.
- OCR upstream comments without this project's marker.
- Kodus comments.
- Comments from other bots.

## CI Behavior

The CLI should be CI-friendly:

- Use env inference for common GitLab CI variables.
- Never print platform tokens.
- Support `--dry-run` to render comments without publishing.
- Support `--fail-on-publish-error` and default non-fatal inline failures.
- Distinguish review findings from publisher failures.
- Produce a machine-readable publish report when requested.

Version 1 does not need a full GitHub Action. It should provide examples that call the CLI from GitLab CI.

The project itself should use GitHub Actions for quality control:

- Pull request CI runs unit tests, parser fixture tests, golden renderer tests, vet, build, and diff checks.
- Scheduled compatibility CI installs the latest published OCR CLI, runs `ocr review --format json --audience agent` against a small fixture repository, and verifies that the output can still be parsed.
- Release CI runs the full local gate set and requires a manually supplied GitLab e2e/smoke result before publishing a release.

## Quality Gates

The project must not repeat the previous prototype failure mode where API tests passed but real comments looked poor.

Required gates:

- Golden Markdown tests for inline, summary, diagnostics, and managed markers.
- Fixture tests using real OCR JSON output captured from a representative review.
- Compatibility tests that generate fresh OCR output from the latest published OCR package.
- GitLab API shape tests with `httptest`.
- Diff anchor selection tests for context-line starts, added-line ranges, zero lines, deleted files, and unparseable diffs.
- Opt-in GitLab 13.12 e2e tests for publish, update, clear, and rendering.
- Local smoke script that runs OCR against a test repo, publishes to GitLab, fetches comments, and asserts rendered Markdown quality.

Rendered Markdown assertions must check for:

- No raw `suggestion` fence unless explicitly enabled by a future feature.
- Existing code inside a fenced code block.
- Language-aware fence for known file extensions.
- No raw GitLab API errors such as `line_code can't be blank` in comments.
- Diagnostics folded under `<details>`.
- No duplicate summary comments after repeated publish.

## Documentation

Documentation should be honest about scope:

- This project wraps OCR output; it does not improve OCR's review intelligence.
- Version 1 supports GitLab only.
- OCR output format changes may require parser updates.
- Category/severity badges appear only when OCR output provides those fields.

## Deferred Work

- GitHub PR publisher.
- GitHub Action packaging.
- GitLab slash command or webhook trigger.
- Server mode.
- True one-click suggestions.
- Category/severity policy gates after OCR exposes stable fields.
- Multi-review-engine support.
