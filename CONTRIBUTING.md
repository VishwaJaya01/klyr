# Contributing

Thanks for helping improve Klyr.

## Quick Start

```bash
make build
make test
make lint
```

## Guidelines

- Keep changes within v0.1 scope (see `agent.md`).
- Prefer Go stdlib; keep dependencies minimal.
- No AI/ML features and no unsafe payload generation.
- Avoid logging secrets (Authorization/Cookie/Set-Cookie values).
- Add tests for new behavior and keep changes small.

## Coding Standards

- Use `context.Context` for request handling.
- Avoid global mutable state.
- No panics on normal code paths.
- Wrap errors with `%w` where useful.

## Submitting

- Use conventional commit messages: `feat:`, `fix:`, `docs:`, `chore:`, `test:`
- Ensure `go test ./...` and `golangci-lint run` pass.
