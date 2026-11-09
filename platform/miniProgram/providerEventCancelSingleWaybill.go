package miniProgram

import (
	"context"
)

// ProviderCancelSingleWaybillEvent is the cancel_single_waybill callback: a
// single-parcel order was cancelled and the courier should void the waybill.
//
// WayBillID may be empty: an order can be cancelled before a waybill number was
// issued, in which case there is nothing to void and the courier only has to
// acknowledge.
type ProviderCancelSingleWaybillEvent struct {
	EventNotify

	// Token is the token the merchant passed when placing the order.
	Token string `xml:"Token" json:"Token"`
	// WXAppID is the merchant's mini program AppID.
	WXAppID string `xml:"WXAppId" json:"WXAppId"`
	// OrderID is the merchant's order identifier.
	OrderID string `xml:"OrderID" json:"OrderID"`
	// WayBillID is the waybill number, when one was already issued.
	WayBillID string `xml:"WayBillID" json:"WayBillID"`
	// CancelID is the cancellation reason id.
	CancelID int `xml:"CancelID" json:"CancelID"`
	// CancelReason describes the cancellation reason.
	CancelReason string `xml:"CancelReason" json:"CancelReason"`
}

// OK reports whether the payload carries something to locate the order with.
func (e *ProviderCancelSingleWaybillEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && (e.Token != "" || e.OrderID != "")
}

// NewReply starts a reply with the header filled in.
//
// The helper expects only the header and the result for this callback, so the
// reply is a plain ProviderReply.
func (e *ProviderCancelSingleWaybillEvent) NewReply() *ProviderReply {
	reply := &ProviderReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ProviderCancelSingleWaybillHandler handles a cancel_single_waybill callback.
type ProviderCancelSingleWaybillHandler func(
	ctx context.Context, evt *ProviderCancelSingleWaybillEvent,
) (*ProviderReply, error)

// OnProviderCancelSingleWaybill registers a handler for the
// cancel_single_waybill callback.
func (a *Application) OnProviderCancelSingleWaybill(fn ProviderCancelSingleWaybillHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventCancelSingleWaybill,
		func() eventPayload { return &ProviderCancelSingleWaybillEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderCancelSingleWaybillEvent))
		},
	)
}
