# RTX5 Go SDK Production Readiness and Roadmap

## Executive Summary

`rtx5-sdk-go` is usable today for a Go CRM backend that needs manager REST operations: connect, create accounts, list groups and symbols, read market data, place/modify/close manager orders, deposit, withdraw, and read reports.

It is not yet complete for the full broker product scope:

- full manager/admin CRM
- client account portal
- IB management
- webterminal trading
- realtime tick-by-tick quotes over WebSocket
- order, position, deal, account, and metrics streams over WebSocket
- pending orders, stops, OCO, and advanced order lifecycle
- secure modern finance endpoints
- SuperAdmin/control-plane onboarding
- production-grade SDK release/security lifecycle

The recommended path is to keep the current manager REST module stable, then add typed professional modules in phases: `manager`, `trader`, `marketdata`, `streaming`, `ib`, `finance`, `reports`, `superadmin`, and `webhooks`.

## Current SDK Coverage

| Capability | Current status | Evidence |
|---|---|---|
| Manager/admin connection | Supported | `Client.Connect`, `Auth().LoginV2`, `POST /api/v2/auth/login` |
| Account creation | Supported | `Accounts().CreateTradingAccount`, `Accounts().CreateAndDeposit` |
| Deposit | Supported through manager legacy endpoint | `Finance().Deposit`, `GET /Deposit` |
| Withdraw | Supported through balance adjustment | `Finance().Withdraw`, `GET /BalanceAdjustment` |
| Group CRUD | Supported | `Groups().List/Get/Create/Update/Delete` |
| Set account group at creation | Supported | `CreateAccountRequest.Group` |
| Move existing account to another group | Partial/missing typed method | Needs account update or manager-specific endpoint contract |
| IB management | Missing | No `IB()` module |
| Manager REST order send | Supported | `Trading().SendOrder`, `GET /OrderSend` |
| Modify SL/TP | Supported | `Trading().ModifyOrder`, `GET /OrderModify` |
| Close order | Supported | `Trading().CloseOrder`, `GET /OrderClose` |
| Open positions/orders | Supported | `Trading().Positions`, `Trading().Orders` |
| Deal/daily history | Supported | `Reports().DealHistory`, daily reports |
| Realtime tick-by-tick WebSocket | Missing | No WebSocket package/module |
| Webterminal trader session | Missing | No trader gateway auth/client |
| WebSocket trading/order lifecycle | Missing | No stream command/event module |
| Pending/stops/OCO typed APIs | Partial/missing | `operation` string can pass values, but no typed lifecycle model |

## Required Credentials and Network Onboarding

Yes, a broker deployment needs real manager credentials and a reachable manager/admin endpoint.

Minimum required values:

```env
RTX5_BASE_URL=https://<broker-manager-or-services-domain>
RTX5_BROKER_ID=<broker-id>
RTX5_MANAGER_LOGIN=<manager-login>
RTX5_MANAGER_PASSWORD=<manager-password>
RTX5_SERVER=<registered-server-or-host:port>
```

Production onboarding should also confirm:

- manager account permissions: `manage_accounts`, `manage_finances`, `manage_trades`, `manage_groups`, `view_reports`, and `manage_ib` if IB is enabled
- source IP allowlisting for the CRM backend if the manager/admin backend enforces IP restrictions
- allowed frontend origins for browser WebSocket routes
- public domain and TLS certificate for HTTPS/WSS
- broker tenant ID and server identifier exactly matching RTX5 backend configuration
- separate credentials for high-risk modules such as finance, trading, and manager administration
- credential rotation and emergency revocation procedure

Do not put manager credentials in the browser, webterminal frontend, mobile app, or public JavaScript.

## Workspace Endpoint Evidence

The ORRNN workspace already contains several SDK-worthy surfaces beyond the current Go SDK.

### Trader/Webterminal Gateway

Gateway surfaces found in the broker backend include:

- WebSocket: `GET /ws/v2`
- Auth/discovery: `POST /api/v2/auth/login`, `POST /api/v2/auth/refresh`, `GET /api/v2/auth/me`, `GET /api/v2/broker/servers`, `GET /api/v2/broker/config`
- Market data REST: `GET /api/v2/instruments`, `/quotes`, `/quotes/{symbol}`, `/candles/{symbol}`, `/ticks/{symbol}`, `/depth/{symbol}`, `/watchlist`
- Trading REST: `POST /api/v2/orders`, `POST /api/v2/orders/pending`, `PUT /api/v2/orders/pending/{ticket}`, `DELETE /api/v2/orders/pending/{ticket}`, `POST /api/v2/orders/oco`, `DELETE /api/v2/orders/oco/{id}`
- Positions/history: `GET /api/v2/positions/{login}`, `POST /api/v2/positions/{id}/close`, `PUT /api/v2/positions/{id}`, `GET /api/v2/deals/{login}`, `GET /api/v2/orders/{login}`
- Account actions: `GET/POST /api/v2/accounts`, `GET /api/v2/account/{login}`, `PUT /api/v2/accounts/{login}`, `POST /api/v2/accounts/{login}/deposit`, `withdraw`, `credit`
- IB/referral: `GET /api/v2/ib/my-referral`

Gateway WebSocket channels found:

- market: `ticks.{symbol}`, `candles.{symbol}.{tf}`, `depth.{symbol}`
- account: `positions.{login}`, `account.{login}`, `orders.{login}`, `deals.{login}`, `metrics.{login}`

### Market Feed and History

Bridge/feed surfaces:

- WebSocket: `GET /ws/feed`
- Status: `GET /ws/feed/status`
- REST: `/api/v1/market/symbols`, `/api/v1/market/symbol-list`, `/api/v1/market/quote`, `/api/v1/market/ws-info`, `/api/v1/candles`

History surfaces:

- WebSocket: `GET /ws/history`
- REST: `/history/candles/{symbol}`, `/history/ticks/{symbol}`, `/history/trades`, `/history/deals`, `/history/orders`, `/history/balance`, `/history/equity/{account}`, backfill and coverage endpoints

### Manager Admin Backend

Manager/admin surfaces found:

- event WebSockets: `/OnTick`, `/OnQuote`, `/OnPositionUpdate`, `/OnOrderUpdate`, `/OnDealUpdate`, `/OnAccountUpdate`, `/OnGroupUpdate`, `/OnSymbolUpdate`, `/OnMarketWatch`, `/OnTickStat`, `/OnAccountMetrics`
- account/group/manager: `/Accounts`, `/AccountDetails`, `/AccountCreateAndDeposit`, `/GroupList`, `/GroupGet`, `/GroupCreate`, `/GroupUpdate`, `/GroupDelete`, manager role/list/create/update/delete endpoints
- trading/history: `/Positions`, `/OpenedOrders`, `/Orders`, `/BrokerOrders`, `/BrokerDeals`, `/OrderHistory`, `/PendingOrderHistory`, `/DealHistory`
- IB: `/IBList`, `/IBGet`, `/IBCreate`, `/IBUpdate`, `/IBDelete`, `/IBLinkClient`, `/IBUnlinkClient`, `/IBClientList*`, `/IBPlanCreate`, `/IBPlanList`, `/IBCommissions`, `/IBCommissionPay`, `/IBStats`, `/IBTree`
- modern finance: `POST /api/v1/manager/transactions/deposit`, `withdraw`, `credit`, `transfer`

## Professional SDK Module Roadmap

### Phase 1: Harden Current Manager REST SDK

Goal: make the existing manager REST SDK production-safe before expanding.

Tasks:

- Done: bounded response body reads now prevent unbounded manager responses from being loaded into memory.
- Done: typed manager methods now validate login IDs, symbols, comments, actions, time ranges, raw paths, and common order operations before network I/O.
- Add safe typed operation enums for buy, sell, buy limit, sell limit, buy stop, sell stop, stop-limit where backend supports them.
- Add `CloseAll` if `/OrderCloseAll` is required.
- Add typed pending order methods if using manager endpoints: list, history, cancel, activate, modify price.
- Current manager `/OrderModify` is ticket-based and supports explicit SL/TP updates only; price modification should be handled by a dedicated pending-order endpoint when that module is added.
- Add account update/change group method after confirming exact endpoint contract.
- Mark legacy query-string endpoints as compatibility-only in docs.
- Prefer modern POST body finance endpoints where available.
- Keep `Raw()` out of public CRM routes; treat it as an advanced escape hatch.

### Phase 2: Trader/Webterminal SDK

Goal: support client trading and webterminal flows through Gateway APIs, not manager credentials.

Add module: `Trader()`.

Suggested methods:

```text
LoginTrader()
RefreshTraderToken()
Me()
BrokerServers()
BrokerConfig()
ListInstruments()
GetQuote()
GetCandles()
GetTicks()
GetDepth()
PlaceMarketOrder()
PlacePendingOrder()
ModifyPendingOrder()
CancelPendingOrder()
PlaceOCO()
CancelOCO()
ClosePosition()
ModifyPosition()
GetPositions()
GetOrders()
GetDeals()
GetAccountMetrics()
RequestWithdrawal()
RequestLeverageChange()
```

Rules:

- trader module uses trader tokens, not manager credentials
- all browser-facing calls must enforce CRM/backend authorization
- webterminal should use REST for initial state and WebSocket for live patches
- account-specific routes must validate that the authenticated user owns the login or has admin permission

### Phase 3: WebSocket Streaming SDK

Goal: provide realtime quote/account/trade streams professionally.

Add modules:

```text
Streaming()
GatewayStream()
ManagerEvents()
FeedStream()
HistoryStream()
```

Required channels:

- tick-by-tick quotes: `ticks.{symbol}`
- candles: `candles.{symbol}.{tf}`
- depth: `depth.{symbol}`
- positions: `positions.{login}`
- account: `account.{login}`
- orders: `orders.{login}`
- deals: `deals.{login}`
- metrics: `metrics.{login}`
- manager events: `/OnTick`, `/OnQuote`, `/OnPositionUpdate`, `/OnOrderUpdate`, `/OnDealUpdate`, `/OnAccountUpdate`

Required controls:

- require `wss://` by default for public/non-loopback hosts
- authenticate handshake using `Authorization: Bearer` or a secure subprotocol, not manager tokens in query strings
- strict origin allowlist for browser-facing WebSocket routes
- bounded subscription count per connection
- bounded message size
- heartbeat/ping/pong and idle timeout
- reconnect with backoff and resubscribe
- channel authorization before subscribe
- no account channel unless token claims allow that login
- no manager event stream to browser clients

### Phase 4: IB and Referral Module

Goal: support broker IB operations for manager/admin CRM.

Add module: `IB()`.

Suggested methods:

```text
ListIBs()
GetIB()
CreateIB()
UpdateIB()
DeleteIB()
LinkClient()
UnlinkClient()
ListIBClients()
CreatePlan()
ListPlans()
GetCommissions()
PayCommission()
GetIBStats()
GetIBTree()
GetMyReferral()
```

Rules:

- manager/admin IB endpoints require `manage_ib`
- client-facing referral endpoint uses trader token, not manager token
- commission payment must require stable idempotency key
- commission payment must be audited in CRM backend

### Phase 5: Modern Finance Module

Goal: stop relying only on legacy GET finance endpoints.

Add module: `ManagerTransactions()`.

Suggested methods:

```text
Deposit()
Withdraw()
Credit()
Transfer()
BulkDeposit()
BulkWithdraw()
GetFinanceHistory()
ApproveWithdrawal()
RejectWithdrawal()
```

Rules:

- prefer `POST /api/v1/manager/transactions/*` when available
- stable idempotency key is mandatory for money movement
- CRM backend must persist finance request IDs
- all finance actions require approval, audit log, and role checks
- never log full request/response bodies without redaction

### Phase 6: SuperAdmin and Broker Onboarding SDK

Goal: support multi-broker control-plane setup.

Add module: `SuperAdmin()`.

Suggested coverage:

- broker registration and status
- broker server discovery
- manager credential vault/handover
- central LP credential setup
- market-data LPs and feed assignments
- broker FIX accounts
- symbol registry and permissions
- group LP routing
- license plans and usage
- audit log reads

Rules:

- separate SuperAdmin credentials from broker manager credentials
- never mix SuperAdmin tokens with trader or manager modules
- enforce tenant/broker ID on every request

## Security Hardening Plan

### High Priority

1. Patch Go toolchain before release.
   - Resolved on this host with `go1.26.4`.
   - `govulncheck` now reports no reachable vulnerabilities.
   - Release builds should continue to use a patched Go version and run `govulncheck` in CI.

2. Replace legacy query-string mutations where backend supports POST body.
   - Manager password in query is deprecated.
   - Account passwords, finance values, and order actions in URLs can leak to proxy/access logs.
   - Keep legacy methods only for compatibility.

3. Response size limits.
   - Implemented: the client now limits response-body reads and returns typed `ResponseTooLargeError` when the configured maximum is exceeded.
   - Default: 32 MiB. CRM integrations that intentionally pull large history/report ranges should set `MaxResponseBytes` explicitly and paginate/time-slice where possible.

4. Formalize idempotency.
   - Auto-generated keys are fine for one-shot calls.
   - Retried account, finance, and trade operations must use stable CRM request IDs.
   - Professional SDK should make stable keys mandatory for high-risk calls or provide request wrappers that enforce it.

### Medium Priority

5. Harden caller-supplied HTTP clients.
   - `HTTPClient(...)` can bypass default timeout/redirect protections.
   - Add docs and optional validation helper.

6. Add typed validation.
   - Implemented for the current manager REST surface: positive logins/tickets, symbols, comments, order operation allowlist, balance action allowlist, time ranges, group names, and raw paths.
   - Remaining: extend validation as new typed modules are added.

7. Reduce raw leakage.
   - `Session.Raw`, `APIError.APIBody()`, and example output can leak sensitive data if copied into production logs.
   - Add redacted session output and clear example warnings.

8. Add CI gates.
   - `go test ./...`
   - `go vet ./...`
   - `go test -race ./...` on a runner with CGO and a C compiler
   - `govulncheck ./...`
   - secret scanning
   - markdown link check

### WebSocket Security Requirements

Before adding any WebSocket module:

- use `wss://` for non-loopback production
- authenticate the handshake
- validate `Origin` for browser clients
- do not put manager tokens in query strings
- validate every message as untrusted input
- use explicit subscribe/unsubscribe schemas
- limit message size, subscription count, and per-user connection count
- add heartbeat and idle timeouts
- rate-limit subscriptions and commands
- authorize every account-specific channel
- redact tokens from reconnect logs

## Broker CRM Feature Checklist

A professional broker SDK should eventually cover:

- account request and approval lifecycle
- KYC status hooks
- account creation and account update
- group assignment and group migration
- symbol visibility and group-symbol settings
- leverage change requests
- deposit request, withdrawal request, approval, rejection, payout tracking
- manual deposit/withdraw/credit/transfer for admins
- market and pending orders
- SL/TP modification
- close partial/full/all
- OCO orders
- position, order, deal, balance, and equity history
- realtime quotes, depth, candles, account metrics, positions, orders, deals
- IB registration, plan, tree, commission, payout
- broker admin manager roles and permissions
- LP/routing configuration
- reporting exports
- audit log access
- webhooks for account, finance, order, deal, and IB events
- reconciliation tools for finance and trades
- health checks and diagnostics

## Recommended Professional Folder Direction

The current flat package is acceptable for the first manager REST release. For the larger broker SDK, move toward this structure:

```text
rtx5-sdk-go/
  client.go
  errors.go
  options.go
  types.go
  manager/
    accounts.go
    groups.go
    trading.go
    finance.go
    reports.go
    ib.go
  trader/
    auth.go
    accounts.go
    trading.go
    positions.go
    market.go
  streaming/
    gateway.go
    feed.go
    manager_events.go
    subscriptions.go
  superadmin/
    brokers.go
    credentials.go
    routing.go
    marketdata.go
  internal/
    httpx/
    redaction/
    validation/
  examples/
  docs/
```

Do this package split only when adding the next modules. Do not churn the current working SDK just for folder aesthetics.

## Implementation Phases

### Phase A: Current SDK Release Candidate

Exit criteria:

- current manager REST methods stay green
- response size limit added
- validation tightened
- docs updated
- CI uses patched Go and govulncheck
- one live manager login test passes against broker staging

### Phase B: Broker CRM Admin SDK

Exit criteria:

- typed account update/change group
- modern finance POST endpoints
- IB module
- manager roles/permissions reads
- audit/report methods

### Phase C: Trader/Webterminal SDK

Exit criteria:

- trader auth
- gateway REST trading
- typed pending/OCO
- initial state loaders for positions/orders/deals/metrics
- integration examples for webterminal backend

### Phase D: Realtime SDK

Exit criteria:

- WebSocket client
- secure handshake
- channel subscriptions
- reconnect/resubscribe
- typed event payloads
- bounded resource usage
- channel authorization documentation

### Phase E: Control Plane SDK

Exit criteria:

- broker onboarding
- manager credential handover
- LP/routing setup
- symbol permissions
- feed assignments
- license/usage reads

## Current Verification

Latest local checks after upgrading to `go1.26.4` and adding a CGO C compiler:

```powershell
go test -count=1 ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go test -count=1 -race ./...
```

All passed locally. Race tests require `CGO_ENABLED=1` and a C compiler such as WinLibs or MinGW-w64 on Windows.

## External Security References Used

- OWASP WebSocket Security Cheat Sheet: authentication, authorization, message validation, size limits, and rate limiting.
- OWASP API Security Top 10 2023: object-level and object-property authorization risks.
- OWASP Secrets Management Cheat Sheet: centralized storage, auditing, rotation, and lifecycle management.
- OWASP REST Security Cheat Sheet: HTTPS and credential protection in transit.
- Go `govulncheck` documentation: official Go vulnerability scanning.
- Go WebSocket package documentation: origin policy and safe WebSocket upgrade behavior.
