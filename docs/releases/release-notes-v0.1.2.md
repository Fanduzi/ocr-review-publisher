# ocr-review-publisher v0.1.2 Release Notes

## Overview

Patch release after v0.1.1. Improves GitLab CI documentation and adds release binary smoke testing. No changes to parser, render, or publisher behavior.

## What's Changed

### GitLab CI Examples

- Replace the "Basic Example" in `docs/ci.md` with release binary download from GitHub Releases (pinned version, checksum verification, install to `/usr/local/bin`)
- Source build remains documented as a secondary option for developers and custom forks

### Release Binary Smoke Gate

- Add `scripts/smoke-release-binary.sh`: downloads the platform archive from GitHub Releases, extracts the binary, and verifies `--version` and `--help` output
- Add `make smoke-release-binary` target
- Supports `OCR_RELEASE_TAG` to test a specific release and `OCR_RELEASE_SMOKE_DIR` for a custom extraction directory

### GitLab CI Smoke (Real Release Validation)

- `scripts/smoke-gitlab-ci.sh` now downloads the release tarball and checksums from GitHub Releases instead of building from source
- Validates the full flow: download, checksum verification, binary execution, publish, and cleanup
- Fixes checksum verification to preserve the original release filename (matching `checksums.txt` entries)
- Adds configurable proxy support (`OCR_CI_SMOKE_HTTPS_PROXY`) and local cache mode (`OCR_CI_SMOKE_LOCAL_CACHE`)

### Documentation

- Update README.md and README.zh-CN.md with GitHub Releases download instructions (checksum verification included)
- Update `docs/release.md` and `docs/release.zh-CN.md`: document the release binary smoke gate in the "Verify release" step
- Update `docs/quality-gates.md` and `docs/quality-gates.zh-CN.md`: add post-release gate requiring `smoke-release-binary` to pass

## Installation

Download from GitHub Releases:

```bash
# Linux amd64
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.2/ocr-review-publisher_0.1.2_linux_amd64.tar.gz
tar xzf ocr-review-publisher_0.1.2_linux_amd64.tar.gz

# macOS arm64 (Apple Silicon)
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.2/ocr-review-publisher_0.1.2_darwin_arm64.tar.gz
tar xzf ocr-review-publisher_0.1.2_darwin_arm64.tar.gz
```

Verify checksums:

```bash
curl -LO https://github.com/Fanduzi/ocr-review-publisher/releases/download/v0.1.2/ocr-review-publisher_0.1.2_checksums.txt
sha256sum -c ocr-review-publisher_0.1.2_checksums.txt
```

## Compatibility

- Verified OCR version: v1.1.13
- Supported platform: GitLab (including 13.12 self-hosted)
- Supported operating systems: darwin, linux
- Supported architectures: amd64, arm64

## Verification

- `make release-readiness` passed
- `make smoke-release-binary` passed
- GitLab CI smoke: release binary download + checksum verification + publish/cleanup passed

## Known Limitations

- GitLab only: no GitHub PR publishing
- No webhook, server, or slash-command mode
- No Homebrew, npm, or Docker packages
- Category and severity badges render only when OCR output includes those fields
- True one-click GitLab suggestions are not enabled

## Migration Notes

- No breaking changes from v0.1.1
- No parser, render, or publisher behavior changes
- No marker format changes
