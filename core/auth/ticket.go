package auth

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/antsurge/atto-wechat/core/cache"
	"github.com/antsurge/atto-wechat/core/client"
	"github.com/antsurge/atto-wechat/core/codec"
	"github.com/antsurge/atto-wechat/core/errors"
)

// DefaultTicketKeyPrefix is prepended to the ticket type to form a cache key.
const DefaultTicketKeyPrefix = "atto-wechat:ticket:"

// TicketType identifies which ticket endpoint to call.
type TicketType string

const (
	// TicketTypeJSAPI is the jsapi_ticket used to sign JS-SDK config.
	TicketTypeJSAPI TicketType = "jsapi"
	// TicketTypeWxCard is the wx_card api_ticket used to sign card extensions.
	TicketTypeWxCard TicketType = "wx_card"
)

// Ticket is a short-lived credential used to sign front-end calls.
type Ticket struct {
	Ticket    string     `json:"ticket"`
	ExpiresIn int64      `json:"expires_in"`
	ExpiresAt int64      `json:"expires_at"`
	Type      TicketType `json:"type,omitempty"`
}

// Expired reports whether the ticket is unusable at now, minus leeway.
func (t *Ticket) Expired(leeway time.Duration) bool {
	if t == nil || t.Ticket == "" {
		return true
	}
	if t.ExpiresAt == 0 {
		return false
	}

	return time.Now().Add(leeway).Unix() >= t.ExpiresAt
}

// TTL returns how long the ticket may still be cached.
func (t *Ticket) TTL(leeway time.Duration) time.Duration {
	if t == nil || t.ExpiresAt == 0 {
		return 0
	}

	return time.Until(time.Unix(t.ExpiresAt, 0)) - leeway
}

// TicketProvider fetches a fresh ticket of the requested type.
type TicketProvider interface {
	FetchTicket(ctx context.Context, typ TicketType) (*Ticket, error)
}

// ticketResponse is the reply shape shared by the ticket endpoints.
type ticketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int64  `json:"expires_in"`
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
}

// HTTPTicketProvider fetches tickets over HTTP.
type HTTPTicketProvider struct {
	client *client.Client
	path   string
	params url.Values
}

// NewHTTPTicketProvider creates a ticket provider for the given endpoint.
//
// params carries the credential fields the endpoint expects (access_token, or
// appid/secret for the standalone ticket API).
func NewHTTPTicketProvider(
	c *client.Client,
	path string,
	params url.Values,
) *HTTPTicketProvider {
	return &HTTPTicketProvider{
		client: c,
		path:   path,
		params: cloneValues(params),
	}
}

// FetchTicket requests a ticket of type typ.
func (p *HTTPTicketProvider) FetchTicket(
	ctx context.Context,
	typ TicketType,
) (*Ticket, error) {
	if p == nil || p.client == nil || p.path == "" {
		return nil, errInvalidTicketProvider
	}

	q := cloneValues(p.params)
	if q == nil {
		q = make(url.Values, 1)
	}
	q.Set("type", string(typ))

	resp, err := p.client.Get(ctx, p.path, q)
	if err != nil {
		return nil, err
	}

	var result ticketResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	if result.Ticket == "" {
		if result.ErrCode != 0 {
			return nil, errors.Newf(
				errors.CodeInvalidResponse,
				"ticket request failed: errcode %d: %s",
				result.ErrCode,
				result.ErrMsg,
			)
		}

		return nil, errEmptyTicket
	}

	if result.ExpiresIn <= 0 {
		return nil, errInvalidTokenExpiry
	}

	return &Ticket{
		Ticket:    result.Ticket,
		ExpiresIn: result.ExpiresIn,
		ExpiresAt: time.Now().Add(
			time.Duration(result.ExpiresIn) * time.Second,
		).Unix(),
		Type: typ,
	}, nil
}

// TicketManager caches tickets per type and fetches them on demand.
type TicketManager struct {
	cache     cache.Cache
	provider  TicketProvider
	keyPrefix string
	leeway    time.Duration

	mu sync.Mutex
}

// TicketOption mutates a TicketManager.
type TicketOption func(*TicketManager)

// WithTicketKeyPrefix overrides the cache key prefix.
func WithTicketKeyPrefix(prefix string) TicketOption {
	return func(m *TicketManager) {
		if prefix != "" {
			m.keyPrefix = prefix
		}
	}
}

// WithTicketLeeway overrides how early a ticket is considered expired.
func WithTicketLeeway(d time.Duration) TicketOption {
	return func(m *TicketManager) {
		if d >= 0 {
			m.leeway = d
		}
	}
}

// NewTicketManager creates a manager backed by c and p.
func NewTicketManager(
	c cache.Cache,
	p TicketProvider,
	opts ...TicketOption,
) (*TicketManager, error) {
	if c == nil {
		return nil, errInvalidCache
	}
	if p == nil {
		return nil, errInvalidTicketProvider
	}

	m := &TicketManager{
		cache:     c,
		provider:  p,
		keyPrefix: DefaultTicketKeyPrefix,
		leeway:    DefaultLeeway,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}

	return m, nil
}

// Get returns a valid ticket of type typ.
func (m *TicketManager) Get(ctx context.Context, typ TicketType) (string, error) {
	if tok, ok := m.cached(typ); ok {
		return tok.Ticket, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if tok, ok := m.cached(typ); ok {
		return tok.Ticket, nil
	}

	tok, err := m.fetchLocked(ctx, typ)
	if err != nil {
		return "", err
	}

	return tok.Ticket, nil
}

// Refresh discards the cached ticket and fetches a new one.
func (m *TicketManager) Refresh(ctx context.Context, typ TicketType) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tok, err := m.fetchLocked(ctx, typ)
	if err != nil {
		return "", err
	}

	return tok.Ticket, nil
}

// Clear removes the cached ticket of type typ.
func (m *TicketManager) Clear(typ TicketType) {
	m.cache.Delete(m.key(typ))
}

// key builds the cache key for typ.
func (m *TicketManager) key(typ TicketType) string {
	return m.keyPrefix + string(typ)
}

// cached returns the stored ticket when it is still usable.
func (m *TicketManager) cached(typ TicketType) (*Ticket, bool) {
	raw, ok := m.cache.Get(m.key(typ))
	if !ok || len(raw) == 0 {
		return nil, false
	}

	var tok Ticket
	if err := codec.Unmarshal(raw, &tok); err != nil {
		return nil, false
	}

	if tok.Expired(m.leeway) {
		return nil, false
	}

	return &tok, true
}

// fetchLocked fetches and stores a ticket. The caller must hold m.mu.
func (m *TicketManager) fetchLocked(ctx context.Context, typ TicketType) (*Ticket, error) {
	tok, err := m.provider.FetchTicket(ctx, typ)
	if err != nil {
		return nil, err
	}

	if tok == nil || tok.Ticket == "" {
		return nil, errEmptyTicket
	}

	if ttl := tok.TTL(m.leeway); ttl > 0 {
		if raw, err := codec.Marshal(tok); err == nil {
			m.cache.Set(m.key(typ), raw, ttl)
		}
	}

	return tok, nil
}
