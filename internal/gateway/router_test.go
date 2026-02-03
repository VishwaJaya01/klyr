package gateway

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/klyr/klyr/internal/config"
)

func TestRouterMatchLongestPrefix(t *testing.T) {
	cfg := &config.Config{
		Routes: []config.Route{
			{Match: config.RouteMatch{PathPrefix: "/api"}},
			{Match: config.RouteMatch{PathPrefix: "/api/v1"}},
		},
	}

	router, err := NewRouter(cfg)
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}

	req := &http.Request{URL: &url.URL{Path: "/api/v1/users"}, Host: "example.com"}
	route, ok := router.Match(req)
	if !ok {
		t.Fatal("expected route match")
	}
	if route.PathPrefix != "/api/v1" {
		t.Fatalf("expected /api/v1, got %q", route.PathPrefix)
	}
}

func TestRouterMatchHost(t *testing.T) {
	cfg := &config.Config{
		Routes: []config.Route{
			{Match: config.RouteMatch{Host: "example.com", PathPrefix: "/"}},
			{Match: config.RouteMatch{Host: "", PathPrefix: "/"}},
		},
	}

	router, err := NewRouter(cfg)
	if err != nil {
		t.Fatalf("NewRouter error: %v", err)
	}

	req := &http.Request{URL: &url.URL{Path: "/"}, Host: "example.com:8443"}
	route, ok := router.Match(req)
	if !ok {
		t.Fatal("expected route match")
	}
	if route.Host != "example.com" {
		t.Fatalf("expected host match example.com, got %q", route.Host)
	}
}
