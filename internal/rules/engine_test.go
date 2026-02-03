package rules

import (
	"testing"

	"github.com/klyr/klyr/internal/policy"
)

func TestEngineRegexMatch(t *testing.T) {
	matcher, err := NewRegexMatcher("(?i)<script>")
	if err != nil {
		t.Fatalf("regex compile: %v", err)
	}

	engine := &Engine{Rules: []Rule{
		{
			ID:         "xss-1",
			Phase:      PhaseQuery,
			Score:      5,
			Tags:       []string{"xss"},
			Transforms: []Transform{TransformLowercase},
			Matcher:    matcher,
		},
	}}

	ctx := policy.EvalContext{
		Query: policy.Field{Raw: "%3CScRipT%3E"},
	}

	result := engine.Evaluate(ctx)
	if result.Score != 5 {
		t.Fatalf("expected score 5, got %d", result.Score)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result.Matches))
	}
	if result.Matches[0].RuleID != "xss-1" {
		t.Fatalf("unexpected match rule id %q", result.Matches[0].RuleID)
	}
}

func TestEngineAhoMatch(t *testing.T) {
	matcher, err := NewAhoMatcher([]string{"or 1=1"})
	if err != nil {
		t.Fatalf("aho build: %v", err)
	}

	engine := &Engine{Rules: []Rule{
		{
			ID:         "sqli-1",
			Phase:      PhaseQuery,
			Score:      3,
			Tags:       []string{"sqli"},
			Transforms: []Transform{TransformLowercase},
			Matcher:    matcher,
		},
	}}

	ctx := policy.EvalContext{
		Query: policy.Field{Raw: "q=1 Or 1=1"},
	}

	result := engine.Evaluate(ctx)
	if result.Score != 3 {
		t.Fatalf("expected score 3, got %d", result.Score)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(result.Matches))
	}
}
