# Architecture

Klyr is a local-first security gateway that sits in front of an upstream service and enforces deterministic traffic rules.

## Components

- **Gateway**: HTTP reverse proxy with routing by host/path, request size limits, and upstream timeouts.
- **Normalization**: Bounded URL decoding, path normalization, optional lowercase and HTML entity decoding.
- **Rules Engine**: Regex and Aho-Corasick matchers with anomaly scoring per policy.
- **Contracts (Learn → Enforce)**: Observes live traffic to build allowlisted behavior and enforces it with strictness levels.
- **Rate Limiting**: In-memory token bucket keyed by IP or IP+path.
- **Decision Logs**: JSONL records per request with explainable reasons.
- **Observability**: Prometheus metrics exposed on `/metrics` and a starter Grafana dashboard.

## Request Flow

1. **Route match** (host/pathPrefix → upstream + policy)
2. **Limits** (header + body max, timeout)
3. **Rate limit** (if enabled)
4. **Normalization + Rules** (scoring, optional block)
5. **Contract** (learn or enforce)
6. **Proxy** to upstream (if allowed)
7. **Decision log + metrics**

## Modes

- `learn`: build contracts, no blocking from contract violations.
- `enforce`: block contract violations and rules exceeding threshold.
- `shadow`: log-only for rule blocks.
