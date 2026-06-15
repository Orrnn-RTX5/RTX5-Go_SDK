# RTX5 Go SDK Documentation

This folder is the professional documentation set for `rtx5-sdk-go`.

## Start Here

| Document | Purpose |
|---|---|
| [API_COVERAGE.md](API_COVERAGE.md) | Current implemented Go SDK method-to-endpoint coverage. |
| [CRM_INTEGRATION.md](CRM_INTEGRATION.md) | How a CRM backend should use the SDK. |
| [CRM_DEVELOPER_IMPLEMENTATION_GUIDE.md](CRM_DEVELOPER_IMPLEMENTATION_GUIDE.md) | Detailed CRM developer handoff guide with architecture, code examples, endpoints, security, testing, and production checklist. |
| [SECURITY.md](SECURITY.md) | Current security rules and SDK safety notes. |
| [PRODUCTION_READINESS_AND_ROADMAP.md](PRODUCTION_READINESS_AND_ROADMAP.md) | Full professional gap analysis, broker requirements, missing modules, security hardening, and implementation phases. |

## Current Status

The current Go SDK is a manager REST SDK. It can connect with RTX5 manager credentials and perform account, group, symbol, market data, trading, finance, and report actions through manager-style HTTP endpoints.

It is not yet a complete broker CRM or webterminal SDK. The missing professional modules are documented in [PRODUCTION_READINESS_AND_ROADMAP.md](PRODUCTION_READINESS_AND_ROADMAP.md).

## Release Gates

Before shipping this SDK to a broker CRM team:

1. Run `go test ./...`.
2. Run `go vet ./...`.
3. Run `govulncheck ./...` using a patched Go toolchain.
4. Verify live manager login with real broker manager credentials.
5. Verify IP/domain allowlisting for the broker manager/admin backend.
6. Verify that any WebSocket module uses `wss://`, strict origin handling, bounded subscriptions, and authenticated channels.
