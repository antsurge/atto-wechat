package miniProgram

import (
	"context"
	"strings"
	"testing"
)

func TestSendEmployeeRelationMsg(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	const data = `{"data":{"character_string1":{"value":"aaa"},"amount1":{"value":"222"}}}`

	resp, err := app.SendEmployeeRelationMsg(context.Background(), SendEmployeeRelationMsgParams{
		TemplateID: "bV8Jk-XXXXXX",
		Page:       "page/XXX",
		ToUser:     "o_user",
		Data:       data,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}

	if call.method != "POST" || call.path != sendEmployeeRelationMsgPath {
		t.Fatalf("%s %s", call.method, call.path)
	}

	var body struct {
		TemplateID string `json:"template_id"`
		Page       string `json:"page"`
		ToUser     string `json:"touser"`
		Data       string `json:"data"`
	}
	call.decodeBody(t, &body)

	if body.TemplateID != "bV8Jk-XXXXXX" || body.Page != "page/XXX" || body.ToUser != "o_user" {
		t.Fatalf("body = %+v", body)
	}

	// data must travel as a JSON string, not as a nested object.
	if body.Data != data {
		t.Fatalf("data = %q", body.Data)
	}
}

func TestBuildEmployeeRelationMsgData(t *testing.T) {
	got, err := BuildEmployeeRelationMsgData(map[string]string{
		"character_string1": "aaa",
		"amount1":           "222",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Keys are sorted by the encoder, so the output is stable.
	want := `{"data":{"amount1":{"value":"222"},"character_string1":{"value":"aaa"}}}`
	if got != want {
		t.Fatalf("got %s, want %s", got, want)
	}

	// Whatever the helper produces must pass the params check.
	if err := CheckSendEmployeeRelationMsgParams(SendEmployeeRelationMsgParams{
		TemplateID: "t",
		Page:       "p",
		ToUser:     "o",
		Data:       got,
	}); err != nil {
		t.Fatalf("helper output rejected: %v", err)
	}

	if _, err := BuildEmployeeRelationMsgData(nil); err == nil {
		t.Fatal("empty values accepted")
	}
	if _, err := BuildEmployeeRelationMsgData(map[string]string{"": "v"}); err == nil {
		t.Fatal("empty keyword accepted")
	}
}

func TestSendEmployeeRelationMsgValidation(t *testing.T) {
	valid := SendEmployeeRelationMsgParams{
		TemplateID: "t",
		Page:       "p",
		ToUser:     "o",
		Data:       `{"data":{"k":{"value":"v"}}}`,
	}

	if err := CheckSendEmployeeRelationMsgParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*SendEmployeeRelationMsgParams)
	}{
		{name: "no template", mutate: func(p *SendEmployeeRelationMsgParams) { p.TemplateID = "" }},
		{name: "no page", mutate: func(p *SendEmployeeRelationMsgParams) { p.Page = "" }},
		{name: "no touser", mutate: func(p *SendEmployeeRelationMsgParams) { p.ToUser = "" }},
		{name: "no data", mutate: func(p *SendEmployeeRelationMsgParams) { p.Data = "" }},
		{name: "data is not json", mutate: func(p *SendEmployeeRelationMsgParams) { p.Data = `not json` }},
		{name: "data is an array", mutate: func(p *SendEmployeeRelationMsgParams) { p.Data = `[1,2]` }},
		{name: "data is null", mutate: func(p *SendEmployeeRelationMsgParams) { p.Data = `null` }},
	}

	for _, tc := range cases {
		params := valid
		tc.mutate(&params)

		if err := CheckSendEmployeeRelationMsgParams(params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestSendEmployeeRelationMsgUnboundUser(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":40256,"errmsg":"invalid employee relation"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.SendEmployeeRelationMsg(context.Background(), SendEmployeeRelationMsgParams{
		TemplateID: "t",
		Page:       "p",
		ToUser:     "o",
		Data:       `{"data":{"k":{"value":"v"}}}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 40256")
	}
	if err := SendEmployeeRelationMsgError(resp); err == nil ||
		err.Error() != "该用户没有绑定用工关系" {
		t.Fatalf("err = %v", err)
	}
}

func TestUnbindUserB2CAuthInfo(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.UnbindUserB2CAuthInfo(context.Background(), UnbindUserB2CAuthInfoParams{
		OpenIDList: []string{"o_1", "o_2"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if call.path != unbindUserB2CAuthInfoPath {
		t.Fatalf("path = %q", call.path)
	}

	var body struct {
		OpenIDList []string `json:"openid_list"`
	}
	call.decodeBody(t, &body)

	if len(body.OpenIDList) != 2 || body.OpenIDList[1] != "o_2" {
		t.Fatalf("openid_list = %v", body.OpenIDList)
	}
}

func TestUnbindUserB2CAuthInfoValidation(t *testing.T) {
	if err := CheckUnbindUserB2CAuthInfoParams(UnbindUserB2CAuthInfoParams{}); err == nil {
		t.Fatal("empty list accepted")
	}

	err := CheckUnbindUserB2CAuthInfoParams(UnbindUserB2CAuthInfoParams{
		OpenIDList: []string{"o_1", ""},
	})
	if err == nil {
		t.Fatal("empty openid accepted")
	}
	// The index is part of the message: in a long batch it is what locates the
	// offending entry.
	if !strings.Contains(err.Error(), "openid_list[1]") {
		t.Fatalf("err = %v", err)
	}
}

func TestUnbindUserB2CAuthInfoInvalidOpenID(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":40003,"errmsg":"invalid openid"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.UnbindUserB2CAuthInfo(context.Background(), UnbindUserB2CAuthInfoParams{
		OpenIDList: []string{"o_bad"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := UnbindUserB2CAuthInfoError(resp); err == nil || err.Error() != "openid 无效" {
		t.Fatalf("err = %v", err)
	}
}

func TestMsgSecCheck(t *testing.T) {
	reply := `{"errcode":0,"errmsg":"ok",` +
		`"result":{"suggest":"risky","label":20001},` +
		`"detail":[{"strategy":"content_model","errcode":0,"suggest":"risky","label":20006,"prob":90},` +
		`{"strategy":"keyword","errcode":0,"suggest":"pass","label":20006,"level":20,"keyword":"词"}],` +
		`"trace_id":"60ae120f"}`

	srv, call := newKFServer(t, reply)
	app := newApp(t, srv.URL, nil)

	resp, err := app.MsgSecCheck(context.Background(), MsgSecCheckParams{
		Content: "hello world!",
		Scene:   MsgSecCheckSceneProfile,
		OpenID:  "o_user",
	})
	if err != nil {
		t.Fatal(err)
	}

	// A risky verdict is a successful call.
	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if !resp.Risky() || resp.Passed() || resp.Review() {
		t.Fatalf("verdict = %+v", resp.Verdict)
	}
	if resp.Verdict.LabelMsg() != "时政" {
		t.Fatalf("label msg = %q", resp.Verdict.LabelMsg())
	}
	if len(resp.Detail) != 2 || resp.Detail[1].Keyword != "词" || resp.Detail[1].Level != 20 {
		t.Fatalf("detail = %+v", resp.Detail)
	}
	if resp.TraceID != "60ae120f" {
		t.Fatalf("trace_id = %q", resp.TraceID)
	}

	if call.path != msgSecCheckPath {
		t.Fatalf("path = %q", call.path)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	if body["version"] != float64(MsgSecCheckVersion) {
		t.Fatalf("version = %v", body["version"])
	}
	if body["scene"] != float64(MsgSecCheckSceneProfile) {
		t.Fatalf("scene = %v", body["scene"])
	}
	if body["content"] != "hello world!" || body["openid"] != "o_user" {
		t.Fatalf("body = %v", body)
	}
	// The optional fields stay out of the body when unset.
	for _, key := range []string{"title", "nickname", "signature"} {
		if _, ok := body[key]; ok {
			t.Fatalf("%s sent although empty: %v", key, body)
		}
	}
}

func TestMsgSecCheckPassed(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":0,"result":{"suggest":"pass","label":100}}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.MsgSecCheck(context.Background(), MsgSecCheckParams{
		Content: "你好",
		Scene:   MsgSecCheckSceneComment,
		OpenID:  "o_user",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || !resp.Passed() || resp.Risky() {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.Verdict.LabelMsg() != "正常" {
		t.Fatalf("label msg = %q", resp.Verdict.LabelMsg())
	}
}

func TestMsgSecCheckContentLength(t *testing.T) {
	params := MsgSecCheckParams{
		// 2500 Chinese characters are 7500 bytes: the limit counts characters.
		Content: strings.Repeat("好", MsgSecCheckMaxContentLength),
		Scene:   MsgSecCheckSceneForum,
		OpenID:  "o_user",
	}

	if err := CheckMsgSecCheckParams(params); err != nil {
		t.Fatalf("2500 characters rejected: %v", err)
	}

	params.Content += "坏"
	if err := CheckMsgSecCheckParams(params); err == nil {
		t.Fatal("2501 characters accepted")
	}

	if err := CheckMsgSecCheckParams(MsgSecCheckParams{Scene: 1, OpenID: "o"}); err == nil {
		t.Fatal("empty content accepted")
	}
}

func TestMsgSecCheckValidation(t *testing.T) {
	valid := MsgSecCheckParams{Content: "x", Scene: MsgSecCheckSceneProfile, OpenID: "o"}

	if err := CheckMsgSecCheckParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	scenes := []int{0, 5, -1}
	for _, scene := range scenes {
		params := valid
		params.Scene = scene

		if err := CheckMsgSecCheckParams(params); err == nil {
			t.Fatalf("scene %d accepted", scene)
		}
	}

	params := valid
	params.OpenID = ""
	if err := CheckMsgSecCheckParams(params); err == nil {
		t.Fatal("empty openid accepted")
	}

	// Every documented scene must be accepted.
	for scene := 1; scene <= 4; scene++ {
		params := valid
		params.Scene = scene

		if err := CheckMsgSecCheckParams(params); err != nil {
			t.Fatalf("scene %d rejected: %v", scene, err)
		}
	}
}

func TestMsgSecCheckNoVisitRecord(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":61010,"errmsg":"code is expired"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.MsgSecCheck(context.Background(), MsgSecCheckParams{
		Content: "x",
		Scene:   MsgSecCheckSceneProfile,
		OpenID:  "o_user",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 61010")
	}
	if err := MsgSecCheckError(resp); err == nil ||
		err.Error() != "用户访问记录超时（用户需在近两小时访问过小程序）" {
		t.Fatalf("err = %v", err)
	}
}

func TestMediaCheckAsync(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","trace_id":"60f96f1d"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.MediaCheckAsync(context.Background(), MediaCheckAsyncParams{
		MediaURL:  "https://example.com/a.png",
		MediaType: MediaCheckTypeImage,
		Scene:     MsgSecCheckSceneComment,
		OpenID:    "o_user",
	})
	if err != nil {
		t.Fatal(err)
	}

	// OK only says the submission was accepted: the verdict arrives by push.
	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if resp.TraceID != "60f96f1d" {
		t.Fatalf("trace_id = %q", resp.TraceID)
	}

	if call.path != mediaCheckAsyncPath {
		t.Fatalf("path = %q", call.path)
	}

	var body struct {
		MediaURL  string `json:"media_url"`
		MediaType int    `json:"media_type"`
		Version   int    `json:"version"`
		Scene     int    `json:"scene"`
		OpenID    string `json:"openid"`
	}
	call.decodeBody(t, &body)

	if body.MediaURL != "https://example.com/a.png" || body.MediaType != MediaCheckTypeImage {
		t.Fatalf("body = %+v", body)
	}
	if body.Version != MediaCheckVersion {
		t.Fatalf("version = %d", body.Version)
	}
}

func TestMediaCheckAsyncNoTraceID(t *testing.T) {
	// Without a trace_id the submission cannot be matched to its push, so it is
	// not a usable success even though errcode is 0.
	srv, _ := newKFServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.MediaCheckAsync(context.Background(), MediaCheckAsyncParams{
		MediaURL:  "https://example.com/a.png",
		MediaType: MediaCheckTypeImage,
		Scene:     MsgSecCheckSceneComment,
		OpenID:    "o_user",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true without a trace_id")
	}
}

func TestMediaCheckAsyncValidation(t *testing.T) {
	valid := MediaCheckAsyncParams{
		MediaURL:  "https://example.com/a.png",
		MediaType: MediaCheckTypeImage,
		Scene:     MsgSecCheckSceneComment,
		OpenID:    "o_user",
	}

	if err := CheckMediaCheckAsyncParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*MediaCheckAsyncParams)
	}{
		{name: "no url", mutate: func(p *MediaCheckAsyncParams) { p.MediaURL = "" }},
		{name: "relative url", mutate: func(p *MediaCheckAsyncParams) { p.MediaURL = "/a.png" }},
		{name: "file url", mutate: func(p *MediaCheckAsyncParams) { p.MediaURL = "file:///a.png" }},
		{name: "bare host", mutate: func(p *MediaCheckAsyncParams) { p.MediaURL = "example.com/a.png" }},
		{name: "media type 0", mutate: func(p *MediaCheckAsyncParams) { p.MediaType = 0 }},
		{name: "media type 3", mutate: func(p *MediaCheckAsyncParams) { p.MediaType = 3 }},
		{name: "scene 0", mutate: func(p *MediaCheckAsyncParams) { p.Scene = 0 }},
		{name: "scene 5", mutate: func(p *MediaCheckAsyncParams) { p.Scene = 5 }},
		{name: "no openid", mutate: func(p *MediaCheckAsyncParams) { p.OpenID = "" }},
	}

	for _, tc := range cases {
		params := valid
		tc.mutate(&params)

		if err := CheckMediaCheckAsyncParams(params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestMediaCheckAsyncBadMediaType(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":40004,"errmsg":"media type error"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.MediaCheckAsync(context.Background(), MediaCheckAsyncParams{
		MediaURL:  "https://example.com/a.png",
		MediaType: MediaCheckTypeImage,
		Scene:     MsgSecCheckSceneComment,
		OpenID:    "o_user",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := MediaCheckAsyncError(resp); err == nil ||
		err.Error() != "media_type 错误（支持 1 音频、2 图片）" {
		t.Fatalf("err = %v", err)
	}
}

func TestOnMediaCheckResult(t *testing.T) {
	app := newEventApp(t)

	var got *MediaCheckResultEvent

	app.OnMediaCheckResult(func(ctx context.Context, evt *MediaCheckResultEvent) error {
		got = evt

		return nil
	})

	payload := `{"ToUserName":"gh_1","FromUserName":"o_push","CreateTime":1626959646,` +
		`"MsgType":"event","Event":"wxa_media_check","appid":"wxapp",` +
		`"trace_id":"t1","version":2,"errcode":0,"errmsg":"ok",` +
		`"result":{"suggest":"risky","label":20002},` +
		`"detail":[{"strategy":"content_model","errcode":0,"suggest":"risky","label":20002,"prob":90}]}`

	if err := dispatch(t, app, payload); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if got == nil {
		t.Fatal("handler not called")
	}
	if !got.OK() {
		t.Fatalf("OK() = false, evt = %+v", got)
	}
	if !got.Risky() || got.Passed() || got.Review() {
		t.Fatalf("verdict = %+v", got.Verdict)
	}
	if got.TraceID != "t1" || got.AppID != "wxapp" || got.Version != 2 {
		t.Fatalf("evt = %+v", got)
	}
	if len(got.Detail) != 1 || !got.Detail[0].Valid() {
		t.Fatalf("detail = %+v", got.Detail)
	}
}

func TestMediaCheckResultDownloadFailed(t *testing.T) {
	// The push still arrives when WeChat could not fetch the media; it is not a
	// verdict and must not read as "clean".
	evt := &MediaCheckResultEvent{
		EventNotify: EventNotify{ToUserName: "gh_1", Event: MediaCheckEventResult},
		TraceID:     "t2",
		ErrCode:     MediaCheckDownloadFailed,
	}

	if evt.OK() {
		t.Fatal("OK() = true on a download failure")
	}
	if !evt.DownloadFailed() {
		t.Fatal("DownloadFailed() = false")
	}
	if evt.Passed() {
		t.Fatal("Passed() = true on a download failure")
	}
}

func TestGetUserRiskRank(t *testing.T) {
	// risk_rank 0 is the most trustworthy value, and still a real verdict.
	reply := `{"errcode":0,"errmsg":"getuserriskrank succ","risk_rank":0,"unoin_id":123456}`

	srv, call := newKFServer(t, reply)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetUserRiskRank(context.Background(), GetUserRiskRankParams{
		OpenID:   "o_user",
		Scene:    UserRiskSceneMarketing,
		ClientIP: "203.0.113.7",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}

	rank, ok := resp.Rank()
	if !ok || rank != UserRiskRankLowest {
		t.Fatalf("rank = %d, %v", rank, ok)
	}
	if resp.UnionID != 123456 {
		t.Fatalf("unoin_id = %d", resp.UnionID)
	}

	if call.path != getUserRiskRankPath {
		t.Fatalf("path = %q", call.path)
	}

	var body struct {
		AppID    string `json:"appid"`
		OpenID   string `json:"openid"`
		Scene    int    `json:"scene"`
		ClientIP string `json:"client_ip"`
	}
	call.decodeBody(t, &body)

	// appid is filled from the application config when the caller leaves it empty.
	if body.AppID != "wx1" {
		t.Fatalf("appid = %q", body.AppID)
	}
	if body.OpenID != "o_user" || body.Scene != UserRiskSceneMarketing ||
		body.ClientIP != "203.0.113.7" {
		t.Fatalf("body = %+v", body)
	}
}

func TestGetUserRiskRankAppIDOverride(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"risk_rank":3}`)
	app := newApp(t, srv.URL, nil)

	if _, err := app.GetUserRiskRank(context.Background(), GetUserRiskRankParams{
		AppID:    "wxExplicit",
		OpenID:   "o_user",
		Scene:    UserRiskSceneUGC,
		ClientIP: "2001:db8::1",
	}); err != nil {
		t.Fatal(err)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	if body["appid"] != "wxExplicit" {
		t.Fatalf("appid = %v", body["appid"])
	}
}

func TestGetUserRiskRankMissingRank(t *testing.T) {
	// A reply without risk_rank must not be read as "rank 0, most trustworthy".
	srv, _ := newKFServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetUserRiskRank(context.Background(), GetUserRiskRankParams{
		OpenID:   "o_user",
		Scene:    UserRiskSceneUGC,
		ClientIP: "203.0.113.7",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true without risk_rank")
	}
	if _, ok := resp.Rank(); ok {
		t.Fatal("Rank() reported a value that was not sent")
	}
}

func TestGetUserRiskRankValidation(t *testing.T) {
	valid := GetUserRiskRankParams{
		OpenID:   "o_user",
		Scene:    UserRiskSceneRegister,
		ClientIP: "203.0.113.7",
	}

	if err := CheckGetUserRiskRankParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*GetUserRiskRankParams)
	}{
		{name: "no openid", mutate: func(p *GetUserRiskRankParams) { p.OpenID = "" }},
		{name: "scene 3", mutate: func(p *GetUserRiskRankParams) { p.Scene = 3 }},
		{name: "scene -1", mutate: func(p *GetUserRiskRankParams) { p.Scene = -1 }},
		{name: "no ip", mutate: func(p *GetUserRiskRankParams) { p.ClientIP = "" }},
		{name: "ip with port", mutate: func(p *GetUserRiskRankParams) { p.ClientIP = "203.0.113.7:8080" }},
		{name: "forwarded chain", mutate: func(p *GetUserRiskRankParams) { p.ClientIP = "203.0.113.7, 10.0.0.1" }},
		{name: "hostname", mutate: func(p *GetUserRiskRankParams) { p.ClientIP = "example.com" }},
	}

	for _, tc := range cases {
		params := valid
		tc.mutate(&params)

		if err := CheckGetUserRiskRankParams(params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestGetUserRiskRankServiceUnavailable(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":61080,"errmsg":"risk service unavailable"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetUserRiskRank(context.Background(), GetUserRiskRankParams{
		OpenID:   "o_user",
		Scene:    UserRiskSceneUGC,
		ClientIP: "203.0.113.7",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 61080")
	}
	if err := GetUserRiskRankError(resp); err == nil ||
		err.Error() != "当前用户安全服务不可用" {
		t.Fatalf("err = %v", err)
	}
}

func TestSecCheckLabelMsgUnknown(t *testing.T) {
	// An undocumented label returns "", so a caller can tell it apart from 正常.
	if got := SecCheckLabelMsg(99999); got != "" {
		t.Fatalf("label msg = %q", got)
	}
	if got := MsgSecCheckSceneMsg(9); got != "" {
		t.Fatalf("scene msg = %q", got)
	}
	if got := MediaCheckTypeMsg(9); got != "" {
		t.Fatalf("media type msg = %q", got)
	}
	if got := MsgSecCheckErrMsg(999999); got != MsgSecCheckDefaultErrMsg {
		t.Fatalf("errmsg = %q", got)
	}
}
