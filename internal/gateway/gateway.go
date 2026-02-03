package gateway

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/klyr/klyr/internal/config"
	"github.com/klyr/klyr/internal/contract"
	"github.com/klyr/klyr/internal/logging"
	"github.com/klyr/klyr/internal/observability"
	"github.com/klyr/klyr/internal/policy"
	"github.com/klyr/klyr/internal/ratelimit"
	"github.com/klyr/klyr/internal/rules"
)

const defaultBodyMarginBytes = 1024

type Gateway struct {
	router    *Router
	upstreams map[string]*url.URL
	policies  map[string]config.Policy
	proxies   map[string]*httputil.ReverseProxy

	engine      *rules.Engine
	contracts   map[string]*contract.Contract
	decisionLog *logging.DecisionLogger
	metrics     *observability.Metrics
	limiter     *ratelimit.Limiter
	bodyRules   bool

	requestCount uint64
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
	for name, policyCfg := range cfg.Policies {
		policies[name] = policyCfg
	}

	engine, err := rules.BuildEngine(cfg, cfg.BaseDir())
	if err != nil {
		return nil, err
	}

	contracts := make(map[string]*contract.Contract)
	for i, route := range cfg.Routes {
		routeID := fmt.Sprintf("route-%d", i)
		policyCfg, ok := policies[route.Policy]
		if !ok {
			continue
		}
		if policyCfg.Mode == config.ModeLearn {
			contracts[contractKey(routeID, route.Policy)] = contract.New(routeID, route.Policy)
		}
		if policyCfg.Mode == config.ModeEnforce {
			path := cfg.ResolvePath(policyCfg.Contract.Path)
			loaded, err := contract.Load(path)
			if err != nil {
				return nil, fmt.Errorf("load contract for %s: %w", route.Policy, err)
			}
			contracts[contractKey(routeID, route.Policy)] = loaded
		}
	}

	return &Gateway{
		router:    router,
		upstreams: upstreams,
		policies:  policies,
		proxies:   proxies,
		engine:    engine,
		contracts: contracts,
		limiter:   ratelimit.NewLimiter(),
		bodyRules: hasBodyRules(engine),
	}, nil
}

func (g *Gateway) SetDecisionLogger(logger *logging.DecisionLogger) {
	g.decisionLog = logger
}

func (g *Gateway) SetMetrics(metrics *observability.Metrics) {
	g.metrics = metrics
}

func (g *Gateway) Contract(routeID, policyName string) *contract.Contract {
	if g == nil {
		return nil
	}
	return g.contracts[contractKey(routeID, policyName)]
}

func (g *Gateway) SaveContracts(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}

	for i, route := range cfg.Routes {
		routeID := fmt.Sprintf("route-%d", i)
		policyCfg, ok := cfg.Policies[route.Policy]
		if !ok {
			continue
		}
		if policyCfg.Mode != config.ModeLearn {
			continue
		}
		key := contractKey(routeID, route.Policy)
		c, ok := g.contracts[key]
		if !ok {
			continue
		}
		c.Finalize(defaultBodyMarginBytes)
		path := cfg.ResolvePath(policyCfg.Contract.Path)
		if err := contract.Save(path, c); err != nil {
			return err
		}
	}
	return nil
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route, policyCfg, proxy, ok := g.resolveRoute(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	start := time.Now()
	decision := logging.Decision{
		Timestamp: time.Now().UTC(),
		RequestID: g.newRequestID(),
		ClientIP:  clientIP(r),
		Host:      r.Host,
		Method:    r.Method,
		Path:      r.URL.Path,
		Query:     r.URL.RawQuery,
		RouteID:   route.ID,
		Policy:    route.Policy,
		Mode:      policyCfg.Mode,
		Threshold: policyCfg.AnomalyThreshold,
	}

	if exceedsHeaderLimit(r.Header, policyCfg.Limits.MaxHeaderBytes) {
		decision.Action = string(policy.ActionBlock)
		decision.StatusCode = http.StatusRequestHeaderFieldsTooLarge
		g.writeDecision(decision, start, 0, "rule", nil, nil, "")
		http.Error(w, "request headers too large", http.StatusRequestHeaderFieldsTooLarge)
		return
	}

	if policyCfg.Limits.MaxBodyBytes > 0 {
		if r.ContentLength > policyCfg.Limits.MaxBodyBytes {
			decision.Action = string(policy.ActionBlock)
			decision.StatusCode = http.StatusRequestEntityTooLarge
			g.writeDecision(decision, start, 0, "rule", nil, nil, "")
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, policyCfg.Limits.MaxBodyBytes)
	}

	ctx, cancel := context.WithTimeout(r.Context(), policyCfg.Limits.Timeout)
	defer cancel()

	body, bodySize, bodyErr := readBodyIfNeeded(r, policyCfg, g.bodyRules)
	if bodyErr != nil {
		decision.Action = string(policy.ActionBlock)
		decision.StatusCode = http.StatusRequestEntityTooLarge
		g.writeDecision(decision, start, 0, "rule", nil, nil, "")
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return
	}

	rlKey := ""
	ratelimitLabel := ""
	if policyCfg.RateLimit.Enabled {
		rlKey = ratelimitKey(policyCfg.RateLimit.Key, decision.ClientIP, r.URL.Path)
		ratelimitLabel = policyCfg.RateLimit.Key
		allowed := g.limiter.Allow(rlKey, policyCfg.RateLimit.RPS, policyCfg.RateLimit.Burst, time.Now())
		if !allowed {
			decision.RateLimited = true
			decision.Action = string(policy.ActionBlock)
			decision.StatusCode = rateLimitStatus(policyCfg.RateLimit.StatusCode)
			g.writeDecision(decision, start, 0, "ratelimit", nil, nil, ratelimitLabel)
			http.Error(w, "rate limit exceeded", decision.StatusCode)
			return
		}
	}

	evalCtx := buildEvalContext(r, body)
	result := policy.EvaluateRules(g.engine, evalCtx)
	decision.Score = result.Score
	decision.MatchedRules = mapMatches(result.Matches)

	contractViolations := g.checkContract(route.ID, route.Policy, policyCfg, r, bodySize)
	if len(contractViolations) > 0 {
		decision.ContractViolations = mapViolations(contractViolations)
		if policyCfg.Mode == config.ModeEnforce {
			decision.Action = string(policy.ActionBlock)
			decision.StatusCode = blockStatus(policyCfg)
			g.writeDecision(decision, start, 0, "contract", decision.MatchedRules, decision.ContractViolations, ratelimitLabel)
			http.Error(w, policyCfg.Actions.BlockBody, decision.StatusCode)
			return
		}
	}

	action, shouldBlock := policy.DecideAction(policyCfg.Mode, result.Score, policyCfg.AnomalyThreshold)
	decision.Action = string(action)
	if shouldBlock {
		decision.StatusCode = blockStatus(policyCfg)
		g.writeDecision(decision, start, 0, "rule", decision.MatchedRules, decision.ContractViolations, ratelimitLabel)
		http.Error(w, policyCfg.Actions.BlockBody, decision.StatusCode)
		return
	}

	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	req := r.WithContext(ctx)
	proxy.ServeHTTP(rec, req)
	decision.StatusCode = rec.status
	decision.UpstreamMS = time.Since(start).Milliseconds()
	g.writeDecision(decision, start, decision.UpstreamMS, "", decision.MatchedRules, decision.ContractViolations, ratelimitLabel)
}

func (g *Gateway) checkContract(routeID, policyName string, policyCfg config.Policy, r *http.Request, bodySize int64) []contract.Violation {
	key := contractKey(routeID, policyName)
	c, ok := g.contracts[key]
	if !ok {
		return nil
	}

	enforcement := parseEnforcement(policyCfg.Contract.Enforcement)

	switch policyCfg.Mode {
	case config.ModeLearn:
		c.Observe(r, bodySize)
		return nil
	case config.ModeEnforce:
		return contract.Evaluate(c, r, bodySize, enforcement)
	default:
		return nil
	}
}

func (g *Gateway) resolveRoute(r *http.Request) (Route, config.Policy, *httputil.ReverseProxy, bool) {
	route, ok := g.router.Match(r)
	if !ok {
		return Route{}, config.Policy{}, nil, false
	}

	policyCfg, ok := g.policies[route.Policy]
	if !ok {
		return Route{}, config.Policy{}, nil, false
	}
	proxy, ok := g.proxies[route.Upstream]
	if !ok {
		return Route{}, config.Policy{}, nil, false
	}

	return route, policyCfg, proxy, true
}

func (g *Gateway) writeDecision(decision logging.Decision, start time.Time, upstreamMS int64, reason string, matches []logging.MatchedRule, violations []logging.ContractViolation, ratelimitKey string) {
	decision.DurationMS = time.Since(start).Milliseconds()
	decision.UpstreamMS = upstreamMS
	if g.decisionLog != nil {
		_ = g.decisionLog.Write(decision)
	}
	if g.metrics != nil {
		g.metrics.Observe(decision, matches, violations, ratelimitKey, reason)
	}
}

func (g *Gateway) newRequestID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return hex.EncodeToString(buf[:])
	}
	value := atomic.AddUint64(&g.requestCount, 1)
	return fmt.Sprintf("req-%d", value)
}

func mapMatches(matches []rules.Match) []logging.MatchedRule {
	if len(matches) == 0 {
		return nil
	}
	out := make([]logging.MatchedRule, len(matches))
	for i, m := range matches {
		out[i] = logging.MatchedRule{
			ID:       m.RuleID,
			Phase:    string(m.Phase),
			Score:    m.Score,
			Tags:     append([]string(nil), m.Tags...),
			Evidence: redactSecrets(m.Evidence),
		}
	}
	return out
}

func mapViolations(violations []contract.Violation) []logging.ContractViolation {
	out := make([]logging.ContractViolation, len(violations))
	for i, v := range violations {
		out[i] = logging.ContractViolation{Type: v.Type, Field: v.Field}
	}
	return out
}

func buildEvalContext(r *http.Request, body []byte) rules.EvalContext {
	return rules.EvalContext{
		RequestLine: rules.Field{Raw: fmt.Sprintf("%s %s", r.Method, r.URL.Path)},
		Headers:     rules.Field{Raw: headersForEval(r.Header)},
		Query:       rules.Field{Raw: r.URL.RawQuery},
		Body:        rules.Field{Raw: bodyForEval(r, body)},
	}
}

func headersForEval(headers http.Header) string {
	var b strings.Builder
	for name, values := range headers {
		canon := http.CanonicalHeaderKey(name)
		if isSensitiveHeader(canon) {
			b.WriteString(canon)
			b.WriteString(": <redacted>\n")
			continue
		}
		for _, value := range values {
			b.WriteString(canon)
			b.WriteString(": ")
			b.WriteString(value)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func isSensitiveHeader(name string) bool {
	switch strings.ToLower(name) {
	case "authorization", "cookie", "set-cookie":
		return true
	default:
		return false
	}
}

var (
	secretKVPattern     = regexp.MustCompile(`(?i)\b(password|passwd|token|api[_-]?key|secret)\s*=\s*([^\s&]+)`) // key=value
	secretBearerPattern = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+\\-/]+=*`)
)

func redactSecrets(input string) string {
	if input == "" {
		return input
	}
	redacted := secretKVPattern.ReplaceAllString(input, `$1=<redacted>`)
	redacted = secretBearerPattern.ReplaceAllString(redacted, "bearer <redacted>")
	return redacted
}

func bodyForEval(r *http.Request, body []byte) string {
	if len(body) == 0 || r == nil {
		return ""
	}
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(ct, "application/json") {
		text := shallowJSON(body)
		if text != "" {
			return text
		}
	}
	return string(body)
}

func shallowJSON(body []byte) string {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return ""
	}

	obj, ok := value.(map[string]any)
	if !ok {
		return ""
	}

	const maxFields = 50
	var b strings.Builder
	count := 0
	for key, raw := range obj {
		if count >= maxFields {
			break
		}
		count++
		val := formatJSONValue(raw)
		if val == "" {
			continue
		}
		fmt.Fprintf(&b, "%s=%s ", key, redactSecrets(val))
	}
	return strings.TrimSpace(b.String())
}

func formatJSONValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func readBodyIfNeeded(r *http.Request, policyCfg config.Policy, hasBodyRules bool) ([]byte, int64, error) {
	if r.Body == nil {
		return nil, 0, nil
	}

	contentLength := r.ContentLength
	need := hasBodyRules || policyCfg.Mode == config.ModeLearn || policyCfg.Mode == config.ModeEnforce
	if contentLength >= 0 && !need {
		return nil, contentLength, nil
	}
	if contentLength >= 0 && contentLength == 0 {
		return nil, 0, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, 0, err
	}
	if policyCfg.Limits.MaxBodyBytes > 0 && int64(len(body)) > policyCfg.Limits.MaxBodyBytes {
		return nil, int64(len(body)), errors.New("body exceeds limit")
	}

	r.Body = io.NopCloser(bytes.NewReader(body))
	r.ContentLength = int64(len(body))

	return body, int64(len(body)), nil
}

func rateLimitStatus(code int) int {
	if code <= 0 {
		return http.StatusTooManyRequests
	}
	return code
}

func ratelimitKey(mode string, ip string, path string) string {
	switch mode {
	case string(ratelimit.KeyIPPath):
		return ip + "|" + path
	case string(ratelimit.KeyIP):
		fallthrough
	default:
		return ip
	}
}

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func parseEnforcement(level string) contract.Enforcement {
	switch strings.ToLower(level) {
	case string(contract.EnforcementModerate):
		return contract.EnforcementModerate
	case string(contract.EnforcementStrict):
		return contract.EnforcementStrict
	default:
		return contract.EnforcementLenient
	}
}

func blockStatus(policyCfg config.Policy) int {
	if policyCfg.Actions.BlockStatusCode > 0 {
		return policyCfg.Actions.BlockStatusCode
	}
	return http.StatusForbidden
}

func contractKey(routeID, policyName string) string {
	return routeID + "|" + policyName
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func hasBodyRules(engine *rules.Engine) bool {
	if engine == nil {
		return false
	}
	for _, rule := range engine.Rules {
		if rule.Phase == rules.PhaseBody {
			return true
		}
	}
	return false
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
	for _, policyCfg := range cfg.Policies {
		if policyCfg.Limits.Timeout > max {
			max = policyCfg.Limits.Timeout
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
