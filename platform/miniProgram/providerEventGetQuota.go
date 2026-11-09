package miniProgram

import (
	"context"
)

// ProviderGetQuotaEvent is the get_quota callback: WeChat asks how much balance the
// merchant has left before letting it place an order.
type ProviderGetQuotaEvent struct {
	EventNotify

	// BizID is the merchant's customer code at the courier.
	BizID string `xml:"BizID" json:"BizID"`
	// BizPwd is the password that goes with BizID.
	BizPwd string `xml:"BizPwd" json:"BizPwd"`
	// ShopAppID is the merchant's mini program AppID.
	ShopAppID string `xml:"ShopAppID" json:"ShopAppID"`
}

// OK reports whether the payload carries the credentials to look the account up.
func (e *ProviderGetQuotaEvent) OK() bool {
	return e != nil && e.EventNotify.OK() && e.BizID != ""
}

// ProviderGetQuotaReply answers the get_quota callback.
//
// Quota is in fen. Zero means "no balance" and makes WeChat refuse the order, so
// do not leave it unset when the account is fine.
type ProviderGetQuotaReply struct {
	ProviderReply

	// BizID echoes the request.
	BizID string `xml:"BizID" json:"BizID"`
	// Quota is the merchant's remaining balance.
	Quota int64 `xml:"Quota" json:"Quota"`
}

// NewReply starts a reply with the header and BizID filled in.
func (e *ProviderGetQuotaEvent) NewReply() *ProviderGetQuotaReply {
	reply := &ProviderGetQuotaReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
		reply.BizID = e.BizID
	}

	return reply
}

// ProviderGetQuotaHandler handles a get_quota callback.
type ProviderGetQuotaHandler func(
	ctx context.Context, evt *ProviderGetQuotaEvent,
) (*ProviderGetQuotaReply, error)

// OnProviderGetQuota registers a handler for the get_quota callback.
func (a *Application) OnProviderGetQuota(fn ProviderGetQuotaHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventGetQuota,
		func() eventPayload { return &ProviderGetQuotaEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderGetQuotaEvent))
		},
	)
}
