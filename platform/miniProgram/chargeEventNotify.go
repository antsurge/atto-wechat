package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/codec"
	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/event"
)

// 付费管理（charge）的两个告警事件。
//
// 和上面所有事件都不同，这两个报文没有 ToUserName / FromUserName / CreateTime /
// MsgType / Event 头，只有业务字段加一个数字型的 event_type。官方文档把它们写成
// "请求体"里的一张字段表，返回体是 "success 或空字符串（无需加密）"。
//
// 因为没有 MsgType 和 Event，路由只能靠 event_type：解析时落到 event_type.N 这条
// 路由上，注册也注册在同一条路由上。core/event 为此放行了空 MsgType（见
// ParseMessage 与 EventTypeRoute）。

// event_type values the charge alerts carry.
const (
	// ChargeEventTypeQuota is 3: the purchased SPU is running out.
	ChargeEventTypeQuota = 3
	// ChargeEventTypeValidity is 4: the purchased SPU is about to expire.
	ChargeEventTypeValidity = 4
)

// decodeChargeNotify decodes a headerless charge payload and checks that it is the
// event the caller registered for.
//
// A mismatched event_type cannot normally reach here, because routing is keyed by
// it; the check stays so that a future routing change cannot silently hand a
// validity warning to the quota handler.
func decodeChargeNotify(msg *event.Message, want int, dst any) error {
	if msg == nil {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: charge message must not be nil")
	}

	if len(msg.Raw) == 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: charge payload must not be empty")
	}

	if err := codec.Unmarshal(msg.Raw, dst); err != nil {
		return cerrors.Wrap(cerrors.CodeDecodeFailed, "miniprogram: decode charge payload", err)
	}

	if msg.EventType != want {
		return cerrors.New(cerrors.CodeInvalidResponse, "miniprogram: unexpected charge event_type")
	}

	return nil
}

// onChargeEvent registers a handler on the route a headerless event resolves to.
func (a *Application) onChargeEvent(name string, handle event.Handler) *Application {
	if handle == nil {
		return a
	}

	a.dispatcher.On(name, handle)

	return a
}

// chargeHandler runs fn with a decoded payload.
//
// The two events differ only in the struct they decode into, so the boilerplate of
// "check the message, decode it, call fn" lives here instead of being copied.
func chargeHandler[T any](
	want int, fn func(ctx context.Context, evt *T) error,
) event.Handler {
	return event.Handler(func(ctx context.Context, msg *event.Message) error {
		evt := new(T)

		if err := decodeChargeNotify(msg, want, evt); err != nil {
			return err
		}

		return fn(ctx, evt)
	})
}
