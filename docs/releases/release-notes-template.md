# ocr-review-publisher vX.Y.Z Release Notes

## Overview

Brief summary of this release.

## Highlights

- Key change 1
- Key change 2
- Key change 3

## What's New

### Feature Area 1

- Description of change
- Description of change

### Feature Area 2

- Description of change
- Description of change

## Install / Upgrade

Download from GitHub Releases:

```bash
# Linux amd64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/vX.Y.Z/ocr-review-publisher_X.Y.Z_linux_amd64.tar.gz
tar xzf ocr-review-publisher_X.Y.Z_linux_amd64.tar.gz

# macOS arm64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/vX.Y.Z/ocr-review-publisher_X.Y.Z_darwin_arm64.tar.gz
tar xzf ocr-review-publisher_X.Y.Z_darwin_arm64.tar.gz
```

Verify checksums:

```bash
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/vX.Y.Z/ocr-review-publisher_X.Y.Z_checksums.txt
sha256sum -c ocr-review-publisher_X.Y.Z_checksums.txt
```

## Compatibility

- Verified OCR version range: vX.Y.Z - vA.B.C
- Supported platforms: GitLab (including 13.12 self-hosted)
- Supported OS: darwin, linux
- Supported architectures: amd64, arm64

## Verification

- `make release-readiness` passed
- GitLab e2e tests: passed / skipped (reason)
- OCR compatibility: passed

## Known Limitations

- Only supports Open Code Review output
- Only supports GitLab MR publishing (not GitHub PRs)
- No webhook/server mode
- No one-click platform suggestions

## Migration Notes

- Note any config or marker changes
- Note any breaking changes
