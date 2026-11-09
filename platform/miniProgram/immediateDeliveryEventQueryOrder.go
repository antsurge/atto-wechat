package miniProgram

import (
	"context"
)

// ImmediateDeliveryQueryOrderEvent is the transport_query_order_status callback:
// the merchant asks for a delivery order's current state.
//
// The reply must carry the state and the delivery order id, otherwise the merchant
// cannot reconcile its own record.
type ImmediateDeliveryQueryOrderEvent struct {
	ImmediateDeliveryEvent
}

// NewReply returns a reply with the header and the delivery order id filled in.
//
// The id is copied because WeChat requires it in the reply, and it is the only
// field that can be taken from the request.
func (e *ImmediateDeliveryQueryOrderEvent) NewReply() *ImmediateDeliveryReply {
	reply := &ImmediateDeliveryReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
		reply.WaybillID = e.WaybillID
	}

	return reply
}

// StatusReply answers with a state, filling the description from the documented
// table when msg is empty.
func (e *ImmediateDeliveryQueryOrderEvent) StatusReply(
	status int, msg string,
) *ImmediateDeliveryReply {
	reply := e.NewReply()
	reply.Succeed()
	reply.OrderStatus = status

	if msg == "" {
		msg = ImmediateDeliveryStatusMsg(status)
	}
	reply.ActionMsg = msg

	return reply
}

// ImmediateDeliveryQueryOrderHandler handles a transport_query_order_status
// callback.
type ImmediateDeliveryQueryOrderHandler func(
	ctx context.Context, evt *ImmediateDeliveryQueryOrderEvent,
) (*ImmediateDeliveryReply, error)

// OnImmediateDeliveryQueryOrder registers a handler for the
// transport_query_order_status callback.
func (a *Application) OnImmediateDeliveryQueryOrder(
	fn ImmediateDeliveryQueryOrderHandler,
) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ImmediateDeliveryEventQueryOrder,
		func() eventPayload { return &ImmediateDeliveryQueryOrderEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ImmediateDeliveryQueryOrderEvent))
		},
	)
}
