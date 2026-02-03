package main

import (
	"errors"
	"time"

	"github.com/klyr/klyr/internal/config"
	"github.com/spf13/cobra"
)

func newLearnCmd() *cobra.Command {
	var configPath string
	var duration time.Duration
	var outPath string

	cmd := &cobra.Command{
		Use:   "learn",
		Short: "Run gateway in learn mode for a fixed duration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				return errors.New("config path is required")
			}
			if duration <= 0 {
				return errors.New("duration must be > 0")
			}
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			applyOverrides(cfg, config.ModeLearn, "")
			if outPath != "" {
				applyOverrides(cfg, "", outPath)
			}
			if err := cfg.Validate(); err != nil {
				return err
			}
			return runGateway(cmd.Context(), cfg, true, duration)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to config file")
	cmd.Flags().DurationVar(&duration, "duration", 0, "Learn duration (e.g. 2m)")
	cmd.Flags().StringVar(&outPath, "out", "", "Override contract output path")

	return cmd
}
