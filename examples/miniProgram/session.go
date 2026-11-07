package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/antsurge/atto-wechat/core/session"
	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// sessionContextKey keeps the validated session in the request context.
type sessionContextKey struct{}

// SessionFromContext returns the session stored by RequireSession.
func SessionFromContext(ctx context.Context) (*session.Session, bool) {
	sess, ok := ctx.Value(sessionContextKey{}).(*session.Session)

	return sess, ok
}

// RequireSession 校验请求携带的登录态（Authorization: Bearer <session token>）。
//
// 这里用的是本地校验 app.Session：只查会话是否存在、是否过期，成本低，适合
// 每个已登录请求都执行。如果想确认微信侧的 session_key 也仍然有效，把
// app.Session 换成 app.VerifySession，但注意它会调用 /wxa/checksession，
// 有频率限制，建议只在登录后或低频时机执行。
func RequireSession(app *miniProgram.Application) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			const prefix = "Bearer "

			token := ""
			if header := r.Header.Get("Authorization"); strings.HasPrefix(header, prefix) {
				token = strings.TrimPrefix(header, prefix)
			}

			sess, err := app.Session(token)
			if err != nil {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})

				return
			}

			ctx := context.WithValue(r.Context(), sessionContextKey{}, sess)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
