package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/antsurge/atto-wechat/core/config"
	"github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/middleware"
)

// captured records what the test server received.
type captured struct {
	method string
	path   string
	query  url.Values
	header http.Header
	body   string
}

type recorder struct {
	mu   sync.Mutex
	last captured
}

func (rec *recorder) handler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		rec.mu.Lock()
		rec.last = captured{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.Query(),
			header: r.Header.Clone(),
			body:   string(body),
		}
		rec.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"errcode":0,"errmsg":"ok"}`)
	})
}

func (rec *recorder) lastRequest() captured {
	rec.mu.Lock()
	defer rec.mu.Unlock()

	return rec.last
}

func newTestClient(t *testing.T, srv *httptest.Server, mw ...middleware.Middleware) *Client {
	t.Helper()

	c, err := New(config.New(config.WithBaseURL(srv.URL)), mw...)
	if err != nil {
		t.Fatal(err)
	}

	return c
}

func TestGet(t *testing.T) {
	rec := &recorder{}
	srv := httptest.NewServer(rec.handler(t))
	defer srv.Close()

	c := newTestClient(t, srv)

	resp, err := c.Get(context.Background(), "/api/ping", url.Values{"a": {"1"}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	got := rec.lastRequest()
	if got.method != http.MethodGet {
		t.Errorf("method = %s", got.method)
	}
	if got.path != "/api/ping" {
		t.Errorf("path = %s", got.path)
	}
	if got.query.Get("a") != "1" {
		t.Errorf("query = %v", got.query)
	}
	if got.header.Get("User-Agent") != config.DefaultUserAgent {
		t.Errorf("user-agent = %q", got.header.Get("User-Agent"))
	}
}

func TestPostJSON(t *testing.T) {
	rec := &recorder{}
	srv := httptest.NewServer(rec.handler(t))
	defer srv.Close()

	c := newTestClient(t, srv)

	payload := map[string]string{"title": "<a&b"}
	if _, err := c.PostJSON(context.Background(), "/api/x", nil, payload); err != nil {
		t.Fatal(err)
	}

	got := rec.lastRequest()
	if got.method != http.MethodPost {
		t.Errorf("method = %s", got.method)
	}
	if got.header.Get("Content-Type") != "application/json; charset=utf-8" {
		t.Errorf("content-type = %q", got.header.Get("Content-Type"))
	}
	if !strings.Contains(got.body, "<a&b") || strings.Contains(got.body, `\u0026`) {
		t.Errorf("body = %s", got.body)
	}
}

func TestPostForm(t *testing.T) {
	rec := &recorder{}
	srv := httptest.NewServer(rec.handler(t))
	defer srv.Close()

	c := newTestClient(t, srv)

	if _, err := c.PostForm(context.Background(), "/api/x", nil, url.Values{"k": {"v"}}); err != nil {
		t.Fatal(err)
	}

	got := rec.lastRequest()
	if got.header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Errorf("content-type = %q", got.header.Get("Content-Type"))
	}
	if got.body != "k=v" {
		t.Errorf("body = %q", got.body)
	}
}

func TestPostXML(t *testing.T) {
	rec := &recorder{}
	srv := httptest.NewServer(rec.handler(t))
	defer srv.Close()

	c := newTestClient(t, srv)

	type msg struct {
		Value string `xml:"Value"`
	}

	if _, err := c.PostXML(context.Background(), "/api/x", nil, msg{Value: "v"}); err != nil {
		t.Fatal(err)
	}

	got := rec.lastRequest()
	if got.header.Get("Content-Type") != "application/xml" {
		t.Errorf("content-type = %q", got.header.Get("Content-Type"))
	}
	if !strings.Contains(got.body, "<Value>v</Value>") {
		t.Errorf("body = %s", got.body)
	}
}

func TestPostMultipart(t *testing.T) {
	var (
		mu     sync.Mutex
		field  string
		fname  string
		fbytes []byte
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		file, header, err := r.FormFile("media")
		if err != nil {
			t.Errorf("form file: %v", err)
			w.WriteHeader(http.StatusBadRequest)

			return
		}
		defer file.Close()

		data, _ := io.ReadAll(file)

		mu.Lock()
		field = r.FormValue("description")
		fname = header.Filename
		fbytes = data
		mu.Unlock()

		_, _ = io.WriteString(w, `{"errcode":0}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.PostMultipart(
		context.Background(),
		"/api/upload",
		nil,
		url.Values{"description": {"hello"}},
		&FilePart{
			FieldName: "media",
			FileName:  "a.jpg",
			Content:   []byte("image-bytes"),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()

	if field != "hello" {
		t.Errorf("field = %q", field)
	}
	if fname != "a.jpg" {
		t.Errorf("filename = %q", fname)
	}
	if string(fbytes) != "image-bytes" {
		t.Errorf("content = %q", fbytes)
	}
}

func TestPostMultipartRequiresFileNames(t *testing.T) {
	c, err := NewClient("http://example.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.PostMultipart(context.Background(), "/x", nil, nil, &FilePart{Content: []byte("x")})
	if !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestStatusErrors(t *testing.T) {
	tests := []struct {
		status int
		code   errors.Code
	}{
		{http.StatusUnauthorized, errors.CodeUnauthorized},
		{http.StatusForbidden, errors.CodeForbidden},
		{http.StatusNotFound, errors.CodeNotFound},
		{http.StatusTooManyRequests, errors.CodeRateLimited},
		{http.StatusInternalServerError, errors.CodeSystemBusy},
		{http.StatusBadGateway, errors.CodeSystemBusy},
		{http.StatusBadRequest, errors.CodeRequestFailed},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			c := newTestClient(t, srv)

			resp, err := c.Get(context.Background(), "/x", nil)
			if !errors.Is(err, tt.code) {
				t.Fatalf("err = %v, want %s", err, tt.code)
			}
			if resp == nil || resp.StatusCode != tt.status {
				t.Fatalf("response = %+v", resp)
			}
		})
	}
}

func TestBusinessErrCodeIsNotAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"errcode":40013,"errmsg":"invalid appid"}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	resp, err := c.Get(context.Background(), "/x", nil)
	if err != nil {
		t.Fatalf("a business errcode was turned into an error: %v", err)
	}
	if !strings.Contains(string(resp.Body), "40013") {
		t.Fatal("body was not preserved for the caller")
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := c.Get(ctx, "/slow", nil)
	if !errors.Is(err, errors.CodeTimeout) {
		t.Fatalf("err = %v, want TIMEOUT", err)
	}
}

func TestCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	c := newTestClient(t, srv)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Get(ctx, "/x", nil)
	if !errors.Is(err, errors.CodeCancelled) {
		t.Fatalf("err = %v, want CANCELLED", err)
	}
}

func TestNetworkFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	addr := srv.URL
	srv.Close()

	c, err := NewClient(addr, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Get(context.Background(), "/x", nil)
	if !errors.Is(err, errors.CodeRequestFailed) {
		t.Fatalf("err = %v, want REQUEST_FAILED", err)
	}
}

func TestBaseURLOverride(t *testing.T) {
	var hit string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = r.URL.Path
	}))
	defer srv.Close()

	c, err := NewClient("http://example.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := c.Do(context.Background(), &Request{
		Method:  http.MethodGet,
		Path:    "/img",
		BaseURL: srv.URL,
	}); err != nil {
		t.Fatal(err)
	}
	if hit != "/img" {
		t.Fatalf("path = %q", hit)
	}
}

func TestWithMiddlewareKeepsOriginal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	base := newTestClient(t, srv)

	var calls int
	mw := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *Request) (*Response, error) {
			calls++

			return next(ctx, req)
		}
	}

	derived := base.WithMiddleware(mw)

	if _, err := derived.Get(context.Background(), "/x", nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("middleware calls = %d", calls)
	}

	if _, err := base.Get(context.Background(), "/x", nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("derived middleware leaked into the base client: calls = %d", calls)
	}
}

func TestNewRejectsBadConfig(t *testing.T) {
	if _, err := New(nil); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}

	cfg := config.New(config.WithBaseURL(""))
	if _, err := New(cfg); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("empty base url err = %v", err)
	}
}

func TestEncodeErrors(t *testing.T) {
	c, err := NewClient("http://example.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := c.PostJSON(context.Background(), "/x", nil, make(chan int)); !errors.Is(err, errors.CodeEncodeFailed) {
		t.Fatalf("json err = %v", err)
	}

	if _, err := c.PostXML(context.Background(), "/x", nil, make(chan int)); !errors.Is(err, errors.CodeEncodeFailed) {
		t.Fatalf("xml err = %v", err)
	}
}

func TestCustomUserAgentFromRequest(t *testing.T) {
	rec := &recorder{}
	srv := httptest.NewServer(rec.handler(t))
	defer srv.Close()

	c := newTestClient(t, srv)

	_, err := c.Do(context.Background(), &Request{
		Method: http.MethodGet,
		Path:   "/x",
		Header: http.Header{"User-Agent": {"custom"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if ua := rec.lastRequest().header.Get("User-Agent"); ua != "custom" {
		t.Fatalf("user-agent = %q", ua)
	}
}
