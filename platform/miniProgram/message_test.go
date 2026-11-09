package miniProgram

import (
	"context"
	"strings"
	"testing"

	"github.com/antsurge/atto-wechat/core/errors"
)

// newMessageApp starts a fake server whose reply for path is reply.
func newMessageApp(t *testing.T, path string, reply string) (*Application, *fakeWechat) {
	t.Helper()

	fake := &fakeWechat{
		tokenBody:      `{"access_token":"at","expires_in":7200}`,
		messageReplies: map[string]string{path: reply},
	}

	srv := newFakeServer(fake)
	t.Cleanup(srv.Close)

	return newApp(t, srv.URL, nil), fake
}

// assertMessageError checks that the reply was turned into the documented
// description, and that WeChat's raw errmsg never leaks into it.
func assertMessageError(t *testing.T, got string, code int, errMsgMap map[int]string) {
	t.Helper()

	if desc, ok := errMsgMap[code]; ok {
		if !strings.Contains(got, desc) {
			t.Fatalf("err = %q, want the documented description %q", got, desc)
		}
	} else if !strings.Contains(got, "系统错误，请稍后再试") {
		t.Fatalf("err = %q, want the default message", got)
	}

	if strings.Contains(got, "raw text") || strings.Contains(got, "rid:") {
		t.Fatalf("err = %q, want the raw errmsg to be dropped", got)
	}
}

func TestCreateActivityID(t *testing.T) {
	app, fake := newMessageApp(
		t,
		createActivityIDPath,
		`{"errcode":0,"errmsg":"ok","activity_id":"act-1","expiration_time":1735689600}`,
	)

	result, err := app.CreateActivityID(context.Background(), CreateActivityIDParams{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.ActivityID != "act-1" || result.ExpirationTime != 1735689600 {
		t.Fatalf("response = %+v", result)
	}

	// No sharer means no openid/unionid at all, not an empty parameter.
	query := fake.lastMessageQuery[createActivityIDPath]
	if _, ok := query["openid"]; ok {
		t.Fatalf("query = %v, want openid absent", query)
	}
	if fake.lastMessageQuery[createActivityIDPath].Get("access_token") != "at" {
		t.Fatalf("access_token = %q", query.Get("access_token"))
	}
}

func TestCreateActivityIDPrivate(t *testing.T) {
	app, fake := newMessageApp(
		t,
		createActivityIDPath,
		`{"errcode":0,"errmsg":"ok","activity_id":"act-1","expiration_time":1}`,
	)

	_, err := app.CreateActivityID(context.Background(), CreateActivityIDParams{
		UnionID: "union-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	query := fake.lastMessageQuery[createActivityIDPath]
	if query.Get("unionid") != "union-1" {
		t.Fatalf("unionid = %q", query.Get("unionid"))
	}
	if _, ok := query["openid"]; ok {
		t.Fatalf("query = %v, want openid absent", query)
	}
}

func TestCreateActivityIDErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"busy", `{"errcode":-1,"errmsg":"system busy"}`, -1},
		{"bad token", `{"errcode":40001,"errmsg":"invalid credential"}`, 40001},
		{"expired token", `{"errcode":42001,"errmsg":"access_token expired"}`, 42001},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := newMessageApp(t, createActivityIDPath, tt.body)

			result, err := app.CreateActivityID(context.Background(), CreateActivityIDParams{})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code || result.OK() {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			assertMessageError(t, CreateActivityIDError(result).Error(), tt.code, CreateActivityIDErrCodeMap)
		})
	}
}

func TestSetUpdatableMsg(t *testing.T) {
	app, fake := newMessageApp(t, setUpdatableMsgPath, `{"errcode":0,"errmsg":"ok"}`)

	result, err := app.SetUpdatableMsg(context.Background(), SetUpdatableMsgParams{
		ActivityID:  "act-1",
		TargetState: UpdatableMsgStateNotStarted,
		ParameterList: []UpdatableMsgParameter{
			{Name: UpdatableMsgParamMemberCount, Value: "1"},
			{Name: UpdatableMsgParamRoomLimit, Value: "3"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}

	body := string(fake.lastMessageBody[setUpdatableMsgPath])
	want := `{"activity_id":"act-1","target_state":0,"template_info":{"parameter_list":[` +
		`{"name":"member_count","value":"1"},{"name":"room_limit","value":"3"}]}}`
	if body != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}

func TestSetUpdatableMsgErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"bad activity id", `{"errcode":47501,"errmsg":"invalid activity_id"}`, 47501},
		{"expired activity id", `{"errcode":47504,"errmsg":"expired activity_id"}`, 47504},
		{"bad state", `{"errcode":47502,"errmsg":"invalid target_state"}`, 47502},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := newMessageApp(t, setUpdatableMsgPath, tt.body)

			result, err := app.SetUpdatableMsg(context.Background(), SetUpdatableMsgParams{
				ActivityID:  "act-1",
				TargetState: UpdatableMsgStateStarted,
				ParameterList: []UpdatableMsgParameter{
					{Name: UpdatableMsgParamPath, Value: "pages/index/index"},
					{Name: UpdatableMsgParamVersionType, Value: "release"},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			assertMessageError(t, SetUpdatableMsgError(result).Error(), tt.code, SetUpdatableMsgErrCodeMap)
		})
	}
}

func TestSetUpdatableMsgRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params SetUpdatableMsgParams
	}{
		{"missing activity id", SetUpdatableMsgParams{TargetState: 0}},
		{"bad state", SetUpdatableMsgParams{ActivityID: "act-1", TargetState: 2}},
		{"missing parameters", SetUpdatableMsgParams{ActivityID: "act-1", TargetState: 0}},
		{"unknown name", SetUpdatableMsgParams{
			ActivityID:  "act-1",
			TargetState: UpdatableMsgStateNotStarted,
			ParameterList: []UpdatableMsgParameter{
				{Name: "member", Value: "1"},
			},
		}},
		{"empty value", SetUpdatableMsgParams{
			ActivityID:  "act-1",
			TargetState: UpdatableMsgStateNotStarted,
			ParameterList: []UpdatableMsgParameter{
				{Name: UpdatableMsgParamMemberCount},
				{Name: UpdatableMsgParamRoomLimit, Value: "3"},
			},
		}},
		{"not started without room limit", SetUpdatableMsgParams{
			ActivityID:  "act-1",
			TargetState: UpdatableMsgStateNotStarted,
			ParameterList: []UpdatableMsgParameter{
				{Name: UpdatableMsgParamMemberCount, Value: "1"},
			},
		}},
		{"started without path", SetUpdatableMsgParams{
			ActivityID:  "act-1",
			TargetState: UpdatableMsgStateStarted,
			ParameterList: []UpdatableMsgParameter{
				{Name: UpdatableMsgParamVersionType, Value: "release"},
			},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.SetUpdatableMsg(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestSetChatToolMsg(t *testing.T) {
	app, fake := newMessageApp(t, setChatToolMsgPath, `{"errcode":0,"errmsg":"ok"}`)

	result, err := app.SetChatToolMsg(context.Background(), SetChatToolMsgParams{
		ActivityID:  "act-1",
		TargetState: ChatToolMsgStateRunning,
		TemplateID:  "tpl-1",
		VersionType: ChatToolMsgVersionRelease,
		ParticipatorList: []ChatToolMsgParticipator{
			{GroupOpenID: "group-1", State: ChatToolMsgParticipantJoined},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}

	body := string(fake.lastMessageBody[setChatToolMsgPath])
	want := `{"activity_id":"act-1","target_state":1,"template_id":"tpl-1","version_type":0,` +
		`"participator_info_list":[{"group_openid":"group-1","state":1}]}`
	if body != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}

func TestSetChatToolMsgRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params SetChatToolMsgParams
	}{
		{"missing activity id", SetChatToolMsgParams{TemplateID: "tpl-1", TargetState: 1}},
		{"missing template id", SetChatToolMsgParams{ActivityID: "act-1", TargetState: 1}},
		{"missing state", SetChatToolMsgParams{ActivityID: "act-1", TemplateID: "tpl-1"}},
		{"participator without openid", SetChatToolMsgParams{
			ActivityID:       "act-1",
			TemplateID:       "tpl-1",
			TargetState:      1,
			ParticipatorList: []ChatToolMsgParticipator{{State: 1}},
		}},
		{"participator without state", SetChatToolMsgParams{
			ActivityID:       "act-1",
			TemplateID:       "tpl-1",
			TargetState:      1,
			ParticipatorList: []ChatToolMsgParticipator{{GroupOpenID: "group-1"}},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.SetChatToolMsg(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestDeleteMessageTemplate(t *testing.T) {
	app, fake := newMessageApp(t, deleteMessageTemplatePath, `{"errcode":0,"errmsg":"ok"}`)

	result, err := app.DeleteMessageTemplate(context.Background(), DeleteMessageTemplateParams{
		PriTemplateID: "pri-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}

	if body := string(fake.lastMessageBody[deleteMessageTemplatePath]); body != `{"priTmplId":"pri-1"}` {
		t.Fatalf("body = %s", body)
	}
}

func TestGetCategory(t *testing.T) {
	app, fake := newMessageApp(
		t,
		getCategoryPath,
		`{"errcode":0,"errmsg":"ok","data":[{"id":616,"name":"公交"}]}`,
	)

	result, err := app.GetCategory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() || len(result.Data) != 1 {
		t.Fatalf("response = %+v", result)
	}
	if result.Data[0].ID != 616 || result.Data[0].Name != "公交" {
		t.Fatalf("data = %+v", result.Data)
	}
	if method := fake.lastMessageMethod[getCategoryPath]; method != "GET" {
		t.Fatalf("method = %q", method)
	}
}

func TestGetPubTemplateKeywords(t *testing.T) {
	app, fake := newMessageApp(
		t,
		getPubTemplateKeywordsPath,
		`{"errcode":0,"errmsg":"ok","data":[{"kid":1,"name":"物品名称","example":"名称","rule":"thing"}]}`,
	)

	result, err := app.GetPubTemplateKeywords(context.Background(), GetPubTemplateKeywordsParams{
		TitleID: "99",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() || len(result.Data) != 1 {
		t.Fatalf("response = %+v", result)
	}
	if result.Data[0].Kid != 1 || result.Data[0].Rule != "thing" {
		t.Fatalf("data = %+v", result.Data)
	}
	if query := fake.lastMessageQuery[getPubTemplateKeywordsPath]; query.Get("tid") != "99" {
		t.Fatalf("tid = %q", query.Get("tid"))
	}
}

func TestGetPubTemplateTitles(t *testing.T) {
	app, fake := newMessageApp(
		t,
		getPubTemplateTitlesPath,
		`{"errcode":0,"errmsg":"ok","count":55,"data":[{"tid":"99","title":"付款成功通知","type":2,"scenes":["1"]}]}`,
	)

	result, err := app.GetPubTemplateTitles(context.Background(), GetPubTemplateTitlesParams{
		CategoryIDs: []int64{616, 617},
		Start:       30,
		Limit:       10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() || result.Count != 55 || len(result.Data) != 1 {
		t.Fatalf("response = %+v", result)
	}
	if result.Data[0].TitleID != "99" || result.Data[0].Type != 2 {
		t.Fatalf("data = %+v", result.Data)
	}

	query := fake.lastMessageQuery[getPubTemplateTitlesPath]
	if query.Get("ids") != "616,617" {
		t.Fatalf("ids = %q", query.Get("ids"))
	}
	if query.Get("start") != "30" || query.Get("limit") != "10" {
		t.Fatalf("query = %v", query)
	}
}

func TestGetPubTemplateTitlesOmitsZeroLimit(t *testing.T) {
	app, fake := newMessageApp(t, getPubTemplateTitlesPath, `{"errcode":0,"errmsg":"ok","count":0}`)

	_, err := app.GetPubTemplateTitles(context.Background(), GetPubTemplateTitlesParams{
		CategoryIDs: []int64{616},
	})
	if err != nil {
		t.Fatal(err)
	}

	query := fake.lastMessageQuery[getPubTemplateTitlesPath]
	if _, ok := query["limit"]; ok {
		t.Fatalf("query = %v, want limit absent", query)
	}
}

func TestGetMessageTemplateList(t *testing.T) {
	app, _ := newMessageApp(
		t,
		getMessageTemplateListPath,
		`{"errcode":0,"errmsg":"ok","data":[{"priTmplId":"pri-1","title":"付款成功通知","content":"金额{{amount1.DATA}}","example":"金额10元","type":2}]}`,
	)

	result, err := app.GetMessageTemplateList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() || len(result.Data) != 1 {
		t.Fatalf("response = %+v", result)
	}
	if result.Data[0].PriTemplateID != "pri-1" || result.Data[0].Type != 2 {
		t.Fatalf("data = %+v", result.Data)
	}
}

func TestAddMessageTemplate(t *testing.T) {
	app, fake := newMessageApp(
		t,
		addMessageTemplatePath,
		`{"errcode":0,"errmsg":"ok","priTmplId":"pri-1"}`,
	)

	result, err := app.AddMessageTemplate(context.Background(), AddMessageTemplateParams{
		TitleID:          "99",
		KeywordIDs:       []int64{1, 2},
		SceneDescription: "下单提醒",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() || result.PriTemplateID != "pri-1" {
		t.Fatalf("response = %+v", result)
	}

	body := string(fake.lastMessageBody[addMessageTemplatePath])
	want := `{"tid":"99","kidList":[1,2],"sceneDesc":"下单提醒"}`
	if body != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}

func TestAddMessageTemplateRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params AddMessageTemplateParams
	}{
		{"missing tid", AddMessageTemplateParams{KeywordIDs: []int64{1}}},
		{"missing kid list", AddMessageTemplateParams{TitleID: "99"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.AddMessageTemplate(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestSendSubscribeMessage(t *testing.T) {
	app, fake := newMessageApp(t, sendSubscribeMessagePath, `{"errcode":0,"errmsg":"ok"}`)

	result, err := app.SendSubscribeMessage(context.Background(), SendSubscribeMessageParams{
		ToUser:     "openid-1",
		TemplateID: "pri-1",
		Page:       "pages/index/index",
		Data: map[string]SubscribeMessageValue{
			"thing1": {Value: "黄焖鸡"},
		},
		MiniProgramState: SubscribeMsgStateDeveloper,
		Lang:             SubscribeMsgLangZhCN,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}

	body := string(fake.lastMessageBody[sendSubscribeMessagePath])
	want := `{"touser":"openid-1","template_id":"pri-1","page":"pages/index/index",` +
		`"data":{"thing1":{"value":"黄焖鸡"}},"miniprogram_state":"developer","lang":"zh_CN"}`
	if body != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}

func TestSendSubscribeMessageErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"not subscribed", `{"errcode":43101,"errmsg":"user refuse to accept the msg"}`, 43101},
		{"bad template", `{"errcode":40037,"errmsg":"invalid template_id"}`, 40037},
		{"bad data", `{"errcode":47003,"errmsg":"argument invalid"}`, 47003},
		{"bad page", `{"errcode":41030,"errmsg":"invalid page"}`, 41030},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := newMessageApp(t, sendSubscribeMessagePath, tt.body)

			result, err := app.SendSubscribeMessage(context.Background(), SendSubscribeMessageParams{
				ToUser:     "openid-1",
				TemplateID: "pri-1",
				Data: map[string]SubscribeMessageValue{
					"thing1": {Value: "x"},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			assertMessageError(t, SendSubscribeMessageError(result).Error(), tt.code, SendSubscribeMessageErrCodeMap)
		})
	}
}

func TestSendSubscribeMessageRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	valid := SendSubscribeMessageParams{
		ToUser:     "openid-1",
		TemplateID: "pri-1",
		Data:       map[string]SubscribeMessageValue{"thing1": {Value: "x"}},
	}

	tests := []struct {
		name   string
		mutate func(*SendSubscribeMessageParams)
	}{
		{"missing touser", func(p *SendSubscribeMessageParams) { p.ToUser = "" }},
		{"missing template id", func(p *SendSubscribeMessageParams) { p.TemplateID = "" }},
		{"missing data", func(p *SendSubscribeMessageParams) { p.Data = nil }},
		{"empty data value", func(p *SendSubscribeMessageParams) {
			p.Data = map[string]SubscribeMessageValue{"thing1": {}}
		}},
		{"bad state", func(p *SendSubscribeMessageParams) { p.MiniProgramState = "production" }},
		{"bad lang", func(p *SendSubscribeMessageParams) { p.Lang = "fr_FR" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := valid
			tt.mutate(&params)

			if _, err := app.SendSubscribeMessage(context.Background(), params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestSetUserNotify(t *testing.T) {
	app, fake := newMessageApp(t, setUserNotifyPath, `{"errcode":0,"errmsg":"ok"}`)

	result, err := app.SetUserNotify(context.Background(), SetUserNotifyParams{
		OpenID:      "openid-1",
		NotifyType:  1003,
		NotifyCode:  "code-1",
		ContentJSON: `{"a":1}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}

	body := string(fake.lastMessageBody[setUserNotifyPath])
	want := `{"openid":"openid-1","notify_type":1003,"notify_code":"code-1","content_json":"{\"a\":1}"}`
	if body != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}

func TestSetUserNotifyErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"bad openid", `{"errcode":40003,"errmsg":"invalid openid"}`, 40003},
		{"not admitted", `{"errcode":85461,"errmsg":"notify_type not admitted"}`, 85461},
		{"bad content", `{"errcode":85470,"errmsg":"invalid content_json"}`, 85470},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := newMessageApp(t, setUserNotifyPath, tt.body)

			result, err := app.SetUserNotify(context.Background(), SetUserNotifyParams{
				OpenID:      "openid-1",
				NotifyType:  1003,
				NotifyCode:  "code-1",
				ContentJSON: "{}",
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			assertMessageError(t, SetUserNotifyError(result).Error(), tt.code, SetUserNotifyErrCodeMap)
		})
	}
}

func TestSetUserNotifyExt(t *testing.T) {
	app, fake := newMessageApp(t, setUserNotifyExtPath, `{"errcode":0,"errmsg":"ok"}`)

	_, err := app.SetUserNotifyExt(context.Background(), SetUserNotifyExtParams{
		OpenID:     "openid-1",
		NotifyType: 1003,
		NotifyCode: "code-1",
		ExtJSON:    `{"b":2}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	body := string(fake.lastMessageBody[setUserNotifyExtPath])
	want := `{"openid":"openid-1","notify_type":1003,"notify_code":"code-1","ext_json":"{\"b\":2}"}`
	if body != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}

func TestGetUserNotify(t *testing.T) {
	app, fake := newMessageApp(
		t,
		getUserNotifyPath,
		`{"errcode":0,"errmsg":"ok","data":[{"notify_type":1003,"notify_code":"code-1","content_json":"{}","check_json":"{}"}]}`,
	)

	result, err := app.GetUserNotify(context.Background(), GetUserNotifyParams{
		OpenID: "openid-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() || len(result.Data) != 1 {
		t.Fatalf("response = %+v", result)
	}
	if result.Data[0].NotifyCode != "code-1" {
		t.Fatalf("data = %+v", result.Data)
	}

	// An empty notify_code is dropped rather than sent as "".
	body := string(fake.lastMessageBody[getUserNotifyPath])
	if body != `{"openid":"openid-1"}` {
		t.Fatalf("body = %s", body)
	}
}

func TestSendCustomMessageText(t *testing.T) {
	app, fake := newMessageApp(t, sendCustomMessagePath, `{"errcode":0,"errmsg":"ok"}`)

	result, err := app.SendCustomMessage(context.Background(), SendCustomMessageParams{
		ToUser:  "openid-1",
		MsgType: CustomMsgTypeText,
		Text:    &CustomText{Content: "你好"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}

	if body := string(fake.lastMessageBody[sendCustomMessagePath]); body != `{"touser":"openid-1","msgtype":"text","text":{"content":"你好"}}` {
		t.Fatalf("body = %s", body)
	}
}

func TestSendCustomMessageMiniProgramPage(t *testing.T) {
	app, fake := newMessageApp(t, sendCustomMessagePath, `{"errcode":0,"errmsg":"ok"}`)

	_, err := app.SendCustomMessage(context.Background(), SendCustomMessageParams{
		ToUser:  "openid-1",
		MsgType: CustomMsgTypeMiniProgramPage,
		MiniProgramPage: &CustomMiniProgramPage{
			Title:        "订单详情",
			PagePath:     "pages/order/order?id=1",
			ThumbMediaID: "media-1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := string(fake.lastMessageBody[sendCustomMessagePath])
	want := `{"touser":"openid-1","msgtype":"miniprogrampage",` +
		`"miniprogrampage":{"title":"订单详情","pagepath":"pages/order/order?id=1","thumb_media_id":"media-1"}}`
	if body != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}

func TestSendCustomMessageErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"bad openid", `{"errcode":40003,"errmsg":"invalid openid"}`, 40003},
		{"window closed", `{"errcode":45015,"errmsg":"response out of time limit"}`, 45015},
		{"quota exceeded", `{"errcode":45047,"errmsg":"out of response count limit"}`, 45047},
		{"minor protection", `{"errcode":70000,"errmsg":"minor protection"}`, 70000},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := newMessageApp(t, sendCustomMessagePath, tt.body)

			result, err := app.SendCustomMessage(context.Background(), SendCustomMessageParams{
				ToUser:  "openid-1",
				MsgType: CustomMsgTypeText,
				Text:    &CustomText{Content: "x"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			assertMessageError(t, SendCustomMessageError(result).Error(), tt.code, SendCustomMessageErrCodeMap)
		})
	}
}

func TestSendCustomMessageRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params SendCustomMessageParams
	}{
		{"missing touser", SendCustomMessageParams{MsgType: CustomMsgTypeText, Text: &CustomText{Content: "x"}}},
		{"unknown msgtype", SendCustomMessageParams{ToUser: "openid-1", MsgType: "video"}},
		{"text without content", SendCustomMessageParams{ToUser: "openid-1", MsgType: CustomMsgTypeText}},
		{"image without media id", SendCustomMessageParams{ToUser: "openid-1", MsgType: CustomMsgTypeImage}},
		{"link without title", SendCustomMessageParams{ToUser: "openid-1", MsgType: CustomMsgTypeLink}},
		{"card without title", SendCustomMessageParams{
			ToUser:          "openid-1",
			MsgType:         CustomMsgTypeMiniProgramPage,
			MiniProgramPage: &CustomMiniProgramPage{PagePath: "pages/index/index"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.SendCustomMessage(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestSetTyping(t *testing.T) {
	app, fake := newMessageApp(t, setTypingPath, `{"errcode":0,"errmsg":"ok"}`)

	result, err := app.SetTyping(context.Background(), SetTypingParams{
		ToUser:  "openid-1",
		Command: TypingCommandTyping,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}

	if body := string(fake.lastMessageBody[setTypingPath]); body != `{"touser":"openid-1","command":"Typing"}` {
		t.Fatalf("body = %s", body)
	}
}

func TestSetTypingRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params SetTypingParams
	}{
		{"missing touser", SetTypingParams{Command: TypingCommandTyping}},
		{"bad command", SetTypingParams{ToUser: "openid-1", Command: "typing"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.SetTyping(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestAddKFAccount(t *testing.T) {
	app, fake := newMessageApp(t, addKFAccountPath, `{"errcode":0,"errmsg":"ok"}`)

	_, err := app.AddKFAccount(context.Background(), AddKFAccountParams{
		KFWx:       "kf_wx",
		BusinessID: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if body := string(fake.lastMessageBody[addKFAccountPath]); body != `{"kf_wx":"kf_wx","business_id":1}` {
		t.Fatalf("body = %s", body)
	}
}

func TestGetKFList(t *testing.T) {
	app, fake := newMessageApp(
		t,
		getKFListPath,
		`{"errcode":0,"errmsg":"ok","kf_list":[{"kf_account":"kf1@wx","kf_nick":"客服1","kf_id":"1","kf_headimgurl":"https://x","kf_wx":"wx1","kf_openid":"kf-openid-1"}]}`,
	)

	result, err := app.GetKFList(context.Background(), GetKFListParams{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() || len(result.KFList) != 1 {
		t.Fatalf("response = %+v", result)
	}
	if result.KFList[0].KFOpenID != "kf-openid-1" {
		t.Fatalf("kf_list = %+v", result.KFList)
	}

	if _, ok := fake.lastMessageQuery[getKFListPath]["business_id"]; ok {
		t.Fatalf("query = %v, want business_id absent", fake.lastMessageQuery[getKFListPath])
	}
}

func TestGetOnlineKFList(t *testing.T) {
	app, fake := newMessageApp(
		t,
		getOnlineKFListPath,
		`{"errcode":0,"errmsg":"ok","kf_online_list":[{"kf_account":"kf1@wx","kf_id":"1","kf_openid":"kf-openid-1","status":1,"accepted_case":2}]}`,
	)

	result, err := app.GetOnlineKFList(context.Background(), GetOnlineKFListParams{BusinessID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() || len(result.KFOnlineList) != 1 {
		t.Fatalf("response = %+v", result)
	}
	if result.KFOnlineList[0].Status != 1 || result.KFOnlineList[0].AcceptedCase != 2 {
		t.Fatalf("kf_online_list = %+v", result.KFOnlineList)
	}

	if query := fake.lastMessageQuery[getOnlineKFListPath]; query.Get("business_id") != "1" {
		t.Fatalf("business_id = %q", query.Get("business_id"))
	}
}

func TestDeleteKFAccount(t *testing.T) {
	app, fake := newMessageApp(t, deleteKFAccountPath, `{"errcode":0,"errmsg":"ok"}`)

	_, err := app.DeleteKFAccount(context.Background(), DeleteKFAccountParams{KFOpenID: "kf-openid-1"})
	if err != nil {
		t.Fatal(err)
	}
	if method := fake.lastMessageMethod[deleteKFAccountPath]; method != "GET" {
		t.Fatalf("method = %q", method)
	}
	if query := fake.lastMessageQuery[deleteKFAccountPath]; query.Get("kf_openid") != "kf-openid-1" {
		t.Fatalf("kf_openid = %q", query.Get("kf_openid"))
	}
}

func TestSetKFAdmin(t *testing.T) {
	app, fake := newMessageApp(t, setKFAdminPath, `{"errcode":0,"errmsg":"ok"}`)

	_, err := app.SetKFAdmin(context.Background(), SetKFAdminParams{KFOpenID: "kf-openid-1"})
	if err != nil {
		t.Fatal(err)
	}
	if query := fake.lastMessageQuery[setKFAdminPath]; query.Get("kf_openid") != "kf-openid-1" {
		t.Fatalf("kf_openid = %q", query.Get("kf_openid"))
	}
}

func TestCancelKFAdmin(t *testing.T) {
	app, fake := newMessageApp(t, cancelKFAdminPath, `{"errcode":0,"errmsg":"ok"}`)

	_, err := app.CancelKFAdmin(context.Background(), CancelKFAdminParams{KFOpenID: "kf-openid-1"})
	if err != nil {
		t.Fatal(err)
	}
	if query := fake.lastMessageQuery[cancelKFAdminPath]; query.Get("kf_openid") != "kf-openid-1" {
		t.Fatalf("kf_openid = %q", query.Get("kf_openid"))
	}
}

func TestKFAccountErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{"account exists", `{"errcode":65402,"errmsg":"kf account exists"}`, 65402},
		{"account not found", `{"errcode":65405,"errmsg":"kf account not exists"}`, 65405},
		{"not admin", `{"errcode":65414,"errmsg":"kf account not admin"}`, 65414},
		{"undocumented", `{"errcode":99999,"errmsg":"raw text"}`, 99999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := newMessageApp(t, addKFAccountPath, tt.body)

			result, err := app.AddKFAccount(context.Background(), AddKFAccountParams{KFWx: "kf_wx"})
			if err != nil {
				t.Fatal(err)
			}
			if result.ErrCode != tt.code {
				t.Fatalf("result = %+v, want errcode %d", result, tt.code)
			}

			assertMessageError(t, AddKFAccountError(result).Error(), tt.code, KFAccountErrCodeMap)
		})
	}
}

func TestUploadTempMedia(t *testing.T) {
	app, fake := newMessageApp(
		t,
		uploadTempMediaPath,
		`{"errcode":0,"errmsg":"ok","type":"image","media_id":"media-1","created_at":1735689600}`,
	)

	result, err := app.UploadTempMedia(context.Background(), UploadTempMediaParams{
		MediaType: TempMediaTypeImage,
		Filename:  "cover.png",
		Data:      []byte("png-bytes"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() || result.MediaID != "media-1" {
		t.Fatalf("response = %+v", result)
	}

	if method := fake.lastMessageMethod[uploadTempMediaPath]; method != "POST" {
		t.Fatalf("method = %q", method)
	}
	if query := fake.lastMessageQuery[uploadTempMediaPath]; query.Get("type") != "image" {
		t.Fatalf("type = %q", query.Get("type"))
	}

	// The file travels in a part named "media", with the caller's file name.
	body := string(fake.lastMessageBody[uploadTempMediaPath])
	for _, want := range []string{`name="media"`, `filename="cover.png"`, "png-bytes"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %s, want %s", body, want)
		}
	}
}

func TestUploadTempMediaRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params UploadTempMediaParams
	}{
		{"missing type", UploadTempMediaParams{Filename: "a.png", Data: []byte("x")}},
		{"unknown type", UploadTempMediaParams{MediaType: "file", Filename: "a.png", Data: []byte("x")}},
		{"missing filename", UploadTempMediaParams{MediaType: TempMediaTypeImage, Data: []byte("x")}},
		{"missing data", UploadTempMediaParams{MediaType: TempMediaTypeImage, Filename: "a.png"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.UploadTempMedia(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestGetTempMedia(t *testing.T) {
	fake := &fakeWechat{
		tokenBody:           `{"access_token":"at","expires_in":7200}`,
		messageContentTypes: map[string]string{getTempMediaPath: "image/png"},
		messageBodies:       map[string][]byte{getTempMediaPath: []byte("png-bytes")},
	}
	srv := newFakeServer(fake)
	defer srv.Close()

	app := newApp(t, srv.URL, nil)

	result, err := app.GetTempMedia(context.Background(), GetTempMediaParams{MediaID: "media-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK() {
		t.Fatalf("response = %+v", result)
	}
	if result.ContentType != "image/png" || string(result.Data) != "png-bytes" {
		t.Fatalf("response = %+v", result)
	}
	if query := fake.lastMessageQuery[getTempMediaPath]; query.Get("media_id") != "media-1" {
		t.Fatalf("media_id = %q", query.Get("media_id"))
	}
}

func TestGetTempMediaErrors(t *testing.T) {
	app, _ := newMessageApp(t, getTempMediaPath, `{"errcode":40007,"errmsg":"invalid media_id"}`)

	result, err := app.GetTempMedia(context.Background(), GetTempMediaParams{MediaID: "media-1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ErrCode != 40007 || result.OK() {
		t.Fatalf("response = %+v", result)
	}

	assertMessageError(t, GetTempMediaError(result).Error(), 40007, GetTempMediaErrCodeMap)
}

func TestGetUserNotifyRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.GetUserNotify(context.Background(), GetUserNotifyParams{}); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetPubTemplateKeywordsRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.GetPubTemplateKeywords(context.Background(), GetPubTemplateKeywordsParams{}); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestGetPubTemplateTitlesRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params GetPubTemplateTitlesParams
	}{
		{"missing ids", GetPubTemplateTitlesParams{}},
		{"negative start", GetPubTemplateTitlesParams{CategoryIDs: []int64{1}, Start: -1}},
		{"negative limit", GetPubTemplateTitlesParams{CategoryIDs: []int64{1}, Limit: -1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.GetPubTemplateTitles(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestDeleteKFAccountRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.DeleteKFAccount(context.Background(), DeleteKFAccountParams{}); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestAddKFAccountRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.AddKFAccount(context.Background(), AddKFAccountParams{}); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestSetKFAdminRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.SetKFAdmin(context.Background(), SetKFAdminParams{}); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestCancelKFAdminRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.CancelKFAdmin(context.Background(), CancelKFAdminParams{}); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestDeleteMessageTemplateRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	if _, err := app.DeleteMessageTemplate(context.Background(), DeleteMessageTemplateParams{}); !errors.Is(err, errors.CodeInvalidArgument) {
		t.Fatalf("err = %v", err)
	}
}

func TestSetUserNotifyRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params SetUserNotifyParams
	}{
		{"missing openid", SetUserNotifyParams{NotifyType: 1003, NotifyCode: "c", ContentJSON: "{}"}},
		{"missing notify type", SetUserNotifyParams{OpenID: "o", NotifyCode: "c", ContentJSON: "{}"}},
		{"missing notify code", SetUserNotifyParams{OpenID: "o", NotifyType: 1003, ContentJSON: "{}"}},
		{"missing content", SetUserNotifyParams{OpenID: "o", NotifyType: 1003, NotifyCode: "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.SetUserNotify(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestSetUserNotifyExtRequiresValidParams(t *testing.T) {
	app := newApp(t, "http://example.invalid", nil)

	tests := []struct {
		name   string
		params SetUserNotifyExtParams
	}{
		{"missing openid", SetUserNotifyExtParams{NotifyType: 1003, NotifyCode: "c", ExtJSON: "{}"}},
		{"missing notify type", SetUserNotifyExtParams{OpenID: "o", NotifyCode: "c", ExtJSON: "{}"}},
		{"missing notify code", SetUserNotifyExtParams{OpenID: "o", NotifyType: 1003, ExtJSON: "{}"}},
		{"missing ext json", SetUserNotifyExtParams{OpenID: "o", NotifyType: 1003, NotifyCode: "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := app.SetUserNotifyExt(context.Background(), tt.params); !errors.Is(err, errors.CodeInvalidArgument) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestSendCustomImageMessageBody(t *testing.T) {
	app, fake := newMessageApp(t, sendCustomMessagePath, `{"errcode":0,"errmsg":"ok"}`)

	_, err := app.SendCustomMessage(context.Background(), SendCustomMessageParams{
		ToUser:  "openid-1",
		MsgType: CustomMsgTypeImage,
		Image:   &CustomImage{MediaID: "media-1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if body := string(fake.lastMessageBody[sendCustomMessagePath]); body != `{"touser":"openid-1","msgtype":"image","image":{"media_id":"media-1"}}` {
		t.Fatalf("body = %s", body)
	}
}

func TestSendCustomLinkMessageBody(t *testing.T) {
	app, fake := newMessageApp(t, sendCustomMessagePath, `{"errcode":0,"errmsg":"ok"}`)

	_, err := app.SendCustomMessage(context.Background(), SendCustomMessageParams{
		ToUser:  "openid-1",
		MsgType: CustomMsgTypeLink,
		Link: &CustomLink{
			Title:       "标题",
			Description: "描述",
			URL:         "https://example.com",
			ThumbURL:    "https://example.com/a.png",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := string(fake.lastMessageBody[sendCustomMessagePath])
	want := `{"touser":"openid-1","msgtype":"link","link":{"title":"标题","description":"描述",` +
		`"url":"https://example.com","thumb_url":"https://example.com/a.png"}}`
	if body != want {
		t.Fatalf("body = %s, want %s", body, want)
	}
}
