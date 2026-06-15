package rtx5sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultRequestTimeout = 30 * time.Second
	defaultConnectTimeout = 10 * time.Second
	defaultMaxResponse    = 32 << 20
	maxOpaqueTokenLen     = 4096
)

var idempotencyCounter uint64

type Session struct {
	Token    string
	BrokerID string
	Raw      Value
}

type Client struct {
	http            *http.Client
	baseURL         *url.URL
	brokerID        string
	managerLogin    string
	managerPassword string
	server          string
	maxResponse     int64

	mu      sync.RWMutex
	session *Session
}

type ClientBuilder struct {
	baseURL           string
	brokerID          string
	managerLogin      string
	managerPassword   string
	server            string
	httpClient        *http.Client
	allowInsecureHTTP bool
	maxResponse       int64
}

func New(baseURL string) (*Client, error) {
	return Builder().BaseURL(baseURL).Build()
}

func Builder() *ClientBuilder {
	return &ClientBuilder{}
}

func (b *ClientBuilder) BaseURL(baseURL string) *ClientBuilder {
	b.baseURL = baseURL
	return b
}

func (b *ClientBuilder) BrokerID(brokerID string) *ClientBuilder {
	b.brokerID = brokerID
	return b
}

func (b *ClientBuilder) ManagerLogin(login string) *ClientBuilder {
	b.managerLogin = login
	return b
}

func (b *ClientBuilder) ManagerPassword(password string) *ClientBuilder {
	b.managerPassword = password
	return b
}

func (b *ClientBuilder) Server(server string) *ClientBuilder {
	b.server = server
	return b
}

func (b *ClientBuilder) HTTPClient(client *http.Client) *ClientBuilder {
	b.httpClient = client
	return b
}

func (b *ClientBuilder) AllowInsecureHTTP(allow bool) *ClientBuilder {
	b.allowInsecureHTTP = allow
	return b
}

func (b *ClientBuilder) MaxResponseBytes(maxBytes int64) *ClientBuilder {
	b.maxResponse = maxBytes
	return b
}

func (b *ClientBuilder) Build() (*Client, error) {
	if b.baseURL == "" {
		return nil, MissingConfigError{Name: "base_url"}
	}
	parsed, err := url.Parse(b.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid base url: %s", b.baseURL)
	}
	isHTTPS := parsed.Scheme == "https"
	if !isHTTPS && !b.allowInsecureHTTP && !isLoopback(parsed) {
		return nil, InsecureBaseURLError{URL: parsed.String()}
	}
	httpClient := b.httpClient
	if httpClient == nil {
		httpClient = defaultHTTPClient(isHTTPS)
	}
	maxResponse := b.maxResponse
	if maxResponse == 0 {
		maxResponse = defaultMaxResponse
	}
	if maxResponse < 0 {
		return nil, InvalidInputError{Message: "max response bytes must not be negative"}
	}
	return &Client{
		http:            httpClient,
		baseURL:         parsed,
		brokerID:        b.brokerID,
		managerLogin:    b.managerLogin,
		managerPassword: b.managerPassword,
		server:          b.server,
		maxResponse:     maxResponse,
	}, nil
}

func defaultHTTPClient(httpsOnly bool) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{
		Timeout: defaultConnectTimeout,
	}).DialContext
	client := &http.Client{
		Timeout:   defaultRequestTimeout,
		Transport: transport,
	}
	if httpsOnly {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "https" {
				return fmt.Errorf("refusing plaintext redirect to %s", req.URL.String())
			}
			return nil
		}
	}
	return client
}

func (c *Client) BaseURL() *url.URL {
	copyURL := *c.baseURL
	return &copyURL
}

func (c *Client) Connect(ctx context.Context) (*Session, error) {
	req, err := c.managerLoginRequest()
	if err != nil {
		return nil, err
	}
	return c.Auth().LoginV2(ctx, req)
}

func (c *Client) Disconnect(ctx context.Context) (Value, error) {
	value, err := c.getWithSession(ctx, "/Disconnect", nil)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.session = nil
	c.mu.Unlock()
	return value, nil
}

func (c *Client) Session() *Session {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.session == nil {
		return nil
	}
	cp := *c.session
	return &cp
}

func (c *Client) SessionToken() (string, error) {
	session := c.Session()
	if session == nil {
		return "", ErrNotConnected
	}
	return session.Token, nil
}

func (c *Client) saveSession(session *Session) {
	if session == nil {
		return
	}
	cp := *session
	c.mu.Lock()
	c.session = &cp
	c.mu.Unlock()
}

func (c *Client) managerLoginRequest() (ManagerLoginRequest, error) {
	if c.managerLogin == "" {
		return ManagerLoginRequest{}, MissingConfigError{Name: "manager_login"}
	}
	if c.managerPassword == "" {
		return ManagerLoginRequest{}, MissingConfigError{Name: "manager_password"}
	}
	if c.server == "" {
		return ManagerLoginRequest{}, MissingConfigError{Name: "server"}
	}
	return ManagerLoginRequest{
		User:     c.managerLogin,
		Password: c.managerPassword,
		Server:   c.server,
		BrokerID: c.brokerID,
	}, nil
}

func (c *Client) getWithSession(ctx context.Context, path string, query url.Values) (Value, error) {
	token, err := c.bearerToken()
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodGet, path, query, nil, token, nil)
}

func (c *Client) getWithSessionIdempotent(ctx context.Context, path string, query url.Values, key string) (Value, error) {
	headers, err := idempotencyHeaders(key)
	if err != nil {
		return nil, err
	}
	token, err := c.bearerToken()
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodGet, path, query, nil, token, headers)
}

func (c *Client) postJSONWithSession(ctx context.Context, path string, query url.Values, body any) (Value, error) {
	return c.postJSONWithSessionIdempotent(ctx, path, query, body, "")
}

func (c *Client) postJSONWithSessionIdempotent(ctx context.Context, path string, query url.Values, body any, key string) (Value, error) {
	headers, err := idempotencyHeaders(key)
	if err != nil {
		return nil, err
	}
	token, err := c.bearerToken()
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodPost, path, query, body, token, headers)
}

func (c *Client) bearerToken() (string, error) {
	token, err := c.SessionToken()
	if err != nil {
		return "", err
	}
	return accessTokenPart(token), nil
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, bearerToken string, headers http.Header) (Value, error) {
	endpoint := c.urlFor(path, query)
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	text, err := c.readResponseBody(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, APIError{StatusCode: resp.StatusCode, Body: string(text)}
	}
	return parseResponse(text, resp.Header.Get("Content-Type"))
}

func (c *Client) readResponseBody(body io.Reader) ([]byte, error) {
	if c.maxResponse <= 0 {
		return io.ReadAll(body)
	}
	limited := io.LimitReader(body, c.maxResponse+1)
	text, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(text)) > c.maxResponse {
		return nil, ResponseTooLargeError{MaxBytes: c.maxResponse}
	}
	return text, nil
}

func (c *Client) urlFor(path string, query url.Values) *url.URL {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimLeft(path, "/")
	endpoint.RawQuery = query.Encode()
	return &endpoint
}

func tokenFromValue(value Value) string {
	obj, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"access_token", "token", "session_token", "id"} {
		if v, ok := obj[key].(string); ok {
			return v
		}
	}
	return ""
}

func brokerIDFromValue(value Value) string {
	obj, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	if v, ok := obj["broker_id"].(string); ok {
		return v
	}
	return ""
}

func parseResponse(body []byte, contentType string) (Value, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return map[string]any{}, nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
		return decoded, nil
	}
	if mediatype, _, err := mime.ParseMediaType(contentType); err == nil && strings.Contains(mediatype, "html") {
		return nil, InvalidInputError{Message: ErrInvalidResponse.Error() + " (looks like an HTML/error page, not a token or JSON payload)"}
	}
	if len(trimmed) > maxOpaqueTokenLen || looksLikeMarkup(trimmed) {
		return nil, InvalidInputError{Message: ErrInvalidResponse.Error() + " (looks like an HTML/error page, not a token or JSON payload)"}
	}
	return map[string]any{"token": trimmed, "raw": trimmed}, nil
}

func looksLikeMarkup(body string) bool {
	head := strings.TrimSpace(body)
	if strings.HasPrefix(head, "<") {
		return true
	}
	return strings.Contains(body, "<") && strings.Contains(body, ">")
}

func isLoopback(u *url.URL) bool {
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func accessTokenPart(token string) string {
	parts := strings.Split(token, "|")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0]
	}
	return token
}

func genIdempotencyKey() string {
	seq := atomic.AddUint64(&idempotencyCounter, 1) - 1
	return fmt.Sprintf("rtx5-%032x-%016x", time.Now().UnixNano(), seq)
}

func idempotencyHeaders(key string) (http.Header, error) {
	if key == "" {
		key = genIdempotencyKey()
	}
	if strings.TrimSpace(key) == "" {
		return nil, InvalidInputError{Message: "idempotency key must not be empty"}
	}
	if !validHeaderValue(key) {
		return nil, InvalidInputError{Message: "idempotency key contains invalid bytes"}
	}
	headers := make(http.Header)
	headers.Set("Idempotency-Key", key)
	return headers, nil
}

func validHeaderValue(value string) bool {
	for _, r := range value {
		if r == '\t' {
			continue
		}
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func trimEmpty(value string) bool {
	return strings.TrimSpace(value) == ""
}
