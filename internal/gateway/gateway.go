package gateway

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/klyr/klyr/internal/config"
)

type Gateway struct {
	router    *Router
	upstreams map[string]*url.URL
	policies  map[string]config.Policy
	proxies   map[string]*httputil.ReverseProxy
}

func New(cfg *config.Config) (*Gateway, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	router, err := NewRouter(cfg)
	if err != nil {
		return nil, err
	}

	upstreams := make(map[string]*url.URL, len(cfg.Upstreams))
	for _, upstream := range cfg.Upstreams {
		parsed, err := url.Parse(upstream.URL)
		if err != nil {
			return nil, fmt.Errorf("parse upstream %s: %w", upstream.Name, err)
		}
		upstreams[upstream.Name] = parsed
	}

	maxTimeout := maxPolicyTimeout(cfg)
	transport := newTransport(maxTimeout)

	proxies := make(map[string]*httputil.ReverseProxy, len(upstreams))
	for name, target := range upstreams {
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Transport = transport
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			var maxErr *http.MaxBytesError
			switch {
			case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
				http.Error(w, "upstream timeout", http.StatusGatewayTimeout)
			case errors.As(err, &maxErr):
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			default:
				http.Error(w, "upstream error", http.StatusBadGateway)
			}
		}
		proxies[name] = proxy
	}

	policies := make(map[string]config.Policy, len(cfg.Policies))
	for name, policy := range cfg.Policies {
		policies[name] = policy
	}

	return &Gateway{
		router:    router,
		upstreams: upstreams,
		policies:  policies,
		proxies:   proxies,
	}, nil
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, policy, proxy, ok := g.resolveRoute(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if exceedsHeaderLimit(r.Header, policy.Limits.MaxHeaderBytes) {
		http.Error(w, "request headers too large", http.StatusRequestHeaderFieldsTooLarge)
		return
	}

	if policy.Limits.MaxBodyBytes > 0 {
		if r.ContentLength > policy.Limits.MaxBodyBytes {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, policy.Limits.MaxBodyBytes)
	}

	ctx, cancel := context.WithTimeout(r.Context(), policy.Limits.Timeout)
	defer cancel()

	req := r.WithContext(ctx)
	proxy.ServeHTTP(w, req)
}

func (g *Gateway) resolveRoute(r *http.Request) (Route, config.Policy, *httputil.ReverseProxy, bool) {
	route, ok := g.router.Match(r)
	if !ok {
		return Route{}, config.Policy{}, nil, false
	}

	policy, ok := g.policies[route.Policy]
	if !ok {
		return Route{}, config.Policy{}, nil, false
	}
	proxy, ok := g.proxies[route.Upstream]
	if !ok {
		return Route{}, config.Policy{}, nil, false
	}

	return route, policy, proxy, true
}

func exceedsHeaderLimit(headers http.Header, maxBytes int64) bool {
	if maxBytes <= 0 {
		return false
	}

	var total int64
	for name, values := range headers {
		for _, value := range values {
			total += int64(len(name) + len(value) + 2)
			if total > maxBytes {
				return true
			}
		}
	}

	return total > maxBytes
}

func maxPolicyTimeout(cfg *config.Config) time.Duration {
	var max time.Duration
	for _, policy := range cfg.Policies {
		if policy.Limits.Timeout > max {
			max = policy.Limits.Timeout
		}
	}
	if max <= 0 {
		max = 5 * time.Second
	}
	return max
}

func newTransport(timeout time.Duration) *http.Transport {
	dialer := &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   timeout,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: timeout,
	}
}
