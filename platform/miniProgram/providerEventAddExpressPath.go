package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/event"
)

// ProviderAddExpressPathEvent is the add_express_path callback: a batch waybill got
// new tracking nodes.
//
// Unlike the other courier callbacks this one is a plain push: the helper expects
// "success" or an empty body back, so the handler returns no reply. Read the nodes
// and return nil.
type ProviderAddExpressPathEvent struct {
	EventNotify

	// DeliveryID is the courier's id, e.g. "SF".
	DeliveryID string `xml:"DeliveryID" json:"DeliveryID"`
	// WaybillID is the waybill number.
	WaybillID string `xml:"WayBillId" json:"WayBillId"`
	// OrderID is the merchant's order identifier.
	OrderID string `xml:"OrderId" json:"OrderId"`
	// Version is the tracking version; WeChat only accepts a version newer than
	// the one it has, so a replayed push should be dropped.
	Version int `xml:"Version" json:"Version"`
	// Count is the number of nodes in Actions.
	Count int `xml:"Count" json:"Count"`
	// Actions are the new tracking nodes.
	Actions []ProviderExpressPathAction `xml:"Actions" json:"Actions"`
}

// ProviderExpressPathAction is one tracking node of the add_express_path callback.
type ProviderExpressPathAction struct {
	// ActionTime is when the node happened, as a Unix timestamp.
	ActionTime int64 `xml:"ActionTime" json:"ActionTime"`
	// ActionType is the node type, one of the ProviderAction* constants.
	ActionType int `xml:"ActionType" json:"ActionType"`
	// ActionMsg is the text shown on the tracking page.
	ActionMsg string `xml:"ActionMsg" json:"ActionMsg"`
}

// OK reports whether the payload carries a waybill and at least one node.
func (e *ProviderAddExpressPathEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && e.WaybillID != "" && len(e.Actions) > 0
}

// ProviderAddExpressPathHandler handles an add_express_path callback.
//
// Returning nil leaves the default "success" body, which is what this callback
// expects; returning an error is only for failures worth retrying.
type ProviderAddExpressPathHandler func(
	ctx context.Context, evt *ProviderAddExpressPathEvent,
) error

// OnProviderAddExpressPath registers a handler for the add_express_path callback.
func (a *Application) OnProviderAddExpressPath(fn ProviderAddExpressPathHandler) *Application {
	if fn == nil {
		return a
	}

	a.dispatcher.On(
		EventNotifyMsgType+"."+ProviderEventAddExpressPath,
		event.Handler(func(ctx context.Context, msg *event.Message) error {
			evt := &ProviderAddExpressPathEvent{}

			if err := decodeDispatched(msg, []string{ProviderEventAddExpressPath}, evt); err != nil {
				return err
			}

			return fn(ctx, evt)
		}),
	)

	return a
}
