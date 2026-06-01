# Release Process

This project publishes a wrapper around [Open Code Review (OCR)](https://github.com/alibaba/open-code-review) output. A release must prove both local code correctness and compatibility with the OCR CLI output contract.

## Current Status

Formal release packaging (GoReleaser, binary artifacts, automated GitHub Releases) is planned but not yet implemented. Currently, users build from source. The gates and workflow below describe the intended release process.

## Local Gates

Before tagging a release, run:

```bash
make check
make test-compat
```

These run without secrets or network access. `make check` includes format, test, vet, build, and compatibility checks.

For stricter pre-release validation:

```bash
make release-readiness
```

This runs `make check` plus race detection tests, whitespace checks, and sensitive pattern scanning.

If GitLab credentials are available, also run:

```bash
make test-e2e-gitlab
```

Then run the local smoke flow against a real test MR:

```bash
scripts/gitlab-smoke.example.sh publish
scripts/gitlab-smoke.example.sh check
scripts/gitlab-smoke.example.sh cleanup
```

The smoke script builds the current publisher, publishes comments to a real GitLab MR, and asserts rendered comment quality.

## GitHub Actions Gates

Required workflows:

- **CI** (`.github/workflows/ci.yml`): runs `make check` on push to main and pull requests.
- **OCR Compatibility** (`.github/workflows/ocr-compatibility.yml`): runs fixture compatibility weekly; optional live capture on manual dispatch when LLM secrets are configured.
- **Release Readiness** (`.github/workflows/release-readiness.yml`): runs `make release-readiness` on manual dispatch only. Does not publish, does not create tags/releases.

The release readiness workflow does not publish anything. It only verifies that the codebase passes all pre-release gates. Real GitLab e2e/smoke remains a manual opt-in step that requires platform credentials.

## Release Blockers

Do not release if:

- parser compatibility fails for the latest verified OCR output;
- JSON mode includes human chatter that breaks machine parsing;
- rendered comments fail the quality checklist;
- GitLab clear/update can delete unmarked comments;
- repeated publish creates duplicate summaries;
- tokens or local environment details appear in tracked files;
- release notes do not state the verified OCR version range;
- README.md and README.zh-CN.md are missing, stale, or inconsistent;
- README badges do not follow the local readme-badges skill.

## Release Notes

Each release should include:

- publisher version;
- verified OCR version range;
- supported platform scope;
- known OCR output compatibility limitations;
- whether GitLab e2e/smoke was run;
- English and Chinese README update status;
- any migration notes for config or markers.
