# CRM Developer Implementation Guide

This guide explains how a CRM backend developer should use `rtx5-sdk-go` to integrate a broker CRM with RTX5 manager/admin APIs.

The SDK is a Go library. It is not a running service, not a frontend package, and not a complete CRM. The CRM backend imports it, stores RTX5 manager credentials securely, calls RTX5 manager/admin endpoints through the SDK, and exposes CRM-owned routes to the CRM frontend.

## Current Integration Scope

Use this SDK now for manager/admin CRM workflows:

- test RTX5 manager connection
- list accounts, groups, and symbols
- create trading accounts
- create account and deposit in one approved backend action
- deposit, withdraw, credit, correction, and bonus balance actions
- send, close, and modify manager-side orders
- read open positions and pending orders
- read deal history and daily reports
- create, update, and delete groups
- use raw manager endpoints from backend-only admin code when a typed method does not exist yet

Do not use this SDK yet for:

- browser-side calls
- webterminal trader login
- client portal trading sessions
- realtime WebSocket streams
- IB/referral management
- SuperAdmin/control-plane onboarding
- modern POST-body finance APIs unless you call them through carefully reviewed backend-only `Raw()` methods

## Architecture

Use this request path:

```text
CRM frontend
  -> CRM backend API
    -> rtx5-sdk-go
      -> RTX5 broker manager/admin backend
```

Never use this path:

```text
CRM frontend
  -> rtx5-sdk-go or RTX5 manager/admin backend
```

Manager credentials and manager session tokens must stay inside the CRM backend.

## Prerequisites

Use a patched Go toolchain and run the SDK gates before release:

```powershell
go version
go test -count=1 ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go test -count=1 -race ./...
```

Race tests require CGO and a C compiler on Windows. On this host, WinLibs is installed at:

```text
C:\tools\winlibs-16.1.0-ucrt-posix\mingw64\bin
```

The CRM backend also needs:

- HTTPS access to the RTX5 manager/admin backend
- exact broker tenant ID
- exact registered server value
- manager account with the required permissions
- source IP allowlisting if RTX5 enforces it
- secure secret storage
- CRM-side auth, RBAC, audit logs, and database persistence

## Required Environment Variables

Store these in the CRM backend environment or secret manager:

```env
RTX5_BASE_URL=https://broker-backend.example.com
RTX5_BROKER_ID=gta_broker
RTX5_MANAGER_LOGIN=9001
RTX5_MANAGER_PASSWORD=change-me
RTX5_SERVER=broker.example.com:443
RTX5_DEFAULT_GROUP=demo/STD
RTX5_DEFAULT_CURRENCY=USD
RTX5_DEFAULT_LEVERAGE=100
```

Optional values for smoke tests:

```env
RTX5_TEST_EMAIL=sdk-client@example.com
RTX5_TEST_LOGIN=123456
RTX5_TRADER_PASSWORD=<generated-strong-password>
```

Rules:

- Never expose `RTX5_MANAGER_PASSWORD` to frontend, mobile apps, logs, browser storage, or public JavaScript.
- Encrypt stored manager credentials if the CRM lets admins edit RTX5 settings.
- Use separate manager credentials for finance/trading if the broker wants tighter blast-radius control.
- Rotate manager credentials after developer handoff or staging tests.

## Install The SDK

In the CRM backend module:

```bash
go get github.com/YoForex005/RTX5-Go_SDK
```

For local development when the CRM backend is beside this SDK folder:

```bash
go mod edit -replace github.com/YoForex005/RTX5-Go_SDK=../rtx5-sdk-go
go mod tidy
```

Do not commit a local `replace` unless the repository is intentionally developed as a monorepo.

## Suggested CRM Backend Structure

```text
crm-backend/
  cmd/api/main.go
  internal/config/config.go
  internal/rtx5/client.go
  internal/rtx5/service.go
  internal/http/admin_rtx5_handlers.go
  internal/http/account_handlers.go
  internal/audit/audit.go
  internal/db/
  go.mod
```

Keep RTX5-specific code behind a small internal service layer. Do not call the SDK directly from every HTTP handler.

## Load Configuration

Example config type:

```go
package config

import (
	"fmt"
	"os"
	"strconv"
)

type RTX5Config struct {
	BaseURL         string
	BrokerID        string
	ManagerLogin    string
	ManagerPassword string
	Server          string
	DefaultGroup    string
	DefaultCurrency string
	DefaultLeverage uint32
}

func LoadRTX5() (RTX5Config, error) {
	lev, err := strconv.ParseUint(env("RTX5_DEFAULT_LEVERAGE", "100"), 10, 32)
	if err != nil {
		return RTX5Config{}, fmt.Errorf("invalid RTX5_DEFAULT_LEVERAGE: %w", err)
	}

	cfg := RTX5Config{
		BaseURL:         required("RTX5_BASE_URL"),
		BrokerID:        required("RTX5_BROKER_ID"),
		ManagerLogin:    required("RTX5_MANAGER_LOGIN"),
		ManagerPassword: required("RTX5_MANAGER_PASSWORD"),
		Server:          required("RTX5_SERVER"),
		DefaultGroup:    required("RTX5_DEFAULT_GROUP"),
		DefaultCurrency: env("RTX5_DEFAULT_CURRENCY", "USD"),
		DefaultLeverage: uint32(lev),
	}
	return cfg, nil
}

func required(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(key + " is required")
	}
	return value
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
```

## Create An SDK Client Factory

Build one client per process or per service object. The SDK stores the current manager session in memory and protects it with a mutex.

```go
package rtx5

import (
	"net/http"
	"time"

	rtx5sdk "github.com/YoForex005/RTX5-Go_SDK"
	"your-crm/internal/config"
)

func NewClient(cfg config.RTX5Config) (*rtx5sdk.Client, error) {
	httpClient := &http.Client{Timeout: 30 * time.Second}

	return rtx5sdk.Builder().
		BaseURL(cfg.BaseURL).
		BrokerID(cfg.BrokerID).
		ManagerLogin(cfg.ManagerLogin).
		ManagerPassword(cfg.ManagerPassword).
		Server(cfg.Server).
		HTTPClient(httpClient).
		MaxResponseBytes(32 << 20).
		Build()
}
```

HTTPS is required by default. Plain HTTP works only for loopback or if you explicitly call `AllowInsecureHTTP(true)`. Do not enable insecure HTTP for production internet traffic.

## Connect And Reconnect

Call `Connect(ctx)` before session-backed SDK calls.

```go
session, err := client.Connect(ctx)
if err != nil {
	return err
}
_ = session
```

Do not return `session.Token` to the frontend.

Recommended pattern:

- Connect at startup to fail fast if credentials are wrong.
- Reconnect if a call returns an auth/permission response from RTX5.
- Expose a CRM admin "test connection" button that calls `Connect` and returns only safe metadata.
- Call `Disconnect(ctx)` on shutdown if your app has a graceful shutdown hook.

## Wrap The SDK In A Service

Example service:

```go
package rtx5

import (
	"context"
	"fmt"

	rtx5sdk "github.com/YoForex005/RTX5-Go_SDK"
)

type Service struct {
	client *rtx5sdk.Client
}

func NewService(client *rtx5sdk.Client) *Service {
	return &Service{client: client}
}

func (s *Service) TestConnection(ctx context.Context) error {
	if _, err := s.client.Connect(ctx); err != nil {
		return fmt.Errorf("rtx5 connect failed: %w", err)
	}
	if _, err := s.client.Groups().List(ctx); err != nil {
		return fmt.Errorf("rtx5 group list failed: %w", err)
	}
	return nil
}
```

## Handle SDK Responses

Most SDK methods return `rtx5sdk.Value`, which is an alias for `any`. This is intentional because the manager backend has mixed legacy and JSON response shapes.

CRM code should normalize the response into CRM-owned DTOs before returning anything to the frontend.

Example helper:

```go
package rtx5

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func asMap(value any) (map[string]any, error) {
	obj, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected RTX5 response type %T", value)
	}
	return obj, nil
}

func int64Field(obj map[string]any, keys ...string) (int64, error) {
	for _, key := range keys {
		value, ok := obj[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case float64:
			return int64(v), nil
		case int64:
			return v, nil
		case int:
			return int64(v), nil
		case string:
			return strconv.ParseInt(v, 10, 64)
		case json.Number:
			return v.Int64()
		}
	}
	return 0, fmt.Errorf("missing integer field %v", keys)
}
```

## CRM Endpoint Design

These are CRM backend routes, not RTX5 routes. The frontend should only call these CRM routes.

```http
POST /admin/rtx5/test-connection
GET  /admin/rtx5/groups
GET  /admin/rtx5/groups/{group}
POST /admin/rtx5/groups
PATCH /admin/rtx5/groups/{group}
DELETE /admin/rtx5/groups/{group}
GET  /admin/rtx5/symbols?group=demo/STD
GET  /admin/rtx5/symbols/{symbol}
GET  /admin/rtx5/quotes?symbols=EURUSD,XAUUSD
GET  /admin/accounts
GET  /admin/accounts/{login}
POST /accounts/request
POST /admin/accounts/{request_id}/approve
POST /admin/accounts/{login}/deposit
POST /admin/accounts/{login}/withdraw
POST /admin/accounts/{login}/credit
GET  /admin/accounts/{login}/positions
GET  /admin/accounts/{login}/orders
GET  /admin/accounts/{login}/deals?from=...&to=...
GET  /admin/reports/daily?group=demo/STD&from=...&to=...
```

Every admin route must check CRM auth and RBAC before calling the SDK.

## Implement Test Connection

Handler flow:

1. Check CRM admin permission.
2. Build or load SDK client.
3. Call `Connect`.
4. Call a low-risk read endpoint such as `Groups().List`.
5. Return safe status only.

```go
func (s *Service) TestConnection(ctx context.Context) (map[string]any, error) {
	session, err := s.client.Connect(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := s.client.Groups().List(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"connected": true,
		"broker_id": session.BrokerID,
		"groups": groups,
	}, nil
}
```

Do not include token, password, or raw auth response in the HTTP response.

## Implement Groups And Symbols

List groups:

```go
groups, err := client.Groups().List(ctx)
```

Get one group:

```go
group, err := client.Groups().Get(ctx, "demo/STD")
```

Create a group:

```go
leverage := uint32(100)
marginCall := 100.0
stopOut := 50.0

cfg := rtx5sdk.NewGroupConfig("demo/STD")
cfg.Currency = "USD"
cfg.Leverage = &leverage
cfg.MarginCallLevel = &marginCall
cfg.StopOutLevel = &stopOut

result, err := client.Groups().Create(ctx, cfg)
```

Update a group:

```go
spreadMarkup := 1.5

cfg := rtx5sdk.NewGroupConfig("demo/STD")
cfg.Spread = &spreadMarkup

result, err := client.Groups().Update(ctx, cfg)
```

The SDK sends group update target as `?group=demo/STD` and sends update fields in the JSON body.

List symbols visible to a group:

```go
symbols, err := client.Symbols().ListForGroup(ctx, "demo/STD")
```

Get all enabled symbol names:

```go
symbols, err := client.Symbols().AllNames(ctx)
```

Get one symbol:

```go
symbol, err := client.Symbols().Get(ctx, "EURUSD")
```

## Implement Account Request And Approval

Recommended CRM database flow:

1. User requests a trading account.
2. CRM stores request as `pending`.
3. Admin reviews KYC/risk/payment status.
4. Admin approves.
5. CRM backend creates the account through RTX5.
6. CRM stores returned RTX5 login.
7. CRM creates an audit log.
8. CRM notifies the user.

Suggested table:

```sql
CREATE TABLE account_requests (
  id BIGSERIAL PRIMARY KEY,
  user_id BIGINT NOT NULL,
  status TEXT NOT NULL,
  requested_group TEXT NOT NULL,
  currency TEXT NOT NULL,
  leverage INTEGER NOT NULL,
  rtx5_login BIGINT,
  admin_id BIGINT,
  rejection_reason TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Approval code:

```go
func (s *Service) ApproveAccount(ctx context.Context, req AccountRequest, traderPassword string) (int64, error) {
	leverage := uint32(req.Leverage)

	value, err := s.client.Accounts().CreateTradingAccountWithIdempotencyKey(
		ctx,
		rtx5sdk.CreateAccountRequest{
			MasterPassword: traderPassword,
			Group:          req.Group,
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			Email:          req.Email,
			Leverage:       &leverage,
			Currency:       req.Currency,
			AccountMode:    rtx5sdk.AccountModeHedging,
			Comment:        "CRM account request " + req.IDString(),
		},
		"account-request-"+req.IDString(),
	)
	if err != nil {
		return 0, err
	}

	obj, err := asMap(value)
	if err != nil {
		return 0, err
	}
	login, err := int64Field(obj, "login", "Login", "account_login")
	if err != nil {
		return 0, err
	}
	return login, nil
}
```

Use a generated trader password. Store it according to your CRM policy. Do not send it through logs or analytics.

## Implement Account Reads

List account logins:

```go
accounts, err := client.Accounts().List(ctx)
```

Get account details:

```go
account, err := client.Accounts().Details(ctx, login)
```

Read open positions:

```go
positions, err := client.Trading().Positions(ctx, login)
```

Read pending orders:

```go
orders, err := client.Trading().Orders(ctx, login)
```

The SDK sends `logins[]` for positions and orders to match the manager backend contract.

## Implement Finance Actions

Finance actions must be admin-only and audited. The SDK sends an `Idempotency-Key` header on state-changing calls.

Suggested table:

```sql
CREATE TABLE finance_requests (
  id BIGSERIAL PRIMARY KEY,
  account_login BIGINT NOT NULL,
  type TEXT NOT NULL,
  amount NUMERIC(18, 2) NOT NULL,
  currency TEXT NOT NULL DEFAULT 'USD',
  status TEXT NOT NULL,
  requested_by BIGINT,
  approved_by BIGINT,
  idempotency_key TEXT NOT NULL UNIQUE,
  rtx5_response JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Deposit:

```go
value, err := client.Finance().DepositWithIdempotencyKey(
	ctx,
	login,
	100.00,
	"CRM approved deposit request 123",
	"finance-request-123",
)
```

Withdraw:

```go
value, err := client.Finance().WithdrawWithIdempotencyKey(
	ctx,
	login,
	50.00,
	"CRM approved withdrawal request 124",
	"finance-request-124",
)
```

Credit or correction:

```go
value, err := client.Finance().AdjustBalanceWithIdempotencyKey(
	ctx,
	rtx5sdk.BalanceAdjustmentRequest{
		Login:   login,
		Amount:  25.00,
		Action:  rtx5sdk.BalanceActionCredit,
		Comment: "CRM approved credit request 125",
	},
	"finance-request-125",
)
```

Retry rule:

- Use the same idempotency key only when retrying the same logical CRM request.
- Do not reuse a key for a different account, amount, action, or request.

## Implement Trading Admin Actions

Only expose these to trusted CRM admin roles.

Send order:

```go
value, err := client.Trading().SendOrderWithIdempotencyKey(
	ctx,
	rtx5sdk.OrderSendRequest{
		Login:     login,
		Symbol:    "EURUSD",
		Operation: "BUY",
		Lots:      0.10,
	},
	"trade-request-456",
)
```

Allowed operation strings:

```text
BUY
SELL
BUY_LIMIT
SELL_LIMIT
BUY_STOP
SELL_STOP
```

Close order or position by ticket:

```go
value, err := client.Trading().CloseOrderWithIdempotencyKey(
	ctx,
	rtx5sdk.OrderCloseRequest{
		Ticket: 123456789,
	},
	"close-request-789",
)
```

Modify stop loss and take profit:

```go
sl := 1.0500
tp := 1.1200

value, err := client.Trading().ModifyOrderWithIdempotencyKey(
	ctx,
	rtx5sdk.OrderModifyRequest{
		Ticket:     123456789,
		StopLoss:   &sl,
		TakeProfit: &tp,
	},
	"modify-request-790",
)
```

Current limitation:

- `OrderModifyRequest.Price` is intentionally rejected by the SDK because the current manager `/OrderModify` path supports explicit SL/TP updates only.
- Add a dedicated typed pending-order method later if the CRM needs pending price modification.

## Implement Reports

Prefer the `time.Time` helpers for all time-window calls. The SDK accepts
RFC3339 strings, `YYYY-MM-DDTHH:MM:SS`, `YYYY-MM-DD`, and Unix epoch timestamps,
but it normalizes them before sending `from` and `to` to the manager backend.

Deal history:

```go
from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
to := time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC)

deals, err := client.Reports().DealHistory(ctx, rtx5sdk.NewLoginTimeRange(login, from, to))
```

Daily report for one login:

```go
daily, err := client.Reports().DailyForLogin(ctx, rtx5sdk.NewLoginTimeRange(login, from, to))
```

Daily report for a group:

```go
daily, err := client.Reports().DailyForGroup(ctx, rtx5sdk.NewGroupTimeRange("demo/STD", from, to))
```

For large reports, keep the SDK default response limit in mind. If a legitimate report is larger than 32 MiB, increase `MaxResponseBytes` only for the backend job that needs it, and prefer time slicing or pagination.

## Use Raw Endpoints Carefully

`Raw()` is an escape hatch for manager/admin endpoints that are not typed yet.

Example:

```go
query := url.Values{}
query.Set("some_param", "value")

value, err := client.Raw().Get(ctx, "/SomeManagerEndpoint", query)
```

Rules:

- Never expose `Raw()` directly to frontend route parameters.
- Keep a server-side allowlist of raw paths.
- Validate all input before passing it to `Raw()`.
- Add typed SDK methods once a raw endpoint becomes important to the CRM.

## Error Handling

The SDK returns typed errors for common local failures:

- `MissingConfigError`: required SDK config was not provided.
- `InvalidInputError`: local input validation failed before network I/O.
- `InsecureBaseURLError`: plaintext public HTTP was rejected.
- `ErrNotConnected`: call `Connect` before session-backed methods.
- `APIError`: RTX5 returned non-2xx.
- `ResponseTooLargeError`: response exceeded configured maximum.

Example:

```go
value, err := client.Accounts().Details(ctx, login)
if err != nil {
	var apiErr rtx5sdk.APIError
	if errors.As(err, &apiErr) {
		// Log raw body only in secure backend logs with redaction.
		// Do not return apiErr.APIBody() directly to frontend.
		return nil, fmt.Errorf("rtx5 rejected request: status=%d", apiErr.StatusCode)
	}
	return nil, err
}
_ = value
```

HTTP response guidance:

```text
SDK/config error             -> 500 or admin setup error
CRM validation error         -> 400
CRM auth/RBAC failure        -> 401 or 403
RTX5 permission failure      -> 403
RTX5 not found               -> 404
RTX5 transient/network error -> 502 or 503
```

## Security Requirements

Minimum rules:

- The frontend never sees manager credentials or manager tokens.
- The CRM backend authenticates every CRM user.
- Admin routes enforce CRM RBAC before SDK calls.
- Finance and trading actions require approval or admin confirmation.
- Every account, finance, group, and trading operation writes an audit log.
- Never log manager password, trader password, session token, or raw account passwords.
- Use HTTPS for `RTX5_BASE_URL`.
- Use a secret manager or encrypted database fields for credentials.
- Redact `APIError.APIBody()` before logging.
- Do not put manager tokens into query strings.
- Use stable idempotency keys for retryable money/trade/account mutations.

Suggested audit table:

```sql
CREATE TABLE audit_events (
  id BIGSERIAL PRIMARY KEY,
  actor_user_id BIGINT,
  action TEXT NOT NULL,
  entity_type TEXT NOT NULL,
  entity_id TEXT,
  idempotency_key TEXT,
  request_json JSONB,
  response_json JSONB,
  ip_address TEXT,
  user_agent TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Redact secrets before writing `request_json` and `response_json`.

## Live Smoke Test Plan

Run this before giving the CRM integration to a broker:

1. Set real staging manager credentials.
2. Run `go run ./examples/connect`.
3. Verify `Groups().List`.
4. Verify `Symbols().ListForGroup` for the default group.
5. Create one test account in a demo group.
6. Read account details for the returned login.
7. Run a small test deposit with a stable idempotency key.
8. Read deal history for the test login.
9. Read positions and orders for the test login.
10. Disconnect or rotate test credentials if needed.

PowerShell example from this SDK folder:

```powershell
$env:RTX5_BASE_URL="https://broker-backend.example.com"
$env:RTX5_BROKER_ID="gta_broker"
$env:RTX5_MANAGER_LOGIN="9001"
$env:RTX5_MANAGER_PASSWORD="your-manager-password"
$env:RTX5_SERVER="broker.example.com:443"
$env:RTX5_DEFAULT_GROUP="demo/STD"
$env:RTX5_TRADER_PASSWORD="<generated-strong-password>"
$env:RTX5_TEST_EMAIL="sdk-client@example.com"

go run ./examples/connect
go run ./examples/create_account
```

## CI Checklist

The SDK or CRM backend pipeline should run:

```powershell
go test -count=1 ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go test -count=1 -race ./...
```

Also add:

- secret scanning
- dependency update checks
- linting
- migration tests for CRM-owned tables
- integration tests against an RTX5 staging tenant

## Production Handoff Checklist

Before production:

- Go toolchain is patched.
- SDK tests, vet, vulnerability scan, and race tests pass.
- CRM backend uses HTTPS RTX5 base URL.
- Manager credentials are in secret storage.
- Broker ID and server value match RTX5 configuration.
- Manager account permissions are scoped and confirmed.
- CRM source IP is allowlisted if required.
- CRM auth and RBAC are implemented.
- Finance/trading approval flow is implemented.
- Audit logs are implemented.
- Stable idempotency keys are stored for mutations.
- Live staging smoke test passed.
- Rollback plan exists for CRM deployment.
- Credential rotation process is documented.

## What To Build First

Recommended CRM MVP order:

1. RTX5 settings screen with test connection.
2. Group and symbol read-only screens.
3. Account request table.
4. Admin approval to create trading account.
5. Account details page.
6. Deposit and withdrawal approval flow.
7. Deal history and daily reports.
8. Positions and pending orders read-only views.
9. Admin-only group update controls.
10. Optional manager trading controls if the broker explicitly needs them.

Keep webterminal, realtime streams, IB, and SuperAdmin outside this MVP unless the SDK is extended for those modules.
