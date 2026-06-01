# GitLab Usage

This document covers detailed GitLab configuration and usage for `ocr-review-publisher`.

## Token Requirements

### Creating a Token

1. Go to GitLab > User Settings > Access Tokens
2. Create a token with `api` scope
3. For self-hosted GitLab, ensure the token has access to the target project

### Required Permissions

- **api** - Create, update, and delete merge request notes
- **read_repository** - Read merge request diffs and versions

### CI/CD Token

In GitLab CI, use the built-in `CI_JOB_TOKEN` or a project/group access token:

```yaml
variables:
  GITLAB_TOKEN: ${CI_JOB_TOKEN}
```

Or use a dedicated token stored in CI/CD variables:

```yaml
variables:
  GITLAB_TOKEN: ${OCR_GITLAB_TOKEN}
```

## Project Identification

### Project ID

Use the numeric project ID:

```bash
--project-id 123
```

Find the project ID in GitLab > Project > Settings > General.

### Namespace Path

Use the URL-encoded namespace path:

```bash
--project-id group/subgroup/project
```

The publisher automatically URL-encodes the path (e.g., `group%2Fsubgroup%2Fproject`).

## Merge Request IID

The MR IID (internal ID) is the number shown in the MR URL:

```
https://gitlab.example.com/group/project/-/merge_requests/42
                                                  ^^
                                                  MR IID = 42
```

This is not the same as the global MR ID.

## Self-Hosted GitLab

For self-hosted GitLab instances:

```bash
./ocr-review-publisher publish \
  --platform gitlab \
  --gitlab-base-url https://gitlab.internal.example.com \
  --project-id group/project \
  --mr 42 \
  --input ocr-result.json
```

Or via environment variable:

```bash
export OCR_GITLAB_BASE_URL=https://gitlab.internal.example.com
```

### GitLab 13.12 Compatibility

The publisher supports GitLab 13.12 self-hosted instances:

- Uses `/diffs` endpoint for changed files
- Falls back to `/changes?access_raw_diffs=true` if `/diffs` returns 404
- Paginates discussions with `per_page=100` and exact `X-Next-Page` header

## Inline Comments

### How Anchors Work

The publisher creates inline comments on added lines in the diff:

1. Fetches changed files and their diffs
2. Fetches diff version SHAs for position references
3. For each finding, selects the first added line in the range
4. Creates an inline discussion at that position

### When Comments Are Skipped

Inline comments are skipped (included in summary diagnostics) when:

- Finding has no path or no positive line range
- No added line exists in the finding range
- Diff is unavailable or unparseable
- Diff version is missing
- GitLab rejects the inline position

All findings remain in the summary even when inline is skipped.

## Summary Comments

### Create Behavior

When no existing summary marker note exists, the publisher creates a new note with:

- Product header
- Total findings count
- Findings list with path and line range
- Diagnostics for skipped/failed inline comments
- Summary marker

### Update Behavior

When an existing summary marker note exists, the publisher updates it instead of creating a duplicate. This ensures repeated publish runs don't create multiple summaries.

## Clear Behavior

### Scope Options

- `--scope inline` - Delete only inline marker notes
- `--scope summary` - Delete only summary marker notes
- `--scope all` - Delete both inline and summary marker notes

### Safety

Clear operations only delete notes containing publisher markers. User comments and other bot comments are never deleted.

## Troubleshooting

### Invalid Line Anchors

**Symptom:** Inline comments are skipped with "no added line in range"

**Cause:** The finding targets a line that is context-only (not added) in the diff.

**Fix:** Ensure findings target lines that were actually added or modified in the MR.

### No Diff Version

**Symptom:** All inline comments fail with "no GitLab diff version available"

**Cause:** The MR has no diff versions (may happen with very old or empty MRs).

**Fix:** Ensure the MR has actual changes. The summary will still be created.

### Token Permissions

**Symptom:** 403 Forbidden errors

**Cause:** Token lacks required permissions.

**Fix:** Ensure the token has `api` scope and access to the target project.

### Self-Hosted Base URL

**Symptom:** Connection refused or DNS errors

**Cause:** Incorrect base URL.

**Fix:** Ensure the base URL includes the protocol (https://) and does not end with a trailing slash.

### Comments Skipped but Summary Created

**Symptom:** Summary exists but no inline comments

**Cause:** Findings don't target added lines in the diff, or diff is unavailable.

**Fix:** This is expected behavior. Check the summary diagnostics for details on which findings were skipped and why.
