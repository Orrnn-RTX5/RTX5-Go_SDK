# RTX5 Manager SDK for Go

Go SDK for broker CRM backends that need to call RTX5 manager APIs.

This package mirrors the Rust `rtx5-sdk` concept. It is a backend library, not a standalone CRM and not a deployed service. The intended flow is:

```text
CRM frontend
  -> CRM backend
    -> rtx5-sdk-go
      -> RTX5 broker backend
```

Manager credentials and manager session tokens must stay on the CRM backend.

## Current Scope

This SDK currently covers the manager REST surface. It is enough for a Go CRM backend to connect with manager credentials and perform supported manager-side account, group, finance, trading, market-data, and reporting actions.

It does not yet include the full professional broker SDK surface for trader/webterminal REST, WebSocket realtime streams, IB management, SuperAdmin onboarding, or modern POST-body finance endpoints. Those gaps and the implementation sequence are documented in [Production readiness and roadmap](docs/PRODUCTION_READINESS_AND_ROADMAP.md).

## Install

```bash
go get github.com/YoForex005/RTX5-Go_SDK
```

For private broker delivery, pin a tag or commit.

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"os"

	rtx5sdk "github.com/YoForex005/RTX5-Go_SDK"
)

func main() {
	client, err := rtx5sdk.Builder().
		BaseURL(os.Getenv("RTX5_BASE_URL")).
		BrokerID(os.Getenv("RTX5_BROKER_ID")).
		ManagerLogin(os.Getenv("RTX5_MANAGER_LOGIN")).
		ManagerPassword(os.Getenv("RTX5_MANAGER_PASSWORD")).
		Server(os.Getenv("RTX5_SERVER")).
		Build()
	if err != nil {
		panic(err)
	}

	if _, err := client.Connect(context.Background()); err != nil {
		panic(err)
	}

	account, err := client.Accounts().CreateTradingAccount(context.Background(), rtx5sdk.CreateAccountRequest{
		MasterPassword: os.Getenv("RTX5_TRADER_PASSWORD"),
		Group:          "demo/STD",
		Email:          "client@example.com",
		Currency:       "USD",
		AccountMode:    rtx5sdk.AccountModeHedging,
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("%#v\n", account)
}
```

## Main Modules

| Module | Purpose |
|---|---|
| `Auth()` | Manager connect/login and session storage |
| `Accounts()` | Account creation, list, and account details |
| `Finance()` | Deposit, withdrawal, credit, correction, bonus |
| `Groups()` | List, get, create, update, delete trading groups |
| `Symbols()` | Symbol lists, group symbols, symbol details |
| `MarketData()` | Last tick, candles, tick history |
| `Trading()` | Send, close, modify orders; read positions/orders |
| `Reports()` | Deal history and daily reports |
| `Raw()` | Low-level fallback for endpoints not yet wrapped |

## Runtime Flow

```text
Broker CRM backend
  -> rtx5-sdk-go
    -> POST /api/v2/auth/login with RTX5 manager credentials
      -> manager token/session
        -> broker-scoped manager APIs with Authorization: Bearer
```

`Connect` defaults to secure body-based login. The legacy `GET /Connect` path is available as `Auth().ConnectManagerLegacy` for old backends, but it sends the manager password in the URL and should not be used for new integrations.

## Example Commands

```powershell
Copy-Item .env.example .env
$env:RTX5_BASE_URL="https://broker.example.com"
$env:RTX5_BROKER_ID="gta_broker"
$env:RTX5_MANAGER_LOGIN="9001"
$env:RTX5_MANAGER_PASSWORD="your-manager-password"
$env:RTX5_SERVER="broker.example.com:443"
$env:RTX5_TRADER_PASSWORD="<generated-strong-password>"
go run ./examples/connect
go run ./examples/create_account
```

## Documentation

- [Documentation index](docs/README.md)
- [Broker CRM integration guide](docs/CRM_INTEGRATION.md)
- [Security guide](docs/SECURITY.md)
- [API coverage](docs/API_COVERAGE.md)
- [Production readiness and roadmap](docs/PRODUCTION_READINESS_AND_ROADMAP.md)

## Notes

- HTTPS is required by default. Plaintext HTTP is allowed only for loopback or with `AllowInsecureHTTP(true)`.
- Session-backed requests use `Authorization: Bearer` only; the token is not appended as `?id=`.
- State-changing account, finance, and trading calls send an `Idempotency-Key` header.
- Responses are returned as untyped Go values because RTX5 manager endpoints return mixed legacy and JSON shapes.
- This SDK does not implement CRM auth, RBAC, MFA, database storage, audit logging, Docker, or deployment.
