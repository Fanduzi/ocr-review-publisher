package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Fanduzi/ocr-review-publisher/internal/ocroutput"
	"github.com/Fanduzi/ocr-review-publisher/internal/render"
	"github.com/Fanduzi/ocr-review-publisher/internal/review"
)

func runRender(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	input := fs.String("input", "", "OCR output file, or - for stdin")
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if *input == "" {
		fmt.Fprintf(stderr, "error: --input is required\n")
		return 1
	}

	data, err := readInput(*input, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "error reading input: %v\n", err)
		return 1
	}

	result, err := ocroutput.Parse(data)
	if err != nil {
		fmt.Fprintf(stderr, "error parsing OCR output: %v\n", err)
		return 1
	}

	switch *format {
	case "text":
		printRenderText(stdout, *result)
	case "json":
		printRenderJSON(stdout, stderr, *result)
	default:
		fmt.Fprintf(stderr, "error: unknown format %q (supported: text, json)\n", *format)
		return 1
	}
	return 0
}

func printRenderText(w io.Writer, result review.Result) {
	summary := render.Summary(result, nil, render.SummaryOptions{IncludeMarker: false})
	fmt.Fprintln(w, summary)
	for _, f := range result.Findings {
		fmt.Fprintln(w, "---")
		fmt.Fprintln(w, render.InlineComment(f, render.InlineOptions{IncludeMarker: false}))
	}
}

func printRenderJSON(w io.Writer, stderr io.Writer, result review.Result) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(stderr, "error encoding JSON: %v\n", err)
	}
}

func readInput(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(stdin)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return data, nil
}
