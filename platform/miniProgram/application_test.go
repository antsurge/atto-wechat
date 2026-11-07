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
	tokenCalls              atomic.Int32
	tokenBody               string
	code2Session            string
	checkSessionKey         string
	resetUserSessionKey     string
	getPhoneNumber          string
	getPluginOpenPID        string
	checkEncryptedData      string
	getPaidUnionID          string
	getUserEncryptKey       string
	getQRCode               []byte
	getQRCodeContentType    string
	getUnlimitedQRCode      []byte
	getUnlimitedQRCodeType  string
	createQRCode            []byte
	createQRCodeContentType string
	queryScheme             string
	generateScheme          string
	generateNFCScheme       string
	generateURLLink         string
	queryURLLink            string
	generateShortLink       string
	lastAPIToken            string
	lastCode2Query          url.Values
	lastCheckSessionQuery   url.Values
	lastResetUserQuery      url.Values
	lastGetPhoneQuery       url.Values
	lastGetPhoneBody        []byte
	lastPluginOpenPIDQuery  url.Values
	lastPluginOpenPIDBody   []byte
	lastEncryptedQuery      url.Values
	lastEncryptedBody       []byte
	lastPaidUnionQuery      url.Values
	lastEncryptKeyMethod    string
	lastEncryptKeyQuery     url.Values
	lastEncryptKeyBody      []byte
	lastQRCodeQuery         url.Values
	lastQRCodeBody          []byte
	lastUnlimitedQuery      url.Values
	lastUnlimitedBody       []byte
	lastCreateQuery         url.Values
	lastCreateBody          []byte
	lastQuerySchemeQuery    url.Values
	lastQuerySchemeBody     []byte
	lastGenerateQuery       url.Values
	lastGenerateBody        []byte
	lastNFCQuery            url.Values
	lastNFCBody             []byte
	lastURLLinkQuery        url.Values
	lastURLLinkBody         []byte
	lastQueryURLLinkQuery   url.Values
	lastQueryURLLinkBody    []byte
	lastShortLinkQuery      url.Values
	lastShortLinkBody       []byte
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
		case getPhoneNumberPath:
			f.lastGetPhoneQuery = r.URL.Query()
			f.lastGetPhoneBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, f.getPhoneNumber)
		case getPluginOpenPIDPath:
			f.lastPluginOpenPIDQuery = r.URL.Query()
			f.lastPluginOpenPIDBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, f.getPluginOpenPID)
		case checkEncryptedDataPath:
			f.lastEncryptedQuery = r.URL.Query()
			f.lastEncryptedBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, f.checkEncryptedData)
		case getPaidUnionIDPath:
			f.lastPaidUnionQuery = r.URL.Query()
			_, _ = io.WriteString(w, f.getPaidUnionID)
		case getUserEncryptKeyPath:
			f.lastEncryptKeyMethod = r.Method
			f.lastEncryptKeyQuery = r.URL.Query()
			f.lastEncryptKeyBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, f.getUserEncryptKey)
		case getQRCodePath:
			f.lastQRCodeQuery = r.URL.Query()
			f.lastQRCodeBody, _ = io.ReadAll(r.Body)

			contentType := f.getQRCodeContentType
			if contentType == "" {
				contentType = "application/json"
			}
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write(f.getQRCode)
		case getUnlimitedQRCodePath:
			f.lastUnlimitedQuery = r.URL.Query()
			f.lastUnlimitedBody, _ = io.ReadAll(r.Body)

			contentType := f.getUnlimitedQRCodeType
			if contentType == "" {
				contentType = "application/json"
			}
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write(f.getUnlimitedQRCode)
		case createQRCodePath:
			f.lastCreateQuery = r.URL.Query()
			f.lastCreateBody, _ = io.ReadAll(r.Body)

			contentType := f.createQRCodeContentType
			if contentType == "" {
				contentType = "application/json"
			}
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write(f.createQRCode)
		case querySchemePath:
			f.lastQuerySchemeQuery = r.URL.Query()
			f.lastQuerySchemeBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, f.queryScheme)
		case generateSchemePath:
			f.lastGenerateQuery = r.URL.Query()
			f.lastGenerateBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, f.generateScheme)
		case generateNFCSchemePath:
			f.lastNFCQuery = r.URL.Query()
			f.lastNFCBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, f.generateNFCScheme)
		case generateURLLinkPath:
			f.lastURLLinkQuery = r.URL.Query()
			f.lastURLLinkBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, f.generateURLLink)
		case queryURLLinkPath:
			f.lastQueryURLLinkQuery = r.URL.Query()
			f.lastQueryURLLinkBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, f.queryURLLink)
		case generateShortLinkPath:
			f.lastShortLinkQuery = r.URL.Query()
			f.lastShortLinkBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, f.generateShortLink)
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

func TestGetUserEncryptKey(t *testing.T) {
	fake := &fakeWechat{
		tokenBody: `{"access_token":"at","expires_in":7200}`,
		getUserEncryptKey: `{"errcode":0,"errmsg":"ok","key_info_list":[` +
			`{"encrypt_key":"VI6BpyrK9XH4i4AIGe86tg==","version":10,"expire_in":3597,"iv":"6003f73ec441c386","create_time":1616572301},` +
			`{"encrypt_key":"aoUGAHltcliiL9f23oTKHA==","version":9,"expire_in":0,"iv":"7996656384218dbb","create_time":1616504886}]}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GetUserEncryptKey(context.Background(), "o-user", "sig")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() || len(result.KeyInfoList) != 2 {
		t.Fatalf("response = %+v", result)
	}
	if result.KeyInfoList[0].Version != 10 || result.KeyInfoList[0].ExpireIn != 3597 {
		t.Fatalf("key = %+v", result.KeyInfoList[0])
	}

	// The endpoint is a POST whose parameters all sit in the query and whose
	// body must stay empty: WeChat treats any body as an invalid signature.
	if fake.lastEncryptKeyMethod != http.MethodPost {
		t.Fatalf("method = %q", fake.lastEncryptKeyMethod)
	}
	if len(fake.lastEncryptKeyBody) != 0 {
		t.Fatalf("body = %q, want empty", fake.lastEncryptKeyBody)
	}

	q := fake.lastEncryptKeyQuery
	if q.Get("openid") != "o-user" || q.Get("signature") != "sig" || q.Get("sig_method") != "hmac_sha256" {
		t.Fatalf("query = %v", q)
	}
	// The token middleware must have attached the cached token.
	if q.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", q.Get("access_token"))
	}
}

func TestGetUserEncryptKeyErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"expired session key", `{"errcode":87007,"errmsg":"expired"}`, 87007},
		{"invalid signature", `{"errcode":87009,"errmsg":"invalid signature"}`, 87009},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:         `{"access_token":"at","expires_in":7200}`,
				getUserEncryptKey: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.GetUserEncryptKey(context.Background(), "o-user", "sig")
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := GetUserEncryptKeyError(result).Error()
			switch tt.code {
			case 87007:
				if desc := GetUserEncryptKeyErrCodeMap[87007]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 87009:
				if desc := GetUserEncryptKeyErrCodeMap[87009]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 99999:
				if !strings.Contains(got, GetUserEncryptKeyDefaultErrMsg) {
					t.Fatalf("err = %q, want the default message", got)
				}
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "invalid signature") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestGetUserEncryptKeyRequiresParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name      string
		openID    string
		signature string
	}{
		{"missing openid", "", "sig"},
		{"missing signature", "o-user", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.GetUserEncryptKey(context.Background(), tt.openID, tt.signature); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestGetPaidUnionID(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:      `{"access_token":"at","expires_in":7200}`,
		getPaidUnionID: `{"errcode":0,"errmsg":"ok","unionid":"oTmHYjg-tElZ68xxxxxxxxhy1Rgk"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GetPaidUnionID(context.Background(), PaidUnionIDParams{
		OpenID:        "o-user",
		TransactionID: "4200001234",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.UnionID != "oTmHYjg-tElZ68xxxxxxxxhy1Rgk" {
		t.Fatalf("unionid = %q", result.UnionID)
	}

	// The payment identifiers must travel in the query, not in a body.
	q := fake.lastPaidUnionQuery
	if q.Get("openid") != "o-user" || q.Get("transaction_id") != "4200001234" {
		t.Fatalf("query = %v", q)
	}
	// The token middleware must have attached the cached token.
	if q.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", q.Get("access_token"))
	}
}

func TestGetPaidUnionIDByOutTradeNo(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:      `{"access_token":"at","expires_in":7200}`,
		getPaidUnionID: `{"errcode":0,"errmsg":"ok","unionid":"oTmHYjg-tElZ68xxxxxxxxhy1Rgk"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GetPaidUnionID(context.Background(), PaidUnionIDParams{
		OpenID:     "o-user",
		MchID:      "1900000109",
		OutTradeNo: "merchant-order",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}

	// The merchant pair replaces transaction_id, it must not be sent as well.
	q := fake.lastPaidUnionQuery
	if q.Get("mch_id") != "1900000109" || q.Get("out_trade_no") != "merchant-order" {
		t.Fatalf("query = %v", q)
	}
	if q.Get("transaction_id") != "" {
		t.Fatalf("query = %v, want no transaction_id", q)
	}
}

func TestGetPaidUnionIDErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"not bound", `{"errcode":89002,"errmsg":"not bound"}`, 89002},
		{"invalid trade", `{"errcode":89300,"errmsg":"invalid trade"}`, 89300},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:      `{"access_token":"at","expires_in":7200}`,
				getPaidUnionID: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.GetPaidUnionID(context.Background(), PaidUnionIDParams{
				OpenID:        "o-user",
				TransactionID: "4200001234",
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := GetPaidUnionIDError(result).Error()
			switch tt.code {
			case 89002:
				if desc := GetPaidUnionIDErrCodeMap[89002]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 89300:
				if desc := GetPaidUnionIDErrCodeMap[89300]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 99999:
				if !strings.Contains(got, GetPaidUnionIDDefaultErrMsg) {
					t.Fatalf("err = %q, want the default message", got)
				}
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "invalid trade") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestGetPaidUnionIDRequiresParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params PaidUnionIDParams
	}{
		{"missing openid", PaidUnionIDParams{TransactionID: "4200001234"}},
		{"missing payment proof", PaidUnionIDParams{OpenID: "o-user"}},
		{"incomplete merchant order", PaidUnionIDParams{OpenID: "o-user", MchID: "1900000109"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.GetPaidUnionID(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestGenerateScheme(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:      `{"access_token":"at","expires_in":7200}`,
		generateScheme: `{"errcode":0,"errmsg":"ok","openlink":"weixin://dl/business/?t=XTSkBZlzqmn"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GenerateScheme(context.Background(), GenerateSchemeParams{
		JumpWXA: &JumpWXA{Path: "/pages/index/index"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.OpenLink != "weixin://dl/business/?t=XTSkBZlzqmn" {
		t.Fatalf("openlink = %q", result.OpenLink)
	}

	// A permanent link sets nothing beyond the page, so is_expire and the
	// expire fields must stay off the body and WeChat's defaults apply.
	if body := string(fake.lastGenerateBody); body != `{"jump_wxa":{"path":"/pages/index/index"}}` {
		t.Fatalf("body = %s", body)
	}
	if fake.lastGenerateQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastGenerateQuery.Get("access_token"))
	}
}

func TestGenerateSchemeExpire(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:      `{"access_token":"at","expires_in":7200}`,
		generateScheme: `{"errcode":0,"errmsg":"ok","openlink":"weixin://dl/business/?t=abc"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	_, err := app.GenerateScheme(context.Background(), GenerateSchemeParams{
		JumpWXA: &JumpWXA{
			Path:       "/pages/index/index",
			Query:      "a=1&b=2",
			EnvVersion: "trial",
		},
		IsExpire:       true,
		ExpireType:     1,
		ExpireInterval: 7,
	})
	if err != nil {
		t.Fatal(err)
	}

	body := string(fake.lastGenerateBody)
	for _, want := range []string{
		`"path":"/pages/index/index"`,
		`"query":"a=1&b=2"`,
		`"env_version":"trial"`,
		`"is_expire":true`,
		`"expire_type":1`,
		`"expire_interval":7`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %s, want %s", body, want)
		}
	}
}

func TestGenerateSchemeErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"bad expire time", `{"errcode":85401,"errmsg":"time limit between 1min and 30days"}`, 85401},
		{"bad env version", `{"errcode":85402,"errmsg":"invalid env_version"}`, 85402},
		{"not published", `{"errcode":85079,"errmsg":"miniprogram has no online release"}`, 85079},
		{"daily limit", `{"errcode":44993,"errmsg":"reach max api day frequence limit"}`, 44993},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:      `{"access_token":"at","expires_in":7200}`,
				generateScheme: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.GenerateScheme(context.Background(), GenerateSchemeParams{
				JumpWXA: &JumpWXA{Path: "/pages/index/index"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := GenerateSchemeError(result).Error()
			if desc, ok := GenerateSchemeErrCodeMap[tt.code]; ok {
				if !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			} else if !strings.Contains(got, GenerateSchemeDefaultErrMsg) {
				t.Fatalf("err = %q, want the default message", got)
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "rid:") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestGenerateSchemeRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params GenerateSchemeParams
	}{
		{"path with query", GenerateSchemeParams{
			JumpWXA: &JumpWXA{Path: "/pages/index/index?a=1"},
		}},
		{"query too long", GenerateSchemeParams{
			JumpWXA: &JumpWXA{Path: "/pages/index/index", Query: strings.Repeat("a", 1025)},
		}},
		{"query with unsupported char", GenerateSchemeParams{
			JumpWXA: &JumpWXA{Path: "/pages/index/index", Query: "中文"},
		}},
		{"bad env version", GenerateSchemeParams{
			JumpWXA: &JumpWXA{Path: "/pages/index/index", EnvVersion: "staging"},
		}},
		{"expire time missing", GenerateSchemeParams{
			JumpWXA:    &JumpWXA{Path: "/pages/index/index"},
			IsExpire:   true,
			ExpireType: 0,
		}},
		{"expire interval missing", GenerateSchemeParams{
			JumpWXA:        &JumpWXA{Path: "/pages/index/index"},
			IsExpire:       true,
			ExpireType:     1,
			ExpireInterval: 0,
		}},
		{"expire interval too long", GenerateSchemeParams{
			JumpWXA:        &JumpWXA{Path: "/pages/index/index"},
			IsExpire:       true,
			ExpireType:     1,
			ExpireInterval: 31,
		}},
		{"bad expire type", GenerateSchemeParams{
			JumpWXA:        &JumpWXA{Path: "/pages/index/index"},
			IsExpire:       true,
			ExpireType:     2,
			ExpireInterval: 7,
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.GenerateScheme(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestGenerateNFCScheme(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:         `{"access_token":"at","expires_in":7200}`,
		generateNFCScheme: `{"errcode":0,"errmsg":"ok","openlink":"weixin://dl/business/?t=XTSkBZlzqmn"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GenerateNFCScheme(context.Background(), GenerateNFCSchemeParams{
		JumpWXA: &JumpWXA{Path: "/pages/index/index"},
		ModelID: "model-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.OpenLink != "weixin://dl/business/?t=XTSkBZlzqmn" {
		t.Fatalf("openlink = %q", result.OpenLink)
	}

	// model_id is mandatory; sn stays off the body because a model provisioned
	// one code per model must not receive one.
	if body := string(fake.lastNFCBody); body != `{"jump_wxa":{"path":"/pages/index/index"},"model_id":"model-1"}` {
		t.Fatalf("body = %s", body)
	}
	if fake.lastNFCQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastNFCQuery.Get("access_token"))
	}
}

func TestGenerateNFCSchemeWithSN(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:         `{"access_token":"at","expires_in":7200}`,
		generateNFCScheme: `{"errcode":0,"errmsg":"ok","openlink":"weixin://dl/business/?t=abc"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	_, err := app.GenerateNFCScheme(context.Background(), GenerateNFCSchemeParams{
		JumpWXA: &JumpWXA{
			Path:       "/pages/index/index",
			Query:      "a=1",
			EnvVersion: "trial",
		},
		ModelID: "model-1",
		SN:      "sn-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := string(fake.lastNFCBody)
	for _, want := range []string{
		`"path":"/pages/index/index"`,
		`"query":"a=1"`,
		`"env_version":"trial"`,
		`"model_id":"model-1"`,
		`"sn":"sn-1"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %s, want %s", body, want)
		}
	}
}

func TestGenerateNFCSchemeErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"no permission", `{"errcode":40002,"errmsg":"invalid grant_type"}`, 40002},
		{"bad model id", `{"errcode":9800003,"errmsg":"model_id check failed"}`, 9800003},
		{"sn required", `{"errcode":9800008,"errmsg":"sn is required"}`, 9800008},
		{"quota", `{"errcode":85400,"errmsg":"reach max long time quota limit"}`, 85400},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:         `{"access_token":"at","expires_in":7200}`,
				generateNFCScheme: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.GenerateNFCScheme(context.Background(), GenerateNFCSchemeParams{
				JumpWXA: &JumpWXA{Path: "/pages/index/index"},
				ModelID: "model-1",
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := GenerateNFCSchemeError(result).Error()
			if desc, ok := GenerateNFCSchemeErrCodeMap[tt.code]; ok {
				if !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			} else if !strings.Contains(got, GenerateNFCSchemeDefaultErrMsg) {
				t.Fatalf("err = %q, want the default message", got)
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "rid:") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestGenerateNFCSchemeRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params GenerateNFCSchemeParams
	}{
		{"missing model id", GenerateNFCSchemeParams{
			JumpWXA: &JumpWXA{Path: "/pages/index/index"},
		}},
		{"path with query", GenerateNFCSchemeParams{
			JumpWXA: &JumpWXA{Path: "/pages/index/index?a=1"},
			ModelID: "model-1",
		}},
		{"query too long", GenerateNFCSchemeParams{
			JumpWXA: &JumpWXA{Path: "/pages/index/index", Query: strings.Repeat("a", 1025)},
			ModelID: "model-1",
		}},
		{"query with unsupported char", GenerateNFCSchemeParams{
			JumpWXA: &JumpWXA{Path: "/pages/index/index", Query: "中文"},
			ModelID: "model-1",
		}},
		{"bad env version", GenerateNFCSchemeParams{
			JumpWXA: &JumpWXA{Path: "/pages/index/index", EnvVersion: "staging"},
			ModelID: "model-1",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.GenerateNFCScheme(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestGenerateURLLink(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:       `{"access_token":"at","expires_in":7200}`,
		generateURLLink: `{"errcode":0,"errmsg":"ok","url_link":"https://wxaurl.cn/XTSkBZlzqmn"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GenerateURLLink(context.Background(), GenerateURLLinkParams{
		Path: "/pages/index/index",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.UrlLink != "https://wxaurl.cn/XTSkBZlzqmn" {
		t.Fatalf("url_link = %q", result.UrlLink)
	}

	// A permanent link sets nothing beyond the page, so is_expire and the
	// expire fields must stay off the body.
	if body := string(fake.lastURLLinkBody); body != `{"path":"/pages/index/index"}` {
		t.Fatalf("body = %s", body)
	}
	if fake.lastURLLinkQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastURLLinkQuery.Get("access_token"))
	}
}

func TestGenerateURLLinkExpireWithCloudBase(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:       `{"access_token":"at","expires_in":7200}`,
		generateURLLink: `{"errcode":0,"errmsg":"ok","url_link":"https://wxaurl.cn/abc"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	_, err := app.GenerateURLLink(context.Background(), GenerateURLLinkParams{
		Path:           "/pages/publishHomework/publishHomework",
		Query:          "a=1&b=2",
		EnvVersion:     "trial",
		IsExpire:       true,
		ExpireType:     generateSchemeExpireTypeInterval,
		ExpireInterval: 3,
		CloudBase: &CloudBase{
			Env:    "env-1",
			Domain: "xxx.xx",
			Path:   "/jump-wxa.html",
			Query:  "a=1&b=2",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := string(fake.lastURLLinkBody)
	for _, want := range []string{
		`"path":"/pages/publishHomework/publishHomework"`,
		`"query":"a=1&b=2"`,
		`"env_version":"trial"`,
		`"is_expire":true`,
		`"expire_type":1`,
		`"expire_interval":3`,
		`"cloud_base":{"env":"env-1","domain":"xxx.xx","path":"/jump-wxa.html","query":"a=1&b=2"}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %s, want %s", body, want)
		}
	}
}

func TestGenerateURLLinkErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"no permission", `{"errcode":40002,"errmsg":"invalid grant_type"}`, 40002},
		{"not published", `{"errcode":85079,"errmsg":"miniprogram has no online release"}`, 85079},
		{"cloud not opened", `{"errcode":85088,"errmsg":"cloud base not opened"}`, 85088},
		{"bad expire time", `{"errcode":85401,"errmsg":"time limit between 1min and 30days"}`, 85401},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:       `{"access_token":"at","expires_in":7200}`,
				generateURLLink: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.GenerateURLLink(context.Background(), GenerateURLLinkParams{
				Path: "/pages/index/index",
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := GenerateURLLinkError(result).Error()
			if desc, ok := GenerateURLLinkErrCodeMap[tt.code]; ok {
				if !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			} else if !strings.Contains(got, GenerateURLLinkDefaultErrMsg) {
				t.Fatalf("err = %q, want the default message", got)
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "rid:") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestGenerateURLLinkRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params GenerateURLLinkParams
	}{
		{"path with query", GenerateURLLinkParams{
			Path: "/pages/index/index?a=1",
		}},
		{"query too long", GenerateURLLinkParams{
			Path:  "/pages/index/index",
			Query: strings.Repeat("a", 1025),
		}},
		{"query with unsupported char", GenerateURLLinkParams{
			Path:  "/pages/index/index",
			Query: "中文",
		}},
		{"bad env version", GenerateURLLinkParams{
			Path:       "/pages/index/index",
			EnvVersion: "staging",
		}},
		{"expire time missing", GenerateURLLinkParams{
			Path:       "/pages/index/index",
			IsExpire:   true,
			ExpireType: generateSchemeExpireTypeTime,
		}},
		{"expire interval missing", GenerateURLLinkParams{
			Path:           "/pages/index/index",
			IsExpire:       true,
			ExpireType:     generateSchemeExpireTypeInterval,
			ExpireInterval: 0,
		}},
		{"expire interval too long", GenerateURLLinkParams{
			Path:           "/pages/index/index",
			IsExpire:       true,
			ExpireType:     generateSchemeExpireTypeInterval,
			ExpireInterval: 31,
		}},
		{"bad expire type", GenerateURLLinkParams{
			Path:           "/pages/index/index",
			IsExpire:       true,
			ExpireType:     2,
			ExpireInterval: 7,
		}},
		{"cloud base without env", GenerateURLLinkParams{
			Path:      "/pages/index/index",
			CloudBase: &CloudBase{Domain: "xxx.xx"},
		}},
		{"cloud base path with query", GenerateURLLinkParams{
			Path:      "/pages/index/index",
			CloudBase: &CloudBase{Env: "env-1", Path: "/jump-wxa.html?a=1"},
		}},
		{"cloud base query too long", GenerateURLLinkParams{
			Path:      "/pages/index/index",
			CloudBase: &CloudBase{Env: "env-1", Query: strings.Repeat("a", 1025)},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.GenerateURLLink(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestGenerateShortLink(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:         `{"access_token":"at","expires_in":7200}`,
		generateShortLink: `{"errcode":0,"errmsg":"ok","link":"https://w.url.cn/s/abcdefg"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GenerateShortLink(context.Background(), GenerateShortLinkParams{
		PageURL: "pages/publishHomework/publishHomework?query1=q1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.Link != "https://w.url.cn/s/abcdefg" {
		t.Fatalf("link = %q", result.Link)
	}

	// page_url keeps its own query here, unlike path elsewhere. page_title and
	// a false is_permanent both stay off the body.
	body := string(fake.lastShortLinkBody)
	if body != `{"page_url":"pages/publishHomework/publishHomework?query1=q1"}` {
		t.Fatalf("body = %s", body)
	}
	if fake.lastShortLinkQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastShortLinkQuery.Get("access_token"))
	}
}

func TestGenerateShortLinkPermanent(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:         `{"access_token":"at","expires_in":7200}`,
		generateShortLink: `{"errcode":0,"errmsg":"ok","link":"https://w.url.cn/s/abc"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	_, err := app.GenerateShortLink(context.Background(), GenerateShortLinkParams{
		PageURL:     "pages/index/index",
		PageTitle:   "Homework title",
		IsPermanent: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	body := string(fake.lastShortLinkBody)
	for _, want := range []string{
		`"page_url":"pages/index/index"`,
		`"page_title":"Homework title"`,
		`"is_permanent":true`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %s, want %s", body, want)
		}
	}
}

func TestGenerateShortLinkErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"bad token", `{"errcode":40001,"errmsg":"invalid credential"}`, 40001},
		{"unknown url", `{"errcode":40066,"errmsg":"invalid url"}`, 40066},
		{"bad title", `{"errcode":40225,"errmsg":"invalid page title"}`, 40225},
		{"no permission", `{"errcode":43104,"errmsg":"this appid does not have permission"}`, 43104},
		{"daily quota", `{"errcode":45009,"errmsg":"reach max api daily quota limit"}`, 45009},
		{"permanent quota", `{"errcode":85400,"errmsg":"reach max long time quota limit"}`, 85400},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:         `{"access_token":"at","expires_in":7200}`,
				generateShortLink: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.GenerateShortLink(context.Background(), GenerateShortLinkParams{
				PageURL: "pages/index/index",
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := GenerateShortLinkError(result).Error()
			if desc, ok := GenerateShortLinkErrCodeMap[tt.code]; ok {
				if !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			} else if !strings.Contains(got, GenerateShortLinkDefaultErrMsg) {
				t.Fatalf("err = %q, want the default message", got)
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "rid:") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestGenerateShortLinkRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params GenerateShortLinkParams
	}{
		{"missing page url", GenerateShortLinkParams{}},
		{"page url too long", GenerateShortLinkParams{
			PageURL: strings.Repeat("a", 1025),
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.GenerateShortLink(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestQueryURLLink(t *testing.T) {
	fake := &fakeWechat{
		tokenBody: `{"access_token":"at","expires_in":7200}`,
		queryURLLink: `{"errcode":0,"errmsg":"ok","url_link_info":{"appid":"wx123",` +
			`"path":"pages/index/index","query":"a=hello","create_time":611928113,` +
			`"expire_time":0,"env_version":"release",` +
			`"cloud_base":{"env":"env-1","domain":"xxx.xx","path":"/jump-wxa.html","query":"a=1"}}}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	urlLink := "https://wxaurl.cn/XTSkBZlzqmn"
	result, err := app.QueryURLLink(context.Background(), QueryURLLinkParams{URLLink: urlLink})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.URLLinkInfo == nil || result.QuotaInfo != nil {
		t.Fatalf("url_link_info = %+v, quota_info = %+v", result.URLLinkInfo, result.QuotaInfo)
	}

	info := result.URLLinkInfo
	if info.AppID != "wx123" || info.Path != "pages/index/index" || info.Query != "a=hello" {
		t.Fatalf("url link info = %+v", info)
	}
	if info.EnvVersion != "release" || info.CreateTime != 611928113 || info.ExpireTime != 0 {
		t.Fatalf("url link info = %+v", info)
	}
	if info.CloudBase == nil || info.CloudBase.Env != "env-1" || info.CloudBase.Domain != "xxx.xx" {
		t.Fatalf("cloud_base = %+v", info.CloudBase)
	}

	// query_type 0 is the documented default, so it must be omitted.
	if body := string(fake.lastQueryURLLinkBody); body != `{"url_link":"`+urlLink+`"}` {
		t.Fatalf("body = %s", body)
	}
	if fake.lastQueryURLLinkQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastQueryURLLinkQuery.Get("access_token"))
	}
}

func TestQueryURLLinkQuota(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:    `{"access_token":"at","expires_in":7200}`,
		queryURLLink: `{"errcode":0,"errmsg":"ok","quota_info":{"remain_visit_quota":1000000}}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.QueryURLLink(context.Background(), QueryURLLinkParams{QueryType: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.QuotaInfo == nil || result.URLLinkInfo != nil {
		t.Fatalf("quota_info = %+v, url_link_info = %+v", result.QuotaInfo, result.URLLinkInfo)
	}
	if result.QuotaInfo.RemainVisitQuota != 1000000 {
		t.Fatalf("remain visit quota = %d", result.QuotaInfo.RemainVisitQuota)
	}

	// A quota lookup carries no link at all.
	if body := string(fake.lastQueryURLLinkBody); body != `{"query_type":1}` {
		t.Fatalf("body = %s", body)
	}
}

func TestQueryURLLinkErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"not found", `{"errcode":85403,"errmsg":"not found"}`, 85403},
		{"bad args", `{"errcode":40097,"errmsg":"invalid args"}`, 40097},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:    `{"access_token":"at","expires_in":7200}`,
				queryURLLink: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.QueryURLLink(context.Background(), QueryURLLinkParams{
				URLLink: "https://wxaurl.cn/abc",
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := QueryURLLinkError(result).Error()
			if desc, ok := QueryURLLinkErrCodeMap[tt.code]; ok {
				if !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			} else if !strings.Contains(got, QueryURLLinkDefaultErrMsg) {
				t.Fatalf("err = %q, want the default message", got)
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "not found") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestQueryURLLinkRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params QueryURLLinkParams
	}{
		{"missing url link", QueryURLLinkParams{}},
		{"bad query type", QueryURLLinkParams{URLLink: "https://wxaurl.cn/abc", QueryType: 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.QueryURLLink(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestQueryScheme(t *testing.T) {
	fake := &fakeWechat{
		tokenBody: `{"access_token":"at","expires_in":7200}`,
		queryScheme: `{"errcode":0,"errmsg":"ok","scheme_info":{"appid":"wx123",` +
			`"path":"pages/index/index","query":"a=hello","create_time":611928113,` +
			`"expire_time":0,"env_version":"release"}}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	scheme := "weixin://dl/business/?t=XTSkBZlzqmn&cq=a%3Dhello"
	result, err := app.QueryScheme(context.Background(), QuerySchemeParams{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.SchemeInfo == nil || result.QuotaInfo != nil {
		t.Fatalf("scheme_info = %+v, quota_info = %+v", result.SchemeInfo, result.QuotaInfo)
	}

	info := result.SchemeInfo
	if info.AppID != "wx123" || info.Path != "pages/index/index" || info.Query != "a=hello" {
		t.Fatalf("scheme info = %+v", info)
	}
	if info.EnvVersion != "release" || info.CreateTime != 611928113 || info.ExpireTime != 0 {
		t.Fatalf("scheme info = %+v", info)
	}

	// query_type 0 is the documented default, so it must be omitted while the
	// scheme travels verbatim, percent escapes and all.
	if body := string(fake.lastQuerySchemeBody); body != `{"scheme":"`+scheme+`"}` {
		t.Fatalf("body = %s", body)
	}
	if fake.lastQuerySchemeQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastQuerySchemeQuery.Get("access_token"))
	}
}

func TestQuerySchemeQuota(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:   `{"access_token":"at","expires_in":7200}`,
		queryScheme: `{"errcode":0,"errmsg":"ok","quota_info":{"remain_visit_quota":1000000}}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.QueryScheme(context.Background(), QuerySchemeParams{QueryType: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.QuotaInfo == nil || result.SchemeInfo != nil {
		t.Fatalf("quota_info = %+v, scheme_info = %+v", result.QuotaInfo, result.SchemeInfo)
	}
	if result.QuotaInfo.RemainVisitQuota != 1000000 {
		t.Fatalf("remain visit quota = %d", result.QuotaInfo.RemainVisitQuota)
	}

	// A quota lookup carries no scheme at all.
	if body := string(fake.lastQuerySchemeBody); body != `{"query_type":1}` {
		t.Fatalf("body = %s", body)
	}
}

func TestQuerySchemeErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"not found", `{"errcode":85403,"errmsg":"not found"}`, 85403},
		{"plain scheme disabled", `{"errcode":85405,"errmsg":"appid or path not support plain scheme"}`, 85405},
		{"bad query", `{"errcode":40212,"errmsg":"invalid query"}`, 40212},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:   `{"access_token":"at","expires_in":7200}`,
				queryScheme: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.QueryScheme(context.Background(), QuerySchemeParams{Scheme: "weixin://dl/business/?t=abc"})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := QuerySchemeError(result).Error()
			if desc, ok := QuerySchemeErrCodeMap[tt.code]; ok {
				if !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			} else if !strings.Contains(got, QuerySchemeDefaultErrMsg) {
				t.Fatalf("err = %q, want the default message", got)
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "not found") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestQuerySchemeRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params QuerySchemeParams
	}{
		{"missing scheme", QuerySchemeParams{}},
		{"bad query type", QuerySchemeParams{Scheme: "weixin://dl/business/?t=abc", QueryType: 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.QueryScheme(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestCreateQRCode(t *testing.T) {
	image := []byte{0xff, 0xd8, 0xff, 0xe0}

	fake := &fakeWechat{
		tokenBody:               `{"access_token":"at","expires_in":7200}`,
		createQRCode:            image,
		createQRCodeContentType: "image/jpeg",
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.CreateQRCode(context.Background(), CreateQRCodeParams{Path: "pages/index/index"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if len(result.Image) != len(image) {
		t.Fatalf("image = %v", result.Image)
	}
	if result.ContentType != "image/jpeg" {
		t.Fatalf("content type = %q", result.ContentType)
	}

	// Only the path was set, so width must be omitted and WeChat's default of
	// 430 px applies.
	if body := string(fake.lastCreateBody); body != `{"path":"pages/index/index"}` {
		t.Fatalf("body = %s", body)
	}
	if fake.lastCreateQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastCreateQuery.Get("access_token"))
	}
}

func TestCreateQRCodeRequestBody(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:               `{"access_token":"at","expires_in":7200}`,
		createQRCode:            []byte{0xff, 0xd8, 0xff, 0xe0},
		createQRCodeContentType: "image/jpeg",
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	_, err := app.CreateQRCode(context.Background(), CreateQRCodeParams{
		Path:  "pages/index/index?a=1",
		Width: 1280,
	})
	if err != nil {
		t.Fatal(err)
	}

	body := string(fake.lastCreateBody)
	for _, want := range []string{
		`"path":"pages/index/index?a=1"`,
		`"width":1280`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %s, want %s", body, want)
		}
	}

	// The endpoint takes no styling options, so none of them may leak into the
	// body.
	for _, unwanted := range []string{"auto_color", "line_color", "is_hyaline", "env_version"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("body = %s, want no %s", body, unwanted)
		}
	}
}

func TestCreateQRCodeErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"invalid page", `{"errcode":41030,"errmsg":"invalid page rid: 62c52bd4"}`, 41030},
		{"quota exhausted", `{"errcode":45029,"errmsg":"total count of qrcode reached limit"}`, 45029},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:               `{"access_token":"at","expires_in":7200}`,
				createQRCode:            []byte(tt.body),
				createQRCodeContentType: "application/json",
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.CreateQRCode(context.Background(), CreateQRCodeParams{Path: "pages/index/index"})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() || len(result.Image) != 0 {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := CreateQRCodeError(result).Error()
			if desc, ok := CreateQRCodeErrCodeMap[tt.code]; ok {
				if !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			} else if !strings.Contains(got, CreateQRCodeDefaultErrMsg) {
				t.Fatalf("err = %q, want the default message", got)
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "rid: 62c52bd4") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestCreateQRCodeRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params CreateQRCodeParams
	}{
		{"missing path", CreateQRCodeParams{}},
		// 128 bytes is the documented limit of this endpoint, unlike the
		// 1024-byte limit of /wxa/getwxacode.
		{"path too long", CreateQRCodeParams{Path: strings.Repeat("a", 129)}},
		{"width too small", CreateQRCodeParams{Path: "pages/index/index", Width: 200}},
		{"width too large", CreateQRCodeParams{Path: "pages/index/index", Width: 1281}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.CreateQRCode(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestGetUnlimitedQRCode(t *testing.T) {
	image := []byte{0x89, 0x50, 0x4e, 0x47}

	fake := &fakeWechat{
		tokenBody:              `{"access_token":"at","expires_in":7200}`,
		getUnlimitedQRCode:     image,
		getUnlimitedQRCodeType: "image/jpeg",
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GetUnlimitedQRCode(context.Background(), GetUnlimitedQRCodeParams{Scene: "a=1"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if len(result.Image) != len(image) {
		t.Fatalf("image = %v", result.Image)
	}
	if result.ContentType != "image/jpeg" {
		t.Fatalf("content type = %q", result.ContentType)
	}

	// Only scene was set, so page and check_path must stay off the body and
	// WeChat's own defaults (home page, check_path true) apply.
	if body := string(fake.lastUnlimitedBody); body != `{"scene":"a=1"}` {
		t.Fatalf("body = %s", body)
	}
	if fake.lastUnlimitedQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastUnlimitedQuery.Get("access_token"))
	}
}

func TestGetUnlimitedQRCodeRequestBody(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:              `{"access_token":"at","expires_in":7200}`,
		getUnlimitedQRCode:     []byte{0x89, 0x50, 0x4e, 0x47},
		getUnlimitedQRCodeType: "image/png",
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	checkPath := false
	_, err := app.GetUnlimitedQRCode(context.Background(), GetUnlimitedQRCodeParams{
		Scene:      "a=1",
		Page:       "pages/index/index",
		CheckPath:  &checkPath,
		EnvVersion: "release",
		Width:      280,
		LineColor:  &RGBColor{R: 1, G: 2, B: 3},
		IsHyaline:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	body := string(fake.lastUnlimitedBody)
	for _, want := range []string{
		`"scene":"a=1"`,
		`"page":"pages/index/index"`,
		`"check_path":false`,
		`"env_version":"release"`,
		`"width":280`,
		`"line_color":{"r":1,"g":2,"b":3}`,
		`"is_hyaline":true`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %s, want %s", body, want)
		}
	}
}

func TestGetUnlimitedQRCodeErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"invalid scene", `{"errcode":40129,"errmsg":"invalid scene rid: 62c52bd4"}`, 40129},
		{"invalid page", `{"errcode":41030,"errmsg":"invalid page rid: 62c52bd4"}`, 41030},
		{"reserved field", `{"errcode":85096,"errmsg":"not allow include scancode_time field"}`, 85096},
		{"bad scene length", `{"errcode":40169,"errmsg":"invalid length for scene"}`, 40169},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:              `{"access_token":"at","expires_in":7200}`,
				getUnlimitedQRCode:     []byte(tt.body),
				getUnlimitedQRCodeType: "application/json",
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.GetUnlimitedQRCode(context.Background(), GetUnlimitedQRCodeParams{Scene: "a=1"})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() || len(result.Image) != 0 {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := GetUnlimitedQRCodeError(result).Error()
			if desc, ok := GetUnlimitedQRCodeErrCodeMap[tt.code]; ok {
				if !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			} else if !strings.Contains(got, GetUnlimitedQRCodeDefaultErrMsg) {
				t.Fatalf("err = %q, want the default message", got)
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "rid: 62c52bd4") || strings.Contains(got, "invalid length for scene") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestGetUnlimitedQRCodeRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params GetUnlimitedQRCodeParams
	}{
		{"missing scene", GetUnlimitedQRCodeParams{}},
		{"scene too long", GetUnlimitedQRCodeParams{Scene: strings.Repeat("a", 33)}},
		{"scene with unsupported char", GetUnlimitedQRCodeParams{Scene: "中文"}},
		{"page with leading slash", GetUnlimitedQRCodeParams{Scene: "a=1", Page: "/pages/index/index"}},
		{"page with query", GetUnlimitedQRCodeParams{Scene: "a=1", Page: "pages/index/index?a=1"}},
		{"page with reserved field", GetUnlimitedQRCodeParams{Scene: "a=1", Page: "pages/index/index?scancode_time=1"}},
		{"width too small", GetUnlimitedQRCodeParams{Scene: "a=1", Width: 200}},
		{"width too large", GetUnlimitedQRCodeParams{Scene: "a=1", Width: 1281}},
		{"bad env version", GetUnlimitedQRCodeParams{Scene: "a=1", EnvVersion: "staging"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.GetUnlimitedQRCode(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestGetQRCode(t *testing.T) {
	image := []byte{0x89, 0x50, 0x4e, 0x47}

	fake := &fakeWechat{
		tokenBody:            `{"access_token":"at","expires_in":7200}`,
		getQRCode:            image,
		getQRCodeContentType: "image/jpeg",
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GetQRCode(context.Background(), GetQRCodeParams{Path: "pages/index/index"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if len(result.Image) != len(image) {
		t.Fatalf("image = %v", result.Image)
	}
	if result.ContentType != "image/jpeg" {
		t.Fatalf("content type = %q", result.ContentType)
	}

	// Only the path was set, so every optional field must be omitted and
	// WeChat's own defaults apply.
	if body := string(fake.lastQRCodeBody); body != `{"path":"pages/index/index"}` {
		t.Fatalf("body = %s", body)
	}
	if fake.lastQRCodeQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastQRCodeQuery.Get("access_token"))
	}
}

func TestGetQRCodeRequestBody(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:            `{"access_token":"at","expires_in":7200}`,
		getQRCode:            []byte{0x89, 0x50, 0x4e, 0x47},
		getQRCodeContentType: "image/png",
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	_, err := app.GetQRCode(context.Background(), GetQRCodeParams{
		Path:       "pages/index/index?question_id=22579",
		Width:      280,
		LineColor:  &RGBColor{R: 1, G: 2, B: 3},
		IsHyaline:  true,
		EnvVersion: "release",
	})
	if err != nil {
		t.Fatal(err)
	}

	body := string(fake.lastQRCodeBody)
	for _, want := range []string{
		`"width":280`,
		`"is_hyaline":true`,
		`"env_version":"release"`,
		`"line_color":{"r":1,"g":2,"b":3}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %s, want %s", body, want)
		}
	}
}

func TestGetQRCodeErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"invalid page", `{"errcode":41030,"errmsg":"invalid page rid: 62c52bd4"}`, 41030},
		{"empty path", `{"errcode":40159,"errmsg":"invalid length for path"}`, 40159},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:            `{"access_token":"at","expires_in":7200}`,
				getQRCode:            []byte(tt.body),
				getQRCodeContentType: "application/json",
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.GetQRCode(context.Background(), GetQRCodeParams{Path: "pages/index/index"})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() || len(result.Image) != 0 {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := GetQRCodeError(result).Error()
			switch tt.code {
			case 41030:
				if desc := GetQRCodeErrCodeMap[41030]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 40159:
				if desc := GetQRCodeErrCodeMap[40159]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 99999:
				if !strings.Contains(got, GetQRCodeDefaultErrMsg) {
					t.Fatalf("err = %q, want the default message", got)
				}
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "rid: 62c52bd4") || strings.Contains(got, "invalid length for path") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestGetQRCodeRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params GetQRCodeParams
	}{
		{"missing path", GetQRCodeParams{}},
		{"path too long", GetQRCodeParams{Path: strings.Repeat("a", 1025)}},
		{"width too small", GetQRCodeParams{Path: "pages/index/index", Width: 200}},
		{"width too large", GetQRCodeParams{Path: "pages/index/index", Width: 1281}},
		{"bad env version", GetQRCodeParams{Path: "pages/index/index", EnvVersion: "staging"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.GetQRCode(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestEncryptedMsgHash(t *testing.T) {
	// sha256("hello") is a well-known test vector, and the digest must be hex.
	if got := EncryptedMsgHash([]byte("hello")); got != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("hash = %q", got)
	}
}

func TestCheckEncryptedData(t *testing.T) {
	const hash = "657edd868c9715a9bebe42b833269a557a48498785397a796f1568c29a200b2c"

	fake := &fakeWechat{
		tokenBody:          `{"access_token":"at","expires_in":7200}`,
		checkEncryptedData: `{"errcode":0,"errmsg":"ok","vaild":true,"create_time":1629121902}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.CheckEncryptedData(context.Background(), hash)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.CreateTime != 1629121902 {
		t.Fatalf("create_time = %d", result.CreateTime)
	}

	// WeChat ships the field as "vaild", so the reply tag must match it.
	if body := string(fake.lastEncryptedBody); body != `{"encrypted_msg_hash":"`+hash+`"}` {
		t.Fatalf("body = %s", body)
	}
	// The token middleware must have attached the cached token.
	if fake.lastEncryptedQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastEncryptedQuery.Get("access_token"))
	}
}

func TestCheckEncryptedDataRejectsUnknownData(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:          `{"access_token":"at","expires_in":7200}`,
		checkEncryptedData: `{"errcode":0,"errmsg":"ok","vaild":false,"create_time":0}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.CheckEncryptedData(context.Background(), "hash")
	if err != nil {
		t.Fatal(err)
	}
	if result.OK() {
		t.Fatalf("response = %+v, want the payload rejected", result)
	}
	// A rejected payload is a successful call, not an API error.
	if result.ErrCode != 0 {
		t.Fatalf("errcode = %d, want 0", result.ErrCode)
	}
}

func TestCheckEncryptedDataErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"data not found", `{"errcode":1,"errmsg":"data not exit"}`, 1},
		{"invalid args", `{"errcode":40097,"errmsg":"invalid args"}`, 40097},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:          `{"access_token":"at","expires_in":7200}`,
				checkEncryptedData: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.CheckEncryptedData(context.Background(), "hash")
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := CheckEncryptedDataError(result).Error()
			switch tt.code {
			case 1:
				if desc := CheckEncryptedDataErrCodeMap[1]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 40097:
				if desc := CheckEncryptedDataErrCodeMap[40097]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 99999:
				if !strings.Contains(got, CheckEncryptedDataDefaultErrMsg) {
					t.Fatalf("err = %q, want the default message", got)
				}
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "data not exit") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestCheckEncryptedDataRequiresHash(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.CheckEncryptedData(context.Background(), ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetPluginOpenPID(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:        `{"access_token":"at","expires_in":7200}`,
		getPluginOpenPID: `{"errcode":0,"errmsg":"ok","openpid":"GACo74wkDIkDzEhkwRwgjGt1pqlk"}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GetPluginOpenPID(context.Background(), "plugin-code")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.OpenPID != "GACo74wkDIkDzEhkwRwgjGt1pqlk" {
		t.Fatalf("openpid = %q", result.OpenPID)
	}

	// The plugin login code must travel in the JSON body, not in the query.
	if body := string(fake.lastPluginOpenPIDBody); body != `{"code":"plugin-code"}` {
		t.Fatalf("body = %s", body)
	}
	// The token middleware must have attached the cached token.
	if fake.lastPluginOpenPIDQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastPluginOpenPIDQuery.Get("access_token"))
	}
}

func TestGetPluginOpenPIDErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"missing openpid", `{"errcode":1007,"errmsg":"rid: 123"}`, 1007},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:        `{"access_token":"at","expires_in":7200}`,
				getPluginOpenPID: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.GetPluginOpenPID(context.Background(), "plugin-code")
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := GetPluginOpenPIDError(result).Error()
			switch tt.code {
			case 1007:
				if desc := GetPluginOpenPIDErrCodeMap[1007]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 99999:
				if !strings.Contains(got, GetPluginOpenPIDDefaultErrMsg) {
					t.Fatalf("err = %q, want the default message", got)
				}
			}
			if strings.Contains(got, "raw text") || strings.Contains(got, "rid: 123") {
				t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
			}
		})
	}
}

func TestGetPluginOpenPIDRequiresCode(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.GetPluginOpenPID(context.Background(), ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetPhoneNumber(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:      `{"access_token":"at","expires_in":7200}`,
		getPhoneNumber: `{"errcode":0,"errmsg":"ok","phone_info":{"phoneNumber":"+8613800000000","purePhoneNumber":"13800000000","countryCode":"86","watermark":{"timestamp":1637744274,"appid":"wx1"}}}`,
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GetPhoneNumber(context.Background(), "phone-code")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.PurePhoneNumber != "13800000000" || result.CountryCode != "86" {
		t.Fatalf("response = %+v", result)
	}
	if result.Watermark == nil || result.Watermark.AppID != "wx1" {
		t.Fatalf("watermark = %+v", result.Watermark)
	}

	// The phone number code must travel in the JSON body, not in the query.
	if body := string(fake.lastGetPhoneBody); body != `{"code":"phone-code"}` {
		t.Fatalf("body = %s", body)
	}
	// The token middleware must have attached the cached token.
	if fake.lastGetPhoneQuery.Get("access_token") != "at" {
		t.Fatalf("access_token = %q", fake.lastGetPhoneQuery.Get("access_token"))
	}
}

func TestGetPhoneNumberErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"invalid code", `{"errcode":40029,"errmsg":"invalid code"}`, 40029},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeWechat{
				tokenBody:      `{"access_token":"at","expires_in":7200}`,
				getPhoneNumber: tt.body,
			}
			srv := newFakeServer(fake)
			defer srv.Close()

			app := newApp(t, srv.URL, nil)

			result, err := app.GetPhoneNumber(context.Background(), "phone-code")
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			got := GetPhoneNumberError(result).Error()
			switch tt.code {
			case 40029:
				if desc := GetPhoneNumberErrCodeMap[40029]; !strings.Contains(got, desc) {
					t.Fatalf("err = %q, want the documented description %q", got, desc)
				}
			case 99999:
				if !strings.Contains(got, GetPhoneNumberDefaultErrMsg) {
					t.Fatalf("err = %q, want the default message", got)
				}
				if strings.Contains(got, "raw text") {
					t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
				}
			}
		})
	}
}

func TestGetPhoneNumberRequiresCode(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.GetPhoneNumber(context.Background(), ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
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
