#!/usr/bin/env bash
set -euo pipefail

# Real OCR smoke gate for GitLab MR publishing.
# This script runs actual OCR against a fixture repository and publishes
# results to GitLab, then verifies comment quality.
#
# Usage:
#   OCR_SMOKE_REPO=~/path/to/fixture make smoke-gitlab-real-ocr
#
# Required environment:
#   OCR_SMOKE_REPO              Path to fixture repository
#   OCR_E2E_GITLAB_URL          GitLab base URL
#   OCR_E2E_GITLAB_TOKEN        GitLab token
#   OCR_E2E_GITLAB_PROJECT_ID   Project ID or namespace path
#   OCR_E2E_GITLAB_MR_IID       Merge request IID
#
# Optional environment:
#   OCR_SMOKE_OCR_BIN           OCR binary (default: ocr)
#   OCR_SMOKE_PUBLISHER_BIN     Publisher binary (default: ./ocr-review-publisher)
#   OCR_SMOKE_FROM              Base ref (default: main)
#   OCR_SMOKE_TO                Head ref (default: HEAD)
#   OCR_SMOKE_CONCURRENCY       Concurrency (default: 2)
#   OCR_SMOKE_TIMEOUT           Timeout in minutes (default: 15)
#   OCR_SMOKE_CLEANUP           Cleanup after run (default: 1)
#   OCR_SMOKE_KEEP_OUTPUT       Keep output files (default: 0)

# Configuration with defaults
OCR_SMOKE_REPO="${OCR_SMOKE_REPO:-}"
OCR_SMOKE_OCR_BIN="${OCR_SMOKE_OCR_BIN:-ocr}"
OCR_SMOKE_PUBLISHER_BIN="${OCR_SMOKE_PUBLISHER_BIN:-./ocr-review-publisher}"
OCR_SMOKE_FROM="${OCR_SMOKE_FROM:-main}"
OCR_SMOKE_TO="${OCR_SMOKE_TO:-HEAD}"
OCR_SMOKE_CONCURRENCY="${OCR_SMOKE_CONCURRENCY:-2}"
OCR_SMOKE_TIMEOUT="${OCR_SMOKE_TIMEOUT:-15}"
OCR_SMOKE_CLEANUP="${OCR_SMOKE_CLEANUP:-1}"
OCR_SMOKE_KEEP_OUTPUT="${OCR_SMOKE_KEEP_OUTPUT:-0}"

GITLAB_URL="${OCR_E2E_GITLAB_URL:-}"
GITLAB_TOKEN="${OCR_E2E_GITLAB_TOKEN:-}"
PROJECT_ID="${OCR_E2E_GITLAB_PROJECT_ID:-}"
MR_IID="${OCR_E2E_GITLAB_MR_IID:-}"

# Marker constants (must match internal/render/markers.go)
INLINE_MARKER="<!-- ocr-review-publisher:inline -->"
SUMMARY_MARKER="<!-- ocr-review-publisher:summary -->"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
  printf "%b[INFO]%b %s\n" "$GREEN" "$NC" "$1"
}

log_warn() {
  printf "%b[WARN]%b %s\n" "$YELLOW" "$NC" "$1"
}

log_error() {
  printf "%b[ERROR]%b %s\n" "$RED" "$NC" "$1" >&2
}

# Get absolute path for temp dir
abs_temp_dir() {
  cd "$TEMP_DIR" && pwd
}

# --- GitLab API helpers ---

# Fetch a GitLab API page with HTTP status validation.
# Usage: gitlab_fetch <url> <output_file>
# Exits on non-2xx status.
gitlab_fetch() {
  local url="$1"
  local output_file="$2"
  local http_code

  http_code=$(curl -s -w '%{http_code}' \
    -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
    "$url" -o "$output_file")

  if [[ ! "$http_code" =~ ^2[0-9][0-9]$ ]]; then
    log_error "GitLab API returned HTTP $http_code for: ${url%%\?*}"
    if [[ -f "$output_file" ]]; then
      # Print first 200 chars of response body for debugging (no token)
      head -c 200 "$output_file" >&2
    fi
    exit 1
  fi
}

# Fetch all discussions with pagination.
# Usage: fetch_discussions <output_file>
fetch_discussions() {
  local output_file="$1"
  local ad
  ad=$(abs_temp_dir)
  local page=1
  local all_discussions="[]"

  while true; do
    local page_file="$ad/discussions-page-$page.json"
    gitlab_fetch \
      "$GITLAB_URL/api/v4/projects/$PROJECT_ID/merge_requests/$MR_IID/discussions?per_page=100&page=$page" \
      "$page_file"

    if ! jq empty "$page_file" 2>/dev/null; then
      log_error "Invalid JSON in discussions page $page"
      exit 1
    fi

    local page_count
    page_count=$(jq 'length' "$page_file")
    if [[ $page_count -eq 0 ]]; then
      break
    fi

    all_discussions=$(jq -s '.[0] + .[1]' <(echo "$all_discussions") "$page_file")
    page=$((page + 1))
  done

  echo "$all_discussions" > "$output_file"
}

# Extract notes from discussions file.
# Usage: extract_notes <discussions_file> <notes_file>
extract_notes() {
  local discussions_file="$1"
  local notes_file="$2"
  jq '[.[] | .notes[] | select(.body != null)]' "$discussions_file" > "$notes_file"
}

# Write marker-filtered notes to separate files.
# Usage: write_marker_notes <notes_file> <inline_file> <summary_file>
write_marker_notes() {
  local notes_file="$1"
  local inline_file="$2"
  local summary_file="$3"
  jq --arg marker "$INLINE_MARKER" '[.[] | select(.body | contains($marker))]' "$notes_file" > "$inline_file"
  jq --arg marker "$SUMMARY_MARKER" '[.[] | select(.body | contains($marker))]' "$notes_file" > "$summary_file"
}

# Assert marker counts match expected values.
# Usage: assert_marker_counts <inline_file> <summary_file> <expected_inline> <expected_summary> <label>
assert_marker_counts() {
  local inline_file="$1"
  local summary_file="$2"
  local expected_inline="$3"
  local expected_summary="$4"
  local label="$5"

  local actual_inline
  actual_inline=$(jq 'length' "$inline_file")
  local actual_summary
  actual_summary=$(jq 'length' "$summary_file")

  log_info "$label: inline=$actual_inline, summary=$actual_summary"

  local failed=false
  if [[ "$actual_inline" != "$expected_inline" ]]; then
    log_error "ASSERTION FAILED: $label inline marker count: expected $expected_inline, got $actual_inline"
    failed=true
  fi
  if [[ "$actual_summary" != "$expected_summary" ]]; then
    log_error "ASSERTION FAILED: $label summary marker count: expected $expected_summary, got $actual_summary"
    failed=true
  fi

  if [[ "$failed" == "true" ]]; then
    exit 1
  fi
}

# --- Core functions ---

# Check prerequisites
check_prerequisites() {
  local missing=()

  if [[ -z "$OCR_SMOKE_REPO" ]]; then
    missing+=("OCR_SMOKE_REPO")
  fi

  if [[ -z "$GITLAB_URL" ]]; then
    missing+=("OCR_E2E_GITLAB_URL")
  fi

  if [[ -z "$GITLAB_TOKEN" ]]; then
    missing+=("OCR_E2E_GITLAB_TOKEN")
  fi

  if [[ -z "$PROJECT_ID" ]]; then
    missing+=("OCR_E2E_GITLAB_PROJECT_ID")
  fi

  if [[ -z "$MR_IID" ]]; then
    missing+=("OCR_E2E_GITLAB_MR_IID")
  fi

  if [[ ${#missing[@]} -gt 0 ]]; then
    log_error "Missing required environment variables:"
    for var in "${missing[@]}"; do
      log_error "  - $var"
    done
    exit 1
  fi

  # Check if OCR binary exists
  if ! command -v "$OCR_SMOKE_OCR_BIN" &>/dev/null; then
    # Try npx as fallback
    if command -v npx &>/dev/null; then
      log_info "OCR binary not found, will use npx @alibaba-group/open-code-review"
      OCR_SMOKE_OCR_BIN="npx @alibaba-group/open-code-review"
    else
      log_error "OCR binary not found: $OCR_SMOKE_OCR_BIN"
      log_error "Install OCR or ensure npx is available"
      exit 1
    fi
  fi

  # Check if publisher binary exists
  if [[ ! -x "$OCR_SMOKE_PUBLISHER_BIN" ]]; then
    log_error "Publisher binary not found or not executable: $OCR_SMOKE_PUBLISHER_BIN"
    log_error "Run 'make build' first"
    exit 1
  fi

  # Check if fixture repo exists
  if [[ ! -d "$OCR_SMOKE_REPO" ]]; then
    log_error "Fixture repository not found: $OCR_SMOKE_REPO"
    exit 1
  fi

  # Check if jq is available for JSON processing
  if ! command -v jq &>/dev/null; then
    log_error "jq is required but not found"
    log_error "Install jq: https://stedolan.github.io/jq/download/"
    exit 1
  fi
}

# Create temporary directory for output files
setup_temp_dir() {
  if [[ "$OCR_SMOKE_KEEP_OUTPUT" == "1" ]]; then
    TEMP_DIR=".local/smoke"
    mkdir -p "$TEMP_DIR"
    log_info "Keeping output files in: $TEMP_DIR"
  else
    TEMP_DIR=$(mktemp -d)
    trap 'cleanup_temp_dir' EXIT
    log_info "Temporary directory: $TEMP_DIR"
  fi
}

cleanup_temp_dir() {
  if [[ "$OCR_SMOKE_KEEP_OUTPUT" != "1" && -d "$TEMP_DIR" ]]; then
    rm -rf "$TEMP_DIR"
    log_info "Cleaned up temporary directory"
  fi
}

# Pre-clear all publisher-owned comments to ensure clean test environment.
# This always runs regardless of OCR_SMOKE_CLEANUP setting.
pre_clear_markers() {
  log_info "Pre-clearing existing publisher-owned comments..."

  local clear_output
  clear_output=$(abs_temp_dir)/pre-clear-output.txt

  if ! "$OCR_SMOKE_PUBLISHER_BIN" clear \
    --platform gitlab \
    --gitlab-base-url "$GITLAB_URL" \
    --project-id "$PROJECT_ID" \
    --mr "$MR_IID" \
    --token "$GITLAB_TOKEN" \
    --scope all \
    > "$clear_output" 2>&1; then
    log_error "Pre-clear failed"
    cat "$clear_output" >&2
    exit 1
  fi

  log_info "Pre-clear output:"
  cat "$clear_output"

  # Verify markers are gone
  log_info "Verifying pre-clear marker counts..."
  local ad
  ad=$(abs_temp_dir)
  local discussions_file="$ad/post-preclear-discussions.json"
  local notes_file="$ad/post-preclear-notes.json"
  local inline_file="$ad/post-preclear-inline.json"
  local summary_file="$ad/post-preclear-summary.json"

  fetch_discussions "$discussions_file"
  extract_notes "$discussions_file" "$notes_file"
  write_marker_notes "$notes_file" "$inline_file" "$summary_file"
  assert_marker_counts "$inline_file" "$summary_file" "0" "0" "Pre-clear"
}

# Run OCR review on fixture repository
run_ocr_review() {
  log_info "Running OCR review on fixture repository..."
  log_info "  Repository: $OCR_SMOKE_REPO"
  log_info "  From: $OCR_SMOKE_FROM"
  log_info "  To: $OCR_SMOKE_TO"

  local ad
  ad=$(abs_temp_dir)
  local ocr_output="$ad/ocr-output.json"
  local ocr_stderr="$ad/ocr-stderr.txt"

  cd "$OCR_SMOKE_REPO"

  # Run OCR and capture output
  if ! $OCR_SMOKE_OCR_BIN review \
    --from "$OCR_SMOKE_FROM" \
    --to "$OCR_SMOKE_TO" \
    --format json \
    --audience agent \
    > "$ocr_output" 2>"$ocr_stderr"; then
    log_error "OCR review failed"
    cat "$ocr_stderr" >&2
    exit 1
  fi

  cd - > /dev/null

  # Remove OCR summary line (starts with [ocr])
  local clean_output="$ad/ocr-output-clean.json"
  grep -v "^\[ocr\]" "$ocr_output" > "$clean_output" || true

  # Validate JSON
  if ! jq empty "$clean_output" 2>/dev/null; then
    log_error "OCR output is not valid JSON"
    cat "$clean_output" >&2
    exit 1
  fi

  # Check status
  local status
  status=$(jq -r '.status // empty' "$clean_output")
  if [[ "$status" != "success" ]]; then
    log_error "OCR review status is not success: $status"
    exit 1
  fi

  local comment_count
  comment_count=$(jq '.comments | length' "$clean_output")
  log_info "OCR review completed: $comment_count comment(s)"

  # Copy clean output to final location
  cp "$clean_output" "$ad/ocr-output-final.json"
}

# Publish OCR results to GitLab
publish_to_gitlab() {
  log_info "Publishing OCR results to GitLab..."

  local ad
  ad=$(abs_temp_dir)
  local ocr_input="$ad/ocr-output-final.json"
  local publish_output="$ad/publish-output.txt"

  if ! "$OCR_SMOKE_PUBLISHER_BIN" publish \
    --platform gitlab \
    --gitlab-base-url "$GITLAB_URL" \
    --project-id "$PROJECT_ID" \
    --mr "$MR_IID" \
    --token "$GITLAB_TOKEN" \
    --input "$ocr_input" \
    --format text \
    > "$publish_output" 2>&1; then
    log_error "Publisher failed"
    cat "$publish_output" >&2
    exit 1
  fi

  log_info "Publish output:"
  cat "$publish_output"
}

# Verify post-publish comment quality
verify_post_publish() {
  log_info "Verifying post-publish comments..."

  local ad
  ad=$(abs_temp_dir)
  local discussions_file="$ad/post-publish-discussions.json"
  local notes_file="$ad/post-publish-notes.json"
  local inline_file="$ad/post-publish-inline.json"
  local summary_file="$ad/post-publish-summary.json"

  fetch_discussions "$discussions_file"
  extract_notes "$discussions_file" "$notes_file"

  local total_notes
  total_notes=$(jq 'length' "$notes_file")
  log_info "Total notes found: $total_notes"

  write_marker_notes "$notes_file" "$inline_file" "$summary_file"

  local inline_count
  inline_count=$(jq 'length' "$inline_file")
  local summary_count
  summary_count=$(jq 'length' "$summary_file")

  log_info "Post-publish marker counts: inline=$inline_count, summary=$summary_count"

  # Core assertions: must have at least some markers
  if [[ $inline_count -eq 0 && $summary_count -eq 0 ]]; then
    log_error "ASSERTION FAILED: No OCR marker comments found after publish"
    exit 1
  fi

  if [[ $summary_count -ne 1 ]]; then
    log_error "ASSERTION FAILED: Expected exactly 1 summary marker, got $summary_count"
    exit 1
  fi

  # Quality assertions on the notes
  local assertions_passed=true

  # No 'line_code can't be blank' errors
  local line_code_errors
  line_code_errors=$(jq --arg pattern "line_code can't be blank" '[.[] | select(.body | contains($pattern))]' "$notes_file")
  if [[ $(echo "$line_code_errors" | jq 'length') -gt 0 ]]; then
    log_error "ASSERTION FAILED: Found 'line_code can't be blank' errors"
    assertions_passed=false
  fi

  # No suggestion fence
  local suggestion_pattern='```suggestion'
  local suggestion_fences
  suggestion_fences=$(jq --arg pattern "$suggestion_pattern" '[.[] | select(.body | contains($pattern))]' "$notes_file")
  if [[ $(echo "$suggestion_fences" | jq 'length') -gt 0 ]]; then
    log_error "ASSERTION FAILED: Found suggestion fences"
    assertions_passed=false
  fi

  # Go files should have language fence
  if [[ $inline_count -gt 0 ]]; then
    local go_fence='```go'
    local go_files
    go_files=$(jq --arg fence "$go_fence" '[.[] | select(.body | contains($fence))]' "$inline_file")
    local go_file_count
    go_file_count=$(echo "$go_files" | jq 'length')

    # Check if any inline comments are for .go files
    local has_go_comments
    has_go_comments=$(jq '[.[] | select(.position != null and .position.new_path != null and (.position.new_path | test("\\.go$")))]' "$inline_file")
    local has_go_count
    has_go_count=$(echo "$has_go_comments" | jq 'length')

    if [[ $has_go_count -gt 0 && $go_file_count -eq 0 ]]; then
      log_error "ASSERTION FAILED: Go file comments missing language fence"
      assertions_passed=false
    fi
  fi

  # Existing code in fenced blocks
  if [[ $inline_count -gt 0 ]]; then
    local fence_marker='```'
    local fenced_blocks
    fenced_blocks=$(jq --arg marker "$fence_marker" '[.[] | select(.body | contains($marker))]' "$inline_file")
    local fenced_count
    fenced_count=$(echo "$fenced_blocks" | jq 'length')

    if [[ $fenced_count -eq 0 ]]; then
      log_error "ASSERTION FAILED: No fenced code blocks found in inline comments"
      assertions_passed=false
    fi
  fi

  # Review context details block
  if [[ $inline_count -gt 0 ]]; then
    local context_marker='<details><summary>Review context'
    local context_blocks
    context_blocks=$(jq --arg marker "$context_marker" '[.[] | select(.body | contains($marker))]' "$inline_file")
    local context_count
    context_count=$(echo "$context_blocks" | jq 'length')

    if [[ $context_count -eq 0 ]]; then
      log_error "ASSERTION FAILED: No Review context details block found"
      assertions_passed=false
    fi
  fi

  # Diagnostics block if warnings exist
  if [[ $summary_count -gt 0 ]]; then
    local diagnostics_marker='<details><summary>Publish diagnostics'
    local diagnostics_blocks
    diagnostics_blocks=$(jq --arg marker "$diagnostics_marker" '[.[] | select(.body | contains($marker))]' "$summary_file")
    local diagnostics_count
    diagnostics_count=$(echo "$diagnostics_blocks" | jq 'length')

    if [[ $diagnostics_count -eq 0 ]]; then
      log_warn "No Publish diagnostics block found - may be OK if no warnings"
    fi
  fi

  if [[ "$assertions_passed" != "true" ]]; then
    log_error "Quality assertions FAILED"
    exit 1
  fi

  log_info "All quality assertions PASSED"
}

# Cleanup publisher-owned comments
cleanup_comments() {
  if [[ "$OCR_SMOKE_CLEANUP" != "1" ]]; then
    log_info "Skipping final cleanup - OCR_SMOKE_CLEANUP=0"
    return
  fi

  log_info "Cleaning up publisher-owned comments..."

  local clear_output
  clear_output=$(abs_temp_dir)/cleanup-output.txt

  if ! "$OCR_SMOKE_PUBLISHER_BIN" clear \
    --platform gitlab \
    --gitlab-base-url "$GITLAB_URL" \
    --project-id "$PROJECT_ID" \
    --mr "$MR_IID" \
    --token "$GITLAB_TOKEN" \
    --scope all \
    > "$clear_output" 2>&1; then
    log_error "Cleanup failed"
    cat "$clear_output" >&2
    exit 1
  fi

  log_info "Cleanup output:"
  cat "$clear_output"

  # Verify markers are gone after cleanup
  log_info "Verifying post-cleanup marker counts..."
  local ad
  ad=$(abs_temp_dir)
  local discussions_file="$ad/post-cleanup-discussions.json"
  local notes_file="$ad/post-cleanup-notes.json"
  local inline_file="$ad/post-cleanup-inline.json"
  local summary_file="$ad/post-cleanup-summary.json"

  fetch_discussions "$discussions_file"
  extract_notes "$discussions_file" "$notes_file"
  write_marker_notes "$notes_file" "$inline_file" "$summary_file"
  assert_marker_counts "$inline_file" "$summary_file" "0" "0" "Post-cleanup"
}

# Main execution
main() {
  log_info "Starting real OCR smoke gate..."

  check_prerequisites
  setup_temp_dir

  log_info "Configuration:"
  log_info "  Fixture repo: $OCR_SMOKE_REPO"
  log_info "  OCR binary: $OCR_SMOKE_OCR_BIN"
  log_info "  Publisher binary: $OCR_SMOKE_PUBLISHER_BIN"
  log_info "  GitLab URL: $GITLAB_URL"
  log_info "  Project ID: $PROJECT_ID"
  log_info "  MR IID: $MR_IID"
  log_info "  Cleanup: $OCR_SMOKE_CLEANUP"

  pre_clear_markers
  run_ocr_review
  publish_to_gitlab
  verify_post_publish
  cleanup_comments

  log_info "Real OCR smoke gate completed successfully!"
}

# Run main function
main
