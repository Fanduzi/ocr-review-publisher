package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/Fanduzi/ocr-review-publisher/internal/platform/gitlab"
	"github.com/Fanduzi/ocr-review-publisher/internal/review"
)

func runClear(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("clear", flag.ContinueOnError)
	platform := fs.String("platform", "", "publishing platform (gitlab)")
	baseURL := fs.String("gitlab-base-url", "", "GitLab base URL")
	project := fs.String("project-id", "", "GitLab project ID or namespace path")
	mr := fs.Int("mr", 0, "GitLab merge request IID")
	token := fs.String("token", "", "GitLab token")
	scope := fs.String("scope", "", "scope to clear: inline, summary, or all")
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if *platform == "" {
		*platform = "gitlab"
	}
	if *platform != "gitlab" {
		fmt.Fprintf(stderr, "error: unsupported platform %q (v1 supports gitlab only)\n", *platform)
		return 1
	}

	if *scope == "" {
		fmt.Fprintf(stderr, "error: --scope is required (inline, summary, or all)\n")
		return 1
	}
	if *scope != "inline" && *scope != "summary" && *scope != "all" {
		fmt.Fprintf(stderr, "error: invalid scope %q (supported: inline, summary, all)\n", *scope)
		return 1
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

	client := gitlab.NewClient(cfg.BaseURL, cfg.Token, nil)
	pub := gitlab.NewPublisher(client, gitlab.PublishOptions{
		Project:         cfg.Project,
		MergeRequestIID: cfg.MRIID,
	})

	return clearWithPublisher(pub, *scope, *format, stdout, stderr)
}

func clearWithPublisher(pub publisher, scope, format string, stdout, stderr io.Writer) int {
	ctx := context.Background()
	var report *review.PublishReport
	var err error

	switch scope {
	case "inline":
		report, err = pub.ClearInline(ctx)
	case "summary":
		report, err = pub.ClearSummary(ctx)
	case "all":
		report, err = pub.ClearInline(ctx)
		if err != nil {
			fmt.Fprintf(stderr, "error clearing inline: %v\n", err)
			return 1
		}
		r2, err2 := pub.ClearSummary(ctx)
		if err2 != nil {
			fmt.Fprintf(stderr, "error clearing summary: %v\n", err2)
			return 1
		}
		report.InlineDeleted += r2.InlineDeleted
		report.SummaryDeleted += r2.SummaryDeleted
		report.Warnings = append(report.Warnings, r2.Warnings...)
	}

	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return printClearReport(report, format, stdout, stderr)
}

func printClearReport(report *review.PublishReport, format string, stdout, stderr io.Writer) int {
	switch format {
	case "json":
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintf(stderr, "error encoding JSON: %v\n", err)
			return 1
		}
	case "text":
		fmt.Fprintf(stdout, "Cleared: %d inline, %d summary\n", report.InlineDeleted, report.SummaryDeleted)
		if len(report.Warnings) > 0 {
			fmt.Fprintf(stderr, "Warnings: %d issue(s)\n", len(report.Warnings))
		}
	default:
		fmt.Fprintf(stderr, "error: unknown format %q (supported: text, json)\n", format)
		return 1
	}
	return 0
}
