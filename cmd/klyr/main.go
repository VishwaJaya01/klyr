package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/klyr/klyr/internal/config"
	"github.com/spf13/cobra"
)

var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:          "klyr",
		Short:        "Klyr security gateway",
		SilenceUsage: true,
	}

	root.AddCommand(newValidateCmd())
	root.AddCommand(stubCmd("run"))
	root.AddCommand(stubCmd("learn"))
	root.AddCommand(stubCmd("enforce"))
	root.AddCommand(stubCmd("report"))
	root.AddCommand(newVersionCmd())

	if err := root.Execute(); err != nil {
		var verr *config.ValidationError
		if errors.As(err, &verr) {
			for _, msg := range verr.Problems {
				fmt.Fprintln(os.Stderr, msg)
			}
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func newValidateCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a Klyr configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				return errors.New("config path is required")
			}
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			if err := cfg.Validate(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "config ok")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to config file")

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "version=%s commit=%s buildDate=%s\n", version, commit, buildDate)
		},
	}
}

func stubCmd(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s (not implemented yet)", name),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("%s command not implemented yet", name)
		},
	}
}
