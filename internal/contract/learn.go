package contract

import (
	"net/http"
	"strings"
)

func (c *Contract) Observe(req *http.Request, bodySize int64) {
	if req == nil {
		return
	}

	c.Samples++
	c.Methods[req.Method] = true

	if ct := parseContentType(req.Header.Get("Content-Type")); ct != "" {
		c.ContentTypes[ct] = true
	}

	for name := range req.URL.Query() {
		c.QueryParams[name] = true
	}

	for name := range req.Header {
		c.HeaderNames[http.CanonicalHeaderKey(name)] = true
	}

	if bodySize > c.ObservedMax {
		c.ObservedMax = bodySize
	}
}

func parseContentType(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ";")
	return strings.TrimSpace(strings.ToLower(parts[0]))
}
