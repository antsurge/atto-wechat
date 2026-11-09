package miniProgram

import (
	"context"
)

// ProviderNotifySingleWaybillPayEvent is the notify_single_waybill_pay callback: the
// buyer paid for a single-parcel order, so the courier can pick the parcel up.
//
// The courier only acknowledges: the money already moved, and a non-zero
// ResultCode here does not reverse anything. It tells WeChat the courier failed to
// record the payment, which makes the order page show an unpaid state.
type ProviderNotifySingleWaybillPayEvent struct {
	EventNotify

	// Token is the token the merchant passed when placing the order.
	Token string `xml:"Token" json:"Token"`
	// WXAppID is the merchant's mini program AppID.
	WXAppID string `xml:"WXAppId" json:"WXAppId"`
	// OrderID is the merchant's order identifier.
	OrderID string `xml:"OrderID" json:"OrderID"`
	// WayBillID is the waybill number.
	WayBillID string `xml:"WayBillID" json:"WayBillID"`
	// Fee is the amount paid in fen.
	Fee int64 `xml:"Fee" json:"Fee"`
	// OriginalFee is the undiscounted amount in fen.
	OriginalFee int64 `xml:"OriginalFee" json:"OriginalFee"`
	// PayOrderID is WeChat's payment order number, the id to reconcile against.
	PayOrderID string `xml:"PayOrderId" json:"PayOrderId"`
	// PayFinishTime is when the payment succeeded, as a Unix timestamp.
	PayFinishTime int64 `xml:"PayFinishTime" json:"PayFinishTime"`
}

// OK reports whether the payload carries an amount and a payment id.
func (e *ProviderNotifySingleWaybillPayEvent) OK() bool {
	return e != nil &&
		e.EventNotify.OK() &&
		(e.Token != "" || e.OrderID != "") &&
		e.PayOrderID != ""
}

// NewReply starts a reply with the header filled in.
//
// The helper expects only the header and the result for this callback, so the
// reply is a plain ProviderReply.
func (e *ProviderNotifySingleWaybillPayEvent) NewReply() *ProviderReply {
	reply := &ProviderReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ProviderNotifySingleWaybillPayHandler handles a notify_single_waybill_pay
// callback.
type ProviderNotifySingleWaybillPayHandler func(
	ctx context.Context, evt *ProviderNotifySingleWaybillPayEvent,
) (*ProviderReply, error)

// OnProviderNotifySingleWaybillPay registers a handler for the
// notify_single_waybill_pay callback.
func (a *Application) OnProviderNotifySingleWaybillPay(
	fn ProviderNotifySingleWaybillPayHandler,
) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventNotifySingleWaybillPay,
		func() eventPayload { return &ProviderNotifySingleWaybillPayEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderNotifySingleWaybillPayEvent))
		},
	)
}
