# ocr-review-publisher v0.1.1 Release Notes

## Overview

Maintenance release after v0.1.0. Fixes release body publication, OCR compatibility live capture, and adds Chinese documentation.

## What's Changed

### Release Infrastructure

- Fix release workflow to publish bilingual release notes body via `gh release edit` after GoReleaser
- Update GitHub repository description and topics

### OCR Compatibility

- Fix live capture env contract: use `OCR_LLM_URL`, `OCR_LLM_TOKEN`, `OCR_LLM_MODEL` (verified against Open Code Review resolver)
- Add direct captured-output parsing via `TestCapturedOCROutputParses`
- Add sanitized real OCR fixture (`ocr-v1.1.13-live.json`) for parser regression coverage
- Manual live capture now fails explicitly when secrets are missing

### Documentation

- Add Chinese documentation for all public docs
- Add external CLI contract quality gate
- Ignore local IDE files in `.gitignore`

## Installation

Download from GitHub Releases:

```bash
# Linux amd64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.1/ocr-review-publisher_0.1.1_linux_amd64.tar.gz
tar xzf ocr-review-publisher_0.1.1_linux_amd64.tar.gz

# macOS arm64 (Apple Silicon)
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.1/ocr-review-publisher_0.1.1_darwin_arm64.tar.gz
tar xzf ocr-review-publisher_0.1.1_darwin_arm64.tar.gz
```

Verify checksums:

```bash
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.1/ocr-review-publisher_0.1.1_checksums.txt
sha256sum -c ocr-review-publisher_0.1.1_checksums.txt
```

## Compatibility

- Verified OCR version: v1.1.13
- Supported platform: GitLab (including 13.12 self-hosted)
- Supported operating systems: darwin, linux
- Supported architectures: amd64, arm64

## Known Limitations

- GitLab only: no GitHub PR publishing
- No webhook, server, or slash-command mode
- No Homebrew, npm, or Docker packages
- Category and severity badges render only when OCR output includes those fields
- True one-click GitLab suggestions are not enabled

## Migration Notes

- No breaking changes from v0.1.0
- No marker format changes
- No GitLab publishing behavior changes
