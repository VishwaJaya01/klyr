package gateway

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/klyr/klyr/internal/config"
)

type Route struct {
	ID         string
	Host       string
	PathPrefix string
	Upstream   string
	Policy     string
}

type Router struct {
	routes []Route
}

func NewRouter(cfg *config.Config) (*Router, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	routes := make([]Route, 0, len(cfg.Routes))
	for i, route := range cfg.Routes {
		routes = append(routes, Route{
			ID:         fmt.Sprintf("route-%d", i),
			Host:       strings.ToLower(strings.TrimSpace(route.Match.Host)),
			PathPrefix: route.Match.PathPrefix,
			Upstream:   route.Upstream,
			Policy:     route.Policy,
		})
	}

	sort.SliceStable(routes, func(i, j int) bool {
		if len(routes[i].PathPrefix) == len(routes[j].PathPrefix) {
			return routes[i].ID < routes[j].ID
		}
		return len(routes[i].PathPrefix) > len(routes[j].PathPrefix)
	})

	return &Router{routes: routes}, nil
}

func (r *Router) Match(req *http.Request) (Route, bool) {
	if req == nil {
		return Route{}, false
	}

	host := strings.ToLower(stripPort(req.Host))
	path := req.URL.Path

	for _, route := range r.routes {
		if route.Host != "" && route.Host != host {
			continue
		}
		if strings.HasPrefix(path, route.PathPrefix) {
			return route, true
		}
	}

	return Route{}, false
}

func stripPort(hostport string) string {
	if hostport == "" {
		return ""
	}

	if host, _, err := net.SplitHostPort(hostport); err == nil {
		return host
	}

	return hostport
}
