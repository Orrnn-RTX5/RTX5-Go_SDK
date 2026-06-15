# API Coverage

This Go SDK mirrors the Rust `rtx5-sdk` v1 manager API coverage.

| SDK method | RTX5 endpoint |
|---|---|
| `client.Connect()` / `Auth().LoginV2(...)` | `POST /api/v2/auth/login` |
| `client.Disconnect()` | `GET /Disconnect` |
| `Auth().ConnectManagerLegacy(...)` | `GET /Connect` |
| `Accounts().CreateTradingAccount(...)` | `GET /AccountCreate` |
| `Accounts().CreateAndDeposit(...)` | `GET /AccountCreateAndDeposit` |
| `Accounts().List()` | `GET /Accounts` |
| `Accounts().Details(login)` | `GET /AccountDetails` |
| `Groups().List()` | `GET /GroupList` |
| `Groups().Get(group)` | `GET /GroupGet` |
| `Groups().Create(GroupConfig)` | `POST /GroupCreate` |
| `Groups().Update(GroupConfig)` | `POST /GroupUpdate` |
| `Groups().CreateRaw(body)` / `UpdateRaw(body)` | `POST /GroupCreate` / `/GroupUpdate` |
| `Groups().Delete(group)` | `GET /GroupDelete` |
| `Symbols().ListForGroup(group)` | `GET /SymbolList` |
| `Symbols().AllNames()` | `GET /SymbolsList` |
| `Symbols().Get(symbol)` | `GET /SymbolGet` |
| `MarketData().LastTick(symbols)` | `GET /TickLast` |
| `MarketData().Candles(symbol, range)` | `GET /ChartRequest` |
| `MarketData().TickHistory(symbol, range)` | `GET /TickHistory` |
| `Trading().SendOrder(...)` | `GET /OrderSend` |
| `Trading().CloseOrder(...)` | `GET /OrderClose` |
| `Trading().ModifyOrder(...)` | `GET /OrderModify` |
| `Trading().Positions(login)` | `GET /Positions` |
| `Trading().Orders(login)` | `GET /Orders` |
| `Finance().Deposit(...)` | `GET /Deposit` |
| `Finance().AdjustBalance(...)` | `GET /BalanceAdjustment` |
| `Reports().DealHistory(...)` | `GET /DealHistory` |
| `Reports().DailyForLogin(...)` | `GET /DailyRequest` |
| `Reports().DailyForGroup(...)` | `GET /DailyRequestByGroup` |
| `Raw().Get(...)` | Any manager GET endpoint with session token |
| `Raw().PostJSON(...)` | Any manager POST endpoint with session token |
