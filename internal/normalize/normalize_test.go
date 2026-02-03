package normalize

import "testing"

func TestApplyDecodeDepth(t *testing.T) {
	res := Apply("%252e%252e%252f", Options{MaxDecodeDepth: 2})
	if res.Normalized != "%2e%2e/" {
		t.Fatalf("expected partial decode, got %q", res.Normalized)
	}

	res = Apply("%252e%252e%252f", Options{MaxDecodeDepth: 3})
	if res.Normalized != "../" {
		t.Fatalf("expected full decode, got %q", res.Normalized)
	}
}

func TestApplyTransforms(t *testing.T) {
	res := Apply("%3CScRipT%3E", Options{MaxDecodeDepth: 1, Lowercase: true})
	if res.Normalized != "<script>" {
		t.Fatalf("expected lowercase, got %q", res.Normalized)
	}

	res = Apply("&lt;div&gt;", Options{HTMLEntity: true})
	if res.Normalized != "<div>" {
		t.Fatalf("expected html decode, got %q", res.Normalized)
	}
}

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{
		"/a//b/./c":  "/a/b/c",
		"/a/b/../c":  "/a/c",
		"../a/../b":  "b",
		"/../a":      "/a",
		"/a/b/":      "/a/b/",
		"":           "/",
		"/":          "/",
		"/a/../../b": "/b",
	}

	for input, expected := range cases {
		got := NormalizePath(input)
		if got != expected {
			t.Fatalf("NormalizePath(%q) expected %q, got %q", input, expected, got)
		}
	}
}
