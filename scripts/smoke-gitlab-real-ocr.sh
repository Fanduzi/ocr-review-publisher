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

# Run OCR review on fixture repository
run_ocr_review() {
  log_info "Running OCR review on fixture repository..."
  log_info "  Repository: $OCR_SMOKE_REPO"
  log_info "  From: $OCR_SMOKE_FROM"
  log_info "  To: $OCR_SMOKE_TO"

  # Use absolute path for output files
  local abs_temp_dir
  abs_temp_dir=$(cd "$TEMP_DIR" && pwd)
  local ocr_output="$abs_temp_dir/ocr-output.json"
  local ocr_stderr="$abs_temp_dir/ocr-stderr.txt"

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
  local clean_output="$abs_temp_dir/ocr-output-clean.json"
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
  cp "$clean_output" "$abs_temp_dir/ocr-output-final.json"
}

# Publish OCR results to GitLab
publish_to_gitlab() {
  log_info "Publishing OCR results to GitLab..."

  local abs_temp_dir
  abs_temp_dir=$(cd "$TEMP_DIR" && pwd)
  local ocr_input="$abs_temp_dir/ocr-output-final.json"
  local publish_output="$abs_temp_dir/publish-output.txt"

  "$OCR_SMOKE_PUBLISHER_BIN" publish \
    --platform gitlab \
    --gitlab-base-url "$GITLAB_URL" \
    --project-id "$PROJECT_ID" \
    --mr "$MR_IID" \
    --token "$GITLAB_TOKEN" \
    --input "$ocr_input" \
    --format text \
    > "$publish_output" 2>&1

  local exit_code=$?
  if [[ $exit_code -ne 0 ]]; then
    log_error "Publisher failed with exit code $exit_code"
    cat "$publish_output" >&2
    exit 1
  fi

  log_info "Publish output:"
  cat "$publish_output"
}

# Fetch GitLab comments and verify quality
verify_comments() {
  log_info "Fetching GitLab comments for quality verification..."

  local abs_temp_dir
  abs_temp_dir=$(cd "$TEMP_DIR" && pwd)
  local discussions_file="$abs_temp_dir/discussions.json"
  local notes_file="$abs_temp_dir/notes.json"

  # Fetch all discussions with pagination
  local page=1
  local all_discussions="[]"
  while true; do
    local page_file="$abs_temp_dir/discussions-page-$page.json"
    curl -s -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
      "$GITLAB_URL/api/v4/projects/$PROJECT_ID/merge_requests/$MR_IID/discussions?per_page=100&page=$page" > "$page_file"

    if ! jq empty "$page_file" 2>/dev/null; then
      log_error "Failed to fetch discussions page $page or invalid JSON"
      cat "$page_file" >&2
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

  # Save all discussions
  echo "$all_discussions" > "$discussions_file"

  # Extract all notes (filter out null bodies)
  jq '[.[] | .notes[] | select(.body != null)]' "$discussions_file" > "$notes_file"

  local total_notes
  total_notes=$(jq 'length' "$notes_file")
  log_info "Total notes found: $total_notes"

  # Find OCR marker comments
  local inline_marker="<!-- ocr-review-publisher:inline -->"
  local summary_marker="<!-- ocr-review-publisher:summary -->"

  local inline_notes="$abs_temp_dir/inline-notes.json"
  local summary_notes="$abs_temp_dir/summary-notes.json"

  jq --arg marker "$inline_marker" '[.[] | select(.body | contains($marker))]' "$notes_file" > "$inline_notes"
  jq --arg marker "$summary_marker" '[.[] | select(.body | contains($marker))]' "$notes_file" > "$summary_notes"

  local inline_count
  inline_count=$(jq 'length' "$inline_notes")
  local summary_count
  summary_count=$(jq 'length' "$summary_notes")

  log_info "OCR marker comments found:"
  log_info "  Inline: $inline_count"
  log_info "  Summary: $summary_count"

  # Quality assertions
  local assertions_passed=true

  # Assertion 1: At least 1 inline or summary comment
  if [[ $inline_count -eq 0 && $summary_count -eq 0 ]]; then
    log_error "ASSERTION FAILED: No OCR marker comments found"
    assertions_passed=false
  fi

  # Assertion 2: Summary marker exists
  if [[ $summary_count -eq 0 ]]; then
    log_error "ASSERTION FAILED: No summary marker comment found"
    assertions_passed=false
  fi

  # Assertion 3: No 'line_code can't be blank' errors
  local line_code_errors
  line_code_errors=$(jq --arg pattern "line_code can't be blank" '[.[] | select(.body | contains($pattern))]' "$notes_file")
  if [[ $(echo "$line_code_errors" | jq 'length') -gt 0 ]]; then
    log_error "ASSERTION FAILED: Found 'line_code can't be blank' errors"
    assertions_passed=false
  fi

  # Assertion 4: No suggestion fence
  local suggestion_pattern='```suggestion'
  local suggestion_fences
  suggestion_fences=$(jq --arg pattern "$suggestion_pattern" '[.[] | select(.body | contains($pattern))]' "$notes_file")
  if [[ $(echo "$suggestion_fences" | jq 'length') -gt 0 ]]; then
    log_error "ASSERTION FAILED: Found suggestion fences"
    assertions_passed=false
  fi

  # Assertion 5: Go files should have language fence
  if [[ $inline_count -gt 0 ]]; then
    local go_fence='```go'
    local go_files
    go_files=$(jq --arg fence "$go_fence" '[.[] | select(.body | contains($fence))]' "$inline_notes")
    local go_file_count
    go_file_count=$(echo "$go_files" | jq 'length')

    # Check if any inline comments are for .go files
    local has_go_comments
    has_go_comments=$(jq '[.[] | select(.position != null and .position.old_path != null and (.position.old_path | test("\\.go$")))]' "$inline_notes")
    local has_go_count
    has_go_count=$(echo "$has_go_comments" | jq 'length')

    if [[ $has_go_count -gt 0 && $go_file_count -eq 0 ]]; then
      log_error "ASSERTION FAILED: Go file comments missing language fence"
      assertions_passed=false
    fi
  fi

  # Assertion 6: Existing code in fenced blocks (check for fenced code in inline notes)
  if [[ $inline_count -gt 0 ]]; then
    local fence_marker='```'
    local fenced_blocks
    fenced_blocks=$(jq --arg marker "$fence_marker" '[.[] | select(.body | contains($marker))]' "$inline_notes")
    local fenced_count
    fenced_count=$(echo "$fenced_blocks" | jq 'length')

    if [[ $fenced_count -eq 0 ]]; then
      log_error "ASSERTION FAILED: No fenced code blocks found in inline comments"
      assertions_passed=false
    fi
  fi

  # Assertion 7: Review context details block
  if [[ $inline_count -gt 0 ]]; then
    local context_marker='<details><summary>Review context'
    local context_blocks
    context_blocks=$(jq --arg marker "$context_marker" '[.[] | select(.body | contains($marker))]' "$inline_notes")
    local context_count
    context_count=$(echo "$context_blocks" | jq 'length')

    if [[ $context_count -eq 0 ]]; then
      log_error "ASSERTION FAILED: No Review context details block found"
      assertions_passed=false
    fi
  fi

  # Assertion 8: Diagnostics block if warnings exist
  if [[ $summary_count -gt 0 ]]; then
    local diagnostics_marker='<details><summary>Publish diagnostics'
    local diagnostics_blocks
    diagnostics_blocks=$(jq --arg marker "$diagnostics_marker" '[.[] | select(.body | contains($marker))]' "$summary_notes")
    local diagnostics_count
    diagnostics_count=$(echo "$diagnostics_blocks" | jq 'length')

    # This is optional - only check if there are warnings
    if [[ $diagnostics_count -eq 0 ]]; then
      log_warn "No Publish diagnostics block found - may be OK if no warnings"
    fi
  fi

  # Assertion 9: Duplicate publish check (optional)
  if [[ $summary_count -gt 1 ]]; then
    log_error "ASSERTION FAILED: Multiple summary notes found: $summary_count"
    assertions_passed=false
  fi

  # Save verification results
  local results_file="$abs_temp_dir/verification-results.json"
  cat > "$results_file" <<EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "inline_count": $inline_count,
  "summary_count": $summary_count,
  "assertions_passed": $assertions_passed
}
EOF

  if [[ "$assertions_passed" != "true" ]]; then
    log_error "Quality assertions FAILED"
    exit 1
  fi

  log_info "All quality assertions PASSED"
}

# Cleanup publisher-owned comments
cleanup_comments() {
  if [[ "$OCR_SMOKE_CLEANUP" != "1" ]]; then
    log_info "Skipping cleanup - OCR_SMOKE_CLEANUP=0"
    return
  fi

  log_info "Cleaning up publisher-owned comments..."

  "$OCR_SMOKE_PUBLISHER_BIN" clear \
    --platform gitlab \
    --gitlab-base-url "$GITLAB_URL" \
    --project-id "$PROJECT_ID" \
    --mr "$MR_IID" \
    --token "$GITLAB_TOKEN" \
    --scope all

  log_info "Cleanup completed"
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

  run_ocr_review
  publish_to_gitlab
  verify_comments
  cleanup_comments

  log_info "Real OCR smoke gate completed successfully!"
}

# Run main function
main
