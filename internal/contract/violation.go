package contract

import "net/http"

type Violation struct {
	Type  string `json:"type"`
	Field string `json:"field"`
}

func Evaluate(c *Contract, req *http.Request, bodySize int64, enforcement Enforcement) []Violation {
	if c == nil || req == nil {
		return nil
	}

	var violations []Violation

	if len(c.Methods) > 0 && !c.Methods[req.Method] {
		violations = append(violations, Violation{Type: "method_unexpected", Field: req.Method})
	}

	if len(c.ContentTypes) > 0 {
		if ct := parseContentType(req.Header.Get("Content-Type")); ct != "" {
			if !c.ContentTypes[ct] {
				violations = append(violations, Violation{Type: "content_type_unexpected", Field: ct})
			}
		}
	}

	if enforcement >= EnforcementModerate {
		for name := range req.URL.Query() {
			if !c.QueryParams[name] {
				violations = append(violations, Violation{Type: "query_param_unexpected", Field: name})
			}
		}
	}

	if enforcement >= EnforcementStrict {
		for name := range req.Header {
			canon := http.CanonicalHeaderKey(name)
			if !c.HeaderNames[canon] {
				violations = append(violations, Violation{Type: "header_unexpected", Field: canon})
			}
		}
	}

	if c.MaxBodyBytes > 0 && bodySize > c.MaxBodyBytes {
		violations = append(violations, Violation{Type: "body_size_exceeded", Field: "body"})
	}

	return violations
}
