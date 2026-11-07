package miniProgram

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/antsurge/atto-wechat/core/client"
	"github.com/antsurge/atto-wechat/core/config"
	"github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/event"
	"github.com/antsurge/atto-wechat/core/response"
	"github.com/antsurge/atto-wechat/core/security"
	"github.com/antsurge/atto-wechat/core/session"
)

type fakeWechat struct {
	tokenCalls            atomic.Int32
	tokenBody             string
	code2Session          string
	checkSessionKey       string
	resetUserSessionKey   string
	lastAPIToken          string
	lastCode2Query        url.Values
	lastCheckSessionQuery url.Values
	lastResetUserQuery    url.Values
}

func newFakeServer(f *fakeWechat) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case tokenPath:
			f.tokenCalls.Add(1)
			_, _ = io.WriteString(w, f.tokenBody)
		case code2SessionPath:
			f.lastCode2Query = r.URL.Query()
			_, _ = io.WriteString(w, f.code2Session)
		case checkSessionKeyPath:
			f.lastCheckSessionQuery = r.URL.Query()
			_, _ = io.WriteString(w, f.checkSessionKey)
		case resetUserSessionKeyPath:
			f.lastResetUserQuery = r.URL.Query()
			_, _ = io.WriteString(w, f.resetUserSessionKey)
		default:
			f.lastAPIToken = r.URL.Query().Get("access_token")
			_, _ = io.WriteString(w, `{"errcode":0}`)
		}
	}))
}

func quietOptions() []config.Option {
	return []config.Option{
		config.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		config.WithLogLevel(slog.LevelWarn),
	}
}

func newApp(t *testing.T, srvURL string, mutate func(*Config)) *Application {
	t.Helper()

	cfg := &Config{
		AppID:     "wx1",
		AppSecret: "secret",
		CoreOptions: append(quietOptions(),
			config.WithBaseURL(srvURL),
		),
	}

	if mutate != nil {
		mutate(cfg)
	}

	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return app
}

func TestNewValidation(t *testing.T) {
	if _, err := New(nil); err == nil {
		t.Fatal("nil config accepted")
	}

	for _, cfg := range []*Config{{AppID: "wx1"}, {AppSecret: "s"}} {
		if _, err := New(cfg); err == nil {
			t.Fatalf("incomplete config accepted: %+v", cfg)
		}
	}
}

func TestAccessTokenIsCached(t *testing.T) {
	fake := &fakeWechat{tokenBody: `{"access_token":"at","expires_in":7200}`}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	for i := 0; i < 5; i++ {
		token, err := app.AccessToken(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if token != "at" {
			t.Fatalf("token = %q", token)
		}
	}

	if fake.tokenCalls.Load() != 1 {
		t.Fatalf("token requests = %d, want 1", fake.tokenCalls.Load())
	}
}

func TestClientInjectsToken(t *testing.T) {
	fake := &fakeWechat{tokenBody: `{"access_token":"at","expires_in":7200}`}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	_, err := app.Client().Do(context.Background(), &client.Request{
		Method: http.MethodGet,
		Path:   "/api/x",
	})
	if err != nil {
		t.Fatal(err)
	}

	if fake.lastAPIToken != "at" {
		t.Fatalf("api call carried token %q, want at", fake.lastAPIToken)
	}
}

func TestCode2SessionAndLogin(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:    `{"access_token":"at","expires_in":7200}`,
		code2Session: `{"openid":"o1","unionid":"u1","session_key":"sk"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.Code2Session(context.Background(), "code-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.OpenID != "o1" || result.UnionID != "u1" || result.SessionKey != "sk" {
		t.Fatalf("result = %+v", result)
	}
	if fake.lastCode2Query.Get("js_code") != "code-1" {
		t.Fatalf("js_code = %q", fake.lastCode2Query.Get("js_code"))
	}
	if fake.lastCode2Query.Get("appid") != "wx1" {
		t.Fatalf("appid = %q", fake.lastCode2Query.Get("appid"))
	}

	s, err := app.Login(context.Background(), "code-1")
	if err != nil {
		t.Fatal(err)
	}

	got, ok := app.Sessions().Get(s.Token)
	if !ok {
		t.Fatal("session not found")
	}
	if got.OpenID != "o1" || got.SessionKey != "sk" {
		t.Fatalf("session = %+v", got)
	}
}

func TestCode2SessionErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"invalid code", `{"errcode":40029,"errmsg":"invalid code"}`, 40029},
		{"used code", `{"errcode":40163,"errmsg":"code been used"}`, 40163},
		{"blocked user", `{"errcode":40226,"errmsg":"code blocked"}`, 40226},
		{"rate limited", `{"errcode":45011,"errmsg":"too frequent"}`, 45011},
		{"busy", `{"errcode":-1,"errmsg":"system busy"}`, -1},
		{"no openid", `{}`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:    `{"access_token":"at","expires_in":7200}`,
				code2Session: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.Code2Session(context.Background(), "code")
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code {
				t.Fatalf("errcode = %d, want %d", result.ErrCode, tt.code)
			}
			if result.OK() {
				t.Fatalf("response reported OK for body %s", tt.body)
			}
		})
	}
}

func TestCode2SessionErrMsg(t *testing.T) {
	if got, want := Code2SessionErrMsg(45011), Code2SessionErrCodeMap[45011]; got != want {
		t.Fatalf("Code2SessionErrMsg(45011) = %q, want %q", got, want)
	}
	if got := Code2SessionErrMsg(99999); got != Code2SessionDefaultErrMsg {
		t.Fatalf("undocumented errcode = %q, want the default message", got)
	}

	err := Code2SessionError(&Code2SessionResponse{
		WechatResponse: response.WechatResponse{ErrCode: 45011},
	})
	if !strings.Contains(err.Error(), Code2SessionErrCodeMap[45011]) {
		t.Fatalf("err = %q, want the documented description", err)
	}
}

func TestCode2SessionRequiresCode(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.Code2Session(context.Background(), ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckSessionKey(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:       `{"access_token":"at","expires_in":7200}`,
		checkSessionKey: `{"errcode":0,"errmsg":"ok"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.CheckSessionKey(context.Background(), "o1", "sig")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}

	query := fake.lastCheckSessionQuery
	if query.Get("openid") != "o1" {
		t.Fatalf("openid = %q", query.Get("openid"))
	}
	if query.Get("signature") != "sig" {
		t.Fatalf("signature = %q", query.Get("signature"))
	}
	if query.Get("sig_method") != checkSessionKeySigMethod {
		t.Fatalf("sig_method = %q", query.Get("sig_method"))
	}

	// The token middleware must have attached the cached token.
	if query.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", query.Get("access_token"))
	}
}

func TestCheckSessionKeyErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"invalid signature", `{"errcode":87009,"errmsg":"invalid signature"}`, 87009},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:       `{"access_token":"at","expires_in":7200}`,
				checkSessionKey: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.CheckSessionKey(context.Background(), "o1", "sig")
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code {
				t.Fatalf("errcode = %d, want %d", result.ErrCode, tt.code)
			}
			if result.OK() {
				t.Fatalf("response reported OK for body %s", tt.body)
			}

			got := CheckSessionKeyError(result).Error()
			switch tt.code {
			case 87009:
				if desc := CheckSessionKeyErrCodeMap[87009]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 99999:
				if !strings.Contains(got, CheckSessionKeyDefaultErrMsg) {
					t.Fatalf("err = %q, want the default message", got)
				}
				if strings.Contains(got, "raw text") {
					t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
				}
			}
		})
	}
}

func TestCheckSessionKeyRequiresArguments(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.CheckSessionKey(context.Background(), "", "sig"); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("empty openid err = %v", err)
	}
	if _, err := app.CheckSessionKey(context.Background(), "o1", ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("empty signature err = %v", err)
	}
}

func TestCheckSessionSignatureMatchesHMAC(t *testing.T) {
	const sessionKey = "session-key"

	mac := hmac.New(sha256.New, []byte(sessionKey))
	want := hex.EncodeToString(mac.Sum(nil)) // HMAC over an empty message.

	if got := CheckSessionSignature(sessionKey); got != want {
		t.Fatalf("signature = %s, want %s", got, want)
	}
	if CheckSessionSignature("other") == want {
		t.Fatal("signature does not depend on the session key")
	}
}

func TestResetUserSessionKey(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:           `{"access_token":"at","expires_in":7200}`,
		resetUserSessionKey: `{"errcode":0,"errmsg":"ok","openid":"o1","session_key":"sk-new"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.ResetUserSessionKey(
		context.Background(),
		"o1",
		CheckSessionSignature("session-key"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.OpenID != "o1" || result.SessionKey != "sk-new" {
		t.Fatalf("openid/session_key = %q/%q", result.OpenID, result.SessionKey)
	}

	query := fake.lastResetUserQuery
	if query.Get("openid") != "o1" {
		t.Fatalf("openid = %q", query.Get("openid"))
	}
	if query.Get("signature") != CheckSessionSignature("session-key") {
		t.Fatalf("signature = %q", query.Get("signature"))
	}
	if query.Get("sig_method") != resetUserSessionKeySigMethod {
		t.Fatalf("sig_method = %q", query.Get("sig_method"))
	}

	// The token middleware must have attached the cached token.
	if query.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", query.Get("access_token"))
	}
}

func TestResetUserSessionKeyErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"invalid signature", `{"errcode":87009,"errmsg":"invalid signature"}`, 87009},
		{"undocumented", `{"errcode":40066,"errmsg":"raw text"}`, 40066},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:           `{"access_token":"at","expires_in":7200}`,
				resetUserSessionKey: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.ResetUserSessionKey(context.Background(), "o1", "sig")
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code {
				t.Fatalf("errcode = %d, want %d", result.ErrCode, tt.code)
			}
			if result.OK() {
				t.Fatalf("response reported OK for body %s", tt.body)
			}

			got := ResetUserSessionKeyError(result).Error()
			switch tt.code {
			case 87009:
				if desc := ResetUserSessionKeyErrCodeMap[87009]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 40066:
				if !strings.Contains(got, ResetUserSessionKeyDefaultErrMsg) {
					t.Fatalf("err = %q, want the default message", got)
				}
				if strings.Contains(got, "raw text") {
					t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
				}
			}
		})
	}
}

func TestResetUserSessionKeyUpdatesLocalSession(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:           `{"access_token":"at","expires_in":7200}`,
		code2Session:        `{"openid":"o1","session_key":"sk-old"}`,
		resetUserSessionKey: `{"errcode":0,"errmsg":"ok","openid":"o1","session_key":"sk-new"}`,
		checkSessionKey:     `{"errcode":0}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	sess, err := app.Login(context.Background(), "code")
	if err != nil {
		t.Fatal(err)
	}

	result, err := app.ResetUserSessionKey(
		context.Background(),
		sess.OpenID,
		CheckSessionSignature(sess.SessionKey),
	)
	if err != nil || !result.OK() {
		t.Fatalf("result = %+v, err = %v", result, err)
	}

	// The local session must adopt the new session_key returned by WeChat.
	token, ok := app.Sessions().Token("o1")
	if !ok {
		t.Fatal("session index missing")
	}
	stored, ok := app.Sessions().Get(token)
	if !ok {
		t.Fatal("session missing")
	}
	if stored.SessionKey != "sk-new" {
		t.Fatalf("stored session_key = %q, want sk-new", stored.SessionKey)
	}

	// Verification afterwards must sign with the new key.
	if _, err := app.VerifySession(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if want := CheckSessionSignature("sk-new"); fake.lastCheckSessionQuery.Get("signature") != want {
		t.Fatalf("signature = %q, want %q", fake.lastCheckSessionQuery.Get("signature"), want)
	}
}

func TestResetUserSessionKeyRequiresArguments(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.ResetUserSessionKey(context.Background(), "", "sig"); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("empty openid err = %v", err)
	}
	if _, err := app.ResetUserSessionKey(context.Background(), "o1", ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("empty signature err = %v", err)
	}
}

func TestSessionRejectsUnknownToken(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.Session("missing"); !errors.Is(err, errors.CodeUnauthorized) {
		t.Fatalf("err = %v", err)
	}
}

func TestVerifySession(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:       `{"access_token":"at","expires_in":7200}`,
		code2Session:    `{"openid":"o1","unionid":"u1","session_key":"sk"}`,
		checkSessionKey: `{"errcode":0,"errmsg":"ok"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	sess, err := app.Login(context.Background(), "code")
	if err != nil {
		t.Fatal(err)
	}

	// The local check accepts a fresh session without calling WeChat.
	if got, err := app.Session(sess.Token); err != nil || got.OpenID != "o1" {
		t.Fatalf("session = %+v, err = %v", got, err)
	}

	got, err := app.VerifySession(context.Background(), sess.Token)
	if err != nil {
		t.Fatal(err)
	}
	if got.OpenID != "o1" {
		t.Fatalf("openid = %q", got.OpenID)
	}

	// The signature sent to /wxa/checksession must come from session_key.
	if want := CheckSessionSignature("sk"); fake.lastCheckSessionQuery.Get("signature") != want {
		t.Fatalf("signature = %q, want %q", fake.lastCheckSessionQuery.Get("signature"), want)
	}
}

func TestVerifySessionRejectsInvalidatedSession(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:       `{"access_token":"at","expires_in":7200}`,
		code2Session:    `{"openid":"o1","session_key":"sk"}`,
		checkSessionKey: `{"errcode":87009,"errmsg":"invalid signature"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	sess, err := app.Login(context.Background(), "code")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := app.VerifySession(context.Background(), sess.Token); !errors.Is(err, errors.CodeUnauthorized) {
		t.Fatalf("err = %v, want UNAUTHORIZED", err)
	}

	// A rejected session must not linger in the session manager.
	if _, ok := app.Sessions().Token("o1"); ok {
		t.Fatal("rejected session was not revoked")
	}
}

func TestCallbackDisabledWithoutCredentials(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if app.Callback() != nil {
		t.Fatal("callback endpoint exists without Token or AESKey")
	}
	if app.Server() != nil {
		t.Fatal("callback server exists without Token or AESKey")
	}
}

func TestCallbackEnabledAndVerifiesURL(t *testing.T) {
	app := newApp(t, "http://example.invalid", func(c *Config) {
		c.Token = "callback-token"
	})

	h := app.Callback()
	if h == nil {
		t.Fatal("callback endpoint missing")
	}

	req := httptest.NewRequest(http.MethodGet, "/callback?"+url.Values{
		"signature": {security.SHA1Signature("callback-token", "123", "n")},
		"timestamp": {"123"},
		"nonce":     {"n"},
		"echostr":   {"ok"},
	}.Encode(), nil)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK || w.Body.String() != "ok" {
		t.Fatalf("status = %d, body = %q", w.Code, w.Body.String())
	}
}

func TestCallbackDeduplicates(t *testing.T) {
	app := newApp(t, "http://example.invalid", func(c *Config) {
		c.Token = "callback-token"
	})

	var calls atomic.Int32
	app.Dispatcher().On("*", func(context.Context, *event.Message) error {
		calls.Add(1)

		return nil
	})

	msg, err := event.ParseMessage([]byte(`<xml><MsgType>text</MsgType><Content>hi</Content><MsgId>9</MsgId></xml>`))
	if err != nil {
		t.Fatal(err)
	}

	if err := app.Dispatcher().Dispatch(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if err := app.Dispatcher().Dispatch(context.Background(), msg); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("duplicate err = %v", err)
	}

	if calls.Load() != 1 {
		t.Fatalf("calls = %d, want 1", calls.Load())
	}
}

func TestSessionsScopedByApp(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if app.Sessions().TTL() != session.DefaultTTL {
		t.Fatalf("ttl = %v", app.Sessions().TTL())
	}

	s, err := app.Sessions().Create("o1", "", "sk")
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Token) < 16 {
		t.Fatalf("token = %q", s.Token)
	}
}
