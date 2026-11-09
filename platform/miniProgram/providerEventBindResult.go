package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

// ProviderBindResultEvent is the update_business_bind_result callback
// (logistics.onBindResultUpdate): the audit result of a courier account the mini
// program bound changed.
//
// It is a plain push -- the helper wants "success" or an empty body back -- so the
// handler returns an error instead of a reply document.
//
// The payload fields are snake_case while the header is PascalCase; the tags
// follow the document.
type ProviderBindResultEvent struct {
	EventNotify

	// ErrCode is 0 when the audit passed, non-zero when it failed.
	ErrCode int `xml:"errcode" json:"errcode"`
	// ErrMsg is the failure reason, "ok" on success.
	ErrMsg string `xml:"errmsg" json:"errmsg"`
	// DeliveryID is the courier's id, e.g. "EMS".
	DeliveryID string `xml:"delivery_id" json:"delivery_id"`
	// BizID is the courier account the merchant bound.
	BizID string `xml:"biz_id" json:"biz_id"`
}

// OK reports whether the payload carries an account and an audit result.
//
// A failed audit is a valid, expected payload: check Succeeded to tell the two
// apart. OK only says the document can be trusted.
func (e *ProviderBindResultEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && e.BizID != ""
}

// Succeeded reports whether the courier accepted the binding.
func (e *ProviderBindResultEvent) Succeeded() bool {
	return e.OK() && e.ErrCode == 0
}

// ProviderBindResultHandler handles an update_business_bind_result callback.
//
// Returning nil leaves the default "success" body, which is what this callback
// expects.
type ProviderBindResultHandler func(
	ctx context.Context, evt *ProviderBindResultEvent,
) error

// OnProviderBindResult registers a handler for the update_business_bind_result
// callback.
func (a *Application) OnProviderBindResult(fn ProviderBindResultHandler) *Application {
	if fn == nil {
		return a
	}

	a.dispatcher.On(
		EventNotifyMsgType+"."+ProviderEventBindResult,
		event.Handler(func(ctx context.Context, msg *event.Message) error {
			evt := &ProviderBindResultEvent{}

			if err := decodeDispatched(msg, []string{ProviderEventBindResult}, evt); err != nil {
				return err
			}

			return fn(ctx, evt)
		}),
	)

	return a
}
