package miniProgram

import (
	"context"
	"strings"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
)

// errWaybillDataPairs reports a malformed call to ProviderWaybillData.
var errWaybillDataPairs = cerrors.New(
	cerrors.CodeInvalidArgument,
	"miniprogram: waybill data takes non-empty key/value pairs",
)

// ProviderAddWaybillEvent is the add_waybill callback: WeChat asks the courier to
// create a waybill for an order.
//
// The fields come from the merchant and are neither guaranteed complete nor
// correct, so the courier has to validate them. For the same BizID and OrderID the
// courier must return the same waybill ID: the push can be repeated when the
// network is unstable, and a second ID would split one order into two parcels.
type ProviderAddWaybillEvent struct {
	EventNotify

	// Token identifies the order in later calls such as ProviderUpdatePath.
	Token string `xml:"Token" json:"Token"`
	// OrderID is the merchant's order identifier.
	OrderID string `xml:"OrderID" json:"OrderID"`
	// BizID is the merchant's customer code at the courier.
	BizID string `xml:"BizID" json:"BizID"`
	// BizPwd is the password that goes with BizID.
	BizPwd string `xml:"BizPwd" json:"BizPwd"`
	// ShopAppID is the merchant's mini program AppID.
	ShopAppID string `xml:"ShopAppID" json:"ShopAppID"`
	// WayBillID is the waybill number, or "0" when WeChat has none and the
	// courier has to generate one.
	WayBillID string `xml:"WayBillID" json:"WayBillID"`
	// Remark is printed on the waybill, e.g. "易碎物品".
	Remark string `xml:"Remark" json:"Remark"`
	// Sender is the shipper.
	Sender *ProviderWaybillPerson `xml:"Sender" json:"Sender"`
	// Receiver is the consignee.
	Receiver *ProviderWaybillPerson `xml:"Receiver" json:"Receiver"`
	// Cargo describes the parcel.
	Cargo *ProviderWaybillCargo `xml:"Cargo" json:"Cargo"`
	// Insured carries the declared value, if any.
	Insured *ProviderWaybillInsured `xml:"Insured" json:"Insured"`
	// Service is the shipping service the merchant picked.
	Service *ProviderWaybillService `xml:"Service" json:"Service"`
}

// ProviderWaybillPerson is a shipper or consignee as the courier sees it.
type ProviderWaybillPerson struct {
	// Name is the person's name.
	Name string `xml:"Name" json:"Name"`
	// Tel is the landline number.
	Tel string `xml:"Tel" json:"Tel"`
	// Mobile is the mobile number.
	Mobile string `xml:"Mobile" json:"Mobile"`
	// Company is the company name.
	Company string `xml:"Company" json:"Company"`
	// PostCode is the postal code.
	PostCode string `xml:"PostCode" json:"PostCode"`
	// Country defaults to 中国.
	Country string `xml:"Country" json:"Country"`
	// Province is the province, e.g. 广东省.
	Province string `xml:"Province" json:"Province"`
	// City is the city, e.g. 广州市.
	City string `xml:"City" json:"City"`
	// Area is the district, e.g. 海珠区.
	Area string `xml:"Area" json:"Area"`
	// Address is the street address.
	Address string `xml:"Address" json:"Address"`
}

// ProviderWaybillCargo describes the parcel's size and weight.
type ProviderWaybillCargo struct {
	// Weight is the total weight in kilograms.
	Weight float64 `xml:"Weight" json:"Weight"`
	// SpaceX is the length in centimetres.
	SpaceX float64 `xml:"Space_X" json:"Space_X"`
	// SpaceY is the width in centimetres.
	SpaceY float64 `xml:"Space_Y" json:"Space_Y"`
	// SpaceZ is the height in centimetres.
	SpaceZ float64 `xml:"Space_Z" json:"Space_Z"`
	// Count is the number of parcels, usually 1.
	Count int `xml:"Count" json:"Count"`
}

// ProviderWaybillInsured is the declared value of the parcel.
type ProviderWaybillInsured struct {
	// UseInsured is 1 when the parcel is insured, 0 otherwise.
	UseInsured int `xml:"UseInsured" json:"UseInsured"`
	// InsuredValue is the declared value in fen, e.g. 10000 for 100 yuan.
	InsuredValue int64 `xml:"InsuredValue" json:"InsuredValue"`
}

// ProviderWaybillService is the shipping service the merchant picked.
type ProviderWaybillService struct {
	// ServiceType is the service id listed in the courier's capabilities.
	ServiceType int `xml:"ServiceType" json:"ServiceType"`
	// ServiceName is the service name.
	ServiceName string `xml:"ServiceName" json:"ServiceName"`
}

// OK reports whether the payload is complete enough to create a waybill.
func (e *ProviderAddWaybillEvent) OK() bool {
	return e != nil &&
		e.EventNotify.OK() &&
		e.Token != "" &&
		e.OrderID != "" &&
		e.BizID != "" &&
		e.Sender != nil &&
		e.Receiver != nil
}

// ProviderAddWaybillReply answers the add_waybill callback.
type ProviderAddWaybillReply struct {
	ProviderReply

	// Token echoes the request; WeChat uses it to tie later calls to the order.
	Token string `xml:"Token" json:"Token"`
	// OrderID echoes the request.
	OrderID string `xml:"OrderID" json:"OrderID"`
	// BizID echoes the request.
	BizID string `xml:"BizID" json:"BizID"`
	// WayBillID is the waybill number the courier assigned.
	WayBillID string `xml:"WayBillID" json:"WayBillID"`
	// WaybillData carries the sorting fields printed on the waybill, built with
	// ProviderWaybillData.
	WaybillData string `xml:"WaybillData" json:"WaybillData"`
}

// NewReply starts a reply with the header and the echoed fields filled in.
//
// WayBillID is copied from the request, so a courier that generates its own number
// has to overwrite it.
func (e *ProviderAddWaybillEvent) NewReply() *ProviderAddWaybillReply {
	reply := &ProviderAddWaybillReply{}
	if e != nil {
		reply.Echo(&e.EventNotify)
		reply.Token = e.Token
		reply.OrderID = e.OrderID
		reply.BizID = e.BizID
		reply.WayBillID = e.WayBillID
	}

	return reply
}

// ProviderWaybillData builds the WaybillData string from key/value pairs.
//
// The format is `##key##value##key##value##`: the key is a placeholder in the
// courier's waybill template and the value is what gets printed there, e.g.
// ProviderWaybillData("ZTO_bagAddr", "广州", "ZTO_mark", "888-666-666").
func ProviderWaybillData(pairs ...string) (string, error) {
	if len(pairs)%2 != 0 {
		return "", errWaybillDataPairs
	}

	var b strings.Builder

	for i := 0; i < len(pairs); i += 2 {
		if pairs[i] == "" {
			return "", errWaybillDataPairs
		}

		b.WriteString("##")
		b.WriteString(pairs[i])
		b.WriteString("##")
		b.WriteString(pairs[i+1])
	}

	b.WriteString("##")

	return b.String(), nil
}

// ProviderAddWaybillHandler handles an add_waybill callback.
type ProviderAddWaybillHandler func(
	ctx context.Context, evt *ProviderAddWaybillEvent,
) (*ProviderAddWaybillReply, error)

// OnProviderAddWaybill registers a handler for the add_waybill callback.
//
// This is the callback that turns a merchant order into a real waybill: it is the
// one handler a courier must implement for the logistics helper to work.
func (a *Application) OnProviderAddWaybill(fn ProviderAddWaybillHandler) *Application {
	if fn == nil {
		return a
	}

	return a.onProviderEvent(
		ProviderEventAddWaybill,
		func() eventPayload { return &ProviderAddWaybillEvent{} },
		func(ctx context.Context, evt eventPayload) (any, error) {
			return fn(ctx, evt.(*ProviderAddWaybillEvent))
		},
	)
}
