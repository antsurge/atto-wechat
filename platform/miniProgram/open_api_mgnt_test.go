package miniProgram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// apiCall is what the fake server saw for the endpoint under test.
type apiCall struct {
	method string
	path   string
	query  url.Values
	body   []byte
}

// newAPIServer serves reply for every endpoint except the token endpoint, which
// answers with a valid token so the token middleware can do its job.
//
// The token call is deliberately not captured: these tests assert on the endpoint
// they call, and letting the middleware overwrite the record would make every
// request look like a token fetch.
func newAPIServer(t *testing.T, reply string) (*httptest.Server, *apiCall) {
	t.Helper()

	call := &apiCall{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == tokenPath {
			_, _ = io.WriteString(w, `{"access_token":"TOKEN","expires_in":7200}`)

			return
		}

		call.method = r.Method
		call.path = r.URL.Path
		call.query = r.URL.Query()
		call.body, _ = io.ReadAll(r.Body)

		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)

	return srv, call
}

// decodeBody decodes the captured request body.
func (c *apiCall) decodeBody(t *testing.T, dst any) {
	t.Helper()

	if err := json.Unmarshal(c.body, dst); err != nil {
		t.Fatalf("decode request body %q: %v", c.body, err)
	}
}

func TestGetStableAccessToken(t *testing.T) {
	srv, call := newAPIServer(t, `{"access_token":"STABLE","expires_in":7200}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetStableAccessToken(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if resp.AccessToken != "STABLE" || resp.ExpiresIn != 7200 {
		t.Fatalf("resp = %+v", resp)
	}

	if call.method != http.MethodPost || call.path != stableTokenPath {
		t.Fatalf("%s %s", call.method, call.path)
	}

	// The endpoint authenticates with appid/secret in the body, so no token may be
	// attached: a token here would be the one this endpoint is supposed to issue.
	if got := call.query.Get("access_token"); got != "" {
		t.Fatalf("access_token = %q, want none", got)
	}

	var body struct {
		GrantType    string `json:"grant_type"`
		AppID        string `json:"appid"`
		Secret       string `json:"secret"`
		ForceRefresh bool   `json:"force_refresh"`
	}
	call.decodeBody(t, &body)

	if body.GrantType != "client_credential" || body.AppID != "wx1" || body.Secret != "secret" {
		t.Fatalf("body = %+v", body)
	}
	if body.ForceRefresh {
		t.Fatal("force_refresh sent for a normal call")
	}
}

func TestGetStableAccessTokenForceRefresh(t *testing.T) {
	srv, call := newAPIServer(t, `{"access_token":"STABLE","expires_in":7200}`)
	app := newApp(t, srv.URL, nil)

	if _, err := app.GetStableAccessToken(context.Background(), true); err != nil {
		t.Fatal(err)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	if body["force_refresh"] != true {
		t.Fatalf("force_refresh = %v, want true", body["force_refresh"])
	}
}

func TestGetStableAccessTokenError(t *testing.T) {
	srv, _ := newAPIServer(t, `{"errcode":40125,"errmsg":"invalid appsecret"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetStableAccessToken(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 40125")
	}
	if err := StableAccessTokenError(resp); err == nil ||
		err.Error() != "appsecret 无效" {
		t.Fatalf("err = %v", err)
	}

	// An undocumented errcode must not leak WeChat's raw errmsg.
	if got := StableAccessTokenErrMsg(43210); got != StableAccessTokenDefaultErrMsg {
		t.Fatalf("errmsg = %q", got)
	}
}

func TestGetAPIQuota(t *testing.T) {
	reply := `{"errcode":0,"errmsg":"ok",` +
		`"quota":{"daily_limit":10000,"used":1000,"remain":9000},` +
		`"rate_limit":{"call_count":500,"refresh_second":60}}`

	srv, call := newAPIServer(t, reply)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetAPIQuota(context.Background(), GetAPIQuotaParams{
		CGIPath: "/cgi-bin/message/custom/send",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if resp.Quota.DailyLimit != 10000 || resp.Quota.Remain != 9000 {
		t.Fatalf("quota = %+v", resp.Quota)
	}
	if resp.RateLimit.CallCount != 500 || resp.RateLimit.RefreshSecond != 60 {
		t.Fatalf("rate limit = %+v", resp.RateLimit)
	}
	// Absent in the reply: the struct must stay nil rather than a zero value that
	// reads like "limit of 0 calls".
	if resp.ComponentRateLimit != nil {
		t.Fatalf("component rate limit = %+v", resp.ComponentRateLimit)
	}

	if call.method != http.MethodPost || call.path != getAPIQuotaPath {
		t.Fatalf("%s %s", call.method, call.path)
	}
	if got := call.query.Get("access_token"); got != "TOKEN" {
		t.Fatalf("access_token = %q", got)
	}

	var body struct {
		CGIPath string `json:"cgi_path"`
	}
	call.decodeBody(t, &body)

	if body.CGIPath != "/cgi-bin/message/custom/send" {
		t.Fatalf("cgi_path = %q", body.CGIPath)
	}
}

func TestGetAPIQuotaValidation(t *testing.T) {
	app := newApp(t, "http://127.0.0.1:1", nil)

	cases := []struct {
		name string
		path string
	}{
		{name: "empty", path: ""},
		{name: "host prefix", path: "https://api.weixin.qq.com/cgi-bin/token"},
		{name: "no leading slash", path: "cgi-bin/token"},
	}

	for _, tc := range cases {
		if _, err := app.GetAPIQuota(
			context.Background(), GetAPIQuotaParams{CGIPath: tc.path},
		); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestGetAPIQuotaError(t *testing.T) {
	srv, _ := newAPIServer(t, `{"errcode":76022,"errmsg":"could not use this cgi_path"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetAPIQuota(context.Background(), GetAPIQuotaParams{
		CGIPath: "/channels/ec/basics/info/get",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 76022")
	}
	if msg := GetAPIQuotaErrMsg(resp.ErrCode); msg != OpenAPIErrCodeMap[76022] {
		t.Fatalf("errmsg = %q", msg)
	}
}

func TestClearQuota(t *testing.T) {
	srv, call := newAPIServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.ClearQuota(context.Background(), ClearQuotaParams{AppID: "wx1"})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if call.method != http.MethodPost || call.path != clearQuotaPath {
		t.Fatalf("%s %s", call.method, call.path)
	}

	var body struct {
		AppID string `json:"appid"`
	}
	call.decodeBody(t, &body)

	if body.AppID != "wx1" {
		t.Fatalf("appid = %q", body.AppID)
	}
}

func TestClearQuotaValidation(t *testing.T) {
	app := newApp(t, "http://127.0.0.1:1", nil)

	if _, err := app.ClearQuota(context.Background(), ClearQuotaParams{}); err == nil {
		t.Fatal("empty appid accepted")
	}
}

func TestClearAPIQuota(t *testing.T) {
	srv, call := newAPIServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.ClearAPIQuota(context.Background(), ClearAPIQuotaParams{
		CGIPath: "/channels/ec/basics/info/get",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if call.path != clearAPIQuotaPath {
		t.Fatalf("path = %q", call.path)
	}

	var body struct {
		CGIPath string `json:"cgi_path"`
	}
	call.decodeBody(t, &body)

	if body.CGIPath != "/channels/ec/basics/info/get" {
		t.Fatalf("cgi_path = %q", body.CGIPath)
	}
}

func TestClearQuotaByAppSecret(t *testing.T) {
	srv, call := newAPIServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.ClearQuotaByAppSecret(context.Background(),
		ClearQuotaByAppSecretParams{AppID: "wx1"})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if call.path != clearQuotaByAppSecretPath {
		t.Fatalf("path = %q", call.path)
	}

	// This endpoint must work when no token can be obtained, so it may not carry
	// one.
	if got := call.query.Get("access_token"); got != "" {
		t.Fatalf("access_token = %q, want none", got)
	}

	var body struct {
		AppID     string `json:"appid"`
		AppSecret string `json:"appsecret"`
	}
	call.decodeBody(t, &body)

	if body.AppID != "wx1" {
		t.Fatalf("appid = %q", body.AppID)
	}
	// The configured secret is the default.
	if body.AppSecret != "secret" {
		t.Fatalf("appsecret = %q", body.AppSecret)
	}
}

func TestClearQuotaByAppSecretOverride(t *testing.T) {
	srv, call := newAPIServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	_, err := app.ClearQuotaByAppSecret(context.Background(),
		ClearQuotaByAppSecretParams{AppID: "wx2", AppSecret: "other"})
	if err != nil {
		t.Fatal(err)
	}

	var body struct {
		AppID     string `json:"appid"`
		AppSecret string `json:"appsecret"`
	}
	call.decodeBody(t, &body)

	if body.AppID != "wx2" || body.AppSecret != "other" {
		t.Fatalf("body = %+v", body)
	}
}

func TestClearQuotaByAppSecretError(t *testing.T) {
	srv, _ := newAPIServer(t, `{"errcode":48006,"errmsg":"forbid to clear quota"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.ClearQuotaByAppSecret(context.Background(),
		ClearQuotaByAppSecretParams{AppID: "wx1"})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 48006")
	}
	if err := ClearQuotaByAppSecretError(resp); err == nil ||
		err.Error() != "重置次数已达上限，禁止重置" {
		t.Fatalf("err = %v", err)
	}
}

func TestGetRIDInfo(t *testing.T) {
	reply := `{"errcode":0,"errmsg":"ok","request":{` +
		`"invoke_time":1700000000,"cost_in_ms":42,` +
		`"request_url":"https://api.weixin.qq.com/wxa/getwxacode?access_token=X",` +
		`"request_body":"{\"path\":\"pages/index\"}",` +
		`"response_body":"{\"errcode\":40013}",` +
		`"client_ip":"1.2.3.4"}}`

	srv, call := newAPIServer(t, reply)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetRIDInfo(context.Background(), GetRIDInfoParams{
		RID: "61725984-6126f6f9-040f19c4",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if resp.Request.CostInMS != 42 || resp.Request.ClientIP != "1.2.3.4" {
		t.Fatalf("request = %+v", resp.Request)
	}
	if resp.Request.InvokeTime != 1700000000 {
		t.Fatalf("invoke time = %d", resp.Request.InvokeTime)
	}

	if call.path != getRIDInfoPath {
		t.Fatalf("path = %q", call.path)
	}

	var body struct {
		RID string `json:"rid"`
	}
	call.decodeBody(t, &body)

	if body.RID != "61725984-6126f6f9-040f19c4" {
		t.Fatalf("rid = %q", body.RID)
	}
}

func TestGetRIDInfoValidation(t *testing.T) {
	app := newApp(t, "http://127.0.0.1:1", nil)

	if _, err := app.GetRIDInfo(context.Background(), GetRIDInfoParams{}); err == nil {
		t.Fatal("empty rid accepted")
	}
}

func TestGetRIDInfoError(t *testing.T) {
	srv, _ := newAPIServer(t, `{"errcode":76001,"errmsg":"rid not found"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetRIDInfo(context.Background(), GetRIDInfoParams{RID: "gone"})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 76001")
	}
	if err := GetRIDInfoError(resp); err == nil ||
		err.Error() != "rid 不存在（rid 仅保留 7 天）" {
		t.Fatalf("err = %v", err)
	}
}

func TestCallbackCheck(t *testing.T) {
	reply := `{"errcode":0,"errmsg":"ok",` +
		`"dns":[{"ip":"1.2.3.4","real_operator":"CHINANET"}],` +
		`"ping":[{"ip":"1.2.3.4","from_operator":"DEFAULT","package_loss":"100%"}]}`

	srv, call := newAPIServer(t, reply)
	app := newApp(t, srv.URL, nil)

	resp, err := app.CallbackCheck(context.Background(), CallbackCheckParams{
		Action:        CallbackCheckActionAll,
		CheckOperator: CallbackCheckOperatorDefault,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if len(resp.DNS) != 1 || resp.DNS[0].IP != "1.2.3.4" {
		t.Fatalf("dns = %+v", resp.DNS)
	}
	if len(resp.Ping) != 1 || !resp.Ping[0].Lost() {
		t.Fatalf("ping = %+v", resp.Ping)
	}
	if CallbackCheckOperatorMsg(resp.Ping[0].FromOperator) != "自动选择" {
		t.Fatalf("operator msg failed")
	}

	if call.path != callbackCheckPath {
		t.Fatalf("path = %q", call.path)
	}

	var body struct {
		Action        string `json:"action"`
		CheckOperator string `json:"check_operator"`
	}
	call.decodeBody(t, &body)

	if body.Action != "all" || body.CheckOperator != "DEFAULT" {
		t.Fatalf("body = %+v", body)
	}
}

func TestCallbackCheckValidation(t *testing.T) {
	app := newApp(t, "http://127.0.0.1:1", nil)

	// Both parameters are enumerations; an unknown value is rejected locally
	// instead of coming back as 40202 / 40203.
	if _, err := app.CallbackCheck(context.Background(), CallbackCheckParams{
		Action:        "traceroute",
		CheckOperator: CallbackCheckOperatorDefault,
	}); err == nil {
		t.Fatal("unknown action accepted")
	}

	if _, err := app.CallbackCheck(context.Background(), CallbackCheckParams{
		Action:        CallbackCheckActionDNS,
		CheckOperator: "MOBILE",
	}); err == nil {
		t.Fatal("unknown check_operator accepted")
	}

	if CallbackCheckActionMsg("traceroute") != "" {
		t.Fatal("undocumented action must have no description")
	}
}

func TestGetAPIDomainIP(t *testing.T) {
	srv, call := newAPIServer(t, `{"errcode":0,"errmsg":"ok",`+
		`"ip_list":["1.2.3.4","5.6.7.0/24"]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetAPIDomainIP(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if len(resp.IPList) != 2 || resp.IPList[1] != "5.6.7.0/24" {
		t.Fatalf("ip list = %v", resp.IPList)
	}

	// The token middleware must be in play: unlike the stable token endpoint this
	// one is authenticated by access_token.
	if call.method != http.MethodGet || call.path != getAPIDomainIPPath {
		t.Fatalf("%s %s", call.method, call.path)
	}
	if got := call.query.Get("access_token"); got != "TOKEN" {
		t.Fatalf("access_token = %q", got)
	}
}

func TestGetCallbackIP(t *testing.T) {
	srv, call := newAPIServer(t, `{"errcode":0,"errmsg":"ok","ip_list":["1.2.3.4"]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetCallbackIP(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if len(resp.IPList) != 1 || resp.IPList[0] != "1.2.3.4" {
		t.Fatalf("ip list = %v", resp.IPList)
	}

	if call.method != http.MethodGet || call.path != getCallbackIPPath {
		t.Fatalf("%s %s", call.method, call.path)
	}
}

func TestGetCallbackIPError(t *testing.T) {
	srv, _ := newAPIServer(t, `{"errcode":40013,"errmsg":"invalid appid"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetCallbackIP(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 40013")
	}
	if err := GetCallbackIPError(resp); err == nil || err.Error() != "appid 无效" {
		t.Fatalf("err = %v", err)
	}
}
