package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

// 授权用户信息变更的三个事件名。
//
// 平台把三件事推成同一个报文结构，只在 Event 上区分：资料被清理、用户撤回
// 授权、用户注销账号。后两者要求开发者删除对应用户的数据，所以分开提供
// 判断方法，避免调用方自己比字符串。
const (
	// UserInfoChangeEventName is "user_info_modified": the platform cleaned up
	// profile fields it considers risky.
	UserInfoChangeEventName = "user_info_modified"
	// UserInfoRevokeEventName is "user_authorization_revoke": the user withdrew
	// an authorization, and the data it covered must be deleted.
	UserInfoRevokeEventName = "user_authorization_revoke"
	// UserInfoCancellationEventName is "user_authorization_cancellation": the
	// user deleted their account.
	UserInfoCancellationEventName = "user_authorization_cancellation"
)

// UserInfoRevokeInfoMap holds the RevokeInfo values WeChat documents, keyed by
// the value as it appears in the push.
var UserInfoRevokeInfoMap = map[string]string{
	"1":  "车牌号",
	"2":  "地址",
	"3":  "发票信息",
	"4":  "蓝牙",
	"5":  "麦克风",
	"6":  "昵称和头像",
	"7":  "摄像头",
	"8":  "手机号",
	"12": "微信运动步数",
	"13": "位置信息",
	"14": "选中的图片或视频",
	"15": "选中的文件",
	"16": "邮箱地址",
	"18": "选择的位置信息",
	"19": "昵称输入键盘中选择的微信昵称",
	"20": "获取用户头像组件中选择的微信头像",
}

// UserInfoChangeEvent is the payload of the three authorization events.
//
// RevokeInfo, PluginID and OpenPID are only meaningful for a revocation; they
// are empty for the other two events.
type UserInfoChangeEvent struct {
	EventNotify

	// OpenID identifies the authorized user.
	OpenID string `xml:"OpenID" json:"OpenID"`
	// AppID is the mini program the authorization belongs to.
	AppID string `xml:"AppID" json:"AppID"`
	// RevokeInfo lists what the user withdrew, as documented values.
	RevokeInfo string `xml:"RevokeInfo" json:"RevokeInfo"`
	// PluginID is the plugin's AppID when the revocation came from a plugin.
	PluginID string `xml:"PluginID" json:"PluginID"`
	// OpenPID is the user's OpenPID when the revocation came from a plugin.
	OpenPID string `xml:"OpenPID" json:"OpenPID"`
}

// OK reports whether the event identifies a user and an application.
func (e *UserInfoChangeEvent) OK() bool {
	return e != nil &&
		e.EventNotify.OK() &&
		e.OpenID != "" &&
		e.AppID != ""
}

// IsModified reports whether the platform cleaned up the user's profile.
func (e *UserInfoChangeEvent) IsModified() bool {
	return e != nil && e.Event == UserInfoChangeEventName
}

// IsRevoked reports whether the user withdrew an authorization, which obliges
// the developer to delete the data it covered.
func (e *UserInfoChangeEvent) IsRevoked() bool {
	return e != nil && e.Event == UserInfoRevokeEventName
}

// IsCancelled reports whether the user deleted their account.
func (e *UserInfoChangeEvent) IsCancelled() bool {
	return e != nil && e.Event == UserInfoCancellationEventName
}

// RevokeInfoName returns the description of RevokeInfo, or the raw value when it
// is not documented.
func (e *UserInfoChangeEvent) RevokeInfoName() string {
	if e == nil {
		return ""
	}

	if name, ok := UserInfoRevokeInfoMap[e.RevokeInfo]; ok {
		return name
	}

	return e.RevokeInfo
}

// UserInfoChangeHandler handles an authorization event.
type UserInfoChangeHandler func(ctx context.Context, evt *UserInfoChangeEvent) error

// ParseUserInfoChangeEvent decodes an authorization event push.
//
// Both the console's data formats are accepted: XML (the default) and JSON.
func ParseUserInfoChangeEvent(data []byte) (*UserInfoChangeEvent, error) {
	evt := &UserInfoChangeEvent{}

	want := []string{
		UserInfoChangeEventName,
		UserInfoRevokeEventName,
		UserInfoCancellationEventName,
	}
	if err := decodeEventNotify(data, want, evt); err != nil {
		return nil, err
	}

	return evt, nil
}

// OnUserInfoChange registers a handler for the three authorization events.
//
// One handler serves all three because the fields are shared; the handler
// branches on IsModified/IsRevoked/IsCancelled.
//
// The callback server must be configured (Config.Token or Config.AESKey) and
// mounted for the handler to ever run.
func (a *Application) OnUserInfoChange(fn UserInfoChangeHandler) *Application {
	if fn == nil {
		return a
	}

	names := []string{
		UserInfoChangeEventName,
		UserInfoRevokeEventName,
		UserInfoCancellationEventName,
	}

	for _, name := range names {
		a.dispatcher.On(
			EventNotifyMsgType+"."+name,
			event.Handler(func(ctx context.Context, msg *event.Message) error {
				evt := &UserInfoChangeEvent{}
				if err := decodeDispatched(msg, []string{name}, evt); err != nil {
					return err
				}

				return fn(ctx, evt)
			}),
		)
	}

	return a
}
