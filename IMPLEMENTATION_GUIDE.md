# RTX5 Go SDK Implementation Guide

This guide explains how to use `rtx5-sdk-go` inside a Go CRM backend.

## What This SDK Does

`rtx5-sdk-go` lets a Go backend connect to RTX5 manager APIs and perform broker CRM actions:

- test/connect manager credentials
- create trading accounts
- list groups and symbols
- read market data
- deposit, withdraw, and adjust balance
- send, close, and modify trades
- read reports and history

It is not a full CRM. Your CRM must still implement frontend, user login, database, admin panel, audit logs, and credential storage.

## Recommended Architecture

```text
CRM frontend
  -> Go CRM backend
    -> rtx5-sdk-go
      -> RTX5 broker backend
```

Never call this SDK from frontend code. Manager credentials must stay server-side.

## Required Environment Variables

Store these in your CRM backend environment or secret manager:

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

For account creation examples:

```env
RTX5_TRADER_PASSWORD=<generated-strong-password>
RTX5_TEST_EMAIL=sdk-client@example.com
RTX5_TEST_LOGIN=123456
```

## Install In Your Go CRM Backend

```bash
go get github.com/YoForex005/RTX5-Go_SDK
```

If using a private repo, pin a release tag or commit.

## Create The SDK Client

```go
client, err := rtx5sdk.Builder().
    BaseURL(os.Getenv("RTX5_BASE_URL")).
    BrokerID(os.Getenv("RTX5_BROKER_ID")).
    ManagerLogin(os.Getenv("RTX5_MANAGER_LOGIN")).
    ManagerPassword(os.Getenv("RTX5_MANAGER_PASSWORD")).
    Server(os.Getenv("RTX5_SERVER")).
    Build()
if err != nil {
    return err
}
```

## Test RTX5 Connection

Use this in your CRM admin settings page:

```go
session, err := client.Connect(ctx)
if err != nil {
    return err
}

fmt.Println("connected token length:", len(session.Token))
```

Do not return the token to the frontend.

## Account Approval Flow

Recommended CRM flow:

1. User submits an account request.
2. CRM stores request as `pending`.
3. Broker admin approves request.
4. CRM backend connects to RTX5.
5. CRM backend creates trading account with SDK.
6. CRM stores returned RTX5 login/account ID.
7. CRM notifies the user.

Example:

```go
account, err := client.Accounts().CreateTradingAccount(ctx, rtx5sdk.CreateAccountRequest{
    MasterPassword: generatedTraderPassword,
    Group:          os.Getenv("RTX5_DEFAULT_GROUP"),
    Email:          user.Email,
    FirstName:      user.FirstName,
    LastName:       user.LastName,
    Currency:       "USD",
    AccountMode:    rtx5sdk.AccountModeHedging,
})
if err != nil {
    return err
}
```

## Finance Operations

Use finance methods only behind broker-admin approval and audit logs.

```go
result, err := client.Finance().Deposit(ctx, login, 100.00, "CRM approved deposit")
if err != nil {
    return err
}
```

For retries, use stable idempotency keys:

```go
result, err := client.Finance().DepositWithIdempotencyKey(
    ctx,
    login,
    100.00,
    "CRM approved deposit",
    "deposit-request-123",
)
```

Use the same key only for retrying the same logical operation.

## Suggested CRM Backend Endpoints

```http
POST /admin/rtx5/test-connection
POST /admin/rtx5/settings
GET  /admin/rtx5/groups
GET  /admin/rtx5/symbols?group=demo/STD
POST /accounts/request
POST /admin/accounts/{request_id}/approve
POST /admin/accounts/{login}/deposit
POST /admin/accounts/{login}/withdraw
GET  /admin/accounts/{login}/deals
```

These are CRM endpoints. They should call the SDK internally.

## Security Rules

- Never expose manager credentials to frontend/mobile/public JavaScript.
- Store `RTX5_MANAGER_PASSWORD` encrypted or in a secret manager.
- Use HTTPS for `RTX5_BASE_URL`.
- Do not log manager password, trader password, or session token.
- Audit every account, finance, group, and trade operation.
- Use scoped manager credentials for the CRM.
- Keep high-risk actions like finance and trading admin-only.

## What Your CRM Must Implement

This SDK does not provide:

- user login/auth
- admin roles/RBAC
- database models
- account request tables
- credential encryption
- audit log storage
- frontend UI
- email/notification flow
- payment workflow
- deployment setup

Build those in your CRM backend/app.

## Verification

From this SDK folder:

```powershell
go test ./...
go vet ./...
```

For live testing, set real RTX5 manager credentials and run:

```powershell
go run ./examples/connect
go run ./examples/create_account
```
