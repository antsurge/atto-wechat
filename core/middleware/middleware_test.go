package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/antsurge/atto-wechat/core/config"
	"github.com/antsurge/atto-wechat/core/errors"
)

type stubSource struct {
	mu sync.Mutex

	token string
	fresh string

	getCalls     int
	refreshCalls int

	getErr     error
	refreshErr error
}

func (s *stubSource) Get(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.getCalls++
	if s.getErr != nil {
		return "", s.getErr
	}

	return s.token, nil
}

func (s *stubSource) Refresh(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.refreshCalls++
	if s.refreshErr != nil {
		return "", s.refreshErr
	}

	return s.fresh, nil
}

func fixedResponse(status int, body string) Handler {
	return func(context.Context, *Request) (*Response, error) {
		return &Response{StatusCode: status, Body: []byte(body)}, nil
	}
}

func captureReq(dst **Request, body string) Handler {
	return func(_ context.Context, req *Request) (*Response, error) {
		*dst = req

		return &Response{StatusCode: 200, Body: []byte(body)}, nil
	}
}

func TestWithTokenInjectsOnACopy(t *testing.T) {
	var got *Request

	src := &stubSource{token: "tkn"}

	handler := WithToken(src)(captureReq(&got, `{"errcode":0}`))

	original := &Request{Method: "GET", Path: "/x", Query: url.Values{"a": {"1"}}}

	if _, err := handler(context.Background(), original); err != nil {
		t.Fatal(err)
	}

	if got.Query.Get("access_token") != "tkn" {
		t.Fatalf("access_token = %q", got.Query.Get("access_token"))
	}
	if got.Query.Get("a") != "1" {
		t.Fatal("existing query parameters were dropped")
	}

	if original.Query.Get("access_token") != "" {
		t.Fatal("WithToken mutated the caller's request")
	}
	if src.getCalls != 1 {
		t.Fatalf("Get calls = %d, want 1", src.getCalls)
	}
}

func TestWithTokenSkipsPublicRequests(t *testing.T) {
	src := &stubSource{token: "tkn"}

	handler := WithToken(src)(fixedResponse(200, "image"))

	if _, err := handler(context.Background(), &Request{NoAuth: true}); err != nil {
		t.Fatal(err)
	}
	if src.getCalls != 0 {
		t.Fatalf("Get calls = %d, want 0 for NoAuth", src.getCalls)
	}
}

func TestWithTokenSurfacesGetError(t *testing.T) {
	src := &stubSource{getErr: errors.New(errors.CodeUnauthorized, "no credentials")}

	handler := WithToken(src)(fixedResponse(200, "{}"))

	_, err := handler(context.Background(), &Request{})
	if !errors.Is(err, errors.CodeUnauthorized) {
		t.Fatalf("err = %v, want UNAUTHORIZED", err)
	}
}

func TestWithTokenRefreshesAndReplaysOnce(t *testing.T) {
	src := &stubSource{token: "stale", fresh: "fresh"}

	var calls int
	var lastQuery string

	next := func(_ context.Context, req *Request) (*Response, error) {
		calls++
		lastQuery = req.Query.Get("access_token")

		if calls == 1 {
			return &Response{StatusCode: 200, Body: []byte(`{"errcode":40001,"errmsg":"invalid credential"}`)}, nil
		}

		return &Response{StatusCode: 200, Body: []byte(`{"errcode":0}`)}, nil
	}

	handler := WithToken(src)(next)

	resp, err := handler(context.Background(), &Request{})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if lastQuery != "fresh" {
		t.Fatalf("replayed with token %q, want fresh", lastQuery)
	}
	if !strings.Contains(string(resp.Body), `"errcode":0`) {
		t.Fatalf("replay response = %s", resp.Body)
	}
	if src.refreshCalls != 1 {
		t.Fatalf("Refresh calls = %d, want 1", src.refreshCalls)
	}
}

func TestWithTokenReplaysAtMostOnce(t *testing.T) {
	src := &stubSource{token: "a", fresh: "b"}

	rejected := fixedResponse(200, `{"errcode":41001}`)

	var calls int
	next := func(ctx context.Context, req *Request) (*Response, error) {
		calls++

		return rejected(ctx, req)
	}

	handler := WithToken(src)(next)

	if _, err := handler(context.Background(), &Request{}); err != nil {
		t.Fatal(err)
	}

	if calls != 2 {
		t.Fatalf("calls = %d, want 2 (no third attempt)", calls)
	}
	if src.refreshCalls != 1 {
		t.Fatalf("Refresh calls = %d, want 1", src.refreshCalls)
	}
}

func TestWithTokenKeepsResponseWhenRefreshFails(t *testing.T) {
	src := &stubSource{
		token:      "stale",
		refreshErr: errors.New(errors.CodeUnauthorized, "refresh rejected"),
	}

	var calls int
	next := func(ctx context.Context, req *Request) (*Response, error) {
		calls++

		return fixedResponse(200, `{"errcode":40001}`)(ctx, req)
	}

	resp, err := WithToken(src)(next)(context.Background(), &Request{})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if src.refreshCalls != 1 {
		t.Fatalf("Refresh calls = %d, want 1", src.refreshCalls)
	}
	if !strings.Contains(string(resp.Body), "40001") {
		t.Fatal("original rejection was discarded")
	}
}

func TestWithTokenIgnoresUnrelatedErrCodes(t *testing.T) {
	src := &stubSource{token: "t"}

	var calls int
	next := func(ctx context.Context, req *Request) (*Response, error) {
		calls++

		return fixedResponse(200, `{"errcode":40013,"errmsg":"invalid appid"}`)(ctx, req)
	}

	if _, err := WithToken(src)(next)(context.Background(), &Request{}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || src.refreshCalls != 0 {
		t.Fatalf("calls = %d, refreshes = %d; want 1, 0", calls, src.refreshCalls)
	}
}

func TestTokenRejectedParsing(t *testing.T) {
	tests := []struct {
		name string
		resp *Response
		want bool
	}{
		{"nil", nil, false},
		{"empty body", &Response{}, false},
		{"not json", &Response{Body: []byte("success")}, false},
		{"ok", &Response{Body: []byte(`{"errcode":0}`)}, false},
		{"expired", &Response{Body: []byte(`{"errcode":42001}`)}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tokenRejected(tt.resp) != tt.want {
				t.Errorf("tokenRejected = %v, want %v", tokenRejected(tt.resp), tt.want)
			}
		})
	}
}

func TestRetrySucceedsOnThirdAttempt(t *testing.T) {
	cfg := config.New(
		config.WithRetry(2, time.Millisecond),
		config.WithRetryMaxBackoff(5*time.Millisecond),
	)

	var calls int
	next := func(context.Context, *Request) (*Response, error) {
		calls++

		if calls < 3 {
			return nil, errors.New(errors.CodeSystemBusy, "busy")
		}

		return &Response{StatusCode: 200}, nil
	}

	resp, err := Retry(cfg)(next)(context.Background(), &Request{})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
	if !resp.OK() {
		t.Fatal("final response not returned")
	}
}

func TestRetryStopsAtAttemptLimit(t *testing.T) {
	cfg := config.New(config.WithRetry(2, time.Millisecond))

	var calls int
	next := func(context.Context, *Request) (*Response, error) {
		calls++

		return nil, errors.New(errors.CodeRateLimited, "limited")
	}

	_, err := Retry(cfg)(next)(context.Background(), &Request{})
	if !errors.Is(err, errors.CodeRateLimited) {
		t.Fatalf("err = %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3 (1 try + 2 retries)", calls)
	}
}

func TestRetrySkipsNonRetryable(t *testing.T) {
	cfg := config.New(config.WithRetry(5, time.Millisecond))

	var calls int
	next := func(context.Context, *Request) (*Response, error) {
		calls++

		return nil, errors.New(errors.CodeTimeout, "timeout")
	}

	_, err := Retry(cfg)(next)(context.Background(), &Request{})
	if !errors.Is(err, errors.CodeTimeout) {
		t.Fatalf("err = %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestRetryHonoursCancelledContext(t *testing.T) {
	cfg := config.New(config.WithRetry(3, 10*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var calls int
	next := func(context.Context, *Request) (*Response, error) {
		calls++

		return nil, errors.New(errors.CodeSystemBusy, "busy")
	}

	_, err := Retry(cfg)(next)(ctx, &Request{})
	if !errors.Is(err, errors.CodeCancelled) {
		t.Fatalf("err = %v, want CANCELLED", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestNextBackoffDoublesAndCaps(t *testing.T) {
	if got := nextBackoff(10*time.Millisecond, 0); got != 20*time.Millisecond {
		t.Errorf("uncapped = %v", got)
	}
	if got := nextBackoff(10*time.Millisecond, 15*time.Millisecond); got != 15*time.Millisecond {
		t.Errorf("capped = %v", got)
	}
	if got := nextBackoff(0, 0); got != 0 {
		t.Errorf("zero = %v", got)
	}
}

func TestLoggingRedactsSecrets(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := config.New(config.WithLogger(logger))

	handler := Logging(cfg)(fixedResponse(200, "{}"))

	_, err := handler(context.Background(), &Request{
		Method: "GET",
		Path:   "/cgi-bin/token",
		Query: url.Values{
			"access_token": {"secret-token"},
			"appid":        {"wx123"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if strings.Contains(out, "secret-token") {
		t.Fatalf("access token leaked into logs: %s", out)
	}
	// The query is URL-encoded, so the marker appears percent-encoded.
	if !strings.Contains(out, "access_token=%2A%2A%2A") {
		t.Fatalf("redaction marker missing: %s", out)
	}
	if !strings.Contains(out, "wx123") {
		t.Fatalf("harmless parameter missing: %s", out)
	}
}

func TestLoggingSilencedAtWarn(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, nil))
	cfg := config.New(
		config.WithLogger(logger),
		config.WithLogLevel(slog.LevelWarn),
	)

	if _, err := Logging(cfg)(fixedResponse(200, "{}"))(context.Background(), &Request{}); err != nil {
		t.Fatal(err)
	}

	if buf.Len() != 0 {
		t.Fatalf("logs emitted at Warn level: %s", buf.String())
	}
}

func TestRedactQueryEmpty(t *testing.T) {
	if redactQuery(nil) != "" {
		t.Fatal("nil query rendered non-empty")
	}
}

func TestResponseHelpers(t *testing.T) {
	tests := []struct {
		name string
		resp *Response
		ok   bool
	}{
		{"nil", nil, false},
		{"200", &Response{StatusCode: 200}, true},
		{"204", &Response{StatusCode: 204}, true},
		{"302", &Response{StatusCode: 302}, false},
		{"500", &Response{StatusCode: 500}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resp.OK() != tt.ok {
				t.Errorf("OK = %v, want %v", tt.resp.OK(), tt.ok)
			}
		})
	}

	var v struct {
		ErrCode int `json:"errcode"`
	}

	resp := &Response{Body: []byte(`{"errcode":0}`)}
	if err := resp.DecodeJSON(&v); err != nil {
		t.Fatal(err)
	}
	if v.ErrCode != 0 {
		t.Fatalf("decoded %+v", v)
	}

	if err := resp.DecodeJSON(make(chan int)); err == nil {
		t.Fatal("decoding into a bad target returned no error")
	}
	if !errors.Is((&Response{Body: []byte("{")}).DecodeJSON(&v), errors.CodeDecodeFailed) {
		t.Fatal("malformed json not classified as DECODE_FAILED")
	}
	if err := (*Response)(nil).DecodeJSON(&v); !errors.Is(err, errors.CodeInvalidResponse) {
		t.Fatalf("nil response error = %v", err)
	}
}
