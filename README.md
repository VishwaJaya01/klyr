# Klyr

Klyr is a local-first security gateway (reverse proxy) written in Go. It provides request normalization, a deterministic rule engine with anomaly scoring (no AI), Learn → Enforce traffic contracts, and explainable decisions with built-in observability.

Klyr targets individuals and small teams who want practical protection and visibility without enterprise complexity.

**Status**: v0.1.0

## Features

- Reverse proxy routing by host and path prefix
- Deterministic request normalization (bounded decoding)
- Regex and Aho-Corasick rule matching
- Anomaly scoring with explainable block decisions
- Learn → Enforce traffic contracts (behavior allowlisting)
- In-memory rate limiting (token bucket)
- JSONL decision logs and Prometheus metrics

## Quickstart

Use the demo stack with Docker Compose:

```bash
make demo
```

Then follow `docs/quickstart.md` for learn → enforce, sample curls, and Grafana.

## CLI

- `klyr run -c <config>`
- `klyr learn -c <config> --duration 2m --out /state/contract.json`
- `klyr enforce -c <config> --contract /state/contract.json`
- `klyr report --in logs/decisions.jsonl --since 10m --format md --out report.md`
- `klyr validate -c <config>`
- `klyr version`

## Configuration

- Example config: `configs/klyr.example.yaml`
- Demo config: `demo/klyr.demo.yaml`
- Rule patterns: `rules/`

## Metrics

Prometheus metrics are available on `/metrics` and exposed via `metrics.listen`.

## Release

See `docs/release.md` for the tag and publish steps and a release notes template.

## Development

```bash
make build
make test
make lint
```

## License

MIT

## Future Work

See `docs/architecture.md` and `docs/threat-model.md`.
