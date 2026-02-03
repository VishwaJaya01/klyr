# Quickstart

This quickstart uses the built-in Docker Compose demo to show learn → enforce, signature blocks, rate limiting, metrics, and reporting.

## Prerequisites

- Docker + Docker Compose
- Go 1.22+ (for local dev only)

## Demo: Learn → Enforce

1) Start the demo stack:

```bash
make demo
```

2) In a new terminal, run learn mode for 2 minutes. If you are running the binary on your host:

```bash
./bin/klyr learn -c demo/klyr.demo.yaml --duration 2m --out ./state/contract.json
```

If you are running from source, build first:

```bash
make build
```

If you want to run the command inside the container instead:

```bash
docker compose -f demo/compose.yaml exec klyr /bin/klyr learn -c /config/klyr.yaml --duration 2m --out /state/contract.json
```

3) Enforce using the generated contract on the host:

```bash
./bin/klyr enforce -c demo/klyr.demo.yaml --contract ./state/contract.json
```

Or inside the container:

```bash
docker compose -f demo/compose.yaml exec klyr /bin/klyr enforce -c /config/klyr.yaml --contract /state/contract.json
```

4) Send traffic through Klyr:

```bash
# Normal request
curl -i "http://localhost:8443/search?q=hello"

# SQLi-like test (should block)
curl -i "http://localhost:8443/search?q=1%20or%201%3D1"

# XSS-like test (should block)
curl -i "http://localhost:8443/search?q=%3Cscript%3Ealert(1)%3C/script%3E"

# Rate limit demo (hit /login repeatedly)
for i in {1..10}; do curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8443/login; done
```

5) Open Grafana:

- http://localhost:3000 (admin/admin)
- Dashboard: "Klyr (starter)"

## Decision Logs

Decision logs are written to `./logs/decisions.jsonl` on the host (mounted into the container). Each line is a JSON object.

## Troubleshooting

- **`gofmt` fails in CI**: run `gofmt -w .` locally and commit.
- **`go test` fails with missing `go.sum`**: run `go mod tidy`, commit `go.sum`.
- **`golangci-lint` fails**: ensure you have the same golangci-lint version as CI and rerun `golangci-lint run`.
- **Contracts not written**: ensure `state/` exists or the container has write access to `/state`.
- **No metrics**: check `metrics.enabled: true` and `metrics.listen` in config; ensure port `9095` is open.
- **Grafana has no data**: wait for a minute and verify Prometheus is scraping `klyr:9095`.
