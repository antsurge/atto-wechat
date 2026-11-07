// Package client provides the HTTP client shared by all platform modules.
//
// It composes a middleware chain (logging, retry, token injection, ...) and
// performs the round-trip, translating transport failures into the unified
// error type from core/errors.
//
// A WeChat reply that carries a business errcode is returned as a normal
// response: the caller decodes it into a core/response type and decides what
// the code means. The client only reports failures of the call itself.
package client

import (
	"bytes"
	"context"
	stderrors "errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/antsurge/atto-wechat/core/codec"
	"github.com/antsurge/atto-wechat/core/config"
	"github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/middleware"
)

// Request and Response are re-exported from middleware so callers only need to
// import client.
type Request = middleware.Request
type Response = middleware.Response

// Client is the unified HTTP client. It is safe for concurrent use.
type Client struct {
	cfg        *config.CoreConfig
	baseURL    string
	httpClient *http.Client
	middleware []middleware.Middleware
}

// New builds a Client from the given config and middleware.
//
// A nil cfg.HTTPClient is replaced with a default client using cfg.HTTPTimeout.
// An invalid config is reported as an error rather than silently falling back to
// defaults: a bad BaseURL would otherwise surface much later as a confusing
// network failure.
func New(cfg *config.CoreConfig, mw ...middleware.Middleware) (*Client, error) {
	if cfg == nil {
		return nil, errors.New(errors.CodeInvalidArgument, "client: config must not be nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(errors.CodeInvalidArgument, "client: invalid config", err)
	}

	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: cfg.HTTPTimeout}
	}

	return &Client{
		cfg:        cfg,
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: hc,
		middleware: mw,
	}, nil
}

// NewClient builds a Client from a base URL and an optional http.Client.
//
// It is a convenience wrapper for callers that do not need the full
// config.CoreConfig.
func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	opts := []config.Option{config.WithBaseURL(baseURL)}
	if httpClient != nil {
		opts = append(opts, config.WithHTTPClient(httpClient))
	}

	return New(config.New(opts...))
}

// BaseURL returns the API root the client targets.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// WithMiddleware returns a copy of the client with additional middleware
// appended. The first middleware in the resulting chain is the outermost.
func (c *Client) WithMiddleware(mw ...middleware.Middleware) *Client {
	cp := *c
	cp.middleware = append(append([]middleware.Middleware{}, c.middleware...), mw...)

	return &cp
}

// Do runs req through the middleware chain and returns the response.
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	handler := middleware.Handler(c.roundTrip)

	// Compose outermost-first: iterate backwards so mw[0] wraps the others.
	for i := len(c.middleware) - 1; i >= 0; i-- {
		handler = c.middleware[i](handler)
	}

	return handler(ctx, req)
}

// Get issues a GET request.
func (c *Client) Get(ctx context.Context, path string, query url.Values) (*Response, error) {
	return c.Do(ctx, &Request{Method: http.MethodGet, Path: path, Query: query})
}

// PostJSON marshals payload as JSON and issues a POST.
func (c *Client) PostJSON(ctx context.Context, path string, query url.Values, payload any) (*Response, error) {
	body, err := codec.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(errors.CodeEncodeFailed, "marshal json payload", err)
	}

	return c.Do(ctx, &Request{
		Method:   http.MethodPost,
		Path:     path,
		Query:    query,
		Body:     body,
		BodyType: middleware.BodyJSON,
	})
}

// PostForm issues a POST with a url-encoded form body.
func (c *Client) PostForm(ctx context.Context, path string, query url.Values, form url.Values) (*Response, error) {
	return c.Do(ctx, &Request{
		Method:   http.MethodPost,
		Path:     path,
		Query:    query,
		Body:     []byte(form.Encode()),
		BodyType: middleware.BodyForm,
	})
}

// PostXML marshals payload as XML and issues a POST.
func (c *Client) PostXML(ctx context.Context, path string, query url.Values, payload any) (*Response, error) {
	body, err := codec.XMLMarshal(payload)
	if err != nil {
		return nil, errors.Wrap(errors.CodeEncodeFailed, "marshal xml payload", err)
	}

	return c.Do(ctx, &Request{
		Method:   http.MethodPost,
		Path:     path,
		Query:    query,
		Body:     body,
		BodyType: middleware.BodyXML,
	})
}

// roundTrip is the innermost Handler: it performs the real HTTP request.
func (c *Client) roundTrip(ctx context.Context, req *Request) (*Response, error) {
	base := c.baseURL
	if req.BaseURL != "" {
		base = strings.TrimRight(req.BaseURL, "/")
	}

	u := base + req.Path
	if len(req.Query) > 0 {
		u += "?" + req.Query.Encode()
	}

	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, u, body)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInvalidArgument, "build request", err)
	}

	if req.Header != nil {
		httpReq.Header = req.Header.Clone()
	}

	// Apply the configured User-Agent unless the request set one: without this
	// Go would send "Go-http-client/1.1", which WeChat cannot attribute to this
	// SDK.
	if httpReq.Header.Get("User-Agent") == "" && c.cfg != nil && c.cfg.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.cfg.UserAgent)
	}

	if contentType := contentType(req.BodyType); contentType != "" &&
		httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, classifyTransportError(err)
	}
	defer httpResp.Body.Close()

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errors.Wrap(errors.CodeInvalidResponse, "read response body", err)
	}

	resp := &Response{
		StatusCode: httpResp.StatusCode,
		Header:     httpResp.Header.Clone(),
		Body:       data,
	}

	// WeChat reports transport-level refusals with ordinary HTTP status codes,
	// so a non-2xx status is a call failure even when the body looks fine.
	if resp.StatusCode >= 400 {
		return resp, statusError(resp.StatusCode)
	}

	return resp, nil
}

// contentType maps a BodyType to its Content-Type value. BodyMultipart and
// BodyRaw leave the header to the caller, which must encode the body itself.
func contentType(t middleware.BodyType) string {
	switch t {
	case middleware.BodyForm:
		return "application/x-www-form-urlencoded"
	case middleware.BodyXML:
		return "application/xml"
	case middleware.BodyJSON:
		return "application/json; charset=utf-8"
	default:
		return ""
	}
}

// classifyTransportError maps a transport failure to a unified error.
func classifyTransportError(err error) error {
	switch {
	case stderrors.Is(err, context.Canceled):
		return errors.Wrap(errors.CodeCancelled, "request cancelled", err)
	case stderrors.Is(err, context.DeadlineExceeded):
		return errors.Wrap(errors.CodeTimeout, "request timed out", err)
	}

	var netErr net.Error
	if stderrors.As(err, &netErr) && netErr.Timeout() {
		return errors.Wrap(errors.CodeTimeout, "request timed out", err)
	}

	return errors.Wrap(errors.CodeRequestFailed, "request failed", err)
}

// statusError maps a non-2xx HTTP status to a unified error.
func statusError(status int) error {
	var code errors.Code

	switch {
	case status == http.StatusUnauthorized:
		code = errors.CodeUnauthorized
	case status == http.StatusForbidden:
		code = errors.CodeForbidden
	case status == http.StatusNotFound:
		code = errors.CodeNotFound
	case status == http.StatusTooManyRequests:
		code = errors.CodeRateLimited
	case status >= 500:
		code = errors.CodeSystemBusy
	default:
		code = errors.CodeRequestFailed
	}

	e := errors.Newf(code, "unexpected http status %d", status)
	e.HTTPStatus = status

	return e
}
