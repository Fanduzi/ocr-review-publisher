# OCR Output Compatibility

`ocr-review-publisher` depends on the machine-readable output from `ocr review --format json --audience agent`. That output is an external contract owned by Open Code Review, so this project must verify it continuously.

## Compatibility Policy

The parser should support:

- the minimum OCR version documented by this project;
- the latest OCR version verified by scheduled CI; and
- forward-compatible optional fields such as `category`, `severity`, or `confidence`.

The parser should be strict about malformed JSON, but tolerant of harmless wrapper text before the JSON object because some OCR modes may print a summary line before the structured payload.

## Fixture Strategy

Keep captured OCR outputs under `testdata/ocr/` or `testdata/fixtures/`.

Each fixture should record:

- OCR version;
- command used;
- whether the output came from a live LLM run or a sanitized sample;
- expected parser result shape.

Fixtures must not contain secrets, private repository names, private URLs, or local filesystem paths.

## Local Compatibility Flow

The local flow should eventually look like:

```bash
make test-compat
```

For regenerating fixtures:

```bash
scripts/capture-ocr-output.sh --ocr-version latest --output testdata/ocr/latest.json
```

The capture script should run against a small fixture repository with deterministic changes. If live LLM credentials are unavailable, developers should still be able to run parser fixture tests.

## GitHub Actions

The project should use two levels of CI:

- Pull request CI: run parser fixture tests, renderer golden tests, vet, build, and unit tests.
- Scheduled OCR compatibility CI: install the latest published OCR package, capture output when credentials are available, and validate that the parser still accepts the output.

Scheduled compatibility CI should be allowed to fail loudly. Its purpose is to detect upstream OCR output changes before users hit failures in GitLab publishing.

## Release Requirement

Before a public release:

- `make test-compat` must pass.
- The latest OCR compatibility workflow must be green or have a documented known incompatibility.
- Release notes must state the OCR versions verified for the release.
