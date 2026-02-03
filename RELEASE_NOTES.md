## v0.1.0

### Added
- Gateway reverse proxy with routing by host and path prefix
- Deterministic normalization (bounded URL decode, path normalize, optional lowercase/entity)
- Rule engine with regex and Aho-Corasick matching and anomaly scoring
- Learn → Enforce traffic contracts with strictness levels
- In-memory rate limiting (token bucket)
- Decision logs (JSONL) with explainable reasons
- Prometheus metrics + starter Grafana dashboard
- Demo environment (docker compose, demo app, Prometheus, Grafana)
- CLI: run, learn, enforce, report, validate, version
- CI workflow: tests, gofmt, golangci-lint, build

### Fixed
- Secret redaction for headers and JSON body evidence
- Improved CI module download step to avoid missing go.sum
