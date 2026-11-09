package miniProgram

import (
	"context"
	"strings"
	"testing"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/event"
)

// eventEnvelope wraps body in the XML envelope WeChat pushes.
func eventEnvelope(eventName, body string) string {
	return `<xml>` +
		`<ToUserName><![CDATA[gh_mini]]></ToUserName>` +
		`<FromUserName><![CDATA[gh_platform]]></FromUserName>` +
		`<CreateTime>1700000000</CreateTime>` +
		`<MsgType><![CDATA[event]]></MsgType>` +
		`<Event><![CDATA[` + eventName + `]]></Event>` +
		body +
		`</xml>`
}

// newEventApp builds an application that never touches the network.
func newEventApp(t *testing.T) *Application {
	t.Helper()

	app, err := New(&Config{AppID: "wxapp", AppSecret: "secret"})
	if err != nil {
		t.Fatal(err)
	}

	return app
}

// dispatch turns a raw payload into a callback message and delivers it, the way
// the callback server does.
func dispatch(t *testing.T, app *Application, payload string) error {
	t.Helper()

	msg, err := event.ParseMessage([]byte(payload))
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}

	return app.dispatcher.Dispatch(context.Background(), msg)
}

func TestParseUserInfoChangeEvent(t *testing.T) {
	payload := eventEnvelope(UserInfoChangeEventName,
		`<OpenID><![CDATA[o_user]]></OpenID>`+
			`<AppID><![CDATA[wx_app]]></AppID>`+
			`<RevokeInfo><![CDATA[6]]></RevokeInfo>`)

	evt, err := ParseUserInfoChangeEvent([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}

	if !evt.OK() {
		t.Fatalf("OK() = false, event = %+v", evt)
	}
	if evt.OpenID != "o_user" || evt.AppID != "wx_app" {
		t.Fatalf("OpenID/AppID = %q/%q", evt.OpenID, evt.AppID)
	}
	if !evt.IsModified() || evt.IsRevoked() || evt.IsCancelled() {
		t.Fatalf("event = %q, want %q", evt.Event, UserInfoChangeEventName)
	}
	if got := evt.RevokeInfoName(); got != "昵称和头像" {
		t.Fatalf("RevokeInfoName = %q", got)
	}
	if evt.CreateTime != 1700000000 || evt.ToUserName != "gh_mini" {
		t.Fatalf("header = %+v", evt.EventNotify)
	}
}

func TestParseUserInfoChangeEventRevoke(t *testing.T) {
	payload := eventEnvelope(UserInfoRevokeEventName,
		`<OpenID><![CDATA[o_user]]></OpenID>`+
			`<AppID><![CDATA[wx_app]]></AppID>`+
			`<RevokeInfo><![CDATA[8]]></RevokeInfo>`+
			`<PluginID><![CDATA[wxplugin]]></PluginID>`+
			`<OpenPID><![CDATA[o_plugin]]></OpenPID>`)

	evt, err := ParseUserInfoChangeEvent([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}

	if !evt.IsRevoked() || evt.IsModified() || evt.IsCancelled() {
		t.Fatalf("IsRevoked = false, event = %q", evt.Event)
	}
	if evt.RevokeInfoName() != "手机号" || evt.PluginID != "wxplugin" || evt.OpenPID != "o_plugin" {
		t.Fatalf("event = %+v", evt)
	}
}

func TestParseUserInfoChangeEventUndocumentedRevokeInfo(t *testing.T) {
	payload := eventEnvelope(UserInfoCancellationEventName,
		`<OpenID><![CDATA[o_user]]></OpenID>`+
			`<AppID><![CDATA[wx_app]]></AppID>`+
			`<RevokeInfo><![CDATA[99]]></RevokeInfo>`)

	evt, err := ParseUserInfoChangeEvent([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}

	if !evt.IsCancelled() {
		t.Fatalf("IsCancelled = false, event = %q", evt.Event)
	}
	// An undocumented value is returned as is rather than turned into "".
	if got := evt.RevokeInfoName(); got != "99" {
		t.Fatalf("RevokeInfoName = %q, want the raw value", got)
	}
}

func TestParsePunishEvent(t *testing.T) {
	payload := eventEnvelope(PunishEventName,
		`<event_type>2</event_type>`+
			`<punish_id><![CDATA[P-1]]></punish_id>`+
			`<appid><![CDATA[wx_app]]></appid>`+
			`<punish_time>1700000100</punish_time>`+
			`<illegal_reason><![CDATA[违规原因]]></illegal_reason>`+
			`<illegal_content><![CDATA[违规内容]]></illegal_content>`+
			`<rule_name><![CDATA[运营规范]]></rule_name>`+
			`<rule_url><![CDATA[https://rule]]></rule_url>`+
			`<adjust_guide_url><![CDATA[https://guide]]></adjust_guide_url>`+
			`<detail><![CDATA[{"banned_days":[7,0],"banned_function_names":["客服","分享"]}]]></detail>`)

	evt, err := ParsePunishEvent([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}

	if !evt.OK() {
		t.Fatalf("OK() = false, event = %+v", evt)
	}
	if evt.EventType != PunishTypeFunctionBan || evt.TypeName() != "功能封禁" {
		t.Fatalf("EventType = %d (%q)", evt.EventType, evt.TypeName())
	}
	if evt.PunishID != "P-1" || evt.AppID != "wx_app" || evt.PunishTime != 1700000100 {
		t.Fatalf("event = %+v", evt)
	}
	if len(evt.IllegalContent) != 1 || evt.IllegalContent[0] != "违规内容" {
		t.Fatalf("IllegalContent = %#v", evt.IllegalContent)
	}

	detail, err := evt.ParseDetail()
	if err != nil {
		t.Fatal(err)
	}
	if detail.FunctionBan == nil {
		t.Fatalf("detail = %+v, want a function ban", detail)
	}
	if len(detail.FunctionBan.BannedDays) != 2 || detail.FunctionBan.BannedDays[1] != PunishBanDaysPermanent {
		t.Fatalf("BannedDays = %#v", detail.FunctionBan.BannedDays)
	}
	if len(detail.FunctionBan.BannedFunctionNames) != 2 {
		t.Fatalf("BannedFunctionNames = %#v", detail.FunctionBan.BannedFunctionNames)
	}
}

func TestPunishEventParseDetail(t *testing.T) {
	tests := []struct {
		name      string
		eventType int
		detail    string
		assert    func(t *testing.T, detail *PunishDetail)
	}{
		{
			name:      "Warning",
			eventType: PunishTypeWarning,
			detail:    `{"warned_type":2,"rectify_deadline":1700000000,"warned_function_names":["客服"],"warned_ban_days":[7]}`,
			assert: func(t *testing.T, detail *PunishDetail) {
				if detail.Warning == nil || detail.Warning.TypeName() != "警告功能封禁" {
					t.Fatalf("warning = %+v", detail.Warning)
				}
				if detail.Warning.RectifyDeadline != 1700000000 || len(detail.Warning.WarnedBanDays) != 1 {
					t.Fatalf("warning = %+v", detail.Warning)
				}
			},
		},
		{
			name:      "Suspend",
			eventType: PunishTypeSuspend,
			detail:    `{"suspended_days":7}`,
			assert: func(t *testing.T, detail *PunishDetail) {
				if detail.Suspend == nil || detail.Suspend.SuspendedDays != 7 {
					t.Fatalf("suspend = %+v", detail.Suspend)
				}
			},
		},
		{
			name:      "AccountBan",
			eventType: PunishTypeAccountBan,
			detail:    `{"banned_days":0}`,
			assert: func(t *testing.T, detail *PunishDetail) {
				if detail.AccountBan == nil || detail.AccountBan.BannedDays != PunishBanDaysPermanent {
					t.Fatalf("account ban = %+v", detail.AccountBan)
				}
			},
		},
		{
			name:      "PageBan",
			eventType: PunishTypePageBan,
			detail:    `{"path":"pages/index/index"}`,
			assert: func(t *testing.T, detail *PunishDetail) {
				if detail.PageBan == nil || detail.PageBan.Path != "pages/index/index" {
					t.Fatalf("page ban = %+v", detail.PageBan)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := &PunishEvent{
				EventNotify: EventNotify{MsgType: EventNotifyMsgType, Event: PunishEventName},
				EventType:   tt.eventType,
				Detail:      tt.detail,
			}

			detail, err := evt.ParseDetail()
			if err != nil {
				t.Fatal(err)
			}

			tt.assert(t, detail)
		})
	}
}

func TestPunishEventParseDetailErrors(t *testing.T) {
	tests := []struct {
		name string
		evt  *PunishEvent
		code cerrors.Code
	}{
		{
			name: "UnknownEventType",
			evt:  &PunishEvent{EventType: 99, Detail: `{}`},
			code: cerrors.CodeInvalidResponse,
		},
		{
			name: "MissingDetail",
			evt:  &PunishEvent{EventType: PunishTypeWarning},
			code: cerrors.CodeInvalidResponse,
		},
		{
			name: "BrokenDetail",
			evt:  &PunishEvent{EventType: PunishTypeSuspend, Detail: `{`},
			code: cerrors.CodeDecodeFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.evt.ParseDetail(); err == nil {
				t.Fatal("err = nil, want a decode error")
			} else if cerrors.CodeOf(err) != tt.code {
				t.Fatalf("code = %v, want %v", cerrors.CodeOf(err), tt.code)
			}
		})
	}
}

func TestParsePunishEventContentArray(t *testing.T) {
	// The official example pushes illegal_content as a JSON array.
	payload := `{"ToUserName":"gh_mini","FromUserName":"gh_platform","CreateTime":1700000000,` +
		`"MsgType":"event","Event":"wxa_punish_event","event_type":1,"punish_id":"P-2",` +
		`"appid":"wx_app","illegal_content":["违规内容","违规内容2"]}`

	evt, err := ParsePunishEvent([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}

	if len(evt.IllegalContent) != 2 || evt.IllegalContent[1] != "违规内容2" {
		t.Fatalf("IllegalContent = %#v", evt.IllegalContent)
	}
}

func TestParseLiveFollowEvent(t *testing.T) {
	payload := eventEnvelope(LiveFollowEventName,
		`<FollowNotify>`+
			`<room_id>7</room_id>`+
			`<user_openid><![CDATA[o_user]]></user_openid>`+
			`<time>1700000200</time>`+
			`<live_status>101</live_status>`+
			`<action><![CDATA[add_follow]]></action>`+
			`</FollowNotify>`)

	evt, err := ParseLiveFollowEvent([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}

	if !evt.OK() || !evt.IsFollow() || evt.IsUnfollow() {
		t.Fatalf("event = %+v", evt)
	}
	if evt.FollowNotify.RoomID != 7 || evt.FollowNotify.UserOpenID != "o_user" {
		t.Fatalf("FollowNotify = %+v", evt.FollowNotify)
	}
	if evt.LiveStatusName() != "直播中" {
		t.Fatalf("LiveStatusName = %q", evt.LiveStatusName())
	}
}

func TestParseLiveFollowEventUnfollow(t *testing.T) {
	payload := eventEnvelope(LiveFollowEventName,
		`<FollowNotify><room_id>7</room_id><user_openid>o_user</user_openid>`+
			`<live_status>103</live_status><action>del_follow</action></FollowNotify>`)

	evt, err := ParseLiveFollowEvent([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}

	if !evt.IsUnfollow() || evt.IsFollow() {
		t.Fatalf("event = %+v", evt)
	}
	if evt.LiveStatusName() != "已结束" {
		t.Fatalf("LiveStatusName = %q", evt.LiveStatusName())
	}
}

func TestParseLivePushMessageNotifyEvent(t *testing.T) {
	payload := eventEnvelope(LivePushMessageNotifyEventName,
		`<PushMessageApiNotify>`+
			`<message_id><![CDATA[M-1]]></message_id>`+
			`<room_id>7</room_id>`+
			`<total_count>10</total_count>`+
			`<success_count>6</success_count>`+
			`<openid_error_count>1</openid_error_count>`+
			`<relation_error_count>2</relation_error_count>`+
			`<user_recv_limit_count>0</user_recv_limit_count>`+
			`<internal_error_count>1</internal_error_count>`+
			`</PushMessageApiNotify>`)

	evt, err := ParseLivePushMessageNotifyEvent([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}

	if !evt.OK() {
		t.Fatalf("OK() = false, event = %+v", evt)
	}
	if evt.Notify.MessageID != "M-1" || evt.Notify.RoomID != 7 {
		t.Fatalf("Notify = %+v", evt.Notify)
	}
	if got := evt.FailedCount(); got != 4 {
		t.Fatalf("FailedCount = %d, want 4", got)
	}

	sum := evt.Notify.OpenIDErrorCount + evt.Notify.RelationErrorCount +
		evt.Notify.UserRecvLimitCount + evt.Notify.InternalErrorCount
	if sum != evt.FailedCount() {
		t.Fatalf("error counters = %d, want %d", sum, evt.FailedCount())
	}
}

func TestEventNotifyDecodeErrors(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		parse   func([]byte) error
		code    cerrors.Code
	}{
		{
			name:    "EmptyPayload",
			payload: "",
			parse: func(b []byte) error {
				_, err := ParsePunishEvent(b)
				return err
			},
			code: cerrors.CodeInvalidArgument,
		},
		{
			name:    "BrokenXML",
			payload: `<xml><MsgType>event`,
			parse: func(b []byte) error {
				_, err := ParsePunishEvent(b)
				return err
			},
			code: cerrors.CodeDecodeFailed,
		},
		{
			name: "NotAnEvent",
			payload: `<xml><ToUserName>gh_mini</ToUserName><MsgType><![CDATA[text]]></MsgType>` +
				`<Event><![CDATA[wxa_punish_event]]></Event></xml>`,
			parse: func(b []byte) error {
				_, err := ParsePunishEvent(b)
				return err
			},
			code: cerrors.CodeInvalidResponse,
		},
		{
			name:    "OtherEvent",
			payload: eventEnvelope(LiveFollowEventName, ``),
			parse: func(b []byte) error {
				_, err := ParsePunishEvent(b)
				return err
			},
			code: cerrors.CodeInvalidResponse,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.parse([]byte(tt.payload))
			if err == nil {
				t.Fatal("err = nil, want a decode error")
			}
			if cerrors.CodeOf(err) != tt.code {
				t.Fatalf("code = %v, want %v (%v)", cerrors.CodeOf(err), tt.code, err)
			}
		})
	}
}

func TestEventNotifyOKRequiresIdentity(t *testing.T) {
	// The header alone is not enough: a punish event without punish_id/appid is
	// undecodable in practice even though every element is well formed.
	payload := eventEnvelope(PunishEventName, `<event_type>1</event_type>`)

	evt, err := ParsePunishEvent([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}

	if evt.OK() {
		t.Fatal("OK() = true, want false for an event without punish_id/appid")
	}
}

func TestOnPunishDispatch(t *testing.T) {
	app := newEventApp(t)

	var got *PunishEvent
	app.OnPunish(func(ctx context.Context, evt *PunishEvent) error {
		got = evt
		return nil
	})

	payload := eventEnvelope(PunishEventName,
		`<event_type>3</event_type><punish_id><![CDATA[P-3]]></punish_id>`+
			`<appid><![CDATA[wx_app]]></appid><detail><![CDATA[{"suspended_days":7}]]></detail>`)

	if err := dispatch(t, app, payload); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if got == nil {
		t.Fatal("handler was not called")
	}
	if got.Event != PunishEventName || got.PunishID != "P-3" {
		t.Fatalf("event = %+v", got)
	}

	detail, err := got.ParseDetail()
	if err != nil {
		t.Fatal(err)
	}
	if detail.Suspend == nil || detail.Suspend.SuspendedDays != 7 {
		t.Fatalf("detail = %+v", detail)
	}
}

func TestOnUserInfoChangeDispatchEveryEvent(t *testing.T) {
	names := []string{
		UserInfoChangeEventName,
		UserInfoRevokeEventName,
		UserInfoCancellationEventName,
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			app := newEventApp(t)

			var got *UserInfoChangeEvent
			app.OnUserInfoChange(func(ctx context.Context, evt *UserInfoChangeEvent) error {
				got = evt
				return nil
			})

			payload := eventEnvelope(name,
				`<OpenID><![CDATA[o_user]]></OpenID><AppID><![CDATA[wx_app]]></AppID>`)

			if err := dispatch(t, app, payload); err != nil {
				t.Fatalf("Dispatch: %v", err)
			}

			if got == nil || got.Event != name {
				t.Fatalf("event = %+v, want %q", got, name)
			}
		})
	}
}

func TestOnLiveFollowDispatch(t *testing.T) {
	app := newEventApp(t)

	var got *LiveFollowEvent
	app.OnLiveFollow(func(ctx context.Context, evt *LiveFollowEvent) error {
		got = evt
		return nil
	})

	payload := eventEnvelope(LiveFollowEventName,
		`<FollowNotify><room_id>9</room_id><user_openid>o_user</user_openid>`+
			`<live_status>102</live_status><action>add_follow</action></FollowNotify>`)

	if err := dispatch(t, app, payload); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if got == nil || got.FollowNotify.RoomID != 9 || !got.IsFollow() {
		t.Fatalf("event = %+v", got)
	}
}

func TestOnLivePushMessageNotifyDispatch(t *testing.T) {
	app := newEventApp(t)

	var got *LivePushMessageNotifyEvent
	app.OnLivePushMessageNotify(func(ctx context.Context, evt *LivePushMessageNotifyEvent) error {
		got = evt
		return nil
	})

	payload := eventEnvelope(LivePushMessageNotifyEventName,
		`<PushMessageApiNotify><message_id>M-9</message_id><room_id>9</room_id>`+
			`<total_count>2</total_count><success_count>2</success_count></PushMessageApiNotify>`)

	if err := dispatch(t, app, payload); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	if got == nil || got.Notify.MessageID != "M-9" || got.FailedCount() != 0 {
		t.Fatalf("event = %+v", got)
	}
}

func TestOnEventReportsUndecodablePayload(t *testing.T) {
	app := newEventApp(t)

	var called bool
	app.OnPunish(func(ctx context.Context, evt *PunishEvent) error {
		called = true
		return nil
	})

	// MsgType/Event route the message to the handler, but event_type is not an
	// integer, so the handler must not run and the error must surface.
	payload := eventEnvelope(PunishEventName, `<event_type>not-a-number</event_type>`)

	err := dispatch(t, app, payload)
	if err == nil {
		t.Fatal("err = nil, want a decode error")
	}
	if cerrors.CodeOf(err) != cerrors.CodeDecodeFailed {
		t.Fatalf("code = %v, want %v (%v)", cerrors.CodeOf(err), cerrors.CodeDecodeFailed, err)
	}
	if called {
		t.Fatal("handler ran with an undecodable payload")
	}
}

func TestEventNotifyMessageCarriesRaw(t *testing.T) {
	payload := eventEnvelope(PunishEventName, `<event_type>1</event_type>`)

	msg, err := event.ParseMessage([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}

	if string(msg.Raw) != payload {
		t.Fatalf("Raw = %q, want the original payload", msg.Raw)
	}
	if !strings.Contains(string(msg.Raw), "event_type") {
		t.Fatal("Raw lost the event specific fields")
	}
}
