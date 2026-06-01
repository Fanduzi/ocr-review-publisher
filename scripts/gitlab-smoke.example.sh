#!/usr/bin/env bash
set -euo pipefail

# Example smoke script for GitLab MR publishing.
# Copy to .local/gitlab-smoke.sh and configure for your environment.

BINARY="${OCR_SMOKE_BIN:-./ocr-review-publisher}"
GITLAB_URL="${OCR_E2E_GITLAB_URL:-}"
PROJECT_ID="${OCR_E2E_GITLAB_PROJECT_ID:-}"
MR_IID="${OCR_E2E_GITLAB_MR_IID:-}"
INPUT="${OCR_SMOKE_INPUT:-}"

usage() {
  cat <<'USAGE'
Usage:
  gitlab-smoke.sh publish   # build, publish, verify
  gitlab-smoke.sh check     # render and check output
  gitlab-smoke.sh cleanup   # clear publisher-owned comments

Required env:
  OCR_E2E_GITLAB_URL        GitLab base URL
  OCR_E2E_GITLAB_TOKEN      GitLab token
  OCR_E2E_GITLAB_PROJECT_ID Project ID or namespace path
  OCR_E2E_GITLAB_MR_IID     Merge request IID

Optional env:
  OCR_SMOKE_INPUT           OCR output file (or - for stdin)
  OCR_SMOKE_BIN             Path to publisher binary
USAGE
}

check_env() {
  local missing=()
  [[ -z "$GITLAB_URL" ]] && missing+=("OCR_E2E_GITLAB_URL")
  [[ -z "${OCR_E2E_GITLAB_TOKEN:-}" ]] && missing+=("OCR_E2E_GITLAB_TOKEN")
  [[ -z "$PROJECT_ID" ]] && missing+=("OCR_E2E_GITLAB_PROJECT_ID")
  [[ -z "$MR_IID" ]] && missing+=("OCR_E2E_GITLAB_MR_IID")
  if [[ ${#missing[@]} -gt 0 ]]; then
    printf 'Missing required env: %s\n' "${missing[*]}" >&2
    exit 2
  fi
}

publish() {
  check_env
  local args=(
    publish
    --platform gitlab
    --gitlab-base-url "$GITLAB_URL"
    --project-id "$PROJECT_ID"
    --mr "$MR_IID"
    --token "$OCR_E2E_GITLAB_TOKEN"
    --format text
  )
  if [[ -n "$INPUT" ]]; then
    args+=(--input "$INPUT")
  fi
  printf '==> publishing to %s project %s MR %s\n' "$GITLAB_URL" "$PROJECT_ID" "$MR_IID"
  "$BINARY" "${args[@]}"
}

check() {
  check_env
  local args=(
    render
    --format text
  )
  if [[ -n "$INPUT" ]]; then
    args+=(--input "$INPUT")
  fi
  printf '==> rendering OCR output\n'
  "$BINARY" "${args[@]}"
}

cleanup() {
  check_env
  printf '==> clearing inline comments\n'
  "$BINARY" clear \
    --platform gitlab \
    --gitlab-base-url "$GITLAB_URL" \
    --project-id "$PROJECT_ID" \
    --mr "$MR_IID" \
    --token "$OCR_E2E_GITLAB_TOKEN" \
    --scope inline

  printf '==> clearing summary comments\n'
  "$BINARY" clear \
    --platform gitlab \
    --gitlab-base-url "$GITLAB_URL" \
    --project-id "$PROJECT_ID" \
    --mr "$MR_IID" \
    --token "$OCR_E2E_GITLAB_TOKEN" \
    --scope summary
}

case "${1:-}" in
  publish)
    publish
    ;;
  check)
    check
    ;;
  cleanup)
    cleanup
    ;;
  ""|-h|--help|help)
    usage
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
