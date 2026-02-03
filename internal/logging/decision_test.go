package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestDecisionLoggerWritesJSONL(t *testing.T) {
	var buf bytes.Buffer
	logger := NewDecisionLogger(&buf)

	decision := Decision{
		Timestamp: time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC),
		RequestID: "req-1",
		Action:    "block",
		MatchedRules: []MatchedRule{{
			ID:       "rule-1",
			Phase:    "query",
			Score:    5,
			Tags:     []string{"sqli"},
			Evidence: strings.Repeat("a", 100),
		}},
	}

	if err := logger.Write(decision); err != nil {
		t.Fatalf("Write error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var parsed Decision
	if err := json.Unmarshal([]byte(lines[0]), &parsed); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if len(parsed.MatchedRules) != 1 {
		t.Fatalf("expected 1 matched rule, got %d", len(parsed.MatchedRules))
	}
	if len(parsed.MatchedRules[0].Evidence) != maxEvidence {
		t.Fatalf("expected evidence length %d, got %d", maxEvidence, len(parsed.MatchedRules[0].Evidence))
	}
}
