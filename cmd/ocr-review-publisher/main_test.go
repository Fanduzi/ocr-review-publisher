package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestHelpFlag(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// --help may exit with code 0 or 2 depending on flag library
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 0 && exitErr.ExitCode() != 2 {
				t.Fatalf("unexpected exit code %d: %s", exitErr.ExitCode(), string(out))
			}
		}
	}
	output := string(out)
	if !strings.Contains(output, "Usage") {
		t.Errorf("help output should contain 'Usage', got:\n%s", output)
	}
	if !strings.Contains(output, "ocr-review-publisher") {
		t.Errorf("help output should contain binary name, got:\n%s", output)
	}
}

func TestVersionSubcommand(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version command failed: %v\n%s", err, string(out))
	}
	output := string(out)
	if !strings.Contains(output, Version) {
		t.Errorf("version output should contain %q, got:\n%s", Version, output)
	}
}

func TestNoArgsShowsHelp(t *testing.T) {
	cmd := exec.Command("go", "run", ".")
	out, _ := cmd.CombinedOutput()
	output := string(out)
	if !strings.Contains(output, "Usage") {
		t.Errorf("running with no args should show usage, got:\n%s", output)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
