package miniProgram

import (
	"context"
)

// ImmediateDeliveryMockUpdateOrderEvent is the mock_update_order_status callback:
// the merchant pushes a status change to test its courier integration.
//
// It carries the same state values as the real one, which is the point: a test
// that passes here should behave the same way in production.
type ImmediateDeliveryMockUpdateOrderEvent struct {
	ImmediateDeliveryEvent

	// OrderStatus is one of the ImmediateDeliveryStatus* constants.
	OrderStatus int `xml:"order_status" json:"order_status"`
	// ActionTime is when the state changed, as a Unix timestamp.
	ActionTime int64 `xml:"action_time" json:"action_time"`
	// ActionMsg is free text, optional.
	ActionMsg string `xml:"action_msg" json:"action_msg"`
}

// StatusMsg returns the documented description of the state, or "" when unknown.
func (e *ImmediateDeliveryMockUpdateOrderEvent) StatusMsg() string {
	if e == nil {
		return ""
	}

	return ImmediateDeliveryStatusMsg(e.OrderStatus)
}

// NewReply returns a reply with the header already filled in.
func (e *ImmediateDeliveryMockUpdateOrderEvent) NewReply() *ImmediateDeliveryReply {
	reply := &ImmediateDeliveryReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ImmediateDeliveryMockUpdateOrderHandler handles a mock_update_order_status
// callback.
type ImmediateDeliveryMockUpdateOrderHandler func(
	ctx context.Context, evt *ImmediateDeliveryMockUpdateOrderEvent,
) (*ImmediateDeliveryReply, error)

// OnImmediateDeliveryMockUpdateOrder registers a handler for the
// mock_update_order_status callback.
func (a *Application) OnImmediateDeliveryMockUpdateOrder(
	fn ImmediateDeliveryMockUpdateOrderHandler,
) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ImmediateDeliveryEventMockUpdateOrder,
		func() eventPayload { return &ImmediateDeliveryMockUpdateOrderEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ImmediateDeliveryMockUpdateOrderEvent))
		},
	)
}
