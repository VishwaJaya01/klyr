package main

import (
	"errors"

	"github.com/klyr/klyr/internal/config"
	"github.com/spf13/cobra"
)

func newEnforceCmd() *cobra.Command {
	var configPath string
	var contractPath string

	cmd := &cobra.Command{
		Use:   "enforce",
		Short: "Run gateway in enforce mode",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				return errors.New("config path is required")
			}
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			applyOverrides(cfg, config.ModeEnforce, "")
			if contractPath != "" {
				applyOverrides(cfg, "", contractPath)
			}
			if err := cfg.Validate(); err != nil {
				return err
			}
			return runGateway(cmd.Context(), cfg, false, 0)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to config file")
	cmd.Flags().StringVar(&contractPath, "contract", "", "Override contract path")

	return cmd
}
