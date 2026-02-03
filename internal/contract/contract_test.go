package contract

import (
	"net/http"
	"net/url"
	"testing"
)

func TestContractObserveAndFinalize(t *testing.T) {
	c := New("route-1", "default")

	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Path:     "/search",
			RawQuery: "q=test&debug=1",
		},
		Header: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
			"X-Test":       []string{"1"},
		},
	}

	c.Observe(req, 128)
	c.Observe(req, 64)

	if c.Samples != 2 {
		t.Fatalf("expected 2 samples, got %d", c.Samples)
	}
	if !c.Methods["POST"] {
		t.Fatalf("expected method POST")
	}
	if !c.ContentTypes["application/json"] {
		t.Fatalf("expected content-type application/json")
	}
	if !c.QueryParams["q"] || !c.QueryParams["debug"] {
		t.Fatalf("expected query params")
	}
	if !c.HeaderNames["Content-Type"] || !c.HeaderNames["X-Test"] {
		t.Fatalf("expected header names")
	}
	if c.ObservedMax != 128 {
		t.Fatalf("expected observed max 128, got %d", c.ObservedMax)
	}

	c.Finalize(32)
	if c.MaxBodyBytes != 160 {
		t.Fatalf("expected max body bytes 160, got %d", c.MaxBodyBytes)
	}
}
