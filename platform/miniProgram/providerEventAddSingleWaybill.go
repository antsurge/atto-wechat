package miniProgram

import (
	"context"
)

// ProviderAddSingleWaybillEvent is the add_single_waybill callback: a merchant
// placed a single-parcel order and the courier should create the waybill.
//
// It is the single-parcel counterpart of ProviderAddWaybillEvent: the merchant
// picked the shipping window itself and the payload carries a settlement method,
// so the courier should not reprice or requote.
type ProviderAddSingleWaybillEvent struct {
	EventNotify

	// Token identifies the order; WeChat recommends it over OrderID.
	Token string `xml:"Token" json:"Token"`
	// WXAppID is the merchant's mini program AppID.
	WXAppID string `xml:"WXAppId" json:"WXAppId"`
	// OrderID is the merchant's order identifier.
	OrderID string `xml:"OrderID" json:"OrderID"`
	// Sender is the shipper.
	Sender *ProviderAddress `xml:"Sender" json:"Sender"`
	// Receiver is the consignee.
	Receiver *ProviderAddress `xml:"Receiver" json:"Receiver"`
	// GoodDetail describes the parcel.
	GoodDetail *ProviderGoods `xml:"GoodDetail" json:"GoodDetail"`
	// Insured carries the declared value, if any.
	Insured *ProviderWaybillInsured `xml:"Insured" json:"Insured"`
	// ExpectStartTime is the start of the requested pickup window, as a Unix
	// timestamp.
	ExpectStartTime int64 `xml:"ExpectStartTime" json:"ExpectStartTime"`
	// ExpectEndTime is the end of the requested pickup window.
	ExpectEndTime int64 `xml:"ExpectEndTime" json:"ExpectEndTime"`
	// Settingment is the settlement method: 0 cash, 1 collect, 2 monthly.
	Settingment int `xml:"Settingment" json:"Settingment"`
	// CompanyID is the large-customer id, required for monthly settlement at ZTO.
	CompanyID string `xml:"CompanyID" json:"CompanyID"`
	// Account is the monthly settlement account, required for method 2.
	Account string `xml:"Account" json:"Account"`
	// AccountPwd is the monthly settlement password, required for method 2.
	AccountPwd string `xml:"AccountPwd" json:"AccountPwd"`
	// Remark is printed on the waybill.
	Remark string `xml:"Remark" json:"Remark"`
	// PickUpStartTime is the start of the pickup slot, as a Unix timestamp.
	PickUpStartTime int64 `xml:"PickUpStartTime" json:"PickUpStartTime"`
	// PickUpEndTime is the end of the pickup slot, as a Unix timestamp.
	PickUpEndTime int64 `xml:"PickUpEndTime" json:"PickUpEndTime"`
	// Scene is the mini program scene value the order came from.
	Scene int `xml:"Scene" json:"Scene"`
}

// Settlement methods the merchant can pick.
const (
	// ProviderSettlementCash is paid on the spot.
	ProviderSettlementCash = 0
	// ProviderSettlementCollect is paid by the consignee on delivery.
	ProviderSettlementCollect = 1
	// ProviderSettlementMonthly is billed to a monthly account.
	ProviderSettlementMonthly = 2
	// ProviderSettlementPayScore is paid through WeChat Pay Score.
	ProviderSettlementPayScore = 3
)

// OK reports whether the payload is complete enough to create a waybill.
func (e *ProviderAddSingleWaybillEvent) OK() bool {
	return e != nil &&
		e.EventNotify.OK() &&
		(e.Token != "" || e.OrderID != "") &&
		e.Sender != nil &&
		e.Receiver != nil
}

// ProviderAddSingleWaybillReply answers the add_single_waybill callback.
type ProviderAddSingleWaybillReply struct {
	ProviderReply

	// WayBillID is the waybill number, once one exists.
	WayBillID string `xml:"WayBillID" json:"WayBillID"`
	// EstimatedDeliveryTime is the expected delivery date, in yyyy-mm-dd.
	EstimatedDeliveryTime string `xml:"EstimatedDeliveryTime" json:"EstimatedDeliveryTime"`
	// JumpPath is the page to open after the order is placed; the AppID has to be
	// registered with WeChat beforehand.
	JumpPath string `xml:"JumpPath" json:"JumpPath"`
	// PickCode is the pickup code shown to the user.
	PickCode string `xml:"PickCode" json:"PickCode"`
}

// NewReply starts a reply with the header filled in.
func (e *ProviderAddSingleWaybillEvent) NewReply() *ProviderAddSingleWaybillReply {
	reply := &ProviderAddSingleWaybillReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
	}

	return reply
}

// ProviderAddSingleWaybillHandler handles an add_single_waybill callback.
type ProviderAddSingleWaybillHandler func(
	ctx context.Context, evt *ProviderAddSingleWaybillEvent,
) (*ProviderAddSingleWaybillReply, error)

// OnProviderAddSingleWaybill registers a handler for the add_single_waybill
// callback.
func (a *Application) OnProviderAddSingleWaybill(fn ProviderAddSingleWaybillHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventAddSingleWaybill,
		func() eventPayload { return &ProviderAddSingleWaybillEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderAddSingleWaybillEvent))
		},
	)
}
