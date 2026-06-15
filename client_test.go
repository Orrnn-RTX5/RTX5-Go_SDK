package rtx5sdk

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildsURLWithQueryParams(t *testing.T) {
	client, err := New("https://broker.example.com")
	if err != nil {
		t.Fatal(err)
	}
	q, err := accountQuery(CreateAccountRequest{
		MasterPassword: "pw",
		Group:          "demo/STD",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := client.urlFor("/AccountCreate", q).String()
	want := "https://broker.example.com/AccountCreate?Group=demo%2FSTD&master_pass=pw"
	if got != want {
		t.Fatalf("url mismatch\nwant %s\ngot  %s", want, got)
	}
}

func TestParseResponseAcceptsJSONAndOpaqueToken(t *testing.T) {
	jsonValue, err := parseResponse([]byte(`{"access_token":"abc"}`), "application/json")
	if err != nil {
		t.Fatal(err)
	}
	if got := tokenFromValue(jsonValue); got != "abc" {
		t.Fatalf("token = %q", got)
	}
	tokenValue, err := parseResponse([]byte("rtx5_rt_deadbeef"), "text/plain")
	if err != nil {
		t.Fatal(err)
	}
	if got := tokenFromValue(tokenValue); got != "rtx5_rt_deadbeef" {
		t.Fatalf("token = %q", got)
	}
}

func TestParseResponseRejectsMarkupAndOversizedToken(t *testing.T) {
	if _, err := parseResponse([]byte("<html>blocked</html>"), "text/html"); err == nil {
		t.Fatal("expected markup rejection")
	}
	if _, err := parseResponse([]byte(strings.Repeat("a", maxOpaqueTokenLen+1)), "text/plain"); err == nil {
		t.Fatal("expected oversized token rejection")
	}
}

func TestRejectsPlaintextPublicHTTPUnlessOptedIn(t *testing.T) {
	_, err := New("http://broker.example.com")
	var insecure InsecureBaseURLError
	if !errors.As(err, &insecure) {
		t.Fatalf("expected InsecureBaseURLError, got %T %[1]v", err)
	}
	if _, err := Builder().BaseURL("http://broker.example.com").AllowInsecureHTTP(true).Build(); err != nil {
		t.Fatal(err)
	}
	if _, err := New("http://127.0.0.1:8090"); err != nil {
		t.Fatal(err)
	}
}

func TestAccessTokenPart(t *testing.T) {
	cases := map[string]string{
		"access|refresh": "access",
		"opaque":         "opaque",
		"a|b|c":          "a|b|c",
		"|refresh":       "|refresh",
		"access|":        "access|",
	}
	for input, want := range cases {
		if got := accessTokenPart(input); got != want {
			t.Fatalf("%q -> %q, want %q", input, got, want)
		}
	}
}

func TestIdempotencyHeaders(t *testing.T) {
	headers, err := idempotencyHeaders("stable-key")
	if err != nil {
		t.Fatal(err)
	}
	if got := headers.Get("Idempotency-Key"); got != "stable-key" {
		t.Fatalf("key = %q", got)
	}
	if _, err := idempotencyHeaders("   "); err == nil {
		t.Fatal("expected blank key rejection")
	}
	if _, err := idempotencyHeaders("bad\r\nkey"); err == nil {
		t.Fatal("expected invalid byte rejection")
	}
	auto, err := idempotencyHeaders("")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(auto.Get("Idempotency-Key"), "rtx5-") {
		t.Fatalf("auto key = %q", auto.Get("Idempotency-Key"))
	}
}

func TestAPIErrorRedactsBody(t *testing.T) {
	err := APIError{StatusCode: 401, Body: "token=SECRET alice@example.com"}
	if strings.Contains(err.Error(), "SECRET") || strings.Contains(err.Error(), "alice@example.com") {
		t.Fatalf("body leaked in error: %s", err.Error())
	}
	if err.APIBody() == "" {
		t.Fatal("body accessor should preserve raw body")
	}
}

func TestAccountValidationAndFormatting(t *testing.T) {
	_, err := accountQuery(CreateAccountRequest{MasterPassword: "", Group: "demo/STD"})
	if err == nil {
		t.Fatal("expected master password error")
	}
	badCredit := math.NaN()
	_, err = accountQuery(CreateAccountRequest{MasterPassword: "pw", Group: "demo/STD", Credit: &badCredit})
	if err == nil {
		t.Fatal("expected non-finite credit error")
	}
	if got := formatAmount(0.1 + 0.2); got != "0.3" {
		t.Fatalf("formatAmount = %q", got)
	}
}

func TestGroupValidation(t *testing.T) {
	zero := uint32(0)
	cfg := NewGroupConfig("demo/STD")
	cfg.Leverage = &zero
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected leverage error")
	}
	mc, so := 50.0, 80.0
	cfg.Leverage = nil
	cfg.MarginCallLevel = &mc
	cfg.StopOutLevel = &so
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected stop-out error")
	}
}

func TestGroupConfigBodyUsesManagerBackendJSONKeys(t *testing.T) {
	leverage := uint32(100)
	spread, contractSize, marginCall, stopOut := 1.5, 100000.0, 100.0, 50.0
	cfg := NewGroupConfig("demo/STD")
	cfg.Currency = "USD"
	cfg.Leverage = &leverage
	cfg.Spread = &spread
	cfg.ContractSize = &contractSize
	cfg.MarginCallLevel = &marginCall
	cfg.StopOutLevel = &stopOut

	body, err := cfg.body()
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"name", "currency", "leverage", "spreadMarkup", "contractSize", "marginCallLevel", "stopOutLevel"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("missing backend JSON key %q in %#v", key, body)
		}
	}
	for _, key := range []string{"spread", "contract_size", "margin_call_level", "stop_out_level"} {
		if _, ok := body[key]; ok {
			t.Fatalf("unexpected stale JSON key %q in %#v", key, body)
		}
	}
}

func TestGroupUpdateTargetsGroupInQueryAndSendsOnlyUpdateFields(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(`{"updated":true}`))
	}))
	defer server.Close()

	client, err := Builder().BaseURL(server.URL).Build()
	if err != nil {
		t.Fatal(err)
	}
	client.saveSession(&Session{Token: "access"})

	marginCall := 100.0
	cfg := NewGroupConfig("demo/STD")
	cfg.MarginCallLevel = &marginCall
	if _, err := client.Groups().Update(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "/GroupUpdate?group=demo%2FSTD") {
		t.Fatalf("group update target mismatch: %s", gotPath)
	}
	if _, ok := gotBody["name"]; ok {
		t.Fatalf("update body should not include name target: %#v", gotBody)
	}
	if gotBody["marginCallLevel"] != marginCall {
		t.Fatalf("missing marginCallLevel update field: %#v", gotBody)
	}
}

func TestSessionUsesBearerOnly(t *testing.T) {
	var rawQuery string
	var auth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery = r.URL.RawQuery
		auth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client, err := Builder().BaseURL(server.URL).Build()
	if err != nil {
		t.Fatal(err)
	}
	client.saveSession(&Session{Token: "access|refresh"})
	if _, err := client.Groups().List(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rawQuery, "id=") {
		t.Fatalf("session token leaked into query: %s", rawQuery)
	}
	if auth != "Bearer access" {
		t.Fatalf("auth = %q", auth)
	}
}

func TestPositionsAndOrdersUseBackendLoginsArrayParam(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.String())
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client, err := Builder().BaseURL(server.URL).Build()
	if err != nil {
		t.Fatal(err)
	}
	client.saveSession(&Session{Token: "access"})

	if _, err := client.Trading().Positions(context.Background(), 1001); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Trading().Orders(context.Background(), 1001); err != nil {
		t.Fatal(err)
	}

	if len(requests) != 2 {
		t.Fatalf("requests = %#v", requests)
	}
	for _, got := range requests {
		if !strings.Contains(got, "logins%5B%5D=1001") {
			t.Fatalf("request should use logins[] query param, got %s", got)
		}
		if strings.Contains(got, "login=1001") {
			t.Fatalf("request used stale login query param: %s", got)
		}
	}
}

func TestReturnedSessionDoesNotAliasStoredSession(t *testing.T) {
	client, err := New("http://127.0.0.1:8090")
	if err != nil {
		t.Fatal(err)
	}
	session, err := client.Auth().finishLogin(map[string]any{"access_token": "original"})
	if err != nil {
		t.Fatal(err)
	}
	session.Token = "mutated"
	got, err := client.SessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if got != "original" {
		t.Fatalf("stored session mutated through returned pointer: %q", got)
	}
}

func TestSensitiveStringRedaction(t *testing.T) {
	login := ManagerLoginRequest{User: "9001", Password: "secret", Server: "srv"}
	if strings.Contains(login.String(), "secret") {
		t.Fatal("manager password leaked")
	}
	acct := CreateAccountRequest{MasterPassword: "trader-secret", InvestorPassword: "investor-secret", Group: "demo"}
	if text := acct.String(); strings.Contains(text, "trader-secret") || strings.Contains(text, "investor-secret") {
		t.Fatalf("account password leaked: %s", text)
	}
	session := Session{Token: "secret-token", BrokerID: "gta_broker", Raw: map[string]any{"access_token": "secret-token"}}
	if text := session.String(); strings.Contains(text, "secret-token") || strings.Contains(text, "access_token") {
		t.Fatalf("session leaked sensitive data: %s", text)
	}
}

func TestResponseBodyLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"too":"large"}`))
	}))
	defer server.Close()

	client, err := Builder().BaseURL(server.URL).MaxResponseBytes(4).Build()
	if err != nil {
		t.Fatal(err)
	}
	client.saveSession(&Session{Token: "token"})
	_, err = client.Groups().List(context.Background())
	var tooLarge ResponseTooLargeError
	if !errors.As(err, &tooLarge) {
		t.Fatalf("expected ResponseTooLargeError, got %T %[1]v", err)
	}
}

func TestResponseBodyExactLimitSucceeds(t *testing.T) {
	body := []byte(`{"ok":true}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()

	client, err := Builder().BaseURL(server.URL).MaxResponseBytes(int64(len(body))).Build()
	if err != nil {
		t.Fatal(err)
	}
	client.saveSession(&Session{Token: "token"})
	value, err := client.Groups().List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	obj, ok := value.(map[string]any)
	if !ok || obj["ok"] != true {
		t.Fatalf("unexpected response: %#v", value)
	}
}

func TestTypedValidationRejectsBadTradingInputsBeforeSessionCheck(t *testing.T) {
	client, err := New("http://127.0.0.1:8090")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Trading().SendOrder(context.Background(), OrderSendRequest{
		Login:     0,
		Symbol:    "EURUSD",
		Operation: "buy",
		Lots:      0.1,
	})
	if err == nil {
		t.Fatal("expected login validation error")
	}
	_, err = client.Trading().SendOrder(context.Background(), OrderSendRequest{
		Login:     1,
		Symbol:    "EUR USD",
		Operation: "buy",
		Lots:      0.1,
	})
	if err == nil {
		t.Fatal("expected symbol validation error")
	}
	_, err = client.Trading().SendOrder(context.Background(), OrderSendRequest{
		Login:     1,
		Symbol:    "EURUSD",
		Operation: "custom_op",
		Lots:      0.1,
	})
	if err == nil {
		t.Fatal("expected operation validation error")
	}
}

func TestOrderOperationMatrixMatchesCurrentManagerBackend(t *testing.T) {
	client, err := New("http://127.0.0.1:8090")
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"BUY", "SELL", "BUY_LIMIT", "SELL_LIMIT", "BUY_STOP", "SELL_STOP"} {
		_, err := client.Trading().SendOrder(context.Background(), OrderSendRequest{
			Login:     1,
			Symbol:    "EURUSD",
			Operation: operation,
			Lots:      0.1,
		})
		if !errors.Is(err, ErrNotConnected) {
			t.Fatalf("%s should pass validation and reach session check, got %T %[2]v", operation, err)
		}
	}
	for _, operation := range []string{"BUY_STOP_LIMIT", "SELL_STOP_LIMIT"} {
		_, err := client.Trading().SendOrder(context.Background(), OrderSendRequest{
			Login:     1,
			Symbol:    "EURUSD",
			Operation: operation,
			Lots:      0.1,
		})
		if err == nil || errors.Is(err, ErrNotConnected) {
			t.Fatalf("%s should fail typed validation before session check, got %T %[2]v", operation, err)
		}
	}
}

func TestCloseAndModifyOrderAreTicketBased(t *testing.T) {
	client, err := New("http://127.0.0.1:8090")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Trading().CloseOrder(context.Background(), OrderCloseRequest{Ticket: 123})
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("close should pass ticket-only validation and reach session check, got %T %[1]v", err)
	}
	sl, tp := 1.05, 1.12
	_, err = client.Trading().ModifyOrder(context.Background(), OrderModifyRequest{
		Ticket:     123,
		StopLoss:   &sl,
		TakeProfit: &tp,
	})
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("modify should pass ticket-only validation and reach session check, got %T %[1]v", err)
	}
}

func TestModifyOrderRejectsUnsupportedPriceAndImplicitStopClears(t *testing.T) {
	client, err := New("http://127.0.0.1:8090")
	if err != nil {
		t.Fatal(err)
	}
	price := 1.2
	_, err = client.Trading().ModifyOrder(context.Background(), OrderModifyRequest{
		Ticket: 123,
		Price:  &price,
	})
	if err == nil {
		t.Fatal("expected price validation error")
	}
	sl := 1.05
	_, err = client.Trading().ModifyOrder(context.Background(), OrderModifyRequest{
		Ticket:   123,
		StopLoss: &sl,
	})
	if err == nil {
		t.Fatal("expected explicit stop-loss/take-profit validation error")
	}
}

func TestFinanceAndReportValidation(t *testing.T) {
	client, err := New("http://127.0.0.1:8090")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Finance().Deposit(context.Background(), 0, 1, "")
	if err == nil {
		t.Fatal("expected login validation error")
	}
	_, err = client.Finance().AdjustBalance(context.Background(), BalanceAdjustmentRequest{
		Login:  1,
		Amount: 1,
		Action: BalanceAction("unsupported"),
	})
	if err == nil {
		t.Fatal("expected balance action validation error")
	}
	_, err = client.Reports().DealHistory(context.Background(), LoginTimeRangeRequest{Login: 1, From: "", To: "2"})
	if err == nil {
		t.Fatal("expected time range validation error")
	}
}

func TestRawPathValidation(t *testing.T) {
	client, err := New("http://127.0.0.1:8090")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"AccountCreate", "/../AccountCreate", "/Account Create"} {
		if _, err := client.Raw().Get(context.Background(), path, nil); err == nil {
			t.Fatalf("expected raw path validation error for %q", path)
		}
	}
}
