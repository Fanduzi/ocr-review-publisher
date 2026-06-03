# GitLab CI Smoke Gate

This document describes the reproducible GitLab CI smoke gate for `ocr-review-publisher`.

## Overview

The CI smoke gate validates that the publisher works correctly when run inside a real GitLab CI pipeline. It:

1. Creates or updates a dedicated smoke branch in the test project with a `.gitlab-ci.yml`.
2. Triggers a pipeline via the GitLab API.
3. Waits for the pipeline to complete.
4. Verifies that OCR ran, the publisher ran, and MR comments meet quality assertions.
5. Optionally cleans up marker comments.

The smoke branch and `.gitlab-ci.yml` are retained by default so the CI config and pipeline history are inspectable on the GitLab instance.

## Prerequisites

- A running GitLab instance with a test project.
- A registered Docker runner accessible from the GitLab instance.
- OCR LLM credentials (`OCR_LLM_URL`, `OCR_LLM_TOKEN`, `OCR_LLM_MODEL`).
- An open merge request in the test project.

## Usage

```bash
# Run with cleanup (default): comments are deleted after verification.
make smoke-gitlab-ci

# Run without cleanup: comments are left for manual inspection.
OCR_CI_SMOKE_CLEANUP=0 make smoke-gitlab-ci
```

## Environment Variables

Required (typically in `env.gitlab.local`, auto-sourced by the Makefile target):

| Variable | Description |
|----------|-------------|
| `OCR_E2E_GITLAB_URL` | GitLab base URL |
| `OCR_E2E_GITLAB_TOKEN` | GitLab API token |
| `OCR_E2E_GITLAB_PROJECT_ID` | Project ID or namespace path |
| `OCR_E2E_GITLAB_MR_IID` | Merge request IID |
| `OCR_LLM_URL` | LLM API endpoint for OCR |
| `OCR_LLM_TOKEN` | LLM API token |
| `OCR_LLM_MODEL` | LLM model name |

Optional:

| Variable | Default | Description |
|----------|---------|-------------|
| `OCR_CI_SMOKE_BRANCH` | `ci-smoke/ocr-review-publisher` | Smoke branch name |
| `OCR_CI_SMOKE_CLEANUP` | `1` | Set to `0` to leave comments |
| `OCR_CI_SMOKE_KEEP_COMMENTS` | `0` | Alias: set to `1` for no cleanup |
| `OCR_CI_SMOKE_TIMEOUT` | `900` | Pipeline timeout in seconds |
| `OCR_CI_SMOKE_POLL` | `5` | Poll interval in seconds |

## How It Works

### Smoke Branch

The script creates a branch named `ci-smoke/ocr-review-publisher` (configurable) from the MR's source branch. It writes a `.gitlab-ci.yml` to this branch via the GitLab API. The branch is reset on each run to match the latest MR source HEAD.

The CI config installs Node.js, Go, the OCR CLI, and builds the publisher from source. It then runs `ocr review` and `ocr-review-publisher publish` against the MR.

### Pipeline Trigger

The pipeline is triggered via the GitLab API (not a merge request event), so all required variables are passed explicitly. This avoids reliance on `CI_MERGE_REQUEST_IID` which is not set for API-triggered pipelines.

### Quality Assertions

After the pipeline succeeds, the script fetches MR discussions and asserts:

- Exactly 1 summary marker comment exists.
- Inline marker comments exist (unless OCR found no inline-publishable findings).
- No `line_code can't be blank` errors.
- No `suggestion` fences.
- Go file comments use ` ```go ` language fences.
- No duplicate summary markers.

### Cleanup Behavior

By default (`OCR_CI_SMOKE_CLEANUP=1`), the script deletes all publisher-owned marker comments after verification and asserts the counts return to 0/0.

With `OCR_CI_SMOKE_CLEANUP=0`, comments are left on the MR for manual inspection. The script prints the MR URL so you can review them.

## Runner Infrastructure

The smoke gate requires a Docker runner registered with the GitLab instance. The runner configuration and token live in a local config directory and must not be committed.

To check runner status:

```bash
docker ps --filter name=gitlab-runner
```

To remove the runner (if needed):

```bash
docker stop gitlab-runner && docker rm gitlab-runner
```

## Output

The script prints:

- Pipeline URL and job URL.
- Smoke branch name.
- Cleanup mode.
- Pre-clear, post-publish, and post-cleanup marker counts.
- Final PASSED/FAILED status.
