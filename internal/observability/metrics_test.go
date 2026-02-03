package observability

import (
	"testing"

	"github.com/klyr/klyr/internal/logging"
	"github.com/prometheus/client_golang/prometheus"
)

func TestMetricsObserve(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := NewMetrics(reg)

	decision := logging.Decision{
		RouteID:    "route-1",
		Policy:     "default",
		Action:     "block",
		StatusCode: 403,
		DurationMS: 12,
	}
	metrics.Observe(decision, []logging.MatchedRule{{ID: "r1", Phase: "query", Tags: []string{"sqli"}}}, []logging.ContractViolation{{Type: "query_param_unexpected"}}, "ip", "rule")

	if _, err := reg.Gather(); err != nil {
		t.Fatalf("expected metrics gather to succeed: %v", err)
	}
}
