package miniProgram

import (
	"context"
)

// ProviderRemindWaybillEvent is the remind_waybill callback: the user asked where
// the parcel is, so WeChat gives the courier a chance to answer with a waybill
// number and an expected delivery date.
//
// It is sent before a waybill number exists, which is why both reply fields are
// optional: a courier that has not picked the parcel up yet can answer with 0 and
// no waybill, and send the number later through ProviderUpdatePath.
type ProviderRemindWaybillEvent struct {
	EventNotify

	// Token identifies the order; WeChat recommends it over OrderID.
	Token string `xml:"Token" json:"Token"`
	// WXAppID is the merchant's mini program AppID.
	WXAppID string `xml:"WXAppId" json:"WXAppId"`
	// OrderID is the merchant's order identifier.
	OrderID string `xml:"OrderID" json:"OrderID"`
}

// OK reports whether the payload carries something to locate the order with.
func (e *ProviderRemindWaybillEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && (e.Token != "" || e.OrderID != "")
}

// ProviderRemindWaybillReply answers the remind_waybill callback.
type ProviderRemindWaybillReply struct {
	ProviderReply

	// WayBillID is the waybill number, once one exists.
	WayBillID string `xml:"WayBillID" json:"WayBillID"`
	// EstimatedDeliveryTime is the expected delivery date, in yyyy-mm-dd.
	EstimatedDeliveryTime string `xml:"EstimatedDeliveryTime" json:"EstimatedDeliveryTime"`
}

// NewReply starts a reply with the header filled in.
func (e *ProviderRemindWaybillEvent) NewReply() *ProviderRemindWaybillReply {
	reply := &ProviderRemindWaybillReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ProviderRemindWaybillHandler handles a remind_waybill callback.
type ProviderRemindWaybillHandler func(
	ctx context.Context, evt *ProviderRemindWaybillEvent,
) (*ProviderRemindWaybillReply, error)

// OnProviderRemindWaybill registers a handler for the remind_waybill callback.
func (a *Application) OnProviderRemindWaybill(fn ProviderRemindWaybillHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventRemindWaybill,
		func() eventPayload { return &ProviderRemindWaybillEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderRemindWaybillEvent))
		},
	)
}
