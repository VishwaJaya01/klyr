package normalize

import (
	"html"
	"net/url"
	"strings"
)

type Options struct {
	MaxDecodeDepth int
	Lowercase      bool
	HTMLEntity     bool
	NormalizePath  bool
}

type Result struct {
	Raw        string
	Normalized string
}

func Apply(input string, opts Options) Result {
	res := Result{Raw: input, Normalized: input}

	depth := opts.MaxDecodeDepth
	if depth <= 0 {
		depth = 2
	}

	decoded := res.Normalized
	for i := 0; i < depth; i++ {
		next, ok := decodeOnce(decoded)
		if !ok || next == decoded {
			break
		}
		decoded = next
	}

	res.Normalized = decoded

	if opts.NormalizePath {
		res.Normalized = NormalizePath(res.Normalized)
	}
	if opts.HTMLEntity {
		res.Normalized = html.UnescapeString(res.Normalized)
	}
	if opts.Lowercase {
		res.Normalized = strings.ToLower(res.Normalized)
	}

	return res
}

func decodeOnce(input string) (string, bool) {
	decoded, err := url.PathUnescape(input)
	if err != nil {
		return input, false
	}
	return decoded, true
}
