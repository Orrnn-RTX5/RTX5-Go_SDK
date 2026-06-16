package rtx5sdk

import (
	"context"
	"net/url"
	"strconv"
)

type AuthAPI struct{ client *Client }
type AccountsAPI struct{ client *Client }
type GroupsAPI struct{ client *Client }
type SymbolsAPI struct{ client *Client }
type MarketDataAPI struct{ client *Client }
type TradingAPI struct{ client *Client }
type FinanceAPI struct{ client *Client }
type ReportsAPI struct{ client *Client }
type RawAPI struct{ client *Client }

func (c *Client) Auth() AuthAPI             { return AuthAPI{client: c} }
func (c *Client) Accounts() AccountsAPI     { return AccountsAPI{client: c} }
func (c *Client) Groups() GroupsAPI         { return GroupsAPI{client: c} }
func (c *Client) Symbols() SymbolsAPI       { return SymbolsAPI{client: c} }
func (c *Client) MarketData() MarketDataAPI { return MarketDataAPI{client: c} }
func (c *Client) Trading() TradingAPI       { return TradingAPI{client: c} }
func (c *Client) Finance() FinanceAPI       { return FinanceAPI{client: c} }
func (c *Client) Reports() ReportsAPI       { return ReportsAPI{client: c} }
func (c *Client) Raw() RawAPI               { return RawAPI{client: c} }

func (a AuthAPI) LoginV2(ctx context.Context, req ManagerLoginRequest) (*Session, error) {
	if err := validateLogin(req); err != nil {
		return nil, err
	}
	body := map[string]any{
		"user":     req.User,
		"password": req.Password,
		"server":   req.Server,
	}
	if req.BrokerID != "" {
		body["broker_id"] = req.BrokerID
	}
	raw, err := a.client.do(ctx, "POST", "/api/v2/auth/login", nil, body, "", nil)
	if err != nil {
		return nil, err
	}
	return a.finishLogin(raw)
}

// Deprecated: ConnectManagerLegacy sends the manager password in the URL query
// string. Use Client.Connect or AuthAPI.LoginV2 for new integrations.
func (a AuthAPI) ConnectManagerLegacy(ctx context.Context, req ManagerLoginRequest) (*Session, error) {
	if err := validateLogin(req); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("user", req.User)
	query.Set("password", req.Password)
	query.Set("server", req.Server)
	if req.BrokerID != "" {
		query.Set("broker_id", req.BrokerID)
	}
	raw, err := a.client.do(ctx, "GET", "/Connect", query, nil, "", nil)
	if err != nil {
		return nil, err
	}
	return a.finishLogin(raw)
}

func validateLogin(req ManagerLoginRequest) error {
	if req.User == "" {
		return InvalidInputError{Message: "manager user is required"}
	}
	if req.Password == "" {
		return InvalidInputError{Message: "manager password is required"}
	}
	if req.Server == "" {
		return InvalidInputError{Message: "server is required"}
	}
	return nil
}

func (a AuthAPI) finishLogin(raw Value) (*Session, error) {
	token := tokenFromValue(raw)
	if token == "" {
		return nil, ErrMissingToken
	}
	stored := &Session{
		Token:    token,
		BrokerID: brokerIDFromValue(raw),
		Raw:      raw,
	}
	a.client.saveSession(stored)
	return &Session{
		Token:    stored.Token,
		BrokerID: stored.BrokerID,
		Raw:      stored.Raw,
	}, nil
}

func (a AccountsAPI) CreateTradingAccount(ctx context.Context, req CreateAccountRequest) (Value, error) {
	return a.CreateTradingAccountWithIdempotencyKey(ctx, req, "")
}

func (a AccountsAPI) CreateTradingAccountWithIdempotencyKey(ctx context.Context, req CreateAccountRequest, key string) (Value, error) {
	query, err := accountQuery(req)
	if err != nil {
		return nil, err
	}
	return a.client.getWithSessionIdempotent(ctx, "/AccountCreate", query, key)
}

func (a AccountsAPI) CreateAndDeposit(ctx context.Context, req CreateAccountAndDepositRequest) (Value, error) {
	return a.CreateAndDepositWithIdempotencyKey(ctx, req, "")
}

func (a AccountsAPI) CreateAndDepositWithIdempotencyKey(ctx context.Context, req CreateAccountAndDepositRequest, key string) (Value, error) {
	if err := requirePositiveAmount("amount", req.Amount); err != nil {
		return nil, err
	}
	query, err := accountQuery(req.Account)
	if err != nil {
		return nil, err
	}
	query.Set("amount", formatAmount(req.Amount))
	return a.client.getWithSessionIdempotent(ctx, "/AccountCreateAndDeposit", query, key)
}

func (a AccountsAPI) List(ctx context.Context) (Value, error) {
	return a.client.getWithSession(ctx, "/Accounts", nil)
}

func (a AccountsAPI) Details(ctx context.Context, login int64) (Value, error) {
	if err := requirePositiveInt64("login", login); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("login", strconv.FormatInt(login, 10))
	return a.client.getWithSession(ctx, "/AccountDetails", query)
}

func (g GroupsAPI) List(ctx context.Context) (Value, error) {
	return g.client.getWithSession(ctx, "/GroupList", nil)
}

func (g GroupsAPI) Get(ctx context.Context, group string) (Value, error) {
	if err := validateGroupName(group); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("group", group)
	return g.client.getWithSession(ctx, "/GroupGet", query)
}

func (g GroupsAPI) Create(ctx context.Context, config GroupConfig) (Value, error) {
	body, err := config.body()
	if err != nil {
		return nil, err
	}
	return g.client.postJSONWithSession(ctx, "/GroupCreate", nil, body)
}

func (g GroupsAPI) Update(ctx context.Context, config GroupConfig) (Value, error) {
	body, err := config.body()
	if err != nil {
		return nil, err
	}
	delete(body, "name")
	delete(body, "group")
	delete(body, "group_name")
	query := url.Values{}
	query.Set("group", config.Name)
	return g.client.postJSONWithSession(ctx, "/GroupUpdate", query, body)
}

// Deprecated: CreateRaw bypasses GroupConfig validation. Prefer Create.
func (g GroupsAPI) CreateRaw(ctx context.Context, body any) (Value, error) {
	return g.client.postJSONWithSession(ctx, "/GroupCreate", nil, body)
}

// Deprecated: UpdateRaw bypasses GroupConfig validation. Prefer Update.
func (g GroupsAPI) UpdateRaw(ctx context.Context, body any) (Value, error) {
	return g.client.postJSONWithSession(ctx, "/GroupUpdate", nil, body)
}

func (g GroupsAPI) Delete(ctx context.Context, group string) (Value, error) {
	if err := validateGroupName(group); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("group", group)
	return g.client.getWithSession(ctx, "/GroupDelete", query)
}

func (s SymbolsAPI) ListForGroup(ctx context.Context, group string) (Value, error) {
	if err := validateGroupName(group); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("group", group)
	return s.client.getWithSession(ctx, "/SymbolList", query)
}

func (s SymbolsAPI) AllNames(ctx context.Context) (Value, error) {
	return s.client.getWithSession(ctx, "/SymbolsList", nil)
}

func (s SymbolsAPI) Get(ctx context.Context, symbol string) (Value, error) {
	if err := validateSymbol(symbol); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("symbol", symbol)
	return s.client.getWithSession(ctx, "/SymbolGet", query)
}

func (m MarketDataAPI) LastTick(ctx context.Context, symbols []string) (Value, error) {
	if len(symbols) == 0 {
		return nil, InvalidInputError{Message: "symbols is required"}
	}
	query := url.Values{}
	for _, symbol := range symbols {
		if err := validateSymbol(symbol); err != nil {
			return nil, err
		}
		query.Add("symbols[]", symbol)
	}
	return m.client.getWithSession(ctx, "/TickLast", query)
}

func (m MarketDataAPI) Candles(ctx context.Context, symbol string, req TimeRangeRequest) (Value, error) {
	if err := validateSymbol(symbol); err != nil {
		return nil, err
	}
	from, to, err := normalizeTimeRange(req.From, req.To)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("symbol", symbol)
	query.Set("from", from)
	query.Set("to", to)
	return m.client.getWithSession(ctx, "/ChartRequest", query)
}

func (m MarketDataAPI) TickHistory(ctx context.Context, symbol string, req TimeRangeRequest) (Value, error) {
	if err := validateSymbol(symbol); err != nil {
		return nil, err
	}
	from, to, err := normalizeTimeRange(req.From, req.To)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("symbol", symbol)
	query.Set("from", from)
	query.Set("to", to)
	return m.client.getWithSession(ctx, "/TickHistory", query)
}

func (t TradingAPI) SendOrder(ctx context.Context, req OrderSendRequest) (Value, error) {
	return t.SendOrderWithIdempotencyKey(ctx, req, "")
}

func (t TradingAPI) SendOrderWithIdempotencyKey(ctx context.Context, req OrderSendRequest, key string) (Value, error) {
	if err := requirePositiveInt64("login", req.Login); err != nil {
		return nil, err
	}
	if err := validateSymbol(req.Symbol); err != nil {
		return nil, err
	}
	if err := validateOrderOperation(req.Operation); err != nil {
		return nil, err
	}
	if err := requirePositiveAmount("lots", req.Lots); err != nil {
		return nil, err
	}
	if err := requireOptionalFinite("price", req.Price); err != nil {
		return nil, err
	}
	if err := requireOptionalFinite("stoploss", req.StopLoss); err != nil {
		return nil, err
	}
	if err := requireOptionalFinite("takeprofit", req.TakeProfit); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("login", strconv.FormatInt(req.Login, 10))
	query.Set("symbol", req.Symbol)
	query.Set("operation", req.Operation)
	query.Set("lots", formatAmount(req.Lots))
	setFloat(query, "price", req.Price)
	setFloat(query, "stoploss", req.StopLoss)
	setFloat(query, "takeprofit", req.TakeProfit)
	return t.client.getWithSessionIdempotent(ctx, "/OrderSend", query, key)
}

func (t TradingAPI) CloseOrder(ctx context.Context, req OrderCloseRequest) (Value, error) {
	return t.CloseOrderWithIdempotencyKey(ctx, req, "")
}

func (t TradingAPI) CloseOrderWithIdempotencyKey(ctx context.Context, req OrderCloseRequest, key string) (Value, error) {
	if err := requirePositiveInt64("ticket", req.Ticket); err != nil {
		return nil, err
	}
	if req.Lots != nil {
		if err := requirePositiveAmount("lots", *req.Lots); err != nil {
			return nil, err
		}
	}
	query := url.Values{}
	query.Set("ticket", strconv.FormatInt(req.Ticket, 10))
	setFloat(query, "lots", req.Lots)
	return t.client.getWithSessionIdempotent(ctx, "/OrderClose", query, key)
}

func (t TradingAPI) ModifyOrder(ctx context.Context, req OrderModifyRequest) (Value, error) {
	return t.ModifyOrderWithIdempotencyKey(ctx, req, "")
}

func (t TradingAPI) ModifyOrderWithIdempotencyKey(ctx context.Context, req OrderModifyRequest, key string) (Value, error) {
	if err := requirePositiveInt64("ticket", req.Ticket); err != nil {
		return nil, err
	}
	if req.Price != nil {
		return nil, InvalidInputError{Message: "price is not supported by the current /OrderModify endpoint"}
	}
	if req.StopLoss == nil || req.TakeProfit == nil {
		return nil, InvalidInputError{Message: "stop_loss and take_profit must both be provided; use 0 explicitly to clear either value"}
	}
	if err := requireOptionalFinite("stoploss", req.StopLoss); err != nil {
		return nil, err
	}
	if err := requireOptionalFinite("takeprofit", req.TakeProfit); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("ticket", strconv.FormatInt(req.Ticket, 10))
	setFloat(query, "stoploss", req.StopLoss)
	setFloat(query, "takeprofit", req.TakeProfit)
	return t.client.getWithSessionIdempotent(ctx, "/OrderModify", query, key)
}

func (t TradingAPI) Positions(ctx context.Context, login int64) (Value, error) {
	if err := requirePositiveInt64("login", login); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Add("logins[]", strconv.FormatInt(login, 10))
	return t.client.getWithSession(ctx, "/Positions", query)
}

func (t TradingAPI) Orders(ctx context.Context, login int64) (Value, error) {
	if err := requirePositiveInt64("login", login); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Add("logins[]", strconv.FormatInt(login, 10))
	return t.client.getWithSession(ctx, "/Orders", query)
}

func (f FinanceAPI) Deposit(ctx context.Context, login int64, amount float64, comment string) (Value, error) {
	return f.DepositWithIdempotencyKey(ctx, login, amount, comment, "")
}

func (f FinanceAPI) DepositWithIdempotencyKey(ctx context.Context, login int64, amount float64, comment string, key string) (Value, error) {
	if err := requirePositiveInt64("login", login); err != nil {
		return nil, err
	}
	if err := requirePositiveAmount("amount", amount); err != nil {
		return nil, err
	}
	if err := validateOptionalComment(comment); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("login", strconv.FormatInt(login, 10))
	query.Set("amount", formatAmount(amount))
	if comment != "" {
		query.Set("comment", comment)
	}
	return f.client.getWithSessionIdempotent(ctx, "/Deposit", query, key)
}

func (f FinanceAPI) AdjustBalance(ctx context.Context, req BalanceAdjustmentRequest) (Value, error) {
	return f.AdjustBalanceWithIdempotencyKey(ctx, req, "")
}

func (f FinanceAPI) AdjustBalanceWithIdempotencyKey(ctx context.Context, req BalanceAdjustmentRequest, key string) (Value, error) {
	if err := requirePositiveInt64("login", req.Login); err != nil {
		return nil, err
	}
	if err := requirePositiveAmount("amount", req.Amount); err != nil {
		return nil, err
	}
	if err := validateBalanceAction(req.Action); err != nil {
		return nil, err
	}
	if err := validateOptionalComment(req.Comment); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("login", strconv.FormatInt(req.Login, 10))
	query.Set("amount", formatAmount(req.Amount))
	query.Set("action", string(req.Action))
	if req.Comment != "" {
		query.Set("comment", req.Comment)
	}
	return f.client.getWithSessionIdempotent(ctx, "/BalanceAdjustment", query, key)
}

func (f FinanceAPI) Withdraw(ctx context.Context, login int64, amount float64, comment string) (Value, error) {
	return f.AdjustBalance(ctx, BalanceAdjustmentRequest{
		Login:   login,
		Amount:  amount,
		Action:  BalanceActionWithdraw,
		Comment: comment,
	})
}

func (f FinanceAPI) WithdrawWithIdempotencyKey(ctx context.Context, login int64, amount float64, comment string, key string) (Value, error) {
	return f.AdjustBalanceWithIdempotencyKey(ctx, BalanceAdjustmentRequest{
		Login:   login,
		Amount:  amount,
		Action:  BalanceActionWithdraw,
		Comment: comment,
	}, key)
}

func (r ReportsAPI) DealHistory(ctx context.Context, req LoginTimeRangeRequest) (Value, error) {
	if err := requirePositiveInt64("login", req.Login); err != nil {
		return nil, err
	}
	from, to, err := normalizeTimeRange(req.From, req.To)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("login", strconv.FormatInt(req.Login, 10))
	query.Set("from", from)
	query.Set("to", to)
	return r.client.getWithSession(ctx, "/DealHistory", query)
}

func (r ReportsAPI) DailyForLogin(ctx context.Context, req LoginTimeRangeRequest) (Value, error) {
	if err := requirePositiveInt64("login", req.Login); err != nil {
		return nil, err
	}
	from, to, err := normalizeTimeRange(req.From, req.To)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("login", strconv.FormatInt(req.Login, 10))
	query.Set("from", from)
	query.Set("to", to)
	return r.client.getWithSession(ctx, "/DailyRequest", query)
}

func (r ReportsAPI) DailyForGroup(ctx context.Context, req GroupTimeRangeRequest) (Value, error) {
	if err := validateGroupName(req.Group); err != nil {
		return nil, err
	}
	from, to, err := normalizeTimeRange(req.From, req.To)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("group", req.Group)
	query.Set("from", from)
	query.Set("to", to)
	return r.client.getWithSession(ctx, "/DailyRequestByGroup", query)
}

func (r RawAPI) Get(ctx context.Context, path string, query url.Values) (Value, error) {
	if err := validateRawPath(path); err != nil {
		return nil, err
	}
	return r.client.getWithSession(ctx, path, query)
}

func (r RawAPI) PostJSON(ctx context.Context, path string, query url.Values, body any) (Value, error) {
	if err := validateRawPath(path); err != nil {
		return nil, err
	}
	return r.client.postJSONWithSession(ctx, path, query, body)
}

func accountQuery(req CreateAccountRequest) (url.Values, error) {
	if req.MasterPassword == "" {
		return nil, InvalidInputError{Message: "master_password is required"}
	}
	if req.Group == "" {
		return nil, InvalidInputError{Message: "group is required"}
	}
	for field, value := range map[string]*float64{
		"Credit":          req.Credit,
		"MarginCallLevel": req.MarginCallLevel,
		"StopOutLevel":    req.StopOutLevel,
		"CommissionRate":  req.CommissionRate,
	} {
		if err := requireOptionalFinite(field, value); err != nil {
			return nil, err
		}
	}
	query := url.Values{}
	query.Set("master_pass", req.MasterPassword)
	query.Set("Group", req.Group)
	setString(query, "investor_pass", req.InvestorPassword)
	if req.Leverage != nil {
		query.Set("Leverage", strconv.FormatUint(uint64(*req.Leverage), 10))
	}
	setString(query, "FirstName", req.FirstName)
	setString(query, "LastName", req.LastName)
	setString(query, "Email", req.Email)
	setString(query, "Phone", req.Phone)
	setString(query, "Country", req.Country)
	setString(query, "City", req.City)
	setString(query, "Address", req.Address)
	setString(query, "ZipCode", req.ZipCode)
	setString(query, "Company", req.Company)
	setString(query, "Comment", req.Comment)
	setString(query, "Currency", req.Currency)
	setFloat(query, "Credit", req.Credit)
	setString(query, "Status", req.Status)
	if req.AccountMode != "" {
		query.Set("AccountMode", string(req.AccountMode))
	}
	setFloat(query, "MarginCallLevel", req.MarginCallLevel)
	setFloat(query, "StopOutLevel", req.StopOutLevel)
	setFloat(query, "CommissionRate", req.CommissionRate)
	if req.SwapFree != nil {
		query.Set("SwapFree", strconv.FormatBool(*req.SwapFree))
	}
	if req.NegativeBalanceProtection != nil {
		query.Set("NegativeBalanceProtection", strconv.FormatBool(*req.NegativeBalanceProtection))
	}
	setString(query, "MQID", req.MQID)
	return query, nil
}

func setString(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

func setFloat(query url.Values, key string, value *float64) {
	if value != nil {
		query.Set(key, formatAmount(*value))
	}
}
