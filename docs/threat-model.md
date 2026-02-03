# Threat Model (v0.1)

## In Scope

- Common web attack patterns detectable via deterministic rules (SQLi/XSS/traversal)
- Unexpected methods/content-types/header names/query parameters when contracts are enforced
- Excessive request rates (simple token bucket)

## Out of Scope

- Advanced exploitation chains or 0-days
- Encrypted payload inspection
- Response body inspection
- Distributed rate limiting
- Enterprise CRS compatibility

## Assumptions

- Klyr is deployed in front of a trusted upstream
- Policies and rules are curated by the operator
- TLS termination can occur at Klyr or upstream, per deployment config
