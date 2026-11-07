package auth

import "time"

// Token is a credential returned by a WeChat token endpoint.
//
// ExpiresAt is an absolute unix timestamp computed when the token was fetched.
// Storing the deadline rather than the lifetime means a cached token stays
// correct across processes and restarts, which a relative expires_in does not.
type Token struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	RefreshToken string `json:"refresh_token,omitempty"`
	OpenID       string `json:"openid,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// Expired reports whether the token is unusable at now, treating the last leeway
// as already gone.
//
// The leeway exists because a token that is valid for one more second is not
// usable: the request it is attached to will arrive after it expires, and the
// call fails after a round trip. Expiring early turns that into a refresh.
func (t *Token) Expired(leeway time.Duration) bool {
	if t == nil || t.AccessToken == "" {
		return true
	}

	// A token without a deadline cannot be judged, so trust it: providers that
	// omit expires_in are rare, and refusing the token would make them unusable.
	if t.ExpiresAt == 0 {
		return false
	}

	return time.Now().Add(leeway).Unix() >= t.ExpiresAt
}

// TTL returns how long the token may still be cached, after subtracting leeway.
// A non-positive result means it should not be cached at all.
func (t *Token) TTL(leeway time.Duration) time.Duration {
	if t == nil || t.ExpiresAt == 0 {
		return 0
	}

	return time.Until(time.Unix(t.ExpiresAt, 0)) - leeway
}
