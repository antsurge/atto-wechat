package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/antsurge/atto-wechat/core/cache"
	"github.com/antsurge/atto-wechat/core/client"
	"github.com/antsurge/atto-wechat/core/codec"
	"github.com/antsurge/atto-wechat/core/errors"
)

type fakeProvider struct {
	mu sync.Mutex

	token *Token
	err   error
	calls int
}

func (p *fakeProvider) FetchToken(context.Context) (*Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.calls++

	return p.token, p.err
}

func (p *fakeProvider) callCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.calls
}

func tokenIn(d time.Duration) *Token {
	return &Token{
		AccessToken: "tok",
		ExpiresIn:   int64(d.Seconds()),
		ExpiresAt:   time.Now().Add(d).Unix(),
	}
}

func TestTokenExpired(t *testing.T) {
	tests := []struct {
		name   string
		token  *Token
		leeway time.Duration
		want   bool
	}{
		{"nil", nil, 0, true},
		{"empty", &Token{}, 0, true},
		{"no deadline", &Token{AccessToken: "t"}, time.Minute, false},
		{"far future", tokenIn(time.Hour), 5 * time.Minute, false},
		{"inside leeway", tokenIn(2 * time.Minute), 5 * time.Minute, true},
		{"past", tokenIn(-time.Minute), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.token.Expired(tt.leeway) != tt.want {
				t.Errorf("Expired = %v, want %v", tt.token.Expired(tt.leeway), tt.want)
			}
		})
	}
}

func TestTokenTTL(t *testing.T) {
	if got := (*Token)(nil).TTL(time.Minute); got != 0 {
		t.Errorf("nil ttl = %v", got)
	}

	i := tokenIn(time.Hour)
	if got := i.TTL(5 * time.Minute); got < 54*time.Minute || got > 56*time.Minute {
		t.Errorf("ttl = %v", got)
	}
	if got := i.TTL(2 * time.Hour); got >= 0 {
		t.Errorf("ttl with oversized leeway = %v, want negative", got)
	}
}

func TestAccessTokenManagerCaches(t *testing.T) {
	p := &fakeProvider{token: tokenIn(time.Hour)}

	m, err := NewAccessTokenManager(cache.NewMemoryCache(), p)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		token, err := m.Get(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if token != "tok" {
			t.Fatalf("token = %q", token)
		}
	}

	if p.callCount() != 1 {
		t.Fatalf("provider calls = %d, want 1", p.callCount())
	}
}

func TestAccessTokenManagerConcurrentFetchesOnce(t *testing.T) {
	p := &fakeProvider{token: tokenIn(time.Hour)}

	m, err := NewAccessTokenManager(cache.NewMemoryCache(), p)
	if err != nil {
		t.Fatal(err)
	}

	const workers = 50

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()

			if _, err := m.Get(context.Background()); err != nil {
				t.Errorf("Get: %v", err)
			}
		}()
	}

	wg.Wait()

	// Concurrent fetches would invalidate each other's token at WeChat, so one
	// fetch for the whole burst is the requirement, not an optimization.
	if p.callCount() != 1 {
		t.Fatalf("provider calls = %d, want 1", p.callCount())
	}
}

func TestAccessTokenManagerRefetchesExpiredEntry(t *testing.T) {
	c := cache.NewMemoryCache()

	expired, err := codec.Marshal(&Token{
		AccessToken: "old",
		ExpiresAt:   time.Now().Add(-time.Minute).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	c.Set(DefaultAccessTokenKey, expired, time.Hour)

	p := &fakeProvider{token: tokenIn(time.Hour)}

	m, err := NewAccessTokenManager(c, p)
	if err != nil {
		t.Fatal(err)
	}

	token, err := m.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if token != "tok" {
		t.Fatalf("token = %q, want a fresh fetch", token)
	}
	if p.callCount() != 1 {
		t.Fatalf("provider calls = %d, want 1", p.callCount())
	}
}

func TestAccessTokenManagerIgnoresCorruptCache(t *testing.T) {
	c := cache.NewMemoryCache()
	c.Set(DefaultAccessTokenKey, []byte("not-json"), time.Hour)

	p := &fakeProvider{token: tokenIn(time.Hour)}

	m, err := NewAccessTokenManager(c, p)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.callCount() != 1 {
		t.Fatalf("provider calls = %d, want 1", p.callCount())
	}
}

func TestAccessTokenManagerRefresh(t *testing.T) {
	p := &fakeProvider{token: tokenIn(time.Hour)}

	m, err := NewAccessTokenManager(cache.NewMemoryCache(), p, WithKey("k"))
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Get(context.Background()); err != nil {
		t.Fatal(err)
	}

	if _, err := m.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.callCount() != 2 {
		t.Fatalf("provider calls = %d, want 2", p.callCount())
	}
}

func TestAccessTokenManagerClear(t *testing.T) {
	p := &fakeProvider{token: tokenIn(time.Hour)}

	m, err := NewAccessTokenManager(cache.NewMemoryCache(), p)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	m.Clear()

	if _, err := m.Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.callCount() != 2 {
		t.Fatalf("provider calls = %d, want 2", p.callCount())
	}
}

func TestAccessTokenManagerSkipsCacheWhenLeewayTooLarge(t *testing.T) {
	c := cache.NewMemoryCache()
	p := &fakeProvider{token: tokenIn(time.Minute)}

	m, err := NewAccessTokenManager(c, p, WithLeeway(time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	token, err := m.Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if token != "tok" {
		t.Fatalf("token = %q", token)
	}

	// The token is still returned, but it is too close to expiry to be worth
	// storing: the entry would be evicted before a second caller could read it.
	if _, ok := c.Get(DefaultAccessTokenKey); ok {
		t.Fatal("token cached despite a non-positive ttl")
	}
}

func TestAccessTokenManagerProviderError(t *testing.T) {
	p := &fakeProvider{err: errors.New(errors.CodeUnauthorized, "bad secret")}

	m, err := NewAccessTokenManager(cache.NewMemoryCache(), p)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Get(context.Background()); !errors.Is(err, errors.CodeUnauthorized) {
		t.Fatalf("err = %v", err)
	}
}

func TestNewAccessTokenManagerValidation(t *testing.T) {
	p := &fakeProvider{token: tokenIn(time.Hour)}

	if _, err := NewAccessTokenManager(nil, p); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil cache err = %v", err)
	}
	if _, err := NewAccessTokenManager(cache.NewMemoryCache(), nil); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil provider err = %v", err)
	}
}

func TestHTTPTokenProvider(t *testing.T) {
	var gotQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = io.WriteString(w, `{"access_token":"t","expires_in":7200}`)
	}))
	defer srv.Close()

	c, err := client.NewClient(srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	p := NewHTTPTokenProvider(c, "/cgi-bin/token", url.Values{
		"appid":      {"wx1"},
		"secret":     {"s1"},
		"grant_type": {"client_credential"},
	})

	token, err := p.FetchToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if token.AccessToken != "t" || token.ExpiresIn != 7200 {
		t.Fatalf("token = %+v", token)
	}
	if token.ExpiresAt <= time.Now().Unix() {
		t.Fatalf("ExpiresAt = %d, want the future", token.ExpiresAt)
	}
	if gotQuery.Get("secret") != "s1" || gotQuery.Get("grant_type") != "client_credential" {
		t.Fatalf("query = %v", gotQuery)
	}
}

func TestHTTPTokenProviderClonesParams(t *testing.T) {
	params := url.Values{"appid": {"wx1"}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":"t","expires_in":60}`)
	}))
	defer srv.Close()

	c, err := client.NewClient(srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	p := NewHTTPTokenProvider(c, "/token", params)

	// Mutating the caller's map after construction must not change the provider.
	params.Set("appid", "wx-evil")

	token, err := p.FetchToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if token == nil || token.AccessToken != "t" {
		t.Fatal("fetch failed")
	}
}

func TestHTTPTokenProviderErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code errors.Code
	}{
		{"empty token", `{"expires_in":7200}`, errors.CodeInvalidResponse},
		{"errcode", `{"errcode":40013,"errmsg":"invalid appid"}`, errors.CodeInvalidResponse},
		{"zero expiry", `{"access_token":"t","expires_in":0}`, errors.CodeInvalidResponse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			c, err := client.NewClient(srv.URL, nil)
			if err != nil {
				t.Fatal(err)
			}

			p := NewHTTPTokenProvider(c, "/token", nil)

			if _, err := p.FetchToken(context.Background()); !errors.Is(err, tt.code) {
				t.Fatalf("err = %v, want %s", err, tt.code)
			}
		})
	}
}

func TestHTTPTokenProviderValidation(t *testing.T) {
	c, err := client.NewClient("http://example.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	var p *HTTPTokenProvider
	if _, err := p.FetchToken(context.Background()); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil provider err = %v", err)
	}

	if _, err := NewHTTPTokenProvider(c, "", nil).FetchToken(context.Background()); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("empty path err = %v", err)
	}
}

// ticketProvider is a fake TicketProvider.
type ticketProvider struct {
	mu    sync.Mutex
	calls map[TicketType]int
	fail  error
}

func (p *ticketProvider) FetchTicket(_ context.Context, typ TicketType) (*Ticket, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.calls == nil {
		p.calls = make(map[TicketType]int)
	}
	p.calls[typ]++

	if p.fail != nil {
		return nil, p.fail
	}

	return &Ticket{
		Ticket:    "ticket-" + string(typ),
		ExpiresIn: 7200,
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}, nil
}

func TestTicketManagerCachesPerType(t *testing.T) {
	p := &ticketProvider{}

	m, err := NewTicketManager(cache.NewMemoryCache(), p)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		for _, typ := range []TicketType{TicketTypeJSAPI, TicketTypeWxCard} {
			ticket, err := m.Get(context.Background(), typ)
			if err != nil {
				t.Fatal(err)
			}
			if ticket != "ticket-"+string(typ) {
				t.Fatalf("ticket = %q", ticket)
			}
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.calls[TicketTypeJSAPI] != 1 || p.calls[TicketTypeWxCard] != 1 {
		t.Fatalf("calls = %v, want one per type", p.calls)
	}
}

func TestTicketManagerRefreshAndClear(t *testing.T) {
	p := &ticketProvider{}

	m, err := NewTicketManager(cache.NewMemoryCache(), p)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Get(context.Background(), TicketTypeJSAPI); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Refresh(context.Background(), TicketTypeJSAPI); err != nil {
		t.Fatal(err)
	}
	m.Clear(TicketTypeJSAPI)

	if _, err := m.Get(context.Background(), TicketTypeJSAPI); err != nil {
		t.Fatal(err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.calls[TicketTypeJSAPI] != 3 {
		t.Fatalf("calls = %d, want 3", p.calls[TicketTypeJSAPI])
	}
}

func TestTicketManagerProviderError(t *testing.T) {
	m, err := NewTicketManager(
		cache.NewMemoryCache(),
		&ticketProvider{fail: errors.New(errors.CodeRequestFailed, "boom")},
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := m.Get(context.Background(), TicketTypeJSAPI); !errors.Is(err, errors.CodeRequestFailed) {
		t.Fatalf("err = %v", err)
	}
}

func TestTicketManagerValidation(t *testing.T) {
	if _, err := NewTicketManager(nil, &ticketProvider{}); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil cache err = %v", err)
	}
	if _, err := NewTicketManager(cache.NewMemoryCache(), nil); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil provider err = %v", err)
	}
}

func TestHTTPTicketProvider(t *testing.T) {
	var gotType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotType = r.URL.Query().Get("type")
		_, _ = io.WriteString(w, `{"ticket":"tk","expires_in":7200}`)
	}))
	defer srv.Close()

	c, err := client.NewClient(srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	p := NewHTTPTicketProvider(c, "/cgi-bin/ticket/getticket", url.Values{"access_token": {"t"}})

	ticket, err := p.FetchTicket(context.Background(), TicketTypeWxCard)
	if err != nil {
		t.Fatal(err)
	}
	if ticket.Ticket != "tk" || ticket.Type != TicketTypeWxCard {
		t.Fatalf("ticket = %+v", ticket)
	}
	if gotType != string(TicketTypeWxCard) {
		t.Fatalf("type query = %q", gotType)
	}
}

func TestHTTPTicketProviderErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"errcode":40001,"errmsg":"invalid credential"}`)
	}))
	defer srv.Close()

	c, err := client.NewClient(srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewHTTPTicketProvider(c, "/t", nil).FetchTicket(context.Background(), TicketTypeJSAPI); !errors.Is(err, errors.CodeInvalidResponse) {
		t.Fatalf("err = %v", err)
	}

	var p *HTTPTicketProvider
	if _, err := p.FetchTicket(context.Background(), TicketTypeJSAPI); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil provider err = %v", err)
	}
}

func TestCloneValues(t *testing.T) {
	src := url.Values{"a": {"1", "2"}}
	cp := cloneValues(src)

	cp.Set("a", "3")
	if src.Get("a") != "1" {
		t.Fatal("cloneValues shares the underlying slice")
	}

	if cloneValues(nil) != nil {
		t.Fatal("cloneValues(nil) != nil")
	}
}

