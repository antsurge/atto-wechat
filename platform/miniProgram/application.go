package miniProgram

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/antsurge/atto-wechat/core/auth"
	"github.com/antsurge/atto-wechat/core/cache"
	"github.com/antsurge/atto-wechat/core/client"
	"github.com/antsurge/atto-wechat/core/config"
	"github.com/antsurge/atto-wechat/core/event"
	"github.com/antsurge/atto-wechat/core/middleware"
	"github.com/antsurge/atto-wechat/core/server"
	"github.com/antsurge/atto-wechat/core/session"
)

// tokenPath is the endpoint that issues the app access token.
const tokenPath = "/cgi-bin/token"

// Application is a fully wired mini program client.
type Application struct {
	cfg *Config

	// base speaks to WeChat without a token: it fetches tokens and serves the
	// endpoints that take appid/secret instead.
	base *client.Client
	// api carries the access token on every request.
	api *client.Client

	tokens   *auth.AccessTokenManager
	sessions *session.Manager

	dispatcher *event.Dispatcher
	callback   *server.Server
}

// New validates cfg and assembles the application.
//
// Nothing here performs I/O: the first token fetch happens on the first API call,
// so construction cannot fail because WeChat was unreachable.
func New(cfg *Config) (*Application, error) {
	if cfg == nil {
		return nil, errors.New("miniprogram: config must not be nil")
	}
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, errors.New("miniprogram: AppID and AppSecret are required")
	}

	coreCfg := config.New(append([]config.Option{
		config.WithBaseURL(DefaultBaseURL),
	}, cfg.CoreOptions...)...)

	base, err := client.New(coreCfg)
	if err != nil {
		return nil, err
	}

	store := cfg.Cache
	if store == nil {
		store = cache.NewMemoryCache()
	}

	provider := auth.NewHTTPTokenProvider(base, tokenPath, url.Values{
		"grant_type": {"client_credential"},
		"appid":      {cfg.AppID},
		"secret":     {cfg.AppSecret},
	})

	tokens, err := auth.NewAccessTokenManager(
		store,
		provider,
		// Scope the key by appid: two applications sharing a cache would
		// otherwise overwrite each other's token, and each fetch invalidates the
		// token the other just stored.
		auth.WithKey("atto-wechat:access_token:"+cfg.AppID),
	)
	if err != nil {
		return nil, err
	}

	sessionOpts := []session.Option{
		session.WithKeyPrefix("atto-wechat:session:" + cfg.AppID + ":"),
	}
	if cfg.SessionTTL > 0 {
		sessionOpts = append(sessionOpts, session.WithTTL(cfg.SessionTTL))
	}

	sessions, err := session.NewManager(store, sessionOpts...)
	if err != nil {
		return nil, err
	}

	mw := cfg.Middleware
	if len(mw) == 0 {
		// Defaults are ordered outermost-first: retry the whole call, log it,
		// then attach the token (refreshing it if WeChat rejects it).
		mw = []middleware.Middleware{
			middleware.Retry(coreCfg),
			middleware.Logging(coreCfg),
		}
	}

	api := base.WithMiddleware(append(append([]middleware.Middleware{}, mw...),
		middleware.WithToken(tokens))...)

	dedup := cfg.DedupStore
	if dedup == nil {
		// MemoryCache satisfies SetNXer; any distributed backend that does too
		// works unchanged.
		if nx, ok := store.(cache.SetNXer); ok {
			dedup = nx
		}
	}

	dispatcher := event.NewDispatcher(
		event.WithDedup(dedup, cfg.DedupTTL),
		event.WithLogger(coreCfg.Logger),
		event.WithErrorHandler(cfg.OnHandlerError),
	)

	app := &Application{
		cfg:        cfg,
		base:       base,
		api:        api,
		tokens:     tokens,
		sessions:   sessions,
		dispatcher: dispatcher,
	}

	if cfg.Token != "" || cfg.AESKey != "" {
		cb, err := server.New(&server.Config{
			Token:      cfg.Token,
			AESKey:     cfg.AESKey,
			AppID:      cfg.AppID,
			Dispatcher: dispatcher,
			Logger:     coreCfg.Logger,
		})
		if err != nil {
			return nil, err
		}
		app.callback = cb
	}

	return app, nil
}

// Client returns the HTTP client that injects the access token.
func (a *Application) Client() *client.Client {
	return a.api
}

// BaseClient returns the HTTP client without token injection, for endpoints that
// authenticate with appid/secret.
func (a *Application) BaseClient() *client.Client {
	return a.base
}

// Tokens returns the access token manager.
func (a *Application) Tokens() *auth.AccessTokenManager {
	return a.tokens
}

// Sessions returns the login session manager.
func (a *Application) Sessions() *session.Manager {
	return a.sessions
}

// Dispatcher returns the callback dispatcher, so handlers can be registered.
func (a *Application) Dispatcher() *event.Dispatcher {
	return a.dispatcher
}

// Server returns the callback endpoint, or nil when Token and AESKey are unset.
func (a *Application) Server() *server.Server {
	return a.callback
}

// Callback returns the callback endpoint as an http.Handler, or nil when the
// callback is not configured.
func (a *Application) Callback() http.Handler {
	if a.callback == nil {
		return nil
	}

	return a.callback
}

// AccessToken returns a valid API access token.
func (a *Application) AccessToken(ctx context.Context) (string, error) {
	return a.tokens.Get(ctx)
}

// Login performs Code2Session and starts a session.
//
// It is a convenience for the common case; applications that need a different
// session policy can call Code2Session and session.Manager themselves.
func (a *Application) Login(ctx context.Context, code string) (*session.Session, error) {
	result, err := a.Code2Session(ctx, code)
	if err != nil {
		return nil, err
	}

	return a.sessions.Create(result.OpenID, result.UnionID, result.SessionKey)
}
