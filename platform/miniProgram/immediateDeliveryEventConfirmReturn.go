package miniProgram

import (
	"context"
)

// ImmediateDeliveryConfirmReturnEvent is the transport_confirm_return_to_biz
// callback: the merchant confirms it took back a parcel that could not be
// delivered.
//
// Only 达达, 闪送 and 人人快送 support it, and only for parcels that already went
// through state 304 or 305.
type ImmediateDeliveryConfirmReturnEvent struct {
	ImmediateDeliveryEvent
}

// NewReply returns a reply with the header already filled in.
func (e *ImmediateDeliveryConfirmReturnEvent) NewReply() *ImmediateDeliveryReply {
	reply := &ImmediateDeliveryReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ImmediateDeliveryConfirmReturnHandler handles a
// transport_confirm_return_to_biz callback.
type ImmediateDeliveryConfirmReturnHandler func(
	ctx context.Context, evt *ImmediateDeliveryConfirmReturnEvent,
) (*ImmediateDeliveryReply, error)

// OnImmediateDeliveryConfirmReturn registers a handler for the
// transport_confirm_return_to_biz callback.
func (a *Application) OnImmediateDeliveryConfirmReturn(
	fn ImmediateDeliveryConfirmReturnHandler,
) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ImmediateDeliveryEventConfirmReturn,
		func() eventPayload { return &ImmediateDeliveryConfirmReturnEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ImmediateDeliveryConfirmReturnEvent))
		},
	)
}
