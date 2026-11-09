package miniProgram

import (
	"context"
)

// ImmediateDeliveryPreCancelOrderEvent is the transport_precancel_order callback:
// the merchant asks what cancelling would cost before it decides.
//
// It is optional for the courier to implement. Answering it lets the merchant show
// the penalty up front instead of surprising the user after the fact.
type ImmediateDeliveryPreCancelOrderEvent struct {
	ImmediateDeliveryEvent

	// CancelReasonID is one of the ImmediateDeliveryCancelReason* constants.
	CancelReasonID int `xml:"cancel_reason_id" json:"cancel_reason_id"`
	// CancelReason is the free-text reason, only filled for reason 6.
	CancelReason string `xml:"cancel_reason" json:"cancel_reason"`
}

// NewReply returns a reply with the header already filled in.
func (e *ImmediateDeliveryPreCancelOrderEvent) NewReply() *ImmediateDeliveryReply {
	reply := &ImmediateDeliveryReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ImmediateDeliveryPreCancelOrderHandler handles a transport_precancel_order
// callback.
type ImmediateDeliveryPreCancelOrderHandler func(
	ctx context.Context, evt *ImmediateDeliveryPreCancelOrderEvent,
) (*ImmediateDeliveryReply, error)

// OnImmediateDeliveryPreCancelOrder registers a handler for the
// transport_precancel_order callback.
func (a *Application) OnImmediateDeliveryPreCancelOrder(
	fn ImmediateDeliveryPreCancelOrderHandler,
) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ImmediateDeliveryEventPreCancelOrder,
		func() eventPayload { return &ImmediateDeliveryPreCancelOrderEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ImmediateDeliveryPreCancelOrderEvent))
		},
	)
}
