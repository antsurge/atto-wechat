package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

// LivePushMessageNotifyEventName is "wxalive_push_message_notify": the final
// result of a long-term subscription broadcast.
//
// PushLiveMessage only accepts the request; the per-user outcome arrives later
// through this event, correlated by MessageID.
const LivePushMessageNotifyEventName = "wxalive_push_message_notify"

// LivePushMessageNotify is the business part of a broadcast result.
//
// The five error counters partition the failures: they add up to
// TotalCount-SuccessCount, so a caller can log the breakdown without guessing.
type LivePushMessageNotify struct {
	// MessageID identifies the broadcast, matching the one the push returned.
	MessageID string `xml:"message_id" json:"message_id"`
	// RoomID is the live room the broadcast belonged to.
	RoomID int64 `xml:"room_id" json:"room_id"`
	// TotalCount is how many openids the broadcast targeted.
	TotalCount int `xml:"total_count" json:"total_count"`
	// SuccessCount is how many messages were delivered.
	SuccessCount int `xml:"success_count" json:"success_count"`
	// OpenIDErrorCount counts openids WeChat rejected as invalid.
	OpenIDErrorCount int `xml:"openid_error_count" json:"openid_error_count"`
	// RelationErrorCount counts users who do not follow the mini program.
	RelationErrorCount int `xml:"relation_error_count" json:"relation_error_count"`
	// UserRecvLimitCount counts users who exceeded their message quota.
	UserRecvLimitCount int `xml:"user_recv_limit_count" json:"user_recv_limit_count"`
	// InternalErrorCount counts failures on WeChat's side.
	InternalErrorCount int `xml:"internal_error_count" json:"internal_error_count"`
}

// LivePushMessageNotifyEvent is the payload of a broadcast result.
type LivePushMessageNotifyEvent struct {
	EventNotify

	// Notify carries the broadcast result.
	Notify LivePushMessageNotify `xml:"PushMessageApiNotify" json:"PushMessageApiNotify"`
}

// OK reports whether the event identifies a finished broadcast.
func (e *LivePushMessageNotifyEvent) OK() bool {
	return e != nil &&
		e.EventNotify.OK() &&
		e.Notify.MessageID != "" &&
		e.Notify.RoomID > 0
}

// FailedCount returns how many messages were not delivered.
func (e *LivePushMessageNotifyEvent) FailedCount() int {
	if e == nil {
		return 0
	}

	return e.Notify.TotalCount - e.Notify.SuccessCount
}

// LivePushMessageNotifyHandler handles a broadcast result.
type LivePushMessageNotifyHandler func(ctx context.Context, evt *LivePushMessageNotifyEvent) error

// ParseLivePushMessageNotifyEvent decodes a broadcast result push.
func ParseLivePushMessageNotifyEvent(data []byte) (*LivePushMessageNotifyEvent, error) {
	evt := &LivePushMessageNotifyEvent{}

	if err := decodeEventNotify(data, []string{LivePushMessageNotifyEventName}, evt); err != nil {
		return nil, err
	}

	return evt, nil
}

// OnLivePushMessageNotify registers a handler for broadcast results.
//
// The callback server must be configured (Config.Token or Config.AESKey) and
// mounted for the handler to ever run.
func (a *Application) OnLivePushMessageNotify(fn LivePushMessageNotifyHandler) *Application {
	if fn == nil {
		return a
	}

	a.dispatcher.On(
		EventNotifyMsgType+"."+LivePushMessageNotifyEventName,
		event.Handler(func(ctx context.Context, msg *event.Message) error {
			evt := &LivePushMessageNotifyEvent{}
			if err := decodeDispatched(msg, []string{LivePushMessageNotifyEventName}, evt); err != nil {
				return err
			}

			return fn(ctx, evt)
		}),
	)

	return a
}
