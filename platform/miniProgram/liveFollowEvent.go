package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

const (
	// LiveFollowEventName is "wxalive_follow_notify": a user subscribed to or
	// unsubscribed from the mini program's live streams.
	LiveFollowEventName = "wxalive_follow_notify"

	// LiveFollowActionAdd is the Action of a subscription.
	LiveFollowActionAdd = "add_follow"
	// LiveFollowActionDelete is the Action of an unsubscription.
	LiveFollowActionDelete = "del_follow"
)

// LiveStatus values carried in LiveFollowEvent.LiveStatus.
const (
	// LiveStatusLiving means the room was live when the user acted.
	LiveStatusLiving = 101
	// LiveStatusNotStarted means the room had not started yet.
	LiveStatusNotStarted = 102
	// LiveStatusFinished means the room had already ended.
	LiveStatusFinished = 103
)

// LiveStatusMap holds the LiveStatus descriptions WeChat documents.
var LiveStatusMap = map[int]string{
	LiveStatusLiving:     "直播中",
	LiveStatusNotStarted: "未开始",
	LiveStatusFinished:   "已结束",
}

// LiveFollowNotify is the business part of a follow event.
type LiveFollowNotify struct {
	// RoomID is the live room the user acted on.
	RoomID int64 `xml:"room_id" json:"room_id"`
	// UserOpenID is the subscriber.
	UserOpenID string `xml:"user_openid" json:"user_openid"`
	// Time is when the user acted, as a Unix timestamp in seconds.
	Time int64 `xml:"time" json:"time"`
	// LiveStatus is one of the LiveStatus constants.
	LiveStatus int `xml:"live_status" json:"live_status"`
	// Action is LiveFollowActionAdd or LiveFollowActionDelete.
	Action string `xml:"action" json:"action"`
}

// LiveFollowEvent is the payload of a long-term subscription change.
type LiveFollowEvent struct {
	EventNotify

	// FollowNotify carries the subscription change.
	FollowNotify LiveFollowNotify `xml:"FollowNotify" json:"FollowNotify"`
}

// OK reports whether the event identifies a user and a room.
func (e *LiveFollowEvent) OK() bool {
	return e != nil &&
		e.EventNotify.OK() &&
		e.FollowNotify.RoomID > 0 &&
		e.FollowNotify.UserOpenID != ""
}

// IsFollow reports whether the user subscribed.
func (e *LiveFollowEvent) IsFollow() bool {
	return e != nil && e.FollowNotify.Action == LiveFollowActionAdd
}

// IsUnfollow reports whether the user unsubscribed.
func (e *LiveFollowEvent) IsUnfollow() bool {
	return e != nil && e.FollowNotify.Action == LiveFollowActionDelete
}

// LiveStatusName returns the description of LiveStatus, or the empty string when
// the value is not documented.
func (e *LiveFollowEvent) LiveStatusName() string {
	if e == nil {
		return ""
	}

	return LiveStatusMap[e.FollowNotify.LiveStatus]
}

// LiveFollowHandler handles a subscription change.
type LiveFollowHandler func(ctx context.Context, evt *LiveFollowEvent) error

// ParseLiveFollowEvent decodes a long-term subscription change push.
func ParseLiveFollowEvent(data []byte) (*LiveFollowEvent, error) {
	evt := &LiveFollowEvent{}

	if err := decodeEventNotify(data, []string{LiveFollowEventName}, evt); err != nil {
		return nil, err
	}

	return evt, nil
}

// OnLiveFollow registers a handler for long-term subscription changes.
//
// The callback server must be configured (Config.Token or Config.AESKey) and
// mounted for the handler to ever run.
func (a *Application) OnLiveFollow(fn LiveFollowHandler) *Application {
	if fn == nil {
		return a
	}

	a.dispatcher.On(
		EventNotifyMsgType+"."+LiveFollowEventName,
		event.Handler(func(ctx context.Context, msg *event.Message) error {
			evt := &LiveFollowEvent{}
			if err := decodeDispatched(msg, []string{LiveFollowEventName}, evt); err != nil {
				return err
			}

			return fn(ctx, evt)
		}),
	)

	return a
}
