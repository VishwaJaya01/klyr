# Release Guide

This document describes how to cut a Klyr release.

## Prerequisites

- Clean working tree
- All tests and lint passing
- Updated `CHANGELOG.md`

## Steps

1) Bump version if needed (e.g., v0.1.0 already set):

```bash
git tag v0.1.0
```

2) Push commits and tags:

```bash
git push
git push --tags
```

3) Draft a GitHub release using the tag `v0.1.0` and paste the release notes (see below).

## Release Notes Template

```
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
```
