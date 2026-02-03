package logging

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

const maxEvidence = 64

// Decision is written as a single JSON object per request.
type Decision struct {
	Timestamp          time.Time           `json:"ts"`
	RequestID          string              `json:"request_id"`
	ClientIP           string              `json:"client_ip"`
	Host               string              `json:"host"`
	Method             string              `json:"method"`
	Path               string              `json:"path"`
	Query              string              `json:"query"`
	RouteID            string              `json:"route_id"`
	Policy             string              `json:"policy"`
	Mode               string              `json:"mode"`
	Score              int                 `json:"score"`
	Threshold          int                 `json:"threshold"`
	Action             string              `json:"action"`
	StatusCode         int                 `json:"status_code"`
	MatchedRules       []MatchedRule       `json:"matched_rules"`
	ContractViolations []ContractViolation `json:"contract_violations"`
	RateLimited        bool                `json:"rate_limited"`
	DurationMS         int64               `json:"duration_ms"`
	UpstreamMS         int64               `json:"upstream_ms"`
}

type MatchedRule struct {
	ID       string   `json:"id"`
	Phase    string   `json:"phase"`
	Score    int      `json:"score"`
	Tags     []string `json:"tags"`
	Evidence string   `json:"evidence"`
}

type ContractViolation struct {
	Type  string `json:"type"`
	Field string `json:"field"`
}

type DecisionLogger struct {
	w io.Writer
}

func NewDecisionLogger(w io.Writer) *DecisionLogger {
	return &DecisionLogger{w: w}
}

func OpenDecisionLog(path string) (*DecisionLogger, func() error, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, err
	}
	return NewDecisionLogger(file), file.Close, nil
}

func (l *DecisionLogger) Write(decision Decision) error {
	decision.MatchedRules = sanitizeMatchedRules(decision.MatchedRules)

	data, err := json.Marshal(decision)
	if err != nil {
		return err
	}
	_, err = l.w.Write(append(data, '\n'))
	return err
}

func sanitizeMatchedRules(rules []MatchedRule) []MatchedRule {
	if len(rules) == 0 {
		return nil
	}
	out := make([]MatchedRule, len(rules))
	for i, rule := range rules {
		out[i] = rule
		if len(rule.Evidence) > maxEvidence {
			out[i].Evidence = rule.Evidence[:maxEvidence]
		}
	}
	return out
}
