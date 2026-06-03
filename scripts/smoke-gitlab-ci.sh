#!/usr/bin/env bash
set -euo pipefail

# Reproducible GitLab CI smoke gate for ocr-review-publisher.
#
# Creates/updates a dedicated smoke branch in the test project with a
# .gitlab-ci.yml, triggers a pipeline, verifies OCR + publisher ran in CI,
# and checks MR comment quality via the GitLab API.
#
# The smoke branch and CI config are retained by default for auditability.
# Pass OCR_CI_SMOKE_CLEANUP=0 to also leave MR comments for manual review.
#
# Usage:
#   make smoke-gitlab-ci
#   OCR_CI_SMOKE_CLEANUP=0 make smoke-gitlab-ci
#
# Required environment (auto-sourced from env.gitlab.local by Makefile):
#   OCR_E2E_GITLAB_URL          GitLab base URL (internal, e.g. http://gitlab:80)
#   OCR_E2E_GITLAB_TOKEN        GitLab API token
#   OCR_E2E_GITLAB_PROJECT_ID   Project ID or namespace path
#   OCR_E2E_GITLAB_MR_IID       Merge request IID
#   OCR_LLM_URL                 LLM API endpoint for OCR
#   OCR_LLM_TOKEN               LLM API token
#   OCR_LLM_MODEL               LLM model name
#
# Optional environment:
#   OCR_CI_SMOKE_BRANCH         Smoke branch name (default: ci-smoke/ocr-review-publisher)
#   OCR_CI_SMOKE_CLEANUP        Clean up comments after run (default: 1)
#   OCR_CI_SMOKE_KEEP_COMMENTS  Alias: set to 1 to skip cleanup
#   OCR_CI_SMOKE_TIMEOUT        Pipeline timeout in seconds (default: 900)
#   OCR_CI_SMOKE_POLL           Poll interval in seconds (default: 5)
#   OCR_E2E_GITLAB_INTERNAL_URL GitLab URL reachable from CI job containers
#                                (default: same as OCR_E2E_GITLAB_URL)

# --- Configuration ---

GITLAB_URL="${OCR_E2E_GITLAB_URL:-}"
GITLAB_TOKEN="${OCR_E2E_GITLAB_TOKEN:-}"
PROJECT_ID="${OCR_E2E_GITLAB_PROJECT_ID:-}"
MR_IID="${OCR_E2E_GITLAB_MR_IID:-}"
LLM_URL="${OCR_LLM_URL:-}"
LLM_TOKEN="${OCR_LLM_TOKEN:-}"
LLM_MODEL="${OCR_LLM_MODEL:-}"

SMOKE_BRANCH="${OCR_CI_SMOKE_BRANCH:-ci-smoke/ocr-review-publisher}"
CLEANUP="${OCR_CI_SMOKE_CLEANUP:-1}"
if [[ "${OCR_CI_SMOKE_KEEP_COMMENTS:-0}" == "1" ]]; then
  CLEANUP="0"
fi
TIMEOUT="${OCR_CI_SMOKE_TIMEOUT:-900}"
POLL_INTERVAL="${OCR_CI_SMOKE_POLL:-5}"

# GitLab URL for CI job containers (may differ from host URL if runner uses
# Docker networking where the host URL doesn't resolve inside job containers).
GITLAB_INTERNAL_URL="${OCR_E2E_GITLAB_INTERNAL_URL:-$GITLAB_URL}"

PUBLISHER_REPO="https://github.com/Fanduzi/ocr-review-publisher.git"
GO_VERSION="1.26.1"

# Marker constants (must match internal/render/markers.go)
INLINE_MARKER='<!-- ocr-review-publisher:inline -->'
SUMMARY_MARKER='<!-- ocr-review-publisher:summary -->'

# Temp directory for API responses
WORK_DIR=""

# --- Colors ---

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info()  { printf "%b[INFO]%b  %s\n" "$GREEN" "$NC" "$1"; }
log_warn()  { printf "%b[WARN]%b  %s\n" "$YELLOW" "$NC" "$1"; }
log_error() { printf "%b[ERROR]%b %s\n" "$RED" "$NC" "$1" >&2; }
log_step()  { printf "\n%b=== %s ===%b\n" "$CYAN" "$1" "$NC"; }

die() { log_error "$1"; exit 1; }

# --- Helpers ---

cleanup_work_dir() {
  if [[ -n "$WORK_DIR" && -d "$WORK_DIR" ]]; then
    rm -rf "$WORK_DIR"
  fi
}

require_var() {
  local name="$1" value="$2"
  if [[ -z "$value" ]]; then
    die "Missing required environment variable: $name"
  fi
}

# GitLab API call with HTTP status validation.
# Usage: gitlab_api METHOD URL [DATA]
# Prints response body to stdout.
gitlab_api() {
  local method="$1" url="$2" data="${3:-}"
  local args=(-s -w '\n%{http_code}' -X "$method"
    -H "PRIVATE-TOKEN: $GITLAB_TOKEN"
    -H "Content-Type: application/json")
  [[ -n "$data" ]] && args+=(-d "$data")

  local response
  response=$(curl "${args[@]}" "$url")
  local body http_code
  body=$(echo "$response" | sed '$d')
  http_code=$(echo "$response" | tail -1)

  if [[ ! "$http_code" =~ ^2[0-9][0-9]$ ]]; then
    log_error "GitLab API $method ${url%%\?*} returned HTTP $http_code"
    echo "$body" | head -c 500 >&2
    return 1
  fi
  echo "$body"
}

# URL-encode a string for use in API paths.
urlencode() {
  python3 -c "import urllib.parse; print(urllib.parse.quote('$1', safe=''))"
}

# --- Prerequisite Checks ---

check_prerequisites() {
  require_var "OCR_E2E_GITLAB_URL"        "$GITLAB_URL"
  require_var "OCR_E2E_GITLAB_TOKEN"      "$GITLAB_TOKEN"
  require_var "OCR_E2E_GITLAB_PROJECT_ID" "$PROJECT_ID"
  require_var "OCR_E2E_GITLAB_MR_IID"     "$MR_IID"
  require_var "OCR_LLM_URL"               "$LLM_URL"
  require_var "OCR_LLM_TOKEN"             "$LLM_TOKEN"
  require_var "OCR_LLM_MODEL"             "$LLM_MODEL"

  command -v jq     &>/dev/null || die "jq is required but not found"
  command -v curl   &>/dev/null || die "curl is required but not found"
  command -v docker &>/dev/null || die "docker is required but not found"
}

# --- Runner Check ---

check_runner() {
  log_step "Checking GitLab Runner"

  # Check Docker container
  local runner_status
  runner_status=$(docker ps --filter name=gitlab-runner --format '{{.Status}}' 2>/dev/null || true)
  if [[ -z "$runner_status" ]]; then
    die "gitlab-runner container not found. Start it or register a new runner."
  fi
  log_info "Docker container: $runner_status"

  # Check runner via GitLab API
  local encoded_pid
  encoded_pid=$(urlencode "$PROJECT_ID")
  local runners_json
  runners_json=$(gitlab_api GET "$GITLAB_URL/api/v4/projects/$encoded_pid/runners" || true)

  local runner_info
  runner_info=$(echo "$runners_json" | jq '.[] | select(.name == "gitlab-runner")' 2>/dev/null || true)
  if [[ -z "$runner_info" ]]; then
    die "Runner 'gitlab-runner' not found in project runners."
  fi

  local online active
  online=$(echo "$runner_info" | jq -r '.online')
  active=$(echo "$runner_info" | jq -r '.active')
  log_info "Runner online=$online, active=$active"

  if [[ "$online" != "true" ]]; then
    die "Runner is not online. Check gitlab-runner container logs."
  fi
}

# --- GitLab API Helpers ---

get_mr_info() {
  local encoded_pid
  encoded_pid=$(urlencode "$PROJECT_ID")
  gitlab_api GET \
    "$GITLAB_URL/api/v4/projects/$encoded_pid/merge_requests/$MR_IID"
}

fetch_discussions() {
  local output_file="$1"
  local encoded_pid
  encoded_pid=$(urlencode "$PROJECT_ID")
  local page=1
  local all_discussions="[]"

  while true; do
    local page_data
    page_data=$(gitlab_api GET \
      "$GITLAB_URL/api/v4/projects/$encoded_pid/merge_requests/$MR_IID/discussions?per_page=100&page=$page")

    local page_count
    page_count=$(echo "$page_data" | jq 'length')
    [[ "$page_count" -eq 0 ]] && break

    all_discussions=$(jq -n --argjson a "$all_discussions" --argjson b "$page_data" '$a + $b')
    page=$((page + 1))
  done

  echo "$all_discussions" > "$output_file"
}

count_markers() {
  local discussions_file="$1" marker="$2"
  jq --arg m "$marker" '
    [.[] | .notes[]? | select(.body != null and (.body | contains($m)))] | length
  ' "$discussions_file"
}

# --- Branch and CI Config ---

prepare_smoke_branch() {
  log_step "Preparing smoke branch: $SMOKE_BRANCH"

  local mr_json source_branch
  mr_json=$(get_mr_info)
  source_branch=$(echo "$mr_json" | jq -r '.source_branch')
  log_info "MR source branch: $source_branch"

  local encoded_pid encoded_branch
  encoded_pid=$(urlencode "$PROJECT_ID")
  encoded_branch=$(urlencode "$source_branch")

  # Get latest commit SHA of source branch.
  local branch_json sha
  branch_json=$(gitlab_api GET \
    "$GITLAB_URL/api/v4/projects/$encoded_pid/repository/branches/$encoded_branch")
  sha=$(echo "$branch_json" | jq -r '.commit.id')
  log_info "Source branch HEAD: ${sha:0:12}"

  # Create or reset smoke branch via API.
  local encoded_smoke
  encoded_smoke=$(urlencode "$SMOKE_BRANCH")

  local existing
  existing=$(gitlab_api GET \
    "$GITLAB_URL/api/v4/projects/$encoded_pid/repository/branches/$encoded_smoke" 2>/dev/null || true)

  if echo "$existing" | jq -e '.name' &>/dev/null; then
    log_info "Branch exists, resetting to ${sha:0:12}"
    # Delete and recreate.
    gitlab_api DELETE \
      "$GITLAB_URL/api/v4/projects/$encoded_pid/repository/branches/$encoded_smoke" >/dev/null
  fi

  gitlab_api POST \
    "$GITLAB_URL/api/v4/projects/$encoded_pid/repository/branches" \
    "{\"branch\": \"$SMOKE_BRANCH\", \"ref\": \"$sha\"}" >/dev/null
  log_info "Smoke branch ready at ${sha:0:12}"
}

write_ci_config() {
  log_step "Writing .gitlab-ci.yml to smoke branch"

  local encoded_pid encoded_branch
  encoded_pid=$(urlencode "$PROJECT_ID")
  encoded_branch=$(urlencode "$SMOKE_BRANCH")

  local ci_yaml
  ci_yaml=$(cat <<'YAML_EOF'
stages:
  - review

image: node:22-bookworm

variables:
  GIT_DEPTH: "0"

review:
  stage: review
  before_script:
    - curl -fsSL https://go.dev/dl/goGO_VERSION_PLACEHOLDER.linux-$(dpkg --print-architecture).tar.gz | tar -xz -C /usr/local
    - export PATH=$PATH:/usr/local/go/bin
    - npm install -g @alibaba-group/open-code-review
    - git clone PUBLISHER_REPO_PLACEHOLDER /tmp/ocr-publisher
    - cd /tmp/ocr-publisher && go build -o /usr/local/bin/ocr-review-publisher ./cmd/ocr-review-publisher
    - cd "$CI_PROJECT_DIR"
  script:
    - export PATH=$PATH:/usr/local/go/bin
    - ocr review --from origin/main --to HEAD --format json --audience agent > /tmp/ocr-result.json
    - |
      ocr-review-publisher clear \
        --platform gitlab \
        --gitlab-base-url "$CI_GITLAB_URL" \
        --project-id "$CI_PROJECT_ID" \
        --mr "$CI_MR_IID" \
        --scope all || true
    - |
      ocr-review-publisher publish \
        --platform gitlab \
        --input /tmp/ocr-result.json \
        --gitlab-base-url "$CI_GITLAB_URL" \
        --project-id "$CI_PROJECT_ID" \
        --mr "$CI_MR_IID" \
        --format text
YAML_EOF
  )

  # Substitute placeholders.
  ci_yaml="${ci_yaml//GO_VERSION_PLACEHOLDER/$GO_VERSION}"
  ci_yaml="${ci_yaml//PUBLISHER_REPO_PLACEHOLDER/$PUBLISHER_REPO}"

  local encoded_content
  encoded_content=$(printf '%s' "$ci_yaml" | base64 | tr -d '\n')

  # Check if file exists.
  local file_check
  file_check=$(gitlab_api GET \
    "$GITLAB_URL/api/v4/projects/$encoded_pid/repository/files/.gitlab-ci.yml?ref=$encoded_branch" 2>/dev/null || true)

  local file_sha=""
  if echo "$file_check" | jq -e '.file_path' &>/dev/null; then
    file_sha=$(echo "$file_check" | jq -r '.blob_id // empty')
  fi

  local payload
  payload=$(jq -n \
    --arg branch "$SMOKE_BRANCH" \
    --arg content "$encoded_content" \
    --arg sha "$file_sha" \
    '{
      branch: $branch,
      content: $content,
      encoding: "base64",
      commit_message: "ci: update smoke gate config"
    } + (if $sha != "" then {last_commit_sha: $sha} else {} end)')

  if [[ -n "$file_sha" ]]; then
    gitlab_api PUT \
      "$GITLAB_URL/api/v4/projects/$encoded_pid/repository/files/.gitlab-ci.yml" \
      "$payload" >/dev/null
    log_info "Updated .gitlab-ci.yml on $SMOKE_BRANCH"
  else
    gitlab_api POST \
      "$GITLAB_URL/api/v4/projects/$encoded_pid/repository/files/.gitlab-ci.yml" \
      "$payload" >/dev/null
    log_info "Created .gitlab-ci.yml on $SMOKE_BRANCH"
  fi
}

# --- Pipeline Lifecycle ---

trigger_pipeline() {
  log_step "Triggering CI pipeline"

  local encoded_pid
  encoded_pid=$(urlencode "$PROJECT_ID")

  local payload
  payload=$(jq -n \
    --arg branch "$SMOKE_BRANCH" \
    --arg mr "$MR_IID" \
    --arg pid "$PROJECT_ID" \
    --arg url "$GITLAB_INTERNAL_URL" \
    --arg gitlab_token "$GITLAB_TOKEN" \
    --arg llm_url "$LLM_URL" \
    --arg llm_token "$LLM_TOKEN" \
    --arg llm_model "$LLM_MODEL" \
    '{
      ref: $branch,
      variables: [
        {key: "CI_MR_IID", value: $mr},
        {key: "CI_PROJECT_ID", value: $pid},
        {key: "CI_GITLAB_URL", value: $url},
        {key: "OCR_GITLAB_TOKEN", value: $gitlab_token},
        {key: "OCR_LLM_URL", value: $llm_url},
        {key: "OCR_LLM_TOKEN", value: $llm_token},
        {key: "OCR_LLM_MODEL", value: $llm_model}
      ]
    }')

  local result
  result=$(gitlab_api POST \
    "$GITLAB_URL/api/v4/projects/$encoded_pid/pipeline" \
    "$payload")

  PIPELINE_ID=$(echo "$result" | jq -r '.id')
  PIPELINE_URL=$(echo "$result" | jq -r '.web_url // empty')
  if [[ -z "$PIPELINE_URL" ]]; then
    PIPELINE_URL="${GITLAB_URL}/pipelines/$PIPELINE_ID"
  fi
  log_info "Pipeline $PIPELINE_ID triggered"
  log_info "URL: $PIPELINE_URL"
}

poll_pipeline() {
  log_step "Waiting for pipeline $PIPELINE_ID"

  local encoded_pid
  encoded_pid=$(urlencode "$PROJECT_ID")
  local elapsed=0

  while [[ $elapsed -lt $TIMEOUT ]]; do
    local result
    result=$(gitlab_api GET \
      "$GITLAB_URL/api/v4/projects/$encoded_pid/pipelines/$PIPELINE_ID")

    local status
    status=$(echo "$result" | jq -r '.status')
    printf "  [%3ds] status: %s\r" "$elapsed" "$status"

    case "$status" in
      success)
        echo ""
        log_info "Pipeline succeeded"
        return 0
        ;;
      failed|canceled|skipped)
        echo ""
        log_error "Pipeline finished with status: $status"
        return 1
        ;;
    esac

    sleep "$POLL_INTERVAL"
    elapsed=$((elapsed + POLL_INTERVAL))
  done

  echo ""
  log_error "Pipeline timed out after ${TIMEOUT}s"
  return 1
}

get_job_info() {
  local encoded_pid
  encoded_pid=$(urlencode "$PROJECT_ID")

  local jobs
  jobs=$(gitlab_api GET \
    "$GITLAB_URL/api/v4/projects/$encoded_pid/pipelines/$PIPELINE_ID/jobs")

  JOB_ID=$(echo "$jobs" | jq -r '.[0].id // empty')
  JOB_STATUS=$(echo "$jobs" | jq -r '.[0].status // "unknown"')

  if [[ -n "$JOB_ID" ]]; then
    JOB_URL=$(echo "$jobs" | jq -r '.[0].web_url // empty')
    if [[ -z "$JOB_URL" ]]; then
      JOB_URL="${GITLAB_URL}/jobs/$JOB_ID"
    fi
    log_info "Job $JOB_ID: $JOB_STATUS"
    log_info "Job URL: $JOB_URL"
  fi
}

fetch_job_log() {
  local encoded_pid
  encoded_pid=$(urlencode "$PROJECT_ID")
  local log_output="$WORK_DIR/job-$JOB_ID.log"

  local http_code
  http_code=$(curl -s -w '%{http_code}' \
    -H "PRIVATE-TOKEN: $GITLAB_TOKEN" \
    "$GITLAB_URL/api/v4/projects/$encoded_pid/jobs/$JOB_ID/trace" \
    -o "$log_output")

  if [[ ! "$http_code" =~ ^2[0-9][0-9]$ ]]; then
    log_error "Failed to fetch job log: HTTP $http_code"
    if [[ -f "$log_output" ]]; then
      head -c 200 "$log_output" >&2
    fi
    return 1
  fi

  echo "$log_output"
}

# --- Verification ---

verify_comments() {
  log_step "Verifying MR comments"

  local discussions_file="$WORK_DIR/post-publish-discussions.json"
  fetch_discussions "$discussions_file"

  local inline_count summary_count
  inline_count=$(count_markers "$discussions_file" "$INLINE_MARKER")
  summary_count=$(count_markers "$discussions_file" "$SUMMARY_MARKER")

  log_info "Post-publish markers: inline=$inline_count, summary=$summary_count"
  POST_INLINE=$inline_count
  POST_SUMMARY=$summary_count

  local failed=false

  # Must have exactly 1 summary.
  if [[ "$summary_count" -ne 1 ]]; then
    log_error "ASSERTION: Expected 1 summary marker, got $summary_count"
    failed=true
  fi

  # Must have inline comments (unless OCR found nothing inline-publishable).
  if [[ "$inline_count" -eq 0 ]]; then
    log_warn "No inline markers - checking if OCR had inline-publishable findings"
    # Acceptable only if summary has diagnostics indicating no inline findings.
    local diagnostics
    diagnostics=$(jq --arg m "$SUMMARY_MARKER" '
      [.[] | .notes[]? | select(.body != null and (.body | contains($m)))] |
      .[0].body // ""
    ' "$discussions_file")
    if echo "$diagnostics" | grep -q "inline_published.*0"; then
      log_info "OCR produced no inline-publishable findings - OK"
    else
      log_error "ASSERTION: No inline markers but summary does not indicate 0 published"
      failed=true
    fi
  fi

  # Extract all notes for quality checks.
  local notes_file="$WORK_DIR/notes.json"
  jq '[.[] | .notes[]? | select(.body != null)]' "$discussions_file" > "$notes_file"

  # No 'line_code can't be blank'.
  local line_code_errors
  line_code_errors=$(jq --arg p "line_code can't be blank" \
    '[.[] | select(.body | contains($p))] | length' "$notes_file")
  if [[ "$line_code_errors" -gt 0 ]]; then
    log_error "ASSERTION: Found 'line_code can't be blank' errors"
    failed=true
  fi

  # No suggestion fences.
  local suggestion_fences
  suggestion_fences=$(jq --arg p '```suggestion' \
    '[.[] | select(.body | contains($p))] | length' "$notes_file")
  if [[ "$suggestion_fences" -gt 0 ]]; then
    log_error "ASSERTION: Found suggestion fences"
    failed=true
  fi

  # Go files should use ```go fence.
  if [[ "$inline_count" -gt 0 ]]; then
    local inline_notes="$WORK_DIR/inline-notes.json"
    jq --arg m "$INLINE_MARKER" \
      '[.[] | select(.body != null and (.body | contains($m)))]' \
      "$notes_file" > "$inline_notes"

    local has_go_comments
    has_go_comments=$(jq '
      [.[] | select(.position != null and .position.new_path != null
        and (.position.new_path | test("\\.go$")))] | length
    ' "$inline_notes")

    if [[ "$has_go_comments" -gt 0 ]]; then
      local go_fences
      go_fences=$(jq --arg f '```go' \
        '[.[] | select(.body | contains($f))] | length' "$inline_notes")
      if [[ "$go_fences" -eq 0 ]]; then
        log_error "ASSERTION: Go file comments missing go language fence"
        failed=true
      else
        log_info "Go language fences found: $go_fences"
      fi
    fi
  fi

  # No duplicate summaries.
  if [[ "$summary_count" -gt 1 ]]; then
    log_error "ASSERTION: Duplicate summary markers ($summary_count)"
    failed=true
  fi

  if [[ "$failed" == "true" ]]; then
    log_error "Quality assertions FAILED"
    return 1
  fi

  log_info "All quality assertions PASSED"
}

# --- Cleanup ---

clear_markers() {
  log_step "Clearing publisher-owned comments"

  local encoded_pid
  encoded_pid=$(urlencode "$PROJECT_ID")
  local discussions_file="$WORK_DIR/clear-discussions.json"

  fetch_discussions "$discussions_file"

  local notes_to_delete
  notes_to_delete=$(jq --arg im "$INLINE_MARKER" --arg sm "$SUMMARY_MARKER" '
    [.[] | .notes[]? | select(.body != null and (
      (.body | contains($im)) or (.body | contains($sm))
    )) | {discussion_id: .discussion_id, note_id: .id}]
  ' "$discussions_file")

  local count
  count=$(echo "$notes_to_delete" | jq 'length')
  log_info "Deleting $count marker notes"

  if [[ "$count" -eq 0 ]]; then
    log_info "No markers to delete"
    return 0
  fi

  local deleted=0 failed_del=0
  for i in $(seq 0 $((count - 1))); do
    local disc_id note_id
    disc_id=$(echo "$notes_to_delete" | jq -r ".[$i].discussion_id")
    note_id=$(echo "$notes_to_delete" | jq -r ".[$i].note_id")

    if gitlab_api DELETE \
      "$GITLAB_URL/api/v4/projects/$encoded_pid/merge_requests/$MR_IID/discussions/$disc_id/notes/$note_id" \
      >/dev/null 2>&1; then
      deleted=$((deleted + 1))
    else
      failed_del=$((failed_del + 1))
    fi
  done

  log_info "Deleted: $deleted, Failed: $failed_del"
}

verify_cleanup() {
  log_step "Verifying post-cleanup markers"

  local discussions_file="$WORK_DIR/post-cleanup-discussions.json"
  fetch_discussions "$discussions_file"

  local inline_count summary_count
  inline_count=$(count_markers "$discussions_file" "$INLINE_MARKER")
  summary_count=$(count_markers "$discussions_file" "$SUMMARY_MARKER")

  log_info "Post-cleanup markers: inline=$inline_count, summary=$summary_count"
  POST_CLEANUP_INLINE=$inline_count
  POST_CLEANUP_SUMMARY=$summary_count

  if [[ "$inline_count" -ne 0 || "$summary_count" -ne 0 ]]; then
    log_error "ASSERTION: Markers remain after cleanup"
    return 1
  fi

  log_info "Post-cleanup verified: 0/0"
}

# --- Reporting ---

report() {
  log_step "Report"

  cat <<EOF
Pipeline:     ${PIPELINE_URL:-N/A}
Job:          ${JOB_URL:-N/A}
Branch:       $SMOKE_BRANCH
MR:           ${MR_URL:-N/A}
Cleanup mode: $([ "$CLEANUP" == "1" ] && echo "enabled" || echo "disabled")

Marker counts:
  Pre-clear:        inline=${PRE_INLINE:-?}, summary=${PRE_SUMMARY:-?}
  Post-publish:     inline=${POST_INLINE:-?}, summary=${POST_SUMMARY:-?}
  Post-cleanup:     inline=${POST_CLEANUP_INLINE:-skipped}, summary=${POST_CLEANUP_SUMMARY:-skipped}

Status: PASSED
EOF
}

# --- Main ---

main() {
  log_step "GitLab CI Smoke Gate"
  log_info "Branch: $SMOKE_BRANCH"
  log_info "MR IID: $MR_IID"
  log_info "Cleanup: $CLEANUP"
  log_info "Timeout: ${TIMEOUT}s"

  check_prerequisites
  check_runner

  # Fetch MR URL from API for reporting.
  local mr_json
  mr_json=$(get_mr_info)
  MR_URL=$(echo "$mr_json" | jq -r '.web_url // empty')
  if [[ -z "$MR_URL" ]]; then
    MR_URL="${GITLAB_URL}/merge_requests/$MR_IID"
  fi
  log_info "MR URL: $MR_URL"

  WORK_DIR=$(mktemp -d)
  trap cleanup_work_dir EXIT

  # Pre-clear existing markers.
  log_step "Pre-clearing markers"
  clear_markers

  local preclear_discussions="$WORK_DIR/preclear-discussions.json"
  fetch_discussions "$preclear_discussions"
  PRE_INLINE=$(count_markers "$preclear_discussions" "$INLINE_MARKER")
  PRE_SUMMARY=$(count_markers "$preclear_discussions" "$SUMMARY_MARKER")
  log_info "Pre-clear verified: inline=$PRE_INLINE, summary=$PRE_SUMMARY"

  if [[ "$PRE_INLINE" -ne 0 || "$PRE_SUMMARY" -ne 0 ]]; then
    die "Pre-clear assertion failed: markers remain"
  fi

  # Prepare branch and CI config.
  prepare_smoke_branch
  write_ci_config

  # Trigger and wait.
  trigger_pipeline
  if ! poll_pipeline; then
    get_job_info
    local log_file
    log_file=$(fetch_job_log)
    log_error "=== Last 60 lines of job log ==="
    tail -60 "$log_file" >&2
    exit 1
  fi

  get_job_info

  # Verify.
  verify_comments

  # Cleanup or report.
  if [[ "$CLEANUP" == "1" ]]; then
    clear_markers
    verify_cleanup
  else
    log_info "Comments retained for manual inspection"
    POST_CLEANUP_INLINE="skipped"
    POST_CLEANUP_SUMMARY="skipped"
  fi

  report
}

main "$@"
