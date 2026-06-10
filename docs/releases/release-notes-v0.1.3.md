# ocr-review-publisher v0.1.3 Release Notes

Release date: 2026-06-10

## Overview

Patch release after v0.1.2. Improves reliability of the GitLab CI smoke test when external downloads (OCR npm package) are flaky. No changes to parser, render, or publisher behavior.

## What's Changed

### GitLab CI Smoke

- Add 3-attempt retry with 15s sleep for `npm install` of the OCR binary — the download from github.com is unreliable inside Docker runner containers
- Update default `PUBLISHER_VERSION` from `v0.1.1` to `v0.1.2`

## Compatibility

- Verified OCR version: v1.1.13
- Supported platform: GitLab (including 13.12 self-hosted)
- Supported operating systems: darwin, linux
- Supported architectures: amd64, arm64

## Verification

- `make release-readiness` passed
- `make smoke-release-binary` passed
- `make smoke-gitlab-ci` passed

## Known Limitations

- GitLab only: no GitHub PR publishing
- No webhook, server, or slash-command mode
- No Homebrew, npm, or Docker packages
- Category and severity badges render only when OCR output includes those fields
- True one-click GitLab suggestions are not enabled

## Migration Notes

- No breaking changes from v0.1.2
- No parser, render, or publisher behavior changes
- No marker format changes
