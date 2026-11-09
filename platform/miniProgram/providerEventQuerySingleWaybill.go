package miniProgram

import (
	"context"
)

// ProviderQuerySingleWaybillEvent is the query_single_waybill callback: WeChat asks
// the courier for everything it knows about a waybill -- addresses, parcel,
// tracking, price and payment state.
//
// The reply is what the user sees on the order page, so an incomplete answer shows
// up as missing information rather than as an error.
type ProviderQuerySingleWaybillEvent struct {
	EventNotify

	// Token is the token the merchant passed when placing the order.
	Token string `xml:"Token" json:"Token"`
	// WXAppID is the merchant's mini program AppID.
	WXAppID string `xml:"WXAppId" json:"WXAppId"`
	// WaybillID is the waybill number, when the courier already reported one.
	WaybillID string `xml:"WaybillId" json:"WaybillId"`
	// OrderID is the merchant's order identifier.
	OrderID string `xml:"OrderID" json:"OrderID"`
}

// OK reports whether the payload carries something to locate the order with.
func (e *ProviderQuerySingleWaybillEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && (e.Token != "" || e.OrderID != "")
}

// Payment states a waybill can be in.
const (
	// ProviderPayResultUnpaid means nobody paid yet.
	ProviderPayResultUnpaid = 0
	// ProviderPayResultOnline means the buyer paid in WeChat.
	ProviderPayResultOnline = 1
	// ProviderPayResultOffline means it was paid outside WeChat.
	ProviderPayResultOffline = 2
)

// ProviderQuerySingleWaybillReply answers the query_single_waybill callback.
type ProviderQuerySingleWaybillReply struct {
	ProviderReply

	// WayBillID is the waybill number, required once one exists.
	WayBillID string `xml:"WayBillID" json:"WayBillID"`
	// Sender is the shipper.
	Sender *ProviderAddress `xml:"Sender" json:"Sender"`
	// Receiver is the consignee.
	Receiver *ProviderAddress `xml:"Receiver" json:"Receiver"`
	// GoodDetail describes the parcel.
	GoodDetail *ProviderGoods `xml:"GoodDetail" json:"GoodDetail"`
	// Insured carries the declared value, if any.
	Insured *ProviderWaybillInsured `xml:"Insured" json:"Insured"`
	// PathInfo is the tracking history.
	PathInfo *ProviderPathInfo `xml:"PathInfo" json:"PathInfo"`
	// Fee is what the waybill costs, in fen; required once the price is known.
	Fee int64 `xml:"Fee" json:"Fee"`
	// OriginalFee is the undiscounted price in fen.
	OriginalFee int64 `xml:"OriginalFee" json:"OriginalFee"`
	// OnlinePay is 1 when the courier accepts payment through WeChat.
	OnlinePay int `xml:"OnlinePay" json:"OnlinePay"`
	// PayResult is the payment state, one of the ProviderPayResult* constants.
	PayResult int `xml:"PayResult" json:"PayResult"`
}

// NewReply starts a reply with the header filled in.
func (e *ProviderQuerySingleWaybillEvent) NewReply() *ProviderQuerySingleWaybillReply {
	reply := &ProviderQuerySingleWaybillReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
		reply.WayBillID = e.WaybillID
	}

	return reply
}

// ProviderQuerySingleWaybillHandler handles a query_single_waybill callback.
type ProviderQuerySingleWaybillHandler func(
	ctx context.Context, evt *ProviderQuerySingleWaybillEvent,
) (*ProviderQuerySingleWaybillReply, error)

// OnProviderQuerySingleWaybill registers a handler for the query_single_waybill
// callback.
func (a *Application) OnProviderQuerySingleWaybill(fn ProviderQuerySingleWaybillHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventQuerySingleWaybill,
		func() eventPayload { return &ProviderQuerySingleWaybillEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderQuerySingleWaybillEvent))
		},
	)
}
