# Klyr

Klyr is a local-first security gateway (reverse proxy) written in Go. It provides request normalization, a deterministic rule engine with anomaly scoring (no AI), Learn → Enforce traffic contracts, and explainable decisions with built-in observability.

Klyr targets individuals and small teams who want practical protection and visibility without enterprise complexity.

**Status**: v0.1 in active development. Scope is intentionally small and focused.

**Key Features**
- High-performance reverse proxy with routing by host and path prefix
- Deterministic request normalization pipeline (bounded decoding)
- Rule engine with regex and Aho-Corasick matching
- Anomaly scoring with explainable block decisions
- Learn → Enforce traffic contracts (behavior allowlisting)
- Rate limiting (in-memory token bucket)
- Decision logs (JSONL) and Prometheus metrics

**Quickstart**
- See `docs/quickstart.md` for the full demo flow (Docker Compose, learn → enforce, sample curls, and Grafana).

**Configuration**
- Example config: `configs/klyr.example.yaml`
- Rule pattern files: `rules/`
- Profiles: `configs/profiles/`

**CLI (v0.1)**
- `klyr run`
- `klyr learn`
- `klyr enforce`
- `klyr report`
- `klyr validate`
- `klyr version`

**Observability**
- Prometheus metrics at `/metrics` (configurable listen address)
- Starter Grafana dashboard in `demo/grafana/dashboards/`

**Development**
- Build: `make build`
- Test: `make test`
- Lint: `make lint`
- Demo: `make demo`

**Security & Privacy**
- Klyr never logs secrets (Authorization/Cookie values)
- Decision logs include limited redacted evidence snippets

**License**
- MIT (see `LICENSE`)

**Future Work**
- See `docs/architecture.md` and `docs/threat-model.md` for scope and rationale. Future items will be tracked there.
