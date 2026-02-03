package contract

import (
	"net/http"
	"net/url"
	"testing"
)

func TestEvaluateContractStrictness(t *testing.T) {
	c := &Contract{
		Methods:      map[string]bool{"GET": true},
		ContentTypes: map[string]bool{"application/json": true},
		QueryParams:  map[string]bool{"q": true},
		HeaderNames:  map[string]bool{"X-Test": true},
		MaxBodyBytes: 10,
	}

	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Path:     "/search",
			RawQuery: "q=ok&debug=1",
		},
		Header: http.Header{
			"Content-Type": []string{"text/plain"},
			"X-Test":       []string{"1"},
			"X-Extra":      []string{"1"},
		},
	}

	violations := Evaluate(c, req, 20, EnforcementLenient)
	if len(violations) != 3 {
		t.Fatalf("lenient expected 3 violations, got %d", len(violations))
	}

	violations = Evaluate(c, req, 20, EnforcementModerate)
	if len(violations) != 4 {
		t.Fatalf("moderate expected 4 violations, got %d", len(violations))
	}

	violations = Evaluate(c, req, 20, EnforcementStrict)
	if len(violations) != 5 {
		t.Fatalf("strict expected 5 violations, got %d", len(violations))
	}
}
