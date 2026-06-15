# Broker CRM Integration Guide

The CRM frontend talks only to the CRM backend. The CRM backend imports `rtx5-sdk-go`, stores manager credentials server-side, connects to RTX5, and calls SDK modules.

Required server-side settings:

| Setting | Notes |
|---|---|
| `RTX5_BASE_URL` | RTX5 broker backend URL |
| `RTX5_BROKER_ID` | Broker tenant ID |
| `RTX5_MANAGER_LOGIN` | Manager login issued for CRM integration |
| `RTX5_MANAGER_PASSWORD` | Store encrypted; never expose to frontend |
| `RTX5_SERVER` | Registered RTX5 server address |
| `RTX5_DEFAULT_GROUP` | Default group for account creation |

Recommended account flow:

1. CRM user submits account request.
2. CRM stores request as pending.
3. Broker admin approves request.
4. CRM backend calls `client.Connect(ctx)`.
5. CRM backend calls `client.Accounts().CreateTradingAccount(ctx, ...)`.
6. CRM stores the returned RTX5 account login.
7. CRM notifies the user.

Recommended CRM-owned endpoints:

```http
POST /crm/rtx5/test-connection
POST /crm/rtx5/settings
GET  /crm/rtx5/groups
GET  /crm/rtx5/symbols?group=demo/STD
POST /crm/accounts/request
POST /crm/accounts/{request_id}/approve
POST /crm/accounts/{login}/deposit
POST /crm/accounts/{login}/withdraw
GET  /crm/accounts/{login}/deals
```
