package report

import (
	"testing"
	"time"

	"github.com/klyr/klyr/internal/logging"
)

func TestSummarize(t *testing.T) {
	decisions := []logging.Decision{
		{Timestamp: time.Unix(0, 0), Action: "allow", DurationMS: 10},
		{Timestamp: time.Unix(1, 0), Action: "block", DurationMS: 30, RateLimited: true, ClientIP: "1.1.1.1", MatchedRules: []logging.MatchedRule{{ID: "r1"}}},
		{Timestamp: time.Unix(2, 0), Action: "shadow", DurationMS: 20, ContractViolations: []logging.ContractViolation{{Type: "query_param_unexpected"}}},
	}

	summary := Summarize(decisions)
	if summary.Total != 3 {
		t.Fatalf("expected total 3, got %d", summary.Total)
	}
	if summary.Allowed != 1 || summary.Blocked != 1 || summary.Shadowed != 1 {
		t.Fatalf("unexpected action counts")
	}
	if summary.RateLimited != 1 {
		t.Fatalf("expected 1 rate limited, got %d", summary.RateLimited)
	}
	if len(summary.TopRules) != 1 || summary.TopRules[0].Key != "r1" {
		t.Fatalf("expected top rule r1")
	}
	if len(summary.TopContracts) != 1 || summary.TopContracts[0].Key != "query_param_unexpected" {
		t.Fatalf("expected top contract violation")
	}
}

func TestRenderJSON(t *testing.T) {
	_, err := RenderJSON(Summary{Total: 1})
	if err != nil {
		t.Fatalf("expected json render ok: %v", err)
	}
}
