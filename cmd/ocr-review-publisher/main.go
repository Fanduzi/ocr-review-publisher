package main

import (
	"flag"
	"fmt"
	"os"
)

// Version is set at build time via ldflags.
var Version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "version":
		fmt.Printf("ocr-review-publisher %s\n", Version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: ocr-review-publisher <command> [flags]

A platform publishing layer for Open Code Review output.
Consumes OCR machine-readable review findings and publishes them
as high-quality GitLab merge request comments.

Commands:
  version    Print version information
  help       Show this help message

Future commands (not yet implemented):
  publish    Publish OCR findings to a GitLab MR
  clear      Clear publisher-owned comments from a GitLab MR
  render     Render OCR findings as Markdown without publishing

Flags:`)
	flag.PrintDefaults()
}
