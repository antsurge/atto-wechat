package miniProgram

import (
	"context"
)

// ProviderCheckBizEvent is the check_biz callback: WeChat asks whether the merchant
// account is known to the courier and how much balance it has left.
//
// It is sent when the merchant is about to place its first order, so an unknown
// BizID should be answered with 10001 rather than with an error.
//
// The Shop* and Sender* fields are only filled for EMS.
type ProviderCheckBizEvent struct {
	EventNotify

	// BizID is the merchant's customer code at the courier.
	BizID string `xml:"BizID" json:"BizID"`
	// BizPwd is the password that goes with BizID.
	BizPwd string `xml:"BizPwd" json:"BizPwd"`
	// ShopAppID is the merchant's mini program AppID.
	ShopAppID string `xml:"ShopAppID" json:"ShopAppID"`
	// ShopName is the merchant's mini program nickname (EMS only).
	ShopName string `xml:"ShopName" json:"ShopName"`
	// ShopTelphone is the merchant's contact phone (EMS only).
	ShopTelphone string `xml:"ShopTelphone" json:"ShopTelphone"`
	// ShopContact is the merchant's contact name (EMS only).
	ShopContact string `xml:"ShopContact" json:"ShopContact"`
	// ServiceName is the pre-opened service type (EMS only).
	ServiceName string `xml:"ServiceName" json:"ServiceName"`
	// SenderAddress is the merchant's shipping address (EMS only).
	SenderAddress string `xml:"SenderAddress" json:"SenderAddress"`
	// SenderProvince is the merchant's shipping province (EMS only).
	SenderProvince string `xml:"SenderProvince" json:"SenderProvince"`
	// SenderCity is the merchant's shipping city (EMS only).
	SenderCity string `xml:"SenderCity" json:"SenderCity"`
	// SenderArea is the merchant's shipping district (EMS only).
	SenderArea string `xml:"SenderArea" json:"SenderArea"`
}

// OK reports whether the payload carries the credentials to look the account up.
func (e *ProviderCheckBizEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && e.BizID != ""
}

// ProviderCheckBizReply answers the check_biz callback.
//
// Quota is what the merchant can still spend, in fen. Reporting 0 makes WeChat
// block the order with "insufficient balance", so a courier that does not track
// balances should report a large value rather than zero.
type ProviderCheckBizReply struct {
	ProviderReply

	// BizID echoes the request.
	BizID string `xml:"BizID" json:"BizID"`
	// Quota is the merchant's remaining balance.
	Quota int64 `xml:"Quota" json:"Quota"`
}

// NewReply starts a reply with the header and BizID filled in.
func (e *ProviderCheckBizEvent) NewReply() *ProviderCheckBizReply {
	reply := &ProviderCheckBizReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
		reply.BizID = e.BizID
	}

	return reply
}

// ProviderCheckBizHandler handles a check_biz callback.
type ProviderCheckBizHandler func(
	ctx context.Context, evt *ProviderCheckBizEvent,
) (*ProviderCheckBizReply, error)

// OnProviderCheckBiz registers a handler for the check_biz callback.
func (a *Application) OnProviderCheckBiz(fn ProviderCheckBizHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventCheckBiz,
		func() eventPayload { return &ProviderCheckBizEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderCheckBizEvent))
		},
	)
}
