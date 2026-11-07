package miniProgram

import (
	"context"
	"time"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/session"
)

// Session validates the login session behind token.
//
// The check is local: the session must exist in the session manager and not be
// expired. It costs nothing, so it can run on every request an authenticated
// user makes.
func (a *Application) Session(token string) (*session.Session, error) {
	if token == "" {
		return nil, cerrors.New(cerrors.CodeUnauthorized, "miniprogram: session token must not be empty")
	}

	s, ok := a.sessions.Get(token)
	if !ok || s.Expired(time.Now()) {
		// Drop the entry so the openid index cannot point at a dead session.
		a.sessions.Revoke(token)

		return nil, cerrors.New(cerrors.CodeUnauthorized, "miniprogram: session not found or expired")
	}

	return s, nil
}

// VerifySession validates the session both locally and against WeChat.
//
// It calls /wxa/checksession with a signature derived from the stored
// session_key, so a session that WeChat has invalidated -- the user logged in
// elsewhere, the session_key was reset -- is rejected and removed locally too.
// That also means one WeChat API call per verification, which is subject to the
// endpoint's quota: run it at login and at low frequency, not on every request.
//
// A transport failure is returned as-is instead of revoking the session: WeChat
// being unreachable says nothing about whether the login state is still valid.
func (a *Application) VerifySession(ctx context.Context, token string) (*session.Session, error) {
	s, err := a.Session(token)
	if err != nil {
		return nil, err
	}

	res, err := a.CheckSessionKey(ctx, s.OpenID, CheckSessionSignature(s.SessionKey))
	if err != nil {
		return nil, err
	}

	if !res.OK() {
		a.sessions.Revoke(token)

		return nil, cerrors.Newf(
			cerrors.CodeUnauthorized,
			"miniprogram: session was rejected by wechat: %s",
			CheckSessionKeyErrMsg(res.ErrCode),
		)
	}

	return s, nil
}
