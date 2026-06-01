// Package gitlab_e2e contains opt-in end-to-end tests against a real GitLab instance.
//
// Run with:
//
//	OCR_E2E_GITLAB=1 go test -tags=e2e ./internal/e2e/gitlab -count=1 -v
//
// Tests skip cleanly when OCR_E2E_GITLAB is not set to "1".
package gitlab_e2e
