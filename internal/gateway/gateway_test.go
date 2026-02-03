package gateway

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/klyr/klyr/internal/config"
)

func TestGatewayProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer backend.Close()

	gw, err := New(sampleConfig(backend.URL, 1024, 1024))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()

	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "ok" {
		t.Fatalf("expected body ok, got %q", string(body))
	}
}

func TestGatewayRejectsLargeBody(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	gw, err := New(sampleConfig(backend.URL, 4, 1024))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://example.com/", bytes.NewBufferString("hello"))
	rec := httptest.NewRecorder()

	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestGatewayRejectsLargeHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	gw, err := New(sampleConfig(backend.URL, 1024, 8))
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set("X-Test", "0123456789")
	rec := httptest.NewRecorder()

	gw.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("expected 431, got %d", rec.Code)
	}
}

func sampleConfig(upstreamURL string, maxBodyBytes, maxHeaderBytes int64) *config.Config {
	return &config.Config{
		Upstreams: []config.Upstream{
			{Name: "backend", URL: upstreamURL},
		},
		Routes: []config.Route{
			{
				Match:    config.RouteMatch{PathPrefix: "/"},
				Upstream: "backend",
				Policy:   "default",
			},
		},
		Policies: map[string]config.Policy{
			"default": {
				Mode:             config.ModeShadow,
				AnomalyThreshold: 0,
				Limits: config.Limits{
					MaxBodyBytes:   maxBodyBytes,
					MaxHeaderBytes: maxHeaderBytes,
					Timeout:        2 * time.Second,
				},
			},
		},
	}
}

func TestRateLimitKey(t *testing.T) {
	got := ratelimitKey("ip_path", "203.0.113.1", "/login")
	if got != "203.0.113.1|/login" {
		t.Fatalf("expected ip_path key, got %q", got)
	}

	got = ratelimitKey("ip", "203.0.113.1", "/login")
	if got != "203.0.113.1" {
		t.Fatalf("expected ip key, got %q", got)
	}
}

func TestHeadersForEvalRedactsSensitive(t *testing.T) {
	headers := http.Header{
		"Authorization": []string{"secret"},
		"Cookie":        []string{"a=b"},
		"X-Test":        []string{"ok"},
	}

	out := headersForEval(headers)
	if !strings.Contains(out, "Authorization: <redacted>") {
		t.Fatalf("expected authorization redacted, got %q", out)
	}
	if !strings.Contains(out, "Cookie: <redacted>") {
		t.Fatalf("expected cookie redacted, got %q", out)
	}
	if !strings.Contains(out, "X-Test: ok") {
		t.Fatalf("expected X-Test value, got %q", out)
	}
}

func TestBodyForEvalJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com/", strings.NewReader(`{"user":"alice","password":"secret","count":2}`))
	req.Header.Set("Content-Type", "application/json")
	body, _ := io.ReadAll(req.Body)

	out := bodyForEval(req, body)
	if !strings.Contains(out, "user=alice") {
		t.Fatalf("expected user field, got %q", out)
	}
	if strings.Contains(out, "password=secret") {
		t.Fatalf("expected password redacted, got %q", out)
	}
	if !strings.Contains(out, "password=<redacted>") {
		t.Fatalf("expected password redaction, got %q", out)
	}
}
