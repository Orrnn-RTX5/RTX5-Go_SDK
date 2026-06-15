# Security Guide

Hard rules:

- Do not put RTX5 manager credentials in browser, mobile app, or public JavaScript.
- Store manager credentials only on the broker CRM backend.
- Encrypt manager credentials at rest in the CRM backend.
- Use TLS for SDK traffic.
- Use scoped manager permissions.
- Audit account, group, finance, and trade operations in the CRM backend.

Implementation notes:

- `Connect` posts manager credentials to `/api/v2/auth/login`; it does not put the manager password in the URL.
- `Auth().ConnectManagerLegacy` exists only for older backends and sends the password in the query string.
- Session-backed requests use `Authorization: Bearer` only.
- API error strings redact response bodies; `APIError.APIBody()` exposes the raw body only when caller code explicitly asks for it.
- Password-bearing request types implement redacted `String`/`GoString` output.
- The default client has request/connect timeouts and bounded response-body reads. The default response limit is 32 MiB; raise it with `MaxResponseBytes` only for intentionally large history/report pulls.
- Typed manager methods validate common high-risk inputs before network I/O, including positive logins/tickets, symbols, order operations, balance actions, comments, time ranges, and raw paths.

Not implemented here:

- CRM user auth
- RBAC/ABAC
- MFA/OAuth/SSO
- CORS/CSRF/security headers
- database storage
- encryption-at-rest
- rate limiting/retries/circuit breaking
- logging infrastructure
- metrics
- deployment/runtime service logic

## Production Gates

- Build and release with a patched Go toolchain.
- Run `go test ./...`, `go vet ./...`, and `govulncheck ./...` in CI.
- Run `go test -race ./...` on a runner with CGO and a C compiler.
- Verify live manager login and broker IP/domain allowlisting before handing credentials to a CRM team.
