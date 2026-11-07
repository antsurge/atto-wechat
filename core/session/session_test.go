package session

import (
	"testing"
	"time"

	"github.com/antsurge/atto-wechat/core/cache"
	"github.com/antsurge/atto-wechat/core/errors"
)

func newManager(t *testing.T, opts ...Option) (*Manager, cache.Cache) {
	t.Helper()

	c := cache.NewMemoryCache()

	m, err := NewManager(c, opts...)
	if err != nil {
		t.Fatal(err)
	}

	return m, c
}

func TestCreateAndGet(t *testing.T) {
	m, _ := newManager(t)

	s, err := m.Create("openid-1", "unionid-1", "session-key")
	if err != nil {
		t.Fatal(err)
	}

	if s.Token == "" {
		t.Fatal("token was not generated")
	}
	if len(s.Token) < 16 {
		t.Fatalf("token %q is too short to be a credential", s.Token)
	}

	got, ok := m.Get(s.Token)
	if !ok {
		t.Fatal("session not found")
	}
	if got.OpenID != "openid-1" || got.UnionID != "unionid-1" || got.SessionKey != "session-key" {
		t.Fatalf("session = %+v", got)
	}

	// Get must return a copy: mutating it may not affect later reads.
	got.OpenID = "mutated"

	again, _ := m.Get(s.Token)
	if again.OpenID != "openid-1" {
		t.Fatal("Get returned a shared pointer")
	}
}

func TestCreateRequiresOpenID(t *testing.T) {
	m, _ := newManager(t)

	if _, err := m.Create("", "", ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetUnknownToken(t *testing.T) {
	m, _ := newManager(t)

	if _, ok := m.Get(""); ok {
		t.Fatal("empty token resolved")
	}
	if _, ok := m.Get("nope"); ok {
		t.Fatal("unknown token resolved")
	}
}

func TestTokenIndex(t *testing.T) {
	m, _ := newManager(t)

	s, err := m.Create("openid", "", "")
	if err != nil {
		t.Fatal(err)
	}

	if token, ok := m.Token("openid"); !ok || token != s.Token {
		t.Fatalf("Token = %q, %v", token, ok)
	}

	if _, ok := m.Token(""); ok {
		t.Fatal("empty openid resolved")
	}
}

func TestRefresh(t *testing.T) {
	m, _ := newManager(t, WithTTL(time.Hour))

	s, err := m.Create("openid", "", "key")
	if err != nil {
		t.Fatal(err)
	}

	refreshed, err := m.Refresh(s.Token)
	if err != nil {
		t.Fatal(err)
	}

	if refreshed.Token != s.Token {
		t.Fatal("refresh changed the token")
	}
	if refreshed.ExpiresAt.Before(s.ExpiresAt) {
		t.Fatal("refresh did not extend the expiry")
	}
	if refreshed.SessionKey != "key" {
		t.Fatal("refresh lost session data")
	}

	if _, err := m.Refresh("missing"); !errors.Is(err, errors.CodeTokenExpired) {
		t.Fatalf("err = %v", err)
	}
}

func TestRevoke(t *testing.T) {
	m, _ := newManager(t)

	s, err := m.Create("openid", "", "")
	if err != nil {
		t.Fatal(err)
	}

	m.Revoke(s.Token)

	if _, ok := m.Get(s.Token); ok {
		t.Fatal("session survived revocation")
	}
	if _, ok := m.Token("openid"); ok {
		t.Fatal("openid index survived revocation")
	}

	// Revoking an unknown token is a no-op.
	m.Revoke("missing")
	m.Revoke("")
}

func TestRevokeByOpenID(t *testing.T) {
	m, _ := newManager(t)

	s, err := m.Create("openid", "", "")
	if err != nil {
		t.Fatal(err)
	}

	m.RevokeByOpenID("openid")

	if _, ok := m.Token("openid"); ok {
		t.Fatal("index survived RevokeByOpenID")
	}
	if _, ok := m.Get(s.Token); ok {
		t.Fatal("session survived RevokeByOpenID")
	}
}

func TestLoginReplacesPreviousSession(t *testing.T) {
	m, _ := newManager(t)

	first, err := m.Create("openid", "", "key-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := m.Create("openid", "", "key-2")
	if err != nil {
		t.Fatal(err)
	}

	// The old session_key was replaced by WeChat, so the old session can no
	// longer decrypt anything and must not remain usable.
	if _, ok := m.Get(first.Token); ok {
		t.Fatal("previous session survived a new login")
	}

	got, ok := m.Get(second.Token)
	if !ok || got.SessionKey != "key-2" {
		t.Fatalf("current session = %+v, %v", got, ok)
	}
}

func TestSessionExpiry(t *testing.T) {
	m, _ := newManager(t, WithTTL(20*time.Millisecond))

	s, err := m.Create("openid", "", "")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(40 * time.Millisecond)

	if _, ok := m.Get(s.Token); ok {
		t.Fatal("expired session still resolved")
	}
}

func TestExpired(t *testing.T) {
	tests := []struct {
		name string
		s    *Session
		want bool
	}{
		{"nil", nil, true},
		{"no token", &Session{}, true},
		{"no deadline", &Session{Token: "t"}, false},
		{"future", &Session{Token: "t", ExpiresAt: time.Now().Add(time.Hour)}, false},
		{"past", &Session{Token: "t", ExpiresAt: time.Now().Add(-time.Minute)}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.s.Expired(time.Now()) != tt.want {
				t.Errorf("Expired = %v, want %v", tt.s.Expired(time.Now()), tt.want)
			}
		})
	}
}

func TestNewManagerValidation(t *testing.T) {
	if _, err := NewManager(nil); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil cache err = %v", err)
	}
}

func TestWithKeyPrefix(t *testing.T) {
	m, c := newManager(t, WithKeyPrefix("app:"))

	s, err := m.Create("openid", "", "")
	if err != nil {
		t.Fatal(err)
	}

	// Nothing may be stored under the default prefix once a custom one is set.
	if _, ok := c.Get(DefaultKeyPrefix + s.Token); ok {
		t.Fatal("session used the default prefix")
	}
	if _, ok := c.Get("app:" + s.Token); !ok {
		t.Fatal("session was not stored under the custom prefix")
	}
	if _, ok := c.Get(DefaultKeyPrefix + "openid:openid"); ok {
		t.Fatal("openid index used the default prefix")
	}
}

func TestTokensAreUnique(t *testing.T) {
	m, _ := newManager(t)

	seen := make(map[string]struct{}, 100)

	for i := 0; i < 100; i++ {
		s, err := m.Create("openid-"+string(rune('a'+i%26)), "", "")
		if err != nil {
			t.Fatal(err)
		}
		if _, dup := seen[s.Token]; dup {
			t.Fatalf("duplicate token %q", s.Token)
		}
		seen[s.Token] = struct{}{}
	}
}

func TestTTLAccessor(t *testing.T) {
	m, _ := newManager(t, WithTTL(time.Hour))

	if m.TTL() != time.Hour {
		t.Fatalf("TTL = %v", m.TTL())
	}

	// Non-positive values are ignored rather than disabling sessions.
	m2, _ := newManager(t, WithTTL(0))
	if m2.TTL() != DefaultTTL {
		t.Fatalf("TTL = %v", m2.TTL())
	}
}
