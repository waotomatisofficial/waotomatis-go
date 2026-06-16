// Package waotomatis is the official Go SDK for WAOtomatis — headless WhatsApp
// (WhatsApp Business / Cloud API) for developers.
//
// Construct a client with your API key and scope to a session to send messages:
//
//	client := waotomatis.New(os.Getenv("WAO_API_KEY"))
//
//	msg, _ := client.Sessions("sess_123").Messages.Send(&waotomatis.Message{
//	    To:   "628123456789",
//	    Type: "text",
//	    Text: "Halo dari WAOtomatis 👋",
//	})
//
//	fmt.Println(msg.ID) // msg_abc123
//
// Attach a context.Context per call with waotomatis.WithContext(ctx) for
// cancellation and deadlines. The SDK depends only on the Go standard library.
package waotomatis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is the production API endpoint used when no base URL is set.
const DefaultBaseURL = "https://api.waotomatis.com"

const (
	defaultMaxRetries = 2
	defaultTimeout    = 60 * time.Second
	maxRetryAfter     = 60 * time.Second
	userAgent         = "waotomatis-go/0.1.0"
)

// retryableStatus holds the only statuses that can plausibly succeed on an
// identical retry (the server's sole 409 is a permanent conflict).
var retryableStatus = map[int]bool{408: true, 429: true, 500: true, 502: true, 503: true, 504: true}

// idempotentMethods are safe to retry without an idempotency key.
var idempotentMethods = map[string]bool{http.MethodGet: true, http.MethodHead: true, http.MethodPut: true, http.MethodDelete: true}

// Client is the WAOtomatis API client. It is safe for concurrent use by
// multiple goroutines. Construct it with New.
type Client struct {
	apiKey         string
	baseURL        string
	httpClient     *http.Client
	maxRetries     int
	timeout        time.Duration
	defaultHeaders map[string]string
}

// Option configures a Client. Pass options to New, e.g.
// New(key, WithBaseURL("https://..."), WithMaxRetries(0)).
type Option func(*Client)

// WithBaseURL overrides the API base URL (default DefaultBaseURL). A trailing
// slash is trimmed.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(baseURL, "/") }
}

// WithHTTPClient injects a custom *http.Client (for proxies, custom transports,
// or tests). Its Timeout is ignored — per-request deadlines are handled via
// context and WithTimeout.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithMaxRetries sets the maximum automatic retries for transient failures
// (408/429/5xx and network errors) on idempotent requests. Default 2; 0 disables.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		if n >= 0 {
			c.maxRetries = n
		}
	}
}

// WithTimeout sets the per-request timeout applied on top of the caller's
// context. Default 60s; <= 0 disables the client-level timeout (context still
// applies).
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithDefaultHeader adds a header sent on every request (e.g. "X-Org-Id").
// Call it multiple times to set several headers.
func WithDefaultHeader(key, value string) Option {
	return func(c *Client) {
		if c.defaultHeaders == nil {
			c.defaultHeaders = map[string]string{}
		}
		c.defaultHeaders[key] = value
	}
}

// New creates a Client authenticated with the given API key. It panics if
// apiKey is empty (a programming error). Apply options to customize behavior.
func New(apiKey string, opts ...Option) *Client {
	if apiKey == "" {
		panic("waotomatis: apiKey is required")
	}
	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{},
		maxRetries: defaultMaxRetries,
		timeout:    defaultTimeout,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Sessions scopes the client to a single session:
//
//	client.Sessions("sess_123").Messages.Send(ctx, msg)
//
// See SessionsService for the list/get/delete methods that are not session-scoped.
func (c *Client) Sessions(id string) *SessionResource {
	return newSessionResource(c, id)
}

// SessionList is the non-scoped sessions sub-API (list, get, delete by id).
// Access it via Client.SessionList().
func (c *Client) SessionList() *SessionsService {
	return &SessionsService{client: c}
}

// ── Internal request plumbing ───────────────────────────────────────────────

// callOptions are per-call overrides threaded through resource methods.
type callOptions struct {
	ctx            context.Context
	idempotencyKey string
	headers        map[string]string
}

// CallOption customizes a single request.
type CallOption func(*callOptions)

// WithContext attaches a context.Context to one call for cancellation and
// deadlines. Without it, calls use context.Background():
//
//	client.Sessions(id).Messages.Send(msg, waotomatis.WithContext(ctx))
func WithContext(ctx context.Context) CallOption {
	return func(o *callOptions) { o.ctx = ctx }
}

// WithIdempotencyKey sets the Idempotency-Key header for one call, making a
// non-idempotent verb safe to retry.
func WithIdempotencyKey(key string) CallOption {
	return func(o *callOptions) { o.idempotencyKey = key }
}

// WithHeader merges an extra header onto a single request.
func WithHeader(key, value string) CallOption {
	return func(o *callOptions) {
		if o.headers == nil {
			o.headers = map[string]string{}
		}
		o.headers[key] = value
	}
}

// request is the low-level description of one HTTP call.
type request struct {
	method string
	path   string
	// body, when non-nil, is JSON-encoded.
	body any
	// rawBody, when non-nil, is sent as-is (e.g. multipart). It takes precedence
	// over body. rawContentType is its Content-Type. A request with a rawBody is
	// never auto-retried (the reader is consumed by the first attempt).
	rawBody        []byte
	rawContentType string
	query          map[string]string
	opts           []CallOption
}

func (c *Client) buildURL(path string, query map[string]string) string {
	var b strings.Builder
	b.WriteString(c.baseURL)
	b.WriteString(path)
	if len(query) > 0 {
		first := true
		for k, v := range query {
			if v == "" {
				continue
			}
			if first {
				b.WriteByte('?')
				first = false
			} else {
				b.WriteByte('&')
			}
			b.WriteString(url.QueryEscape(k))
			b.WriteByte('=')
			b.WriteString(url.QueryEscape(v))
		}
	}
	return b.String()
}

// doJSON performs a request and decodes a successful JSON response into out
// (which may be nil to discard the body). Non-2xx responses become typed errors.
func (c *Client) doJSON(req request, out any) error {
	res, err := c.do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, readErr := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return parseErrorBody(res, body)
	}
	if readErr != nil {
		return newConnectionError("failed reading response body: " + readErr.Error())
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return newConnectionError("malformed JSON in a successful response: " + err.Error())
	}
	return nil
}

// do executes a request with timeout + retries (exponential backoff with jitter,
// honoring Retry-After), only retrying idempotent verbs or requests carrying an
// idempotency key. The caller owns closing the returned Response body.
func (c *Client) do(req request) (*http.Response, error) {
	var co callOptions
	for _, o := range req.opts {
		o(&co)
	}
	ctx := co.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	method := req.method
	if method == "" {
		method = http.MethodGet
	}

	// Pre-encode the body once so it can be replayed across retries.
	var bodyBytes []byte
	var contentType string
	switch {
	case req.rawBody != nil:
		bodyBytes = req.rawBody
		contentType = req.rawContentType
	case req.body != nil:
		b, err := json.Marshal(req.body)
		if err != nil {
			return nil, &Error{Code: CodeValidationFailed, Message: "failed to encode request body: " + err.Error()}
		}
		bodyBytes = b
		contentType = "application/json"
	}

	url := c.buildURL(req.path, req.query)

	// A rawBody (multipart) is never auto-retried.
	canRetry := (idempotentMethods[method] || co.idempotencyKey != "") && req.rawBody == nil

	for attempt := 0; ; attempt++ {
		res, err := c.attempt(ctx, method, url, bodyBytes, contentType, co)
		if err != nil {
			// Timeout / network errors: retry if budget allows and verb is safe.
			retryable := isTimeout(err) || isConnection(err)
			if !retryable || !canRetry || attempt >= c.maxRetries {
				return nil, err
			}
			if werr := sleep(ctx, c.backoff(attempt, -1)); werr != nil {
				return nil, werr
			}
			continue
		}
		if res.StatusCode < 300 || !canRetry || attempt >= c.maxRetries || !retryableStatus[res.StatusCode] {
			return res, nil
		}
		retryAfter := parseRetryAfterHeader(res)
		// Drain + close so the connection can be reused.
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if werr := sleep(ctx, c.backoff(attempt, retryAfter)); werr != nil {
			return nil, werr
		}
	}
}

// attempt performs a single HTTP attempt with a combined per-request timeout +
// caller context. It returns the raw Response (never erroring on HTTP status).
func (c *Client) attempt(ctx context.Context, method, url string, body []byte, contentType string, co callOptions) (*http.Response, error) {
	reqCtx := ctx
	var cancel context.CancelFunc
	if c.timeout > 0 {
		reqCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	httpReq, err := http.NewRequestWithContext(reqCtx, method, url, bodyReader)
	if err != nil {
		return nil, newConnectionError("failed building request: " + err.Error())
	}

	for k, v := range c.defaultHeaders {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", userAgent)
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	if co.idempotencyKey != "" {
		httpReq.Header.Set("Idempotency-Key", co.idempotencyKey)
	}
	for k, v := range co.headers {
		httpReq.Header.Set(k, v)
	}

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Distinguish a deadline/cancel from a transport error.
		if ce := ctx.Err(); ce != nil {
			// Caller-initiated cancel/deadline: propagate as-is.
			return nil, ce
		}
		if reqCtx.Err() == context.DeadlineExceeded {
			return nil, newTimeoutError(fmt.Sprintf("request timed out after %s", c.timeout))
		}
		return nil, newConnectionError("network error: " + err.Error())
	}
	return res, nil
}

// backoff computes the wait before a retry, honoring Retry-After (seconds; <0
// means none).
func (c *Client) backoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter >= 0 {
		if retryAfter > maxRetryAfter {
			return maxRetryAfter
		}
		return retryAfter
	}
	base := time.Duration(1<<uint(attempt)) * time.Second
	if base > 20*time.Second {
		base = 20 * time.Second
	}
	// Full-ish jitter: half fixed + half random.
	return base/2 + time.Duration(rand.Int63n(int64(base/2)+1))
}

// sleep waits for d or until ctx is done, returning the context error if it
// fires first.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func parseRetryAfterHeader(res *http.Response) time.Duration {
	raw := res.Header.Get("Retry-After")
	if raw == "" {
		return -1
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
		if secs < 0 {
			secs = 0
		}
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(raw); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d
	}
	return -1
}

// retryAfterSeconds converts a parsed Retry-After to whole seconds for the
// RateLimitError (0 when absent).
func retryAfterSeconds(res *http.Response) int {
	d := parseRetryAfterHeader(res)
	if d < 0 {
		return 0
	}
	return int(d / time.Second)
}

// parseErrorBody maps a non-2xx response to the right typed error, decoding the
// server's { error: { code, message, requestId } } envelope.
func parseErrorBody(res *http.Response, body []byte) error {
	var env struct {
		Error struct {
			Code      ErrorCode `json:"code"`
			Message   string    `json:"message"`
			RequestID string    `json:"requestId"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &env)
	message := env.Error.Message
	if message == "" {
		message = strings.TrimSpace(http.StatusText(res.StatusCode))
	}
	return errorFromStatus(res.StatusCode, env.Error.Code, message, env.Error.RequestID, retryAfterSeconds(res))
}

func isTimeout(err error) bool {
	var e *TimeoutError
	return errors.As(err, &e)
}

func isConnection(err error) bool {
	var e *ConnectionError
	return errors.As(err, &e)
}
