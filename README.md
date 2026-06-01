# OCR Review Publisher

`ocr-review-publisher` is a platform publishing layer for Open Code Review output.

Open Code Review generates review findings. This project consumes its machine-readable output and publishes those findings as high-quality GitLab merge request comments.

Version 1 is intentionally narrow:

- OCR is the only supported review producer.
- GitLab is the only supported publishing platform.
- The project focuses on comment rendering, safe inline anchors, summary updates, marker-scoped clear operations, and CI-friendly execution.

Design documents:

- [Design](docs/2026-06-01-ocr-review-publisher-design.md)
- [Implementation Plan](docs/2026-06-01-ocr-review-publisher-implementation-plan.md)
- [Quality Gates](docs/quality-gates.md)

This repository is not an OCR fork and does not replace OCR's review engine.
