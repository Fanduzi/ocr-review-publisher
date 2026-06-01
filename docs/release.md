# Release Process

This project publishes a wrapper around OCR output. A release must prove both local code correctness and compatibility with the OCR CLI output contract.

## Local Gates

Before tagging a release, run:

```bash
make check
make test-compat
```

If GitLab credentials are available, also run:

```bash
make test-e2e-gitlab
```

Then run the local smoke flow against a real test MR:

```bash
scripts/gitlab-smoke.sh publish
scripts/gitlab-smoke.sh check
scripts/gitlab-smoke.sh cleanup
```

The smoke script should build the current publisher, run OCR against a fixture or test repository, publish comments, fetch them back through the platform API, and assert rendered comment quality.

## GitHub Actions Gates

Required workflows:

- Pull request CI.
- Scheduled OCR compatibility CI.
- Release readiness CI.

The release workflow should not need GitLab tokens for normal unit compatibility checks. Real GitLab publishing remains an opt-in e2e/smoke gate because it requires platform credentials.

## Release Blockers

Do not release if:

- parser compatibility fails for the latest verified OCR output;
- JSON mode includes human chatter that breaks machine parsing;
- rendered comments fail the quality checklist;
- GitLab clear/update can delete unmarked comments;
- repeated publish creates duplicate summaries;
- tokens or local environment details appear in tracked files;
- release notes do not state the verified OCR version range.
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
