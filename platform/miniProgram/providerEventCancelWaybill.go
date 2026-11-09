package miniProgram

import (
	"context"
)

// ProviderCancelWaybillEvent is the cancel_waybill callback: the merchant
// cancelled an order before the parcel was picked up, so the courier should void
// the waybill.
//
// The reply must repeat BizID, OrderID and WayBillID unchanged; WeChat matches the
// three against the order it is waiting on.
type ProviderCancelWaybillEvent struct {
	EventNotify

	// OrderID is the merchant's order identifier.
	OrderID string `xml:"OrderID" json:"OrderID"`
	// BizID is the merchant's customer code at the courier.
	BizID string `xml:"BizID" json:"BizID"`
	// BizPwd is the password that goes with BizID.
	BizPwd string `xml:"BizPwd" json:"BizPwd"`
	// ShopAppID is the merchant's mini program AppID.
	ShopAppID string `xml:"ShopAppID" json:"ShopAppID"`
	// WayBillID is the waybill to cancel.
	WayBillID string `xml:"WayBillID" json:"WayBillID"`
}

// OK reports whether the payload carries everything needed to cancel a waybill.
func (e *ProviderCancelWaybillEvent) OK() bool {
	return e != nil &&
		e.EventNotify.OK() &&
		e.OrderID != "" &&
		e.BizID != "" &&
		e.WayBillID != ""
}

// ProviderCancelWaybillReply answers the cancel_waybill callback.
type ProviderCancelWaybillReply struct {
	ProviderReply

	// BizID echoes the request.
	BizID string `xml:"BizID" json:"BizID"`
	// OrderID echoes the request.
	OrderID string `xml:"OrderID" json:"OrderID"`
	// WayBillID echoes the request.
	WayBillID string `xml:"WayBillID" json:"WayBillID"`
}

// NewReply starts a reply with the header and the echoed fields filled in, so the
// handler only has to set the result.
func (e *ProviderCancelWaybillEvent) NewReply() *ProviderCancelWaybillReply {
	reply := &ProviderCancelWaybillReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
		reply.BizID = e.BizID
		reply.OrderID = e.OrderID
		reply.WayBillID = e.WayBillID
	}

	return reply
}

// ProviderCancelWaybillHandler handles a cancel_waybill callback.
//
// A returned error leaves WeChat with the default "success" body, which it reads
// as a failed reply; return a reply with a non-zero ResultCode instead when the
// cancellation cannot be honoured.
type ProviderCancelWaybillHandler func(
	ctx context.Context, evt *ProviderCancelWaybillEvent,
) (*ProviderCancelWaybillReply, error)

// OnProviderCancelWaybill registers a handler for the cancel_waybill callback.
func (a *Application) OnProviderCancelWaybill(fn ProviderCancelWaybillHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventCancelWaybill,
		func() eventPayload { return &ProviderCancelWaybillEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderCancelWaybillEvent))
		},
	)
}
