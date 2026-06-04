# Release Process

This project publishes a wrapper around [Open Code Review (OCR)](https://github.com/alibaba/open-code-review) output. A release must prove both local code correctness and compatibility with the OCR CLI output contract.

## Release Artifacts

Each release produces:

- `ocr-review-publisher_<version>_darwin_amd64.tar.gz`
- `ocr-review-publisher_<version>_darwin_arm64.tar.gz`
- `ocr-review-publisher_<version>_linux_amd64.tar.gz`
- `ocr-review-publisher_<version>_linux_arm64.tar.gz`
- `ocr-review-publisher_<version>_checksums.txt`

Not supported: Homebrew, npm, Docker.

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

## Release Flow

1. **Prepare bilingual release notes:**
   - Create `docs/releases/release-notes-vX.Y.Z.md` (English)
   - Create `docs/releases/release-notes-vX.Y.Z.zh-CN.md` (Chinese)
   - Use templates in `docs/releases/release-notes-template*.md`

2. **Run local gates:**
   ```bash
   make release-readiness
   make test-e2e-gitlab  # if GitLab credentials available
   ```

3. **Create and push annotated tag:**
   ```bash
   git tag -a vX.Y.Z -m "Release vX.Y.Z"
   git push origin vX.Y.Z
   ```

4. **GitHub Actions release workflow runs automatically:**
   - Runs `make release-readiness`
   - Validates release notes files exist
   - Runs GoReleaser to build and publish artifacts
   - Creates GitHub Release with bilingual notes

5. **Verify release:**
   - Check GitHub Release page for correct artifacts
   - Verify checksums file
   - Run the release binary smoke gate:
     ```bash
     make smoke-release-binary
     # Or verify a specific tag:
     OCR_RELEASE_TAG=v0.1.1 make smoke-release-binary
     ```
   - This downloads the platform-appropriate archive, extracts the binary, and
     verifies `version` and `help` commands produce expected output.

## GitHub Actions Gates

Required workflows:

- **CI** (`.github/workflows/ci.yml`): runs `make check` on push to main and pull requests.
- **OCR Compatibility** (`.github/workflows/ocr-compatibility.yml`): runs fixture compatibility weekly; optional live capture on manual dispatch when LLM secrets are configured.
- **Release Readiness** (`.github/workflows/release-readiness.yml`): runs `make release-readiness` on manual dispatch only. Does not publish, does not create tags/releases.
- **Release** (`.github/workflows/release.yml`): tag-triggered, builds artifacts and publishes GitHub Release.

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

Each release requires bilingual release notes:

- `docs/releases/release-notes-vX.Y.Z.md` (English)
- `docs/releases/release-notes-vX.Y.Z.zh-CN.md` (Chinese)

Notes should include:

- publisher version;
- verified OCR version range;
- supported platform scope;
- known OCR output compatibility limitations;
- whether GitLab e2e/smoke was run;
- installation instructions;
- any migration notes for config or markers.
