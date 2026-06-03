#!/usr/bin/env bash
set -euo pipefail

# Capture OCR output for compatibility testing.
# Requires: Node.js, npx, git, and LLM credentials configured for OCR.
#
# LLM credentials are resolved by Open Code Review's resolver in this order:
#   1. OCR env:  OCR_LLM_URL + OCR_LLM_TOKEN + OCR_LLM_MODEL
#                (optional: OCR_USE_ANTHROPIC=true to use Anthropic protocol)
#   2. Claude Code env: ANTHROPIC_BASE_URL + ANTHROPIC_AUTH_TOKEN + ANTHROPIC_MODEL
#
# At least one complete set must be available.

OCR_VERSION="latest"
OUTPUT_PATH=""

usage() {
  cat <<'USAGE'
Usage:
  capture-ocr-output.sh --ocr-version latest --output testdata/ocr/latest.json
  capture-ocr-output.sh --ocr-version 1.1.8 --output /tmp/ocr-output.json

Options:
  --ocr-version VERSION   OCR npm package version (default: latest)
  --output PATH           Output file path for captured JSON

Required environment (one set):
  Set 1 - OCR env:
    OCR_LLM_URL       LLM API endpoint URL
    OCR_LLM_TOKEN      LLM API token
    OCR_LLM_MODEL      LLM model name
    OCR_USE_ANTHROPIC  optional, set to "true" for Anthropic protocol

  Set 2 - Claude Code env:
    ANTHROPIC_BASE_URL      Anthropic API base URL
    ANTHROPIC_AUTH_TOKEN    Anthropic API token
    ANTHROPIC_MODEL         Anthropic model name

The script creates a temporary sample repository with deterministic changes,
runs OCR against it, and writes the JSON output to the specified path.
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --ocr-version)
      OCR_VERSION="$2"
      shift 2
      ;;
    --output)
      OUTPUT_PATH="$2"
      shift 2
      ;;
    -h|--help|help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$OUTPUT_PATH" ]]; then
  echo "Error: --output is required" >&2
  usage >&2
  exit 2
fi

# Check prerequisites
for cmd in node npx git; do
  if ! command -v "$cmd" &>/dev/null; then
    echo "Error: $cmd is required but not found" >&2
    exit 1
  fi
done

# Validate LLM credentials: require at least one complete set.
has_ocr_env=false
has_cc_env=false

if [[ -n "${OCR_LLM_URL:-}" && -n "${OCR_LLM_TOKEN:-}" && -n "${OCR_LLM_MODEL:-}" ]]; then
  has_ocr_env=true
fi
if [[ -n "${ANTHROPIC_BASE_URL:-}" && -n "${ANTHROPIC_AUTH_TOKEN:-}" && -n "${ANTHROPIC_MODEL:-}" ]]; then
  has_cc_env=true
fi

if [[ "$has_ocr_env" == "false" && "$has_cc_env" == "false" ]]; then
  echo "Error: No complete LLM credentials found." >&2
  echo "Set one of:" >&2
  echo "  OCR_LLM_URL + OCR_LLM_TOKEN + OCR_LLM_MODEL" >&2
  echo "  ANTHROPIC_BASE_URL + ANTHROPIC_AUTH_TOKEN + ANTHROPIC_MODEL" >&2
  exit 1
fi

# Create temporary working directory
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

echo "==> Working directory: $WORK_DIR"

# Create a small sample repository with deterministic changes
SAMPLE_REPO="$WORK_DIR/sample-repo"
mkdir -p "$SAMPLE_REPO"
cd "$SAMPLE_REPO"
git init -q -b main
git config user.email "test@example.com"
git config user.name "Test"

# Initial commit
cat > main.go <<'GO'
package main

import "fmt"

func main() {
	fmt.Println("hello")
}
GO
git add main.go
git commit -q -m "initial commit"

# Create a branch with changes for OCR to review
git checkout -q -b review-branch

cat > main.go <<'GO'
package main

import (
	"fmt"
	"os"
)

func main() {
	name := os.Args[1]
	result, err := fmt.Sprintf("hello, %s!", name)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(result)
}
GO
git add main.go
git commit -q -m "add name parameter"

# Install and run OCR
echo "==> Installing OCR @alibaba-group/open-code-review@$OCR_VERSION"
OCR_PACKAGE="@alibaba-group/open-code-review"
if [[ "$OCR_VERSION" == "latest" ]]; then
  npm install -q "$OCR_PACKAGE" 2>/dev/null
else
  npm install -q "$OCR_PACKAGE@$OCR_VERSION" 2>/dev/null
fi

# Find OCR binary
OCR_BIN=""
if [[ -x node_modules/.bin/ocr ]]; then
  OCR_BIN="node_modules/.bin/ocr"
elif [[ -x node_modules/@alibaba-group/open-code-review/bin/ocr ]]; then
  OCR_BIN="node_modules/@alibaba-group/open-code-review/bin/ocr"
else
  echo "Error: OCR binary not found after installation" >&2
  exit 1
fi

echo "==> Running OCR review"
if ! "$OCR_BIN" review --from main --to HEAD --format json --audience agent > "$WORK_DIR/ocr-output.json" 2>"$WORK_DIR/ocr-stderr.txt"; then
  echo "Error: OCR review failed" >&2
  cat "$WORK_DIR/ocr-stderr.txt" >&2
  exit 1
fi

# Check output exists and is non-empty
if [[ ! -s "$WORK_DIR/ocr-output.json" ]]; then
  echo "Error: OCR produced no output" >&2
  cat "$WORK_DIR/ocr-stderr.txt" >&2
  exit 1
fi

# Check output is valid JSON (strip optional prefix lines)
CLEAN_JSON="$WORK_DIR/ocr-clean.json"
sed -n '/^{/,$p' "$WORK_DIR/ocr-output.json" > "$CLEAN_JSON"
if ! python3 -c "import json; json.load(open('$CLEAN_JSON'))" 2>/dev/null; then
  echo "Warning: OCR output is not valid JSON, saving as-is" >&2
  CLEAN_JSON="$WORK_DIR/ocr-output.json"
fi

# Copy to output path
mkdir -p "$(dirname "$OUTPUT_PATH")"
cp "$CLEAN_JSON" "$OUTPUT_PATH"

echo "==> Captured OCR output to $OUTPUT_PATH"
echo "==> OCR version: $OCR_VERSION"
echo "==> Output size: $(wc -c < "$OUTPUT_PATH") bytes"
