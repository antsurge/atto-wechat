package miniProgram

import (
	"context"
)

// Cancellation reason ids the merchant can send, as documented.
const (
	// ImmediateDeliveryCancelReasonNotNeeded is 1: the parcel is no longer needed.
	ImmediateDeliveryCancelReasonNotNeeded = 1
	// ImmediateDeliveryCancelReasonPrice is 2: the price is not acceptable.
	ImmediateDeliveryCancelReasonPrice = 2
	// ImmediateDeliveryCancelReasonWrongInfo is 3: the order is wrong, it will be
	// placed again.
	ImmediateDeliveryCancelReasonWrongInfo = 3
	// ImmediateDeliveryCancelReasonSlowPickup is 4: the courier did not collect in
	// time.
	ImmediateDeliveryCancelReasonSlowPickup = 4
	// ImmediateDeliveryCancelReasonSlowDelivery is 5: the courier did not deliver in
	// time.
	ImmediateDeliveryCancelReasonSlowDelivery = 5
	// ImmediateDeliveryCancelReasonOther is 6: anything else; CancelReason is only
	// filled for this one.
	ImmediateDeliveryCancelReasonOther = 6
)

// ImmediateDeliveryCancelOrderEvent is the transport_cancel_order callback: the
// merchant cancelled a delivery order.
//
// The reply must state the penalty, so a courier that charges nothing still has to
// set DeductFee explicitly.
type ImmediateDeliveryCancelOrderEvent struct {
	ImmediateDeliveryEvent

	// CancelReasonID is one of the ImmediateDeliveryCancelReason* constants.
	CancelReasonID int `xml:"cancel_reason_id" json:"cancel_reason_id"`
	// CancelReason is the free-text reason, only filled for reason 6.
	CancelReason string `xml:"cancel_reason" json:"cancel_reason"`
}

// NewReply returns a reply with the header already filled in.
func (e *ImmediateDeliveryCancelOrderEvent) NewReply() *ImmediateDeliveryReply {
	reply := &ImmediateDeliveryReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// FreeReply answers that cancelling costs nothing, which is the common case for an
// order the courier has not accepted yet.
func (e *ImmediateDeliveryCancelOrderEvent) FreeReply(desc string) *ImmediateDeliveryReply {
	reply := e.NewReply()
	reply.Succeed()
	reply.Desc = desc

	return reply
}

// ImmediateDeliveryCancelOrderHandler handles a transport_cancel_order callback.
type ImmediateDeliveryCancelOrderHandler func(
	ctx context.Context, evt *ImmediateDeliveryCancelOrderEvent,
) (*ImmediateDeliveryReply, error)

// OnImmediateDeliveryCancelOrder registers a handler for the
// transport_cancel_order callback.
func (a *Application) OnImmediateDeliveryCancelOrder(
	fn ImmediateDeliveryCancelOrderHandler,
) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ImmediateDeliveryEventCancelOrder,
		func() eventPayload { return &ImmediateDeliveryCancelOrderEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ImmediateDeliveryCancelOrderEvent))
		},
	)
}
