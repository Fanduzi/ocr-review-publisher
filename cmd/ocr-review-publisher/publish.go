package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/Fanduzi/ocr-review-publisher/internal/ocroutput"
	"github.com/Fanduzi/ocr-review-publisher/internal/platform/gitlab"
	"github.com/Fanduzi/ocr-review-publisher/internal/review"
)

func runPublish(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	platform := fs.String("platform", "", "publishing platform (gitlab)")
	input := fs.String("input", "", "OCR output file, or - for stdin")
	baseURL := fs.String("gitlab-base-url", "", "GitLab base URL")
	project := fs.String("project-id", "", "GitLab project ID or namespace path")
	mr := fs.Int("mr", 0, "GitLab merge request IID")
	token := fs.String("token", "", "GitLab token")
	noInline := fs.Bool("no-inline", false, "skip inline comments")
	noSummary := fs.Bool("no-summary", false, "skip summary comment")
	clearExisting := fs.Bool("clear-existing", false, "clear existing publisher comments first")
	dryRun := fs.Bool("dry-run", false, "render without publishing")
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if *input == "" {
		fmt.Fprintf(stderr, "error: --input is required\n")
		return 1
	}

	if *platform == "" {
		*platform = "gitlab"
	}
	if *platform != "gitlab" {
		fmt.Fprintf(stderr, "error: unsupported platform %q (v1 supports gitlab only)\n", *platform)
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

	if *dryRun {
		return publishDryRun(*result, *noInline, *noSummary, *format, stdout, stderr)
	}

	cfg := resolveGitLabConfig(*baseURL, *token, *project, *mr)
	if cfg.Token == "" {
		fmt.Fprintf(stderr, "error: GitLab token required (use --token, GITLAB_TOKEN, or OCR_GITLAB_TOKEN)\n")
		return 1
	}
	if cfg.Project == "" {
		fmt.Fprintf(stderr, "error: GitLab project required (use --project-id or CI_PROJECT_ID)\n")
		return 1
	}
	if cfg.MRIID == 0 {
		fmt.Fprintf(stderr, "error: GitLab MR IID required (use --mr or CI_MERGE_REQUEST_IID)\n")
		return 1
	}

	pub := newGitLabPublisher(cfg, *noInline, *noSummary, *clearExisting)
	return publishWithPublisher(pub, *result, *format, stdout, stderr)
}

func publishDryRun(result review.Result, noInline, noSummary bool, format string, stdout, stderr io.Writer) int {
	if format == "json" {
		report := &review.PublishReport{}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(stderr, "error encoding JSON: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stderr, "dry-run: parsed %d finding(s), no publishing performed\n", len(result.Findings))
	return 0
}

func publishWithPublisher(pub publisher, result review.Result, format string, stdout, stderr io.Writer) int {
	report, err := pub.Publish(context.Background(), result)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return printReport(report, format, stdout, stderr)
}

func newGitLabPublisher(cfg gitlabConfig, noInline, noSummary, clearExisting bool) publisher {
	client := gitlab.NewClient(cfg.BaseURL, cfg.Token, nil)
	return gitlab.NewPublisher(client, gitlab.PublishOptions{
		Project:         cfg.Project,
		MergeRequestIID: cfg.MRIID,
		NoInline:        noInline,
		NoSummary:       noSummary,
		ClearExisting:   clearExisting,
	})
}

func printReport(report *review.PublishReport, format string, stdout, stderr io.Writer) int {
	switch format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(stderr, "error encoding JSON: %v\n", err)
			return 1
		}
	case "text":
		fmt.Fprintf(stdout, "Published: %d inline, skipped %d, failed %d\n",
			report.InlinePublished, report.InlineSkipped, report.InlineFailed)
		if report.SummaryCreated {
			fmt.Fprintln(stdout, "Summary: created")
		} else if report.SummaryUpdated {
			fmt.Fprintln(stdout, "Summary: updated")
		}
		if len(report.Warnings) > 0 {
			fmt.Fprintf(stderr, "Warnings: %d issue(s)\n", len(report.Warnings))
		}
	default:
		fmt.Fprintf(stderr, "error: unknown format %q (supported: text, json)\n", format)
		return 1
	}
	return 0
}
