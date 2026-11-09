package miniProgram

import (
	"context"
	"strings"

	"github.com/antsurge/atto-wechat/core/codec"
	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/event"
)

// 事件通知是微信主动推送到开发者服务器的报文，和 Code2Session 这类主动调用方向
// 相反：没有 access_token，也没有 errcode/errmsg 这样的返回头。
//
// 因此这里没有 XxxErrCodeMap：报文不符合预期时只可能是不完整、不是事件、事件名
// 对不上三种情况，错误统一用 core/errors 的语义码表达，调用方按 code 判断即可。

// EventNotifyMsgType is the MsgType carried by every event push.
const EventNotifyMsgType = "event"

// EventNotify holds the header shared by every event push.
//
// Every typed event embeds it, so a handler can log who pushed what and when
// without knowing which event it received.
type EventNotify struct {
	// ToUserName is the mini program's UserName (gh_xxx).
	ToUserName string `xml:"ToUserName" json:"ToUserName"`
	// FromUserName is the platform push account.
	FromUserName string `xml:"FromUserName" json:"FromUserName"`
	// CreateTime is the push time as a Unix timestamp in seconds.
	CreateTime int64 `xml:"CreateTime" json:"CreateTime"`
	// MsgType is always "event".
	MsgType string `xml:"MsgType" json:"MsgType"`
	// Event is the event name, e.g. "wxa_punish_event".
	Event string `xml:"Event" json:"Event"`
}

// OK reports whether the common header is complete enough to act on.
func (e *EventNotify) OK() bool {
	return e != nil && e.ToUserName != "" && e.Event != ""
}

// EventInfo exposes the embedded header to the shared decoder.
func (e *EventNotify) EventInfo() *EventNotify {
	return e
}

// eventPayload is implemented by every typed event through the embedded
// EventNotify, which is what lets decodeEventNotify fill and then inspect a
// caller-supplied struct.
type eventPayload interface {
	EventInfo() *EventNotify
}

// decodeEventNotify decodes data into dst and checks that it is an event whose
// name is one of want.
//
// Both formats the console offers are accepted: a payload starting with '{' is
// decoded as JSON, anything else as XML. WeChat's XML is CDATA-wrapped and its
// JSON equivalent carries the same field names, so one struct serves both.
func decodeEventNotify(data []byte, want []string, dst eventPayload) error {
	if len(data) == 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: event payload must not be empty")
	}

	var err error
	if isJSONPayload(data) {
		err = codec.Unmarshal(data, dst)
	} else {
		err = codec.XMLUnmarshal(data, dst)
	}
	if err != nil {
		return cerrors.Wrap(cerrors.CodeDecodeFailed, "miniprogram: decode event notify", err)
	}

	info := dst.EventInfo()
	if info.MsgType != EventNotifyMsgType {
		return cerrors.New(cerrors.CodeInvalidResponse, "miniprogram: callback MsgType is not event")
	}

	for _, name := range want {
		// WeChat documents several logistics events as case-insensitive, so the
		// comparison ignores case rather than rejecting a differently-cased name.
		if strings.EqualFold(info.Event, name) {
			return nil
		}
	}

	return cerrors.New(
		cerrors.CodeInvalidResponse,
		"miniprogram: unexpected event "+info.Event,
	)
}

// OnEvent registers a handler for one raw event name.
//
// It is the escape hatch for events this package has no typed structure for:
// the handler receives the parsed message and can read msg.Raw itself. Typed
// helpers such as OnPunish are built on top of it.
func (a *Application) OnEvent(name string, fn func(ctx context.Context, msg *event.Message) error) *Application {
	if name == "" || fn == nil {
		return a
	}

	a.dispatcher.On(EventNotifyMsgType+"."+name, event.Handler(fn))

	return a
}

// decodeDispatched decodes the payload of a dispatched message.
//
// A decoding failure is returned to the dispatcher, which logs it and still
// answers WeChat with success: the push was delivered, and a retry would only
// repeat the same undecodable payload.
func decodeDispatched(msg *event.Message, want []string, dst eventPayload) error {
	if msg == nil {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: event message must not be nil")
	}

	return decodeEventNotify(msg.Raw, want, dst)
}

// isJSONPayload reports whether data is a JSON document.
func isJSONPayload(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '{':
			return true
		default:
			return false
		}
	}

	return false
}
