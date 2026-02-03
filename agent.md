# AGENT.MD — Klyr (v0.1) — Codex Agent Instructions

This file is the single source of truth for how to build **Klyr**. Follow it exactly.

Klyr is an open-source, **local-first security gateway** (reverse proxy) written in **Go**. It combines:

- high-performance reverse proxying
- request normalization
- a policy/rule engine with anomaly scoring (no AI)
- **Learn → Enforce** traffic contracts (behavior allowlisting)
- explainable decisions (forensics/why-blocked)
- a built-in verification harness ("Blitz") and observability (Prometheus/Grafana)

Klyr is **not** trying to beat enterprise WAFs. It is a professional-grade OSS tool aimed at individuals and small teams who want practical protection and visibility without heavy complexity.

---

## 0) Hard Constraints (Do Not Violate)

1. **No AI features.** No ML, no anomaly models beyond simple statistics/thresholds.
2. **No unsafe payload generation.** Blitz may include common benign test strings for SQLi/XSS/traversal, but must not contain malware, exploit code, or instructions for illegal activity.
3. **Deterministic builds and tests.** No network calls in unit tests.
4. **Keep dependencies minimal.** Prefer Go stdlib. Use small, well-known libs only when needed.
5. **Streaming & limits.** Never read unbounded request bodies into memory. Always enforce configured limits.
6. **Explainability is a first-class feature.** Every block decision must include reasons (rules and/or contract violations).
7. **v0.1 scope is strict.** Don't add "enterprise" components (Redis/K8s/control-plane/WASM) unless explicitly in scope.

---

## 1) v0.1 Goals (Definition of Done)

### Gateway Core

- Reverse proxy with routing:
  - host match (optional) + pathPrefix → upstream URL
- Timeouts and limits:
  - max header bytes, max body bytes, upstream timeout
- Request normalization pipeline (minimal):
  - URL decode (bounded depth)
  - path normalize (collapse `//`, resolve `.` and `..` safely)
  - lowercase transform (rule-controlled)
  - HTML entity decode (rule-controlled)
- Policy engine:
  - phases: `request_line`, `headers`, `query`, `body`
  - match types: `aho` (pattern list) and `regex`
  - anomaly scoring with per-policy threshold
  - actions: `allow`, `block`, `shadow` (log-only)
- Rate limiting (in-memory token bucket):
  - keys: `ip` and `ip_path`
  - action: 429 (default) or block, configurable

### Learn → Enforce Contracts (Signature Innovation)

- Learn mode builds contract per route/policy:
  - allowed methods
  - observed content-types
  - observed query param names
  - observed header names (presence-only; do not store sensitive values)
  - max observed body size + margin
  - min sample threshold
- Enforce mode blocks contract violations with strictness levels:
  - `lenient`, `moderate`, `strict`
- Contract stored as JSON on disk and reloadable.

### Explainability & Observability

- Decision log: JSONL line per request with:
  - request_id, route_id, policy, action, status_code
  - score/threshold
  - matched rules (ids, phase, tag, score)
  - contract violations (type + field)
  - timing (total duration, upstream duration)
- Prometheus metrics:
  - requests, blocks, rule matches, contract violations, rate-limit hits, latency histogram
- Provide a starter Grafana dashboard JSON.

### CLI (v0.1)

- `klyr run`
- `klyr learn`
- `klyr enforce`
- `klyr report`
- `klyr validate`
- `klyr version`

### Demo (v0.1)

- Docker Compose demo with:
  - demo app (included in repo)
  - klyr gateway
  - prometheus
  - grafana
- One documented demo script to show:
  - learn → enforce
  - basic SQLi/XSS blocks
  - rate limiting
  - metrics dashboard + report output

---

## 2) Non-Goals (Out of Scope for v0.1)

- Distributed rate limiting (Redis)
- Kubernetes / Helm / Ingress controller packaging
- External control plane, signed policy bundles, staged rollout
- Full OWASP CRS / ModSecurity compatibility
- Response-body inspection
- WASM plugin system
- GeoIP / ASN intelligence

If asked to implement any of the above, stop and request a scope change.

---

## 3) Final Tech Stack

### Language

- Go 1.22+

### Dependencies (preferred)

- CLI: `spf13/cobra` (or `urfave/cli/v2`, choose one; default to Cobra)
- YAML config: `gopkg.in/yaml.v3`
- Logging: `rs/zerolog` (preferred) or `uber-go/zap` (pick one; default to zerolog)
- Prometheus: `prometheus/client_golang`
- (Optional) Aho-Corasick:
  - Prefer implementing a small Aho automaton in `internal/policy/rules/aho` if feasible
  - Otherwise use a tiny library with minimal API surface (keep it isolated)

### Tooling

- Lint: `golangci-lint`
- Release: `goreleaser` (later; stub config ok in v0.1)
- Demo: Docker + Compose

---

## 4) Repository Layout (Must Match)

```
klyr/
  cmd/klyr/main.go

  internal/
    gateway/
    policy/
    normalize/
    rules/
    contract/
    ratelimit/
    observability/
    logging/
    report/

  configs/
    klyr.example.yaml
    schema.json
    profiles/
      login_api.yaml
      webhook_receiver.yaml
      admin_dashboard.yaml

  rules/
    sqli.txt
    xss.txt
    traversal.txt

  demo/
    compose.yaml
    app/
      Dockerfile
      main.go
    prometheus/
      prometheus.yml
    grafana/
      dashboards/
        klyr-dashboard.json
      provisioning/

  docs/
    quickstart.md
    threat-model.md
    architecture.md

  .github/workflows/ci.yml

  Makefile
  README.md
  go.mod
  go.sum
```

Also add:

- `LICENSE` (MIT or Apache-2.0; default to MIT)
- `.gitignore` for `logs/`, `state/`, `tmp/`

---

## 5) Config Schema (v0.1) — Source of Truth

### YAML fields (supported in v0.1)

Top-level:

- `configVersion: 1`
- `server.listen: ":8443"`
- `server.tls.enabled: bool`
- `server.tls.certFile`, `server.tls.keyFile` (required if tls.enabled)
- `upstreams[]: { name, url }`
- `routes[]: { match: { host?, pathPrefix }, upstream, policy }`
- `policies.<name>`:
  - `mode: learn|enforce|shadow`
  - `anomalyThreshold: int`
  - `limits: { maxBodyBytes, maxHeaderBytes, timeout }`
  - `contract: { path, learnWindow, minSamples, enforcement }`
  - `rateLimit: { enabled, key, rps, burst, statusCode }`
  - `actions: { blockStatusCode, blockBody }`
- `rules[]`:
  - `id`, `phase`, `score`, `tags`, `transforms`
  - `match`:
    - `type: aho|regex`
    - `pattern` (regex)
    - `patternsFile` (aho)
- `logging: { level, format, decisionLog }`
- `metrics: { enabled, listen }`

### Validation rules

- `configVersion` must be 1
- All referenced upstreams/policies must exist
- `listen` and `metrics.listen` must be valid address strings
- `maxBodyBytes`, `maxHeaderBytes`, `timeout` must be >0
- `anomalyThreshold` must be >=0
- Rule IDs must be unique
- If `match.type=aho`, `patternsFile` must exist
- If `match.type=regex`, `pattern` must compile
- Contract path must be writable on learn, readable on enforce

`klyr validate` enforces all of this.

---

## 6) CLI Behavior (Exact Expectations)

### `klyr run -c <config>`

- Starts gateway in configured mode
- `--mode` overrides all policy modes
- `--contract` overrides policy contract path (for default policy or all policies)
- writes structured logs to stdout; decision log to configured file
- exits non-zero on invalid config

### `klyr learn -c <config> --duration 2m --out state/contract.json`

- Runs gateway in learn mode for the duration
- Emits a contract JSON to `--out`
- After duration, stops automatically with exit code 0 if contract meets minSamples, else non-zero.

### `klyr enforce -c <config> --contract state/contract.json`

- Runs gateway in enforce mode using provided contract
- If contract missing/invalid → fail fast

### `klyr report --in logs/decisions.jsonl --since 10m --format md --out report.md`

- Reads decision log and produces summary:
  - totals allowed/blocked/shadowed
  - top blocked rules (count)
  - top contract violations
  - top rate-limited IPs/paths
  - latency p50/p95/p99 (estimate ok using histogram buckets)
- Support `--format text|md|json`

### `klyr version`

- Prints version, commit, build date (set via ldflags)

---

## 7) Decision Log Format (JSONL)

One JSON object per request (append-only). Example:

```json
{
  "ts":"2026-02-03T10:00:00Z",
  "request_id":"01J0...",
  "client_ip":"203.0.113.10",
  "host":"localhost",
  "method":"GET",
  "path":"/search",
  "query":"q=%27%20or%201%3D1--",
  "route_id":"route-0",
  "policy":"default",
  "mode":"enforce",
  "score":9,
  "threshold":8,
  "action":"block",
  "status_code":403,
  "matched_rules":[
    {"id":"sqli-aho-basic","phase":"query","score":5,"tags":["sqli"],"evidence":"or 1=1"}
  ],
  "contract_violations":[
    {"type":"query_param_unexpected","field":"debug"}
  ],
  "rate_limited":false,
  "duration_ms":12,
  "upstream_ms":0
}
```

Rules for logging:

- Never log sensitive values (Authorization headers, cookies). Log presence only.
- For evidence strings, include only small redacted snippets (max 64 chars) and never secrets.

---

## 8) Prometheus Metrics (v0.1)

Expose at /metrics on metrics.listen.

Required metrics (names fixed):

- `klyr_requests_total{route,policy,action,code}`
- `klyr_blocks_total{route,policy,reason}` (reason: rule, contract, ratelimit)
- `klyr_rule_matches_total{rule_id,tag,phase}`
- `klyr_contract_violations_total{route,policy,type}`
- `klyr_ratelimit_hits_total{route,policy,key}`
- `klyr_request_duration_seconds_bucket{route,policy,le}` (Histogram)
- `klyr_request_duration_seconds_sum{route,policy}`
- `klyr_request_duration_seconds_count{route,policy}`

Keep label cardinality low:

- Route should be route_id or a short name, not raw path
- Do not label by IP or user agent

---

## 9) Normalization Requirements (v0.1)

Normalization must be deterministic and bounded:

- URL decode: apply up to maxDecodeDepth (default 2). Stop early if no changes.
- Path normalize: safely resolve `.` and `..` without allowing path escape anomalies.
- Lowercase: only when rule requests it.
- HTML entity decode: only when rule requests it.

Store both raw and normalized values in evaluation context; do not mutate original request unless necessary.

---

## 10) Rule Engine Requirements (v0.1)

Phases:

- `request_line`: method + path (no body)
- `headers`: header names and (limited) values (never auth/cookie values)
- `query`: decoded query string
- `body`: read up to maxBodyBytes only; for JSON content-type parse shallowly if feasible, otherwise treat as bytes/string

Matching:

- AHO: pattern list from file; case-handling via transforms
- REGEX: compile once at startup; Go regex is safe

Scoring:

- For each match, add rule.score to request score
- If score >= threshold → block (unless shadow mode)

Actions:

- shadow: never block, only log and metrics as if it would block

---

## 11) Demo Setup (Must Work)

### Docker Compose endpoints

- Klyr: http://localhost:8443 (proxied)
- Demo app direct: http://localhost:8080 (bypass)
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin in demo only)

### Demo app (safe "intentionally vulnerable")

Implement minimal endpoints:

- GET / returns ok
- GET /search?q= echoes input (simulated)
- POST /comment accepts JSON {"text":""} and echoes (simulated)
- GET /login dummy endpoint for rate-limit demo

No real DB. No real injection. Just echo to demonstrate detection.

### Demo script (docs/quickstart.md)

Must show:

- run compose
- learn for 2 minutes (or shorter for demo)
- enforce using generated contract
- run curl examples for SQLi/XSS and show block + decision log
- open Grafana dashboard

---

## 12) Development Workflow (Commands)

Create a Makefile that supports:

- `make build` → go build ./cmd/klyr
- `make test` → go test ./...
- `make lint` → golangci-lint run
- `make fmt` → gofmt -w + goimports if used
- `make demo` → docker compose -f demo/compose.yaml up --build
- `make clean` → remove ./bin, ./logs, ./state

CI (GitHub Actions):

- go test
- gofmt check
- golangci-lint
- build klyr binary

---

## 13) Coding Standards (Go)

- Always use context.Context for request-handling and upstream calls.
- Wrap errors with %w and include actionable context.
- Keep packages small; avoid circular dependencies.
- Use interfaces where it enables testing, not everywhere.
- Avoid global mutable state. Use dependency injection via constructors.
- Do not panic in normal code paths.

---

## 14) Security & Privacy Rules

Never log secrets:

- Authorization headers, Cookie header values, Set-Cookie values, tokens

Rate limiting and contract enforcement must not store IPs permanently (only metrics counts).

Decision log may store client IP in demo mode, but allow a config flag later; for v0.1 keep it but document it.

Ensure safe default timeouts and limits.

Ensure deny behavior is explicit and explainable.

---

## 15) Implementation Plan (Order)

Follow this order for best results:

1. Bootstrap repo, CLI skeleton, config parsing + validate
2. Gateway reverse proxy core + routing
3. Normalization module + evaluation context
4. Rules engine (regex first, then AHO)
5. Scoring + action handling + decision logging
6. Rate limiting (in-memory token bucket)
7. Contract learn module + persistence
8. Enforce contract violations + strictness
9. Prometheus metrics + dashboard JSON
10. Demo compose + demo app + quickstart doc
11. klyr report implementation
12. CI workflow + polish

Do not skip validate/tests/docs.

---

## 16) Release & Versioning (v0.x)

Use semantic versioning.

Start at v0.1.0.

v0.x allows breaking config changes, but:

- if config changes, bump minor and clearly document in CHANGELOG.

Add:

- CHANGELOG.md with entries per release
- klyr version must print version from ldflags

---

## 17) What "Done" Means for v0.1

v0.1 is done only when all are true:

- klyr validate works on example config
- klyr run proxies demo traffic
- Learn generates a contract JSON
- Enforce blocks contract violations
- Signature rules block sample SQLi/XSS strings
- Rate limiting triggers on repeated login requests
- Decision log contains explainable reasons
- Prometheus metrics exposed and scraped by demo Prometheus
- Grafana dashboard loads and shows meaningful panels
- klyr report generates a readable summary
- docker compose -f demo/compose.yaml up --build works end-to-end
- README quickstart reproduces the above without missing steps

---

## 18) Communication Style (for PRs/commits)

Commit messages:

- feat: ...
- fix: ...
- docs: ...
- chore: ...
- test: ...

PRs must include:

- what changed
- how to run
- how to verify

---

## 19) If Anything Is Ambiguous

Prefer the simplest interpretation that preserves:

- scope
- safety
- determinism
- explainability

Avoid adding new components; instead document TODOs under a "Future work" section.

---

**End of AGENT.MD**
