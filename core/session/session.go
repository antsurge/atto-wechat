// Package session manages the login sessions created from a WeChat mini
// program's code2Session exchange.
//
// A session is more than a token: it holds the openid, the optional unionid and
// the session_key, the last of which is required to decrypt user data and
// phone numbers. Storing it in the same cache used for access tokens means a
// production deployment can swap both for a shared backend at once, and a
// restart does not log everybody out.
package session

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/antsurge/atto-wechat/core/cache"
	"github.com/antsurge/atto-wechat/core/codec"
	"github.com/antsurge/atto-wechat/core/errors"
)

const (
	// DefaultTTL is how long a session stays valid without being used.
	DefaultTTL = 30 * 24 * time.Hour

	// DefaultKeyPrefix is prepended to the token to form a cache key.
	DefaultKeyPrefix = "atto-wechat:session:"

	// tokenBytes is the entropy of a session token. 16 bytes (128 bits) is far
	// beyond guessing; the value is only ever looked up, never parsed.
	tokenBytes = 16
)

// Session is a logged-in mini program user.
type Session struct {
	// Token is the opaque credential handed to the client.
	Token string `json:"token"`
	// OpenID identifies the user within this application.
	OpenID string `json:"openid"`
	// UnionID identifies the user across applications of one publisher.
	UnionID string `json:"unionid,omitempty"`
	// SessionKey decrypts user data and phone numbers. Never send it to a client.
	SessionKey string `json:"session_key,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Expired reports whether the session is unusable at now.
func (s *Session) Expired(now time.Time) bool {
	if s == nil || s.Token == "" {
		return true
	}
	if s.ExpiresAt.IsZero() {
		return false
	}

	return !now.Before(s.ExpiresAt)
}

// Manager creates, looks up, refreshes and revokes sessions.
//
// It is safe for concurrent use: every operation is a single cache call, so a
// shared distributed cache gives the same behaviour across processes.
type Manager struct {
	cache        cache.Cache
	ttl          time.Duration
	keyPrefix    string
	openIDPrefix string
}

// Option mutates a Manager.
type Option func(*Manager)

// WithTTL sets the session lifetime.
func WithTTL(ttl time.Duration) Option {
	return func(m *Manager) {
		if ttl > 0 {
			m.ttl = ttl
		}
	}
}

// WithKeyPrefix overrides the cache key prefix.
func WithKeyPrefix(prefix string) Option {
	return func(m *Manager) {
		if prefix != "" {
			m.keyPrefix = prefix
			m.openIDPrefix = prefix + "openid:"
		}
	}
}

// NewManager creates a manager backed by c.
func NewManager(c cache.Cache, opts ...Option) (*Manager, error) {
	if c == nil {
		return nil, errInvalidCache
	}

	m := &Manager{
		cache:        c,
		ttl:          DefaultTTL,
		keyPrefix:    DefaultKeyPrefix,
		openIDPrefix: DefaultKeyPrefix + "openid:",
	}

	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}

	return m, nil
}

// Create starts a session for a logged-in user.
//
// A new login revokes the previous session of the same openid: the old session_key
// is replaced by WeChat on every code2Session call, so a session still holding
// it could no longer decrypt anything and would fail in confusing ways.
func (m *Manager) Create(openID, unionID, sessionKey string) (*Session, error) {
	if openID == "" {
		return nil, errors.New(errors.CodeInvalidArgument, "openid must not be empty")
	}

	token, err := newToken()
	if err != nil {
		return nil, err
	}

	if previous, ok := m.tokenForOpenID(openID); ok {
		m.Revoke(previous)
	}

	now := time.Now()

	s := &Session{
		Token:      token,
		OpenID:     openID,
		UnionID:    unionID,
		SessionKey: sessionKey,
		CreatedAt:  now,
		ExpiresAt:  now.Add(m.ttl),
	}

	raw, err := codec.Marshal(s)
	if err != nil {
		return nil, errors.Wrap(errors.CodeEncodeFailed, "marshal session", err)
	}

	m.cache.Set(m.key(token), raw, m.ttl)
	m.cache.Set(m.openIDKey(openID), []byte(token), m.ttl)

	return s, nil
}

// Get returns the session for token, or nil when it does not exist or expired.
//
// The returned session is a copy: callers may modify it freely.
func (m *Manager) Get(token string) (*Session, bool) {
	if token == "" {
		return nil, false
	}

	raw, ok := m.cache.Get(m.key(token))
	if !ok || len(raw) == 0 {
		return nil, false
	}

	var s Session
	if err := codec.Unmarshal(raw, &s); err != nil {
		return nil, false
	}

	if s.Expired(time.Now()) {
		return nil, false
	}

	return &s, true
}

// Token returns the current token for openID.
func (m *Manager) Token(openID string) (string, bool) {
	return m.tokenForOpenID(openID)
}

// Refresh extends a session's lifetime, keeping its token and contents.
func (m *Manager) Refresh(token string) (*Session, error) {
	s, ok := m.Get(token)
	if !ok {
		return nil, errSessionNotFound
	}

	s.ExpiresAt = time.Now().Add(m.ttl)

	raw, err := codec.Marshal(s)
	if err != nil {
		return nil, errors.Wrap(errors.CodeEncodeFailed, "marshal session", err)
	}

	m.cache.Set(m.key(token), raw, m.ttl)
	m.cache.Set(m.openIDKey(s.OpenID), []byte(token), m.ttl)

	return s, nil
}

// SetSessionKey replaces the session_key of openID's current session without
// extending its lifetime.
//
// WeChat issues a new session_key when a login state is reset, with the expiry
// inherited from the key it replaces. The session has to swap its copy,
// otherwise checksession and data decryption would keep using a dead key.
func (m *Manager) SetSessionKey(openID, sessionKey string) bool {
	if openID == "" || sessionKey == "" {
		return false
	}

	token, ok := m.tokenForOpenID(openID)
	if !ok {
		return false
	}

	s, ok := m.Get(token)
	if !ok {
		return false
	}

	ttl := time.Until(s.ExpiresAt)
	if ttl <= 0 {
		m.Revoke(token)

		return false
	}

	s.SessionKey = sessionKey

	raw, err := codec.Marshal(s)
	if err != nil {
		return false
	}

	m.cache.Set(m.key(token), raw, ttl)

	return true
}

// Revoke deletes a session.
//
// The openid index entry is removed only when it still points at this token,
// otherwise logging in elsewhere would delete the newer session's index.
func (m *Manager) Revoke(token string) {
	if token == "" {
		return
	}

	if s, ok := m.Get(token); ok {
		if current, ok := m.tokenForOpenID(s.OpenID); ok && current == token {
			m.cache.Delete(m.openIDKey(s.OpenID))
		}
	}

	m.cache.Delete(m.key(token))
}

// RevokeByOpenID deletes the current session of openID.
func (m *Manager) RevokeByOpenID(openID string) {
	if token, ok := m.tokenForOpenID(openID); ok {
		m.Revoke(token)
	}
}

// TTL returns the configured session lifetime.
func (m *Manager) TTL() time.Duration {
	return m.ttl
}

// key builds the cache key for a session token.
func (m *Manager) key(token string) string {
	return m.keyPrefix + token
}

// openIDKey builds the cache key for the openid index.
func (m *Manager) openIDKey(openID string) string {
	return m.openIDPrefix + openID
}

// tokenForOpenID reads the openid index.
func (m *Manager) tokenForOpenID(openID string) (string, bool) {
	if openID == "" {
		return "", false
	}

	raw, ok := m.cache.Get(m.openIDKey(openID))
	if !ok || len(raw) == 0 {
		return "", false
	}

	return string(raw), true
}

// newToken generates an opaque session token.
func newToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.Wrap(errors.CodeCryptoFailed, "generate session token", err)
	}

	return hex.EncodeToString(buf), nil
}
