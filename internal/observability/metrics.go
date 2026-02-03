package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/klyr/klyr/internal/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	requestsTotal           *prometheus.CounterVec
	blocksTotal             *prometheus.CounterVec
	ruleMatchesTotal        *prometheus.CounterVec
	contractViolationsTotal *prometheus.CounterVec
	ratelimitHitsTotal      *prometheus.CounterVec
	requestDuration         *prometheus.HistogramVec
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "klyr_requests_total", Help: "Total requests"},
			[]string{"route", "policy", "action", "code"},
		),
		blocksTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "klyr_blocks_total", Help: "Total blocked requests"},
			[]string{"route", "policy", "reason"},
		),
		ruleMatchesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "klyr_rule_matches_total", Help: "Total rule matches"},
			[]string{"rule_id", "tag", "phase"},
		),
		contractViolationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "klyr_contract_violations_total", Help: "Total contract violations"},
			[]string{"route", "policy", "type"},
		),
		ratelimitHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: "klyr_ratelimit_hits_total", Help: "Total rate limit hits"},
			[]string{"route", "policy", "key"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "klyr_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"route", "policy"},
		),
	}

	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}
	reg.MustRegister(
		m.requestsTotal,
		m.blocksTotal,
		m.ruleMatchesTotal,
		m.contractViolationsTotal,
		m.ratelimitHitsTotal,
		m.requestDuration,
	)

	return m
}

func (m *Metrics) Handler(reg *prometheus.Registry) http.Handler {
	if reg == nil {
		return promhttp.Handler()
	}
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}

func (m *Metrics) Observe(decision logging.Decision, matches []logging.MatchedRule, violations []logging.ContractViolation, ratelimitKey string, reason string) {
	if m == nil {
		return
	}

	route := decision.RouteID
	policy := decision.Policy

	m.requestsTotal.WithLabelValues(route, policy, decision.Action, intToString(decision.StatusCode)).Inc()
	m.requestDuration.WithLabelValues(route, policy).Observe(time.Duration(decision.DurationMS).Seconds())

	if decision.Action == "block" || reason != "" {
		blockReason := reason
		if blockReason == "" {
			blockReason = "rule"
		}
		m.blocksTotal.WithLabelValues(route, policy, blockReason).Inc()
	}

	for _, match := range matches {
		tag := "none"
		if len(match.Tags) > 0 {
			tag = match.Tags[0]
		}
		m.ruleMatchesTotal.WithLabelValues(match.ID, tag, match.Phase).Inc()
	}

	for _, v := range violations {
		m.contractViolationsTotal.WithLabelValues(route, policy, v.Type).Inc()
	}

	if decision.RateLimited {
		m.ratelimitHitsTotal.WithLabelValues(route, policy, ratelimitKey).Inc()
	}
}

func intToString(code int) string {
	if code == 0 {
		return "0"
	}
	return strconv.Itoa(code)
}
