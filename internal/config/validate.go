package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ValidationError struct {
	Problems []string
}

func (v *ValidationError) Add(format string, args ...any) {
	v.Problems = append(v.Problems, fmt.Sprintf(format, args...))
}

func (v *ValidationError) Error() string {
	return fmt.Sprintf("%d validation error(s)", len(v.Problems))
}

func (c *Config) Validate() error {
	v := &ValidationError{}

	if c.ConfigVersion != 1 {
		v.Add("configVersion must be 1")
	}

	if err := validateListen(c.Server.Listen); err != nil {
		v.Add("server.listen invalid: %v", err)
	}

	if c.Server.TLS.Enabled {
		if c.Server.TLS.CertFile == "" {
			v.Add("server.tls.certFile required when tls.enabled is true")
		}
		if c.Server.TLS.KeyFile == "" {
			v.Add("server.tls.keyFile required when tls.enabled is true")
		}
		if c.Server.TLS.CertFile != "" {
			if err := requireFile(c.resolvePath(c.Server.TLS.CertFile)); err != nil {
				v.Add("server.tls.certFile invalid: %v", err)
			}
		}
		if c.Server.TLS.KeyFile != "" {
			if err := requireFile(c.resolvePath(c.Server.TLS.KeyFile)); err != nil {
				v.Add("server.tls.keyFile invalid: %v", err)
			}
		}
	}

	if c.Metrics.Enabled {
		if err := validateListen(c.Metrics.Listen); err != nil {
			v.Add("metrics.listen invalid: %v", err)
		}
	}

	upstreamNames := map[string]struct{}{}
	for i, upstream := range c.Upstreams {
		if upstream.Name == "" {
			v.Add("upstreams[%d].name is required", i)
		} else if _, exists := upstreamNames[upstream.Name]; exists {
			v.Add("upstreams[%d].name %q is duplicated", i, upstream.Name)
		} else {
			upstreamNames[upstream.Name] = struct{}{}
		}

		if upstream.URL == "" {
			v.Add("upstreams[%d].url is required", i)
		} else if err := validateURL(upstream.URL); err != nil {
			v.Add("upstreams[%d].url invalid: %v", i, err)
		}
	}

	policyNames := map[string]struct{}{}
	for name, policy := range c.Policies {
		if name == "" {
			v.Add("policies has an empty name")
			continue
		}
		if _, exists := policyNames[name]; exists {
			v.Add("policies.%s is duplicated", name)
			continue
		}
		policyNames[name] = struct{}{}

		switch policy.Mode {
		case ModeLearn, ModeEnforce, ModeShadow:
		default:
			v.Add("policies.%s.mode must be learn|enforce|shadow", name)
		}

		if policy.AnomalyThreshold < 0 {
			v.Add("policies.%s.anomalyThreshold must be >= 0", name)
		}

		if policy.Limits.MaxBodyBytes <= 0 {
			v.Add("policies.%s.limits.maxBodyBytes must be > 0", name)
		}
		if policy.Limits.MaxHeaderBytes <= 0 {
			v.Add("policies.%s.limits.maxHeaderBytes must be > 0", name)
		}
		if policy.Limits.Timeout <= 0 {
			v.Add("policies.%s.limits.timeout must be > 0", name)
		}

		if policy.Contract.Path == "" {
			v.Add("policies.%s.contract.path is required", name)
		} else {
			if err := c.validateContractPath(policy.Mode, policy.Contract.Path); err != nil {
				v.Add("policies.%s.contract.path invalid: %v", name, err)
			}
		}

		if policy.RateLimit.Enabled {
			if policy.RateLimit.RPS <= 0 {
				v.Add("policies.%s.rateLimit.rps must be > 0", name)
			}
			if policy.RateLimit.Burst <= 0 {
				v.Add("policies.%s.rateLimit.burst must be > 0", name)
			}
		}
	}

	for i, route := range c.Routes {
		if route.Match.PathPrefix == "" {
			v.Add("routes[%d].match.pathPrefix is required", i)
		}
		if route.Upstream == "" {
			v.Add("routes[%d].upstream is required", i)
		} else if _, exists := upstreamNames[route.Upstream]; !exists {
			v.Add("routes[%d].upstream %q does not exist", i, route.Upstream)
		}
		if route.Policy == "" {
			v.Add("routes[%d].policy is required", i)
		} else if _, exists := policyNames[route.Policy]; !exists {
			v.Add("routes[%d].policy %q does not exist", i, route.Policy)
		}
	}

	ruleIDs := map[string]struct{}{}
	for i, rule := range c.Rules {
		if rule.ID == "" {
			v.Add("rules[%d].id is required", i)
		} else if _, exists := ruleIDs[rule.ID]; exists {
			v.Add("rules[%d].id %q is duplicated", i, rule.ID)
		} else {
			ruleIDs[rule.ID] = struct{}{}
		}

		if rule.Match.Type == "" {
			v.Add("rules[%d].match.type is required", i)
		}

		switch rule.Match.Type {
		case "aho":
			if rule.Match.PatternsFile == "" {
				v.Add("rules[%d].match.patternsFile is required for aho", i)
			} else if err := requireFile(c.resolvePath(rule.Match.PatternsFile)); err != nil {
				v.Add("rules[%d].match.patternsFile invalid: %v", i, err)
			}
		case "regex":
			if rule.Match.Pattern == "" {
				v.Add("rules[%d].match.pattern is required for regex", i)
			} else if _, err := regexp.Compile(rule.Match.Pattern); err != nil {
				v.Add("rules[%d].match.pattern invalid: %v", i, err)
			}
		default:
			v.Add("rules[%d].match.type must be aho|regex", i)
		}
	}

	if len(v.Problems) > 0 {
		sort.Strings(v.Problems)
		return v
	}
	return nil
}

func validateListen(addr string) error {
	if strings.TrimSpace(addr) == "" {
		return errors.New("address is required")
	}
	if _, err := net.ResolveTCPAddr("tcp", addr); err != nil {
		return err
	}
	return nil
}

func validateURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("must include scheme and host")
	}
	return nil
}

func requireFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}

func (c *Config) validateContractPath(mode, path string) error {
	resolved := c.resolvePath(path)

	switch mode {
	case ModeLearn:
		return ensureWritable(resolved)
	case ModeEnforce:
		return ensureReadable(resolved)
	case ModeShadow:
		return nil
	default:
		return fmt.Errorf("unknown mode %q", mode)
	}
}

func ensureReadable(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	return file.Close()
}

func ensureWritable(path string) error {
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	file, err := os.CreateTemp(dir, "klyr-validate-*")
	if err != nil {
		return err
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		return err
	}
	return os.Remove(name)
}
