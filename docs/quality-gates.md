# Quality Gates

This project exists because platform publishing quality matters. A feature is not complete just because the platform API accepted a request. The rendered review experience must be readable, stable, and safe to rerun.

## Definition Of Done

Every implementation task must satisfy the relevant gates below before it is reported complete.

Required for all tasks:

- Unit tests for changed behavior.
- `go test ./... -count=1` passes.
- `go vet ./...` passes.
- `go build ./...` passes.
- `git diff --check` passes.
- No tokens, local hostnames, or local filesystem paths in tracked files.

Required for parser changes:

- Fixture test using real OCR output.
- Test for leading non-JSON text before the JSON object.
- Test for unknown future fields.
- Test for missing optional fields.
- Compatibility test against at least one captured OCR output fixture from a real OCR release.

Required for renderer changes:

- Golden Markdown test.
- Existing code is fenced, never bare text.
- Known file extensions use language-aware fences.
- Unknown file extensions fall back to `text`.
- Diagnostics render inside `<details>`.
- No raw GitLab/GitHub API error text appears in rendered comments.
- No platform-specific suggestion fence is introduced without a dedicated design.

Required for GitLab publishing changes:

- `httptest` coverage for request method, path, query, and body.
- GitLab 13.12 fallback behavior tested.
- Discussion pagination tested with `X-Next-Page`.
- Inline anchor selection tested against diff fixtures.
- Clear/update behavior deletes or updates only marked publisher-owned comments.

Required before public release:

- Pull request CI has passed.
- Scheduled/latest OCR compatibility check has passed, or a known incompatibility is documented before release.
- Opt-in GitLab e2e tests pass against a real GitLab instance.
- Local smoke script publishes real OCR output to a test MR and fetches comments back for assertions.
- Repeated publish updates summary without creating duplicates.
- Clear operations leave unmarked user comments untouched.
- Documentation includes limitations and supported platform/version scope.
- English and Chinese README files are both present and aligned.
- README badges follow the local `readme-badges` skill.

Required after release:

- `make smoke-release-binary` passes: the published archive for the current platform downloads, extracts, and the binary responds correctly to `version` and `help` commands.

## Rendered Comment Checklist

Fetched comments from a real MR must satisfy:

- Product header or badge is visible.
- Finding text is clear without reading raw OCR JSON.
- Location is present when available.
- Suggested change is readable and not misleading.
- Existing code is in a fenced code block.
- Review context is collapsed when verbose.
- Summary shows total findings and inline skipped count.
- Publish diagnostics are folded.
- Stable marker is present but unobtrusive.
- No duplicate summary after repeated publish.

## Release Blockers

Any of the following blocks a release:

- A comment contains raw platform errors such as `line_code can't be blank`.
- Existing code renders as one line of bare prose.
- Repeated publish creates duplicate summary comments.
- Clear deletes an unmarked user or third-party bot comment.
- GitLab 13.12 e2e cannot run or cannot be clearly skipped with documented env requirements.
- The CLI prints tokens or secrets.
- JSON mode prints human chatter to stdout.
- The latest supported OCR output can no longer be parsed.
- The compatibility workflow cannot detect OCR output contract changes.

## OCR Compatibility Gates

OCR is an upstream dependency even though it is executed as a CLI. Its output contract must be tested with real binaries.

Required compatibility coverage:

- A checked-in fixture for the minimum supported OCR version.
- A checked-in fixture for the latest OCR version verified at release time.
- A parser test that accepts both fixtures.
- A local script that can regenerate fixtures from a chosen OCR version.
- A scheduled GitHub Action that installs the latest published OCR package and validates parser compatibility.

Compatibility jobs must not publish comments or require GitLab tokens. If live OCR generation requires LLM credentials, the workflow must separate:

- fixture compatibility, which always runs; and
- live latest-OCR generation, which runs only when credentials are configured.

Any output format change discovered by scheduled CI should produce an issue or failing workflow with the captured redacted output excerpt.

## External CLI Contract Checks

When a workflow or script invokes an external CLI, the following rules apply:

- **Verify env names against the source.** Environment variable names and config keys must be confirmed against the external CLI's source resolver or official documentation. Do not guess or copy from outdated examples.
- **Validate generated artifacts directly.** The workflow or script must parse or validate the file it just generated, not only run against pre-existing checked-in fixtures. If the generation step produces `/tmp/output.json`, the validation step must read `/tmp/output.json`.
- **Fail explicitly on missing credentials.** Manual validation gates that require credentials must fail with a clear error when credentials are absent. Do not silently skip and report success.
- **Never print secret values.** Error messages may name the missing variable but must not echo its value.

Example from this project: the OCR compatibility workflow must use `OCR_LLM_URL`, `OCR_LLM_TOKEN`, and `OCR_LLM_MODEL` because those are the names Open Code Review's resolver reads. After capturing OCR output, the workflow directly parses the captured file via `TestCapturedOCROutputParses`, not just the checked-in fixtures.

## Review Practice

For every worker handoff, include:

- The exact task scope.
- Required tests and smoke checks.
- A reminder that rendered comments must be inspected through fetched platform notes, not only local Markdown strings.
- A final report requiring command outputs, changed files, and skipped gates.

For reviewer handoff, include:

- A link to changed files.
- The exact rendered Markdown examples or golden fixture names.
- Any real GitLab MR smoke result.
- Known limitations and deferred behavior.
