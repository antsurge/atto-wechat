package auth

import (
	"context"
	"sync"
	"time"

	"github.com/antsurge/atto-wechat/core/cache"
	"github.com/antsurge/atto-wechat/core/codec"
)

// DefaultLeeway is how long before real expiry a token is treated as expired.
const DefaultLeeway = 5 * time.Minute

// DefaultAccessTokenKey is the cache key used when none is configured.
const DefaultAccessTokenKey = "atto-wechat:access_token"

// AccessTokenManager caches an access token and fetches a new one on demand.
//
// It is safe for concurrent use, and it deliberately serializes fetching: WeChat
// invalidates the previously issued token when a new one is granted, so two
// concurrent fetches would leave one caller holding a token that is already
// dead. Holding the lock across the fetch is what prevents that, and the cost
// (one blocked call per expiry window) is negligible.
type AccessTokenManager struct {
	cache    cache.Cache
	provider TokenProvider
	key      string
	leeway   time.Duration

	mu sync.Mutex
}

// Option mutates an AccessTokenManager.
type Option func(*AccessTokenManager)

// WithKey overrides the cache key.
//
// Use a distinct key per app: a shared key would make two applications
// overwrite each other's token, and each overwrite invalidates the other's.
func WithKey(key string) Option {
	return func(m *AccessTokenManager) {
		if key != "" {
			m.key = key
		}
	}
}

// WithLeeway overrides how early a token is considered expired.
func WithLeeway(d time.Duration) Option {
	return func(m *AccessTokenManager) {
		if d >= 0 {
			m.leeway = d
		}
	}
}

// NewAccessTokenManager creates a manager backed by c and p.
func NewAccessTokenManager(
	c cache.Cache,
	p TokenProvider,
	opts ...Option,
) (*AccessTokenManager, error) {
	if c == nil {
		return nil, errInvalidCache
	}
	if p == nil {
		return nil, errInvalidTokenProvider
	}

	m := &AccessTokenManager{
		cache:    c,
		provider: p,
		key:      DefaultAccessTokenKey,
		leeway:   DefaultLeeway,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}

	return m, nil
}

// Get returns a currently valid access token.
func (m *AccessTokenManager) Get(ctx context.Context) (string, error) {
	tok, err := m.Token(ctx)
	if err != nil {
		return "", err
	}

	return tok.AccessToken, nil
}

// Refresh discards the cached token and fetches a new one.
//
// It is called after WeChat rejects a token that still looked valid locally:
// revocation is invisible to the expiry check, so the cache cannot be trusted.
func (m *AccessTokenManager) Refresh(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tok, err := m.fetchLocked(ctx)
	if err != nil {
		return "", err
	}

	return tok.AccessToken, nil
}

// Token returns the cached token, fetching one when the cache is empty, corrupt
// or too close to expiry.
func (m *AccessTokenManager) Token(ctx context.Context) (*Token, error) {
	if tok, ok := m.cached(); ok {
		return tok, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Re-check under the lock: another goroutine may have fetched while this one
	// was waiting, and fetching again would invalidate the token it just stored.
	if tok, ok := m.cached(); ok {
		return tok, nil
	}

	return m.fetchLocked(ctx)
}

// Clear removes the cached token.
func (m *AccessTokenManager) Clear() {
	m.cache.Delete(m.key)
}

// fetchLocked fetches and stores a token. The caller must hold m.mu.
func (m *AccessTokenManager) fetchLocked(ctx context.Context) (*Token, error) {
	tok, err := m.provider.FetchToken(ctx)
	if err != nil {
		return nil, err
	}

	if tok == nil || tok.AccessToken == "" {
		return nil, errEmptyToken
	}

	m.store(tok)

	return tok, nil
}

// cached returns the stored token when it is still usable.
//
// A corrupt or unparsable entry is treated as a miss rather than an error: the
// cache is an optimization, and a bad entry should cost a fetch, not a failure.
func (m *AccessTokenManager) cached() (*Token, bool) {
	raw, ok := m.cache.Get(m.key)
	if !ok || len(raw) == 0 {
		return nil, false
	}

	var tok Token
	if err := codec.Unmarshal(raw, &tok); err != nil {
		return nil, false
	}

	if tok.Expired(m.leeway) {
		return nil, false
	}

	return &tok, true
}

// store caches tok, skipping storage when its remaining TTL is non-positive.
//
// Skipping is not the same as storing with a negative TTL: the latter deletes,
// which is right, but it also hides the fact that leeway exceeded the lifetime.
// Either way the token is still returned to the caller.
func (m *AccessTokenManager) store(tok *Token) {
	ttl := tok.TTL(m.leeway)
	if ttl <= 0 {
		return
	}

	raw, err := codec.Marshal(tok)
	if err != nil {
		return
	}

	m.cache.Set(m.key, raw, ttl)
}

// compile-time guarantee that the manager satisfies the middleware contract.
var _ interface {
	Get(ctx context.Context) (string, error)
	Refresh(ctx context.Context) (string, error)
} = (*AccessTokenManager)(nil)
