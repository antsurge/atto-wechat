package miniProgram

import (
	"context"
)

// ProviderQuerySingleWaybillFeeEvent is the query_single_waybill_fee callback:
// WeChat asks how much shipping between two addresses costs, so the merchant can
// show a price before the user pays.
//
// It is the only callback that answers a price for an order that does not exist
// yet, so there is no token or order id in the payload.
type ProviderQuerySingleWaybillFeeEvent struct {
	EventNotify

	// Sender is the shipper.
	Sender *ProviderAddress `xml:"Sender" json:"Sender"`
	// Receiver is the consignee.
	Receiver *ProviderAddress `xml:"Receiver" json:"Receiver"`
	// GoodDetail describes the parcel.
	GoodDetail *ProviderGoods `xml:"GoodDetail" json:"GoodDetail"`
	// Scene is 1 for a consumer order, 2 for a B-side return, 3 for B-side
	// shipping.
	Scene int `xml:"Scene" json:"Scene"`
}

// Scenes the fee query can come from.
const (
	// ProviderFeeSceneConsumer is a consumer order.
	ProviderFeeSceneConsumer = 1
	// ProviderFeeSceneBusinessReturn is a business return.
	ProviderFeeSceneBusinessReturn = 2
	// ProviderFeeSceneBusinessShip is a business shipment.
	ProviderFeeSceneBusinessShip = 3
)

// OK reports whether both addresses are present.
func (e *ProviderQuerySingleWaybillFeeEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && e.Sender != nil && e.Receiver != nil
}

// ProviderQuerySingleWaybillFeeReply answers the query_single_waybill_fee
// callback.
//
// Fee is in fen. WeChat uses it as the price shown to the user, so it has to
// include everything the courier will charge.
type ProviderQuerySingleWaybillFeeReply struct {
	ProviderReply

	// Fee is the shipping price in fen.
	Fee int64 `xml:"Fee" json:"Fee"`
	// OriginalFee is the undiscounted price in fen.
	OriginalFee int64 `xml:"OriginalFee" json:"OriginalFee"`
}

// NewReply starts a reply with the header filled in.
func (e *ProviderQuerySingleWaybillFeeEvent) NewReply() *ProviderQuerySingleWaybillFeeReply {
	reply := &ProviderQuerySingleWaybillFeeReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ProviderQuerySingleWaybillFeeHandler handles a query_single_waybill_fee
// callback.
type ProviderQuerySingleWaybillFeeHandler func(
	ctx context.Context, evt *ProviderQuerySingleWaybillFeeEvent,
) (*ProviderQuerySingleWaybillFeeReply, error)

// OnProviderQuerySingleWaybillFee registers a handler for the
// query_single_waybill_fee callback.
func (a *Application) OnProviderQuerySingleWaybillFee(fn ProviderQuerySingleWaybillFeeHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventQuerySingleWaybillFee,
		func() eventPayload { return &ProviderQuerySingleWaybillFeeEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderQuerySingleWaybillFeeEvent))
		},
	)
}
