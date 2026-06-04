#!/usr/bin/env bash
set -euo pipefail

# Release binary smoke gate for ocr-review-publisher.
#
# Downloads the release archive for the current platform from GitHub Releases,
# extracts the binary, and verifies it runs correctly. No credentials required.
#
# Usage:
#   make smoke-release-binary
#   OCR_RELEASE_TAG=v0.1.1 make smoke-release-binary
#
# Optional environment:
#   OCR_RELEASE_TAG        Release tag to verify (default: v0.1.1)
#   OCR_RELEASE_SMOKE_DIR  Working directory (default: .local/release-smoke)

REPO="Fanduzi/ocr-review-publisher"
TAG="${OCR_RELEASE_TAG:-v0.1.1}"
SMOKE_DIR="${OCR_RELEASE_SMOKE_DIR:-.local/release-smoke}"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { printf "%b[INFO]%b  %s\n" "$GREEN" "$NC" "$1"; }
log_error() { printf "%b[ERROR]%b %s\n" "$RED" "$NC" "$1" >&2; }
log_step()  { printf "\n%b=== %s ===%b\n" "$CYAN" "$1" "$NC"; }

die() { log_error "$1"; exit 1; }

detect_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    *)      die "Unsupported OS: $os (supported: darwin, linux)" ;;
  esac

  case "$arch" in
    x86_64)  arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)       die "Unsupported architecture: $arch (supported: amd64, arm64)" ;;
  esac

  echo "${os}_${arch}"
}

main() {
  log_step "Release Binary Smoke Gate"
  log_info "Tag: $TAG"
  log_info "Smoke dir: $SMOKE_DIR"

  command -v jq &>/dev/null || die "jq is required but not found"

  local platform
  platform="$(detect_platform)"
  log_info "Platform: $platform"

  local archive_name="ocr-review-publisher_${TAG#v}_${platform}.tar.gz"

  # Prepare smoke directory.
  rm -rf "$SMOKE_DIR"
  mkdir -p "$SMOKE_DIR"

  # Download archive. Prefer gh CLI (handles GitHub redirects and auth);
  # fall back to curl if gh is unavailable.
  log_step "Downloading $archive_name"
  if command -v gh &>/dev/null; then
    log_info "Using gh release download"
    if ! gh release download "$TAG" \
        --repo "$REPO" \
        --pattern "$archive_name" \
        --dir "$SMOKE_DIR" \
        --skip-existing 2>&1; then
      die "gh release download failed for $archive_name"
    fi
  elif command -v curl &>/dev/null; then
    log_info "Using curl"
    local download_url="https://github.com/${REPO}/releases/download/${TAG}/${archive_name}"
    local http_code
    http_code=$(curl -sL -w '%{http_code}' -o "${SMOKE_DIR}/${archive_name}" "$download_url")
    if [[ "$http_code" != "200" ]]; then
      die "Download failed: HTTP $http_code from $download_url"
    fi
  else
    die "Neither gh nor curl is available"
  fi
  log_info "Downloaded: $(du -h "${SMOKE_DIR}/${archive_name}" | cut -f1)"

  # Extract.
  log_step "Extracting"
  tar xzf "${SMOKE_DIR}/${archive_name}" -C "$SMOKE_DIR"
  local binary="${SMOKE_DIR}/ocr-review-publisher"
  [[ -f "$binary" ]] || die "Binary not found after extraction: $binary"
  chmod +x "$binary"
  log_info "Binary extracted: $binary"

  # Verify: version command.
  log_step "Verifying: ocr-review-publisher version"
  local version_output
  version_output=$("$binary" version 2>&1) || die "version command failed"
  echo "$version_output"
  if ! echo "$version_output" | grep -qi "ocr-review-publisher\|version"; then
    die "version output does not contain expected content"
  fi
  log_info "version command: OK"

  # Verify: help command.
  log_step "Verifying: ocr-review-publisher help"
  local help_output
  help_output=$("$binary" help 2>&1) || true
  if [[ -z "$help_output" ]]; then
    die "help command produced no output"
  fi

  local missing_cmds=""
  for cmd in publish clear render; do
    if ! echo "$help_output" | grep -q "$cmd"; then
      missing_cmds="${missing_cmds} $cmd"
    fi
  done

  if [[ -n "$missing_cmds" ]]; then
    die "help output missing expected commands:${missing_cmds}"
  fi
  log_info "help command: OK (contains publish, clear, render)"

  # Cleanup.
  log_step "Cleanup"
  rm -rf "$SMOKE_DIR"
  log_info "Smoke directory removed"

  log_step "PASSED"
  echo "Tag:       $TAG"
  echo "Platform:  $platform"
  echo "Archive:   $archive_name"
  echo "Status:    binary smoke gate passed"
}

main "$@"
