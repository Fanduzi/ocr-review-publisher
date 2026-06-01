.PHONY: fmt test vet build check test-compat test-e2e-gitlab

BINARY := ocr-review-publisher
VERSION ?= dev

fmt:
	go fmt ./...

test:
	go test ./... -count=1

vet:
	go vet ./...

build:
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY) ./cmd/$(BINARY)

check: fmt test vet build test-compat
	@echo "All checks passed."

test-compat:
	go test ./internal/compat -count=1 -v

test-e2e-gitlab:
	@echo "Skipping GitLab e2e tests: set OCR_E2E_GITLAB=1 and required env vars to enable."
	@echo "See docs/ for e2e requirements."
