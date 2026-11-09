package miniProgram

import (
	"context"
)

// ImmediateDeliveryOrderStatusEvent is the update_waybill_status callback: the
// courier reported a new state for a delivery order.
//
// This is the merchant-side (小程序方) direction -- the merchant receives it and
// still owes a resultcode reply, so the handler returns a reply like the
// courier-side callbacks do.
type ImmediateDeliveryOrderStatusEvent struct {
	EventNotify

	// ShopID is the merchant id the courier assigned.
	ShopID string `xml:"shopid" json:"shopid"`
	// ShopOrderID is the merchant's own order id.
	ShopOrderID string `xml:"shop_order_id" json:"shop_order_id"`
	// ShopNo is the merchant's store number.
	ShopNo string `xml:"shop_no" json:"shop_no"`
	// WaybillID is the delivery order id.
	WaybillID string `xml:"waybill_id" json:"waybill_id"`
	// ActionTime is when the state changed, as a Unix timestamp.
	ActionTime int64 `xml:"action_time" json:"action_time"`
	// OrderStatus is one of the ImmediateDeliveryStatus* constants.
	OrderStatus int `xml:"order_status" json:"order_status"`
	// ActionMsg is free text the courier attached.
	ActionMsg string `xml:"action_msg" json:"action_msg"`
	// Agent is the courier assigned to the order, when there is one.
	Agent *ImmediateDeliveryAgent `xml:"agent" json:"agent"`
}

// OK reports whether the header and the order identity are present.
func (e *ImmediateDeliveryOrderStatusEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && (e.ShopOrderID != "" || e.WaybillID != "")
}

// StatusMsg returns the documented description of the state, or "" when the
// courier sent a value WeChat does not define.
func (e *ImmediateDeliveryOrderStatusEvent) StatusMsg() string {
	if e == nil {
		return ""
	}

	return ImmediateDeliveryStatusMsg(e.OrderStatus)
}

// Terminal reports whether the order is finished.
func (e *ImmediateDeliveryOrderStatusEvent) Terminal() bool {
	return e != nil && ImmediateDeliveryStatusTerminal(e.OrderStatus)
}

// NewReply returns a reply with the header already filled in.
func (e *ImmediateDeliveryOrderStatusEvent) NewReply() *ImmediateDeliveryReply {
	reply := &ImmediateDeliveryReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ImmediateDeliveryOrderStatusHandler handles an update_waybill_status callback.
type ImmediateDeliveryOrderStatusHandler func(
	ctx context.Context, evt *ImmediateDeliveryOrderStatusEvent,
) (*ImmediateDeliveryReply, error)

// OnImmediateDeliveryOrderStatus registers a handler for the update_waybill_status
// callback.
func (a *Application) OnImmediateDeliveryOrderStatus(
	fn ImmediateDeliveryOrderStatusHandler,
) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ImmediateDeliveryEventOrderStatus,
		func() eventPayload { return &ImmediateDeliveryOrderStatusEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ImmediateDeliveryOrderStatusEvent))
		},
	)
}
