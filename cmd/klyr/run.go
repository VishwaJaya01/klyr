package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/klyr/klyr/internal/config"
	"github.com/klyr/klyr/internal/gateway"
	"github.com/klyr/klyr/internal/logging"
	"github.com/klyr/klyr/internal/observability"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var configPath string
	var modeOverride string
	var contractOverride string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the Klyr gateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath == "" {
				return errors.New("config path is required")
			}
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			applyOverrides(cfg, modeOverride, contractOverride)
			if err := cfg.Validate(); err != nil {
				return err
			}
			return runGateway(cmd.Context(), cfg, false, 0)
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to config file")
	cmd.Flags().StringVar(&modeOverride, "mode", "", "Override policy mode for all policies")
	cmd.Flags().StringVar(&contractOverride, "contract", "", "Override contract path for all policies")

	return cmd
}

func runGateway(ctx context.Context, cfg *config.Config, learnMode bool, duration time.Duration) error {
	gw, err := gateway.New(cfg)
	if err != nil {
		return err
	}

	if cfg.Logging.DecisionLog != "" {
		logger, closer, err := logging.OpenDecisionLog(cfg.ResolvePath(cfg.Logging.DecisionLog))
		if err != nil {
			return err
		}
		defer func() { _ = closer() }()
		gw.SetDecisionLogger(logger)
	}

	metricsSrv, err := startMetricsServer(cfg, gw)
	if err != nil {
		return err
	}
	defer func() {
		if metricsSrv != nil {
			_ = metricsSrv.Shutdown(context.Background())
		}
	}()

	srv := &http.Server{
		Addr:              cfg.Server.Listen,
		Handler:           gw,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		if cfg.Server.TLS.Enabled {
			serverErr <- srv.ListenAndServeTLS(cfg.ResolvePath(cfg.Server.TLS.CertFile), cfg.ResolvePath(cfg.Server.TLS.KeyFile))
			return
		}
		serverErr <- srv.ListenAndServe()
	}()

	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	if learnMode {
		select {
		case <-time.After(duration):
		case <-signalCtx.Done():
		case err := <-serverErr:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		}
	} else {
		select {
		case <-signalCtx.Done():
		case err := <-serverErr:
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	if learnMode {
		if err := gw.SaveContracts(cfg); err != nil {
			return err
		}
		if err := ensureMinSamples(cfg, gw); err != nil {
			return err
		}
	}

	return nil
}

func startMetricsServer(cfg *config.Config, gw *gateway.Gateway) (*http.Server, error) {
	if !cfg.Metrics.Enabled {
		return nil, nil
	}

	reg := prometheus.NewRegistry()
	metrics := observability.NewMetrics(reg)
	gw.SetMetrics(metrics)

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler(reg))

	srv := &http.Server{Addr: cfg.Metrics.Listen, Handler: mux}
	go func() {
		_ = srv.ListenAndServe()
	}()
	return srv, nil
}

func applyOverrides(cfg *config.Config, modeOverride, contractOverride string) {
	for name, policyCfg := range cfg.Policies {
		if modeOverride != "" {
			policyCfg.Mode = modeOverride
		}
		if contractOverride != "" {
			policyCfg.Contract.Path = contractOverride
		}
		cfg.Policies[name] = policyCfg
	}
}

func ensureMinSamples(cfg *config.Config, gw *gateway.Gateway) error {
	for i, route := range cfg.Routes {
		routeID := fmt.Sprintf("route-%d", i)
		policyCfg, ok := cfg.Policies[route.Policy]
		if !ok {
			continue
		}
		if policyCfg.Mode != config.ModeLearn {
			continue
		}
		c := gw.Contract(routeID, route.Policy)
		if c == nil {
			return fmt.Errorf("missing contract for %s", route.Policy)
		}
		if policyCfg.Contract.MinSamples > 0 && c.Samples < policyCfg.Contract.MinSamples {
			return fmt.Errorf("contract for %s has %d samples, need %d", route.Policy, c.Samples, policyCfg.Contract.MinSamples)
		}
	}
	return nil
}
