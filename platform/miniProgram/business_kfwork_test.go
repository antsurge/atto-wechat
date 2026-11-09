package miniProgram

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newKFServer serves reply for every endpoint but the token one, and records what
// the endpoint under test received.
func newKFServer(t *testing.T, reply string) (*httptest.Server, *apiCall) {
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

func TestRegisterBusiness(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","business_id":12}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.RegisterBusiness(context.Background(), RegisterBusinessParams{
		AccountName:      "apple_store",
		Nickname:         "苹果旗舰店",
		IconMediaID:      "media_1",
		TransferToCommKF: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if resp.BusinessID != 12 {
		t.Fatalf("business_id = %d", resp.BusinessID)
	}

	if call.method != http.MethodPost || call.path != registerBusinessPath {
		t.Fatalf("%s %s", call.method, call.path)
	}

	var body struct {
		AccountName      string `json:"account_name"`
		Nickname         string `json:"nickname"`
		IconMediaID      string `json:"icon_media_id"`
		TransferToCommKF bool   `json:"transfer_to_commkf"`
	}
	call.decodeBody(t, &body)

	if body.AccountName != "apple_store" || body.Nickname != "苹果旗舰店" ||
		body.IconMediaID != "media_1" {
		t.Fatalf("body = %+v", body)
	}
	if !body.TransferToCommKF {
		t.Fatal("transfer_to_commkf not sent")
	}
}

func TestRegisterBusinessValidation(t *testing.T) {
	valid := RegisterBusinessParams{
		AccountName: "apple_store",
		Nickname:    "苹果",
		IconMediaID: "media_1",
	}

	cases := []struct {
		name   string
		mutate func(*RegisterBusinessParams)
	}{
		{name: "empty account_name", mutate: func(p *RegisterBusinessParams) { p.AccountName = "" }},
		{name: "account_name too short", mutate: func(p *RegisterBusinessParams) { p.AccountName = "apple" }},
		{name: "account_name too long", mutate: func(p *RegisterBusinessParams) {
			p.AccountName = "abcdefghijklmnopqrstuvwxyz0123456789"
		}},
		{name: "account_name with dash", mutate: func(p *RegisterBusinessParams) { p.AccountName = "apple-store" }},
		{name: "account_name with chinese", mutate: func(p *RegisterBusinessParams) { p.AccountName = "苹果apple" }},
		{name: "empty nickname", mutate: func(p *RegisterBusinessParams) { p.Nickname = "" }},
		{name: "empty icon", mutate: func(p *RegisterBusinessParams) { p.IconMediaID = "" }},
	}

	// The unmodified params must pass, otherwise the cases below prove nothing.
	if err := CheckRegisterBusinessParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	for _, tc := range cases {
		params := valid
		tc.mutate(&params)

		if err := CheckRegisterBusinessParams(params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestRegisterBusinessError(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":45070,"errmsg":"account_name exists"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.RegisterBusiness(context.Background(), RegisterBusinessParams{
		AccountName: "apple_store",
		Nickname:    "苹果",
		IconMediaID: "media_1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 45070")
	}
	if err := RegisterBusinessError(resp); err == nil ||
		err.Error() != "account_name 对应的商户已被创建" {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateBusiness(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.UpdateBusiness(context.Background(), UpdateBusinessParams{
		BusinessID: 12,
		Nickname:   "新昵称",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if call.path != updateBusinessPath {
		t.Fatalf("path = %q", call.path)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	if body["business_id"] != float64(12) {
		t.Fatalf("business_id = %v", body["business_id"])
	}
	if body["nickname"] != "新昵称" {
		t.Fatalf("nickname = %v", body["nickname"])
	}
	// An empty icon must still be present as "": WeChat reads it as "do not
	// update", and the documented example sends it that way.
	if body["icon_media_id"] != "" {
		t.Fatalf("icon_media_id = %v", body["icon_media_id"])
	}
}

func TestUpdateBusinessValidation(t *testing.T) {
	app := newApp(t, "http://127.0.0.1:1", nil)

	if _, err := app.UpdateBusiness(context.Background(), UpdateBusinessParams{
		Nickname: "新昵称",
	}); err == nil {
		t.Fatal("missing business_id accepted")
	}
}

func TestGetBusiness(t *testing.T) {
	reply := `{"errcode":0,"errmsg":"ok","business_info":{` +
		`"business_id":12,"account_name":"apple_store",` +
		`"nickname":"苹果","icon_url":"https://example.com/i.png"}}`

	srv, call := newKFServer(t, reply)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetBusiness(context.Background(), GetBusinessParams{BusinessID: 12})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if resp.BusinessInfo.BusinessID != 12 || resp.BusinessInfo.AccountName != "apple_store" {
		t.Fatalf("info = %+v", resp.BusinessInfo)
	}
	if resp.BusinessInfo.IconURL != "https://example.com/i.png" {
		t.Fatalf("icon_url = %q", resp.BusinessInfo.IconURL)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	// Only the identifier that was set goes out.
	if _, ok := body["account_name"]; ok {
		t.Fatalf("account_name sent alongside business_id: %v", body)
	}
}

func TestGetBusinessByAccountName(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"business_info":{"business_id":12}}`)
	app := newApp(t, srv.URL, nil)

	if _, err := app.GetBusiness(context.Background(), GetBusinessParams{
		AccountName: "apple_store",
	}); err != nil {
		t.Fatal(err)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	if _, ok := body["business_id"]; ok {
		t.Fatalf("business_id sent alongside account_name: %v", body)
	}
	if body["account_name"] != "apple_store" {
		t.Fatalf("account_name = %v", body["account_name"])
	}
}

func TestGetBusinessValidation(t *testing.T) {
	app := newApp(t, "http://127.0.0.1:1", nil)

	// Neither identifier leaves nothing to look up.
	if _, err := app.GetBusiness(context.Background(), GetBusinessParams{}); err == nil {
		t.Fatal("empty params accepted")
	}

	// Both is ambiguous: WeChat documents them as alternatives.
	if _, err := app.GetBusiness(context.Background(), GetBusinessParams{
		BusinessID:  12,
		AccountName: "apple_store",
	}); err == nil {
		t.Fatal("both identifiers accepted")
	}
}

func TestGetBusinessNotFound(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":45071,"errmsg":"business not exist"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetBusiness(context.Background(), GetBusinessParams{BusinessID: 99})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 45071")
	}
	if err := GetBusinessError(resp); err == nil ||
		err.Error() != "business_id 对应的商户不存在" {
		t.Fatalf("err = %v", err)
	}
}

func TestListBusiness(t *testing.T) {
	reply := `{"errcode":0,"list":[` +
		`{"business_id":1,"account_name":"apple","nickname":"苹果"},` +
		`{"business_id":2,"account_name":"banana","nickname":"香蕉"}]}`

	srv, call := newKFServer(t, reply)
	app := newApp(t, srv.URL, nil)

	resp, err := app.ListBusiness(context.Background(), ListBusinessParams{
		Offset: 0,
		Count:  100,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if len(resp.List) != 2 || resp.List[1].AccountName != "banana" {
		t.Fatalf("list = %+v", resp.List)
	}
	if !resp.Done(100) {
		t.Fatal("Done() = false for a short page")
	}
	if resp.Done(2) {
		t.Fatal("Done() = true for a full page")
	}

	if call.path != listBusinessPath {
		t.Fatalf("path = %q", call.path)
	}

	var body struct {
		Offset int `json:"offset"`
		Count  int `json:"count"`
	}
	call.decodeBody(t, &body)

	if body.Offset != 0 || body.Count != 100 {
		t.Fatalf("body = %+v", body)
	}
}

func TestListBusinessEmptyPageIsOK(t *testing.T) {
	// An offset past the end is a valid answer, not an error.
	srv, _ := newKFServer(t, `{"errcode":0,"list":[]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.ListBusiness(context.Background(), ListBusinessParams{Count: 10})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatal("OK() = false for an empty page")
	}
	if !resp.Done(10) {
		t.Fatal("Done() = false for an empty page")
	}
}

func TestListBusinessValidation(t *testing.T) {
	app := newApp(t, "http://127.0.0.1:1", nil)

	cases := []ListBusinessParams{
		{Offset: -1, Count: 10},
		{Offset: 0, Count: 0},
		{Offset: 0, Count: ListBusinessPageSize + 1},
	}

	for _, params := range cases {
		if _, err := app.ListBusiness(context.Background(), params); err == nil {
			t.Fatalf("params accepted: %+v", params)
		}
	}
}

func TestGetKFWorkBound(t *testing.T) {
	reply := `{"errcode":0,"entityName":"某某有限公司",` +
		`"corpid":"ww123","bindTime":1694611289}`

	srv, call := newKFServer(t, reply)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetKFWorkBound(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if !resp.Bound() {
		t.Fatal("Bound() = false for a bound corp")
	}
	if resp.CorpID != "ww123" || resp.BindTime != 1694611289 {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.EntityName != "某某有限公司" {
		t.Fatalf("entityName = %q", resp.EntityName)
	}

	if call.method != http.MethodGet || call.path != getKFWorkBoundPath {
		t.Fatalf("%s %s", call.method, call.path)
	}
	if got := call.query.Get("access_token"); got != "TOKEN" {
		t.Fatalf("access_token = %q", got)
	}
}

func TestGetKFWorkBoundUnbound(t *testing.T) {
	// 未绑定时三个业务字段都不返回，errcode 仍为 0：这不是错误。
	srv, _ := newKFServer(t, `{"errcode":0}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetKFWorkBound(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatal("OK() = false for an unbound reply")
	}
	if resp.Bound() {
		t.Fatal("Bound() = true without a corpid")
	}
}

func TestGetKFWorkBoundNoPermission(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":10130030,"errmsg":"no permission"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetKFWorkBound(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if err := GetKFWorkBoundError(resp); err == nil ||
		err.Error() != "小程序账号无权限，须为非个人主体的小程序" {
		t.Fatalf("err = %v", err)
	}
}

func TestBindKFWork(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.BindKFWork(context.Background(), BindKFWorkParams{CorpID: "ww123"})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if call.method != http.MethodPost || call.path != bindKFWorkPath {
		t.Fatalf("%s %s", call.method, call.path)
	}

	var body struct {
		CorpID string `json:"corpid"`
	}
	call.decodeBody(t, &body)

	if body.CorpID != "ww123" {
		t.Fatalf("corpid = %q", body.CorpID)
	}
}

func TestBindKFWorkValidation(t *testing.T) {
	app := newApp(t, "http://127.0.0.1:1", nil)

	if _, err := app.BindKFWork(context.Background(), BindKFWorkParams{}); err == nil {
		t.Fatal("empty corpid accepted")
	}
}

func TestBindKFWorkAlreadyBound(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":10130603,"errmsg":"already bound"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.BindKFWork(context.Background(), BindKFWorkParams{CorpID: "ww123"})
	if err != nil {
		t.Fatal(err)
	}

	if err := BindKFWorkError(resp); err == nil ||
		err.Error() != "已经绑定微信客服，请解绑后重试" {
		t.Fatalf("err = %v", err)
	}
}

func TestUnbindKFWork(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.UnbindKFWork(context.Background(), UnbindKFWorkParams{CorpID: "ww123"})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if call.path != unbindKFWorkPath {
		t.Fatalf("path = %q", call.path)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	if body["corpid"] != "ww123" {
		t.Fatalf("corpid = %v", body["corpid"])
	}
}

func TestUnbindKFWorkValidation(t *testing.T) {
	app := newApp(t, "http://127.0.0.1:1", nil)

	if _, err := app.UnbindKFWork(context.Background(), UnbindKFWorkParams{}); err == nil {
		t.Fatal("empty corpid accepted")
	}
}

func TestUnbindKFWorkBadCorpID(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":10130600,"errmsg":"invalid corpid"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.UnbindKFWork(context.Background(), UnbindKFWorkParams{CorpID: "bad"})
	if err != nil {
		t.Fatal(err)
	}

	if err := UnbindKFWorkError(resp); err == nil || err.Error() != "corpid 不合法" {
		t.Fatalf("err = %v", err)
	}
}

func TestKFWorkErrMsgFallback(t *testing.T) {
	// An undocumented errcode must not leak WeChat's raw errmsg.
	if got := KFWorkErrMsg(99999999); got != KFWorkDefaultErrMsg {
		t.Fatalf("errmsg = %q", got)
	}
	if got := BusinessErrMsg(99999999); got != BusinessDefaultErrMsg {
		t.Fatalf("errmsg = %q", got)
	}
}

func TestBusinessInfoDecodesNumericID(t *testing.T) {
	// WeChat's parameter table declares business_id a string, but the examples and
	// the sibling endpoints send a number; decoding must follow the examples.
	var info BusinessInfo
	if err := json.Unmarshal([]byte(`{"business_id":12}`), &info); err != nil {
		t.Fatal(err)
	}
	if info.BusinessID != 12 {
		t.Fatalf("business_id = %d", info.BusinessID)
	}
}
