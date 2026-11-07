package oauth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/antsurge/atto-wechat/core/client"
	"github.com/antsurge/atto-wechat/core/errors"
)

// oauthServer fakes the four OAuth endpoints.
type oauthServer struct {
	accessTokenBody  string
	refreshTokenBody string
	userInfoBody     string
	checkBody        string

	lastQuery url.Values
}

func newOAuthServer() (*httptest.Server, *oauthServer) {
	o := &oauthServer{
		accessTokenBody:  `{"access_token":"at","expires_in":7200,"refresh_token":"rt","openid":"o","scope":"snsapi_base"}`,
		refreshTokenBody: `{"access_token":"at2","expires_in":7200,"refresh_token":"rt2","openid":"o","scope":"snsapi_base"}`,
		userInfoBody:     `{"openid":"o","nickname":"nick","sex":1,"province":"p","city":"c","country":"cn","headimgurl":"http://img","privilege":["a"]}`,
		checkBody:        `{"errcode":0,"errmsg":"ok"}`,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o.lastQuery = r.URL.Query()

		switch r.URL.Path {
		case PathAccessToken:
			_, _ = io.WriteString(w, o.accessTokenBody)
		case PathRefreshToken:
			_, _ = io.WriteString(w, o.refreshTokenBody)
		case PathUserInfo:
			_, _ = io.WriteString(w, o.userInfoBody)
		case PathCheckToken:
			_, _ = io.WriteString(w, o.checkBody)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	return srv, o
}

func newClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	c, err := client.NewClient(baseURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	oc, err := NewClient(c, "wx1", "secret1")
	if err != nil {
		t.Fatal(err)
	}

	return oc
}

func TestAuthorizationURL(t *testing.T) {
	oc := newClient(t, "http://example.invalid")

	got := oc.AuthorizationURL("https://app.example/cb?a=1", "", "state-1")

	if !strings.HasPrefix(got, AuthorizationURL+"?") {
		t.Fatalf("url = %s", got)
	}
	if !strings.HasSuffix(got, "#wechat_redirect") {
		t.Fatalf("url = %s", got)
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}

	q := u.Query()

	if q.Get("appid") != "wx1" {
		t.Errorf("appid = %q", q.Get("appid"))
	}
	if q.Get("redirect_uri") != "https://app.example/cb?a=1" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q", q.Get("response_type"))
	}
	if q.Get("scope") != ScopeBase {
		t.Errorf("scope = %q, want the default scope", q.Get("scope"))
	}
	if q.Get("state") != "state-1" {
		t.Errorf("state = %q", q.Get("state"))
	}
}

func TestAuthorizationURLScopeUserInfo(t *testing.T) {
	oc := newClient(t, "http://example.invalid")

	u, err := url.Parse(oc.AuthorizationURL("https://app.example/cb", ScopeUserInfo, "s"))
	if err != nil {
		t.Fatal(err)
	}

	if u.Query().Get("scope") != ScopeUserInfo {
		t.Fatalf("scope = %q", u.Query().Get("scope"))
	}
}

func TestAccessToken(t *testing.T) {
	srv, fake := newOAuthServer()
	defer srv.Close()

	oc := newClient(t, srv.URL)

	token, err := oc.AccessToken(context.Background(), "code-1")
	if err != nil {
		t.Fatal(err)
	}

	if token.AccessToken != "at" || token.RefreshToken != "rt" || token.OpenID != "o" {
		t.Fatalf("token = %+v", token)
	}
	if token.ExpiresAt <= time.Now().Unix() {
		t.Fatalf("ExpiresAt = %d, want the future", token.ExpiresAt)
	}

	if fake.lastQuery.Get("code") != "code-1" {
		t.Fatalf("query = %v", fake.lastQuery)
	}
	if fake.lastQuery.Get("grant_type") != "authorization_code" {
		t.Fatalf("grant_type = %q", fake.lastQuery.Get("grant_type"))
	}
	if fake.lastQuery.Get("secret") != "secret1" {
		t.Fatalf("secret = %q", fake.lastQuery.Get("secret"))
	}
}

func TestAccessTokenRequiresCode(t *testing.T) {
	oc := newClient(t, "http://example.invalid")

	if _, err := oc.AccessToken(context.Background(), ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestAccessTokenErrorCodes(t *testing.T) {
	tests := []struct {
		name string
		body string
		code errors.Code
	}{
		{"invalid code", `{"errcode":40029,"errmsg":"invalid code"}`, errors.CodeInvalidArgument},
		{"code used", `{"errcode":40163,"errmsg":"code been used"}`, errors.CodeInvalidArgument},
		{"bad appid", `{"errcode":40013,"errmsg":"invalid appid"}`, errors.CodeUnauthorized},
		{"no token field", `{}`, errors.CodeInvalidResponse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, fake := newOAuthServer()
			defer srv.Close()

			fake.accessTokenBody = tt.body
			oc := newClient(t, srv.URL)

			if _, err := oc.AccessToken(context.Background(), "c"); !errors.Is(err, tt.code) {
				t.Fatalf("err = %v, want %s", err, tt.code)
			}
		})
	}
}

func TestRefreshToken(t *testing.T) {
	srv, _ := newOAuthServer()
	defer srv.Close()

	oc := newClient(t, srv.URL)

	token, err := oc.RefreshToken(context.Background(), "rt")
	if err != nil {
		t.Fatal(err)
	}
	if token.AccessToken != "at2" || token.RefreshToken != "rt2" {
		t.Fatalf("token = %+v", token)
	}

	if _, err := oc.RefreshToken(context.Background(), ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestUserInfo(t *testing.T) {
	srv, _ := newOAuthServer()
	defer srv.Close()

	oc := newClient(t, srv.URL)

	info, err := oc.UserInfo(context.Background(), &Token{AccessToken: "at", OpenID: "o"})
	if err != nil {
		t.Fatal(err)
	}

	if info.Nickname != "nick" || info.OpenID != "o" || len(info.Privilege) != 1 {
		t.Fatalf("info = %+v", info)
	}
}

func TestUserInfoErrors(t *testing.T) {
	srv, fake := newOAuthServer()
	defer srv.Close()

	oc := newClient(t, srv.URL)

	if _, err := oc.UserInfo(context.Background(), nil); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil token err = %v", err)
	}

	fake.userInfoBody = `{"errcode":40003,"errmsg":"invalid openid"}`

	if _, err := oc.UserInfo(context.Background(), &Token{AccessToken: "at", OpenID: "o"}); !errors.Is(err, errors.CodeInvalidResponse) {
		t.Fatalf("err = %v", err)
	}

	fake.userInfoBody = `{}`

	if _, err := oc.UserInfo(context.Background(), &Token{AccessToken: "at", OpenID: "o"}); !errors.Is(err, errors.CodeInvalidResponse) {
		t.Fatalf("empty err = %v", err)
	}
}

func TestCheckToken(t *testing.T) {
	srv, fake := newOAuthServer()
	defer srv.Close()

	oc := newClient(t, srv.URL)

	if err := oc.CheckToken(context.Background(), &Token{AccessToken: "at", OpenID: "o"}); err != nil {
		t.Fatal(err)
	}

	fake.checkBody = `{"errcode":40001,"errmsg":"invalid credential"}`

	if err := oc.CheckToken(context.Background(), &Token{AccessToken: "at", OpenID: "o"}); !errors.Is(err, errors.CodeTokenExpired) {
		t.Fatalf("err = %v", err)
	}

	if err := oc.CheckToken(context.Background(), nil); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil token err = %v", err)
	}
}

func TestNewClientValidation(t *testing.T) {
	c, err := client.NewClient("http://example.invalid", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := NewClient(nil, "wx", "s"); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("nil client err = %v", err)
	}
	if _, err := NewClient(c, "", "s"); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("empty appid err = %v", err)
	}
	if _, err := NewClient(c, "wx", ""); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("empty secret err = %v", err)
	}
}

func TestTokenExpired(t *testing.T) {
	tests := []struct {
		name  string
		token *Token
		want  bool
	}{
		{"nil", nil, true},
		{"no token", &Token{}, true},
		{"no deadline", &Token{AccessToken: "a"}, false},
		{"future", &Token{AccessToken: "a", ExpiresAt: time.Now().Add(time.Hour).Unix()}, false},
		{"past", &Token{AccessToken: "a", ExpiresAt: time.Now().Add(-time.Minute).Unix()}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.token.Expired() != tt.want {
				t.Errorf("Expired = %v, want %v", tt.token.Expired(), tt.want)
			}
		})
	}
}
