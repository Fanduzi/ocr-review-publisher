.PHONY: fmt test vet build check ci test-compat test-e2e-gitlab release-readiness

BINARY := ocr-review-publisher
VERSION ?= dev

fmt:
	go fmt ./...

test:
	go test ./... -count=1

test-race:
	go test -race ./... -count=1

vet:
	go vet ./...

build:
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY) ./cmd/$(BINARY)

check: fmt test vet build test-compat
	@echo "All checks passed."

ci: check

test-compat:
	go test ./internal/compat -count=1 -v

test-e2e-gitlab:
	go test -tags=e2e ./internal/e2e/gitlab -count=1 -v

release-readiness: check test-race
	@echo "==> Checking for uncommitted changes..."
	@git diff --check || (echo "FAIL: whitespace errors"; exit 1)
	@echo "==> Checking for sensitive patterns..."
	@if rg -n 'localhost:8929|/Users/fan|fm2Z' . --glob '!.git' --glob '!internal/compat/ocr_output_test.go' --glob '!internal/platform/gitlab/client_test.go' --glob '!Makefile' 2>/dev/null; then \
		echo "FAIL: sensitive patterns found"; exit 1; \
	fi
	@echo "All release readiness checks passed."
