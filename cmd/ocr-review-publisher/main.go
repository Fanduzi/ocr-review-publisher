package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/Fanduzi/ocr-review-publisher/internal/review"
)

// Version is set at build time via ldflags.
var Version = "dev"

// publisher is the interface for GitLab publishing operations.
type publisher interface {
	Publish(ctx context.Context, result review.Result) (*review.PublishReport, error)
	ClearInline(ctx context.Context) (*review.PublishReport, error)
	ClearSummary(ctx context.Context) (*review.PublishReport, error)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 0
	}

	switch args[0] {
	case "version":
		fmt.Fprintf(stdout, "ocr-review-publisher %s\n", Version)
		return 0
	case "help", "--help", "-h":
		printUsage(stderr)
		return 0
	case "publish":
		return runPublish(args[1:], stdin, stdout, stderr)
	case "clear":
		return runClear(args[1:], stdin, stdout, stderr)
	case "render":
		return runRender(args[1:], stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 1
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `Usage: ocr-review-publisher <command> [flags]

A platform publishing layer for Open Code Review output.
Consumes OCR machine-readable review findings and publishes them
as high-quality GitLab merge request comments.

Commands:
  publish    Publish OCR findings to a GitLab MR
  clear      Clear publisher-owned comments from a GitLab MR
  render     Render OCR findings as Markdown without publishing
  version    Print version information
  help       Show this help message

Use "ocr-review-publisher <command> --help" for command-specific flags.
`)
}
