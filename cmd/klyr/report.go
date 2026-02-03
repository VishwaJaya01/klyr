package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/klyr/klyr/internal/report"
	"github.com/spf13/cobra"
)

func newReportCmd() *cobra.Command {
	var inputPath string
	var since string
	var format string
	var outPath string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Summarize decision logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if inputPath == "" {
				return errors.New("input path is required")
			}

			reader := report.Reader{}
			if since != "" {
				dur, err := time.ParseDuration(since)
				if err != nil {
					return fmt.Errorf("invalid since duration: %w", err)
				}
				reader.Since = time.Now().Add(-dur)
			}

			decisions, err := reader.Read(inputPath)
			if err != nil {
				return err
			}

			summary := report.Summarize(decisions)
			switch format {
			case "", "text":
				return report.WriteOutput(outPath, []byte(report.RenderText(summary)))
			case "md":
				return report.WriteOutput(outPath, []byte(report.RenderMarkdown(summary)))
			case "json":
				data, err := report.RenderJSON(summary)
				if err != nil {
					return err
				}
				return report.WriteOutput(outPath, data)
			default:
				return fmt.Errorf("unknown format %q", format)
			}
		},
	}

	cmd.Flags().StringVar(&inputPath, "in", "", "Path to decision log JSONL")
	cmd.Flags().StringVar(&since, "since", "", "Only include entries newer than this duration (e.g. 10m)")
	cmd.Flags().StringVar(&format, "format", "text", "Output format: text|md|json")
	cmd.Flags().StringVar(&outPath, "out", "", "Output file path (default stdout)")

	return cmd
}
