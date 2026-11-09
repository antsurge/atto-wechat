package miniProgram

import (
	"encoding/xml"

	"github.com/antsurge/atto-wechat/core/codec"
)

// 即时配送（Immediate Delivery）的事件通知。
//
// 和物流助手一样是一问一答：配送公司把报文推过来，必须回一个带 resultcode 的报文，
// 回 "success" 会被判为失败并重试。所以这些事件的处理函数返回回复结构体，由框架用
// ReplyWith 写回；请求是 XML 还是 JSON，回复就用同样的格式。
//
// 和物流助手的差别只在字段名：这里回复头是 codec.CDATA 包裹的 ToUserName /
// FromUserName，错误码是小写的 resultcode / resultmsg，而不是 ResultCode / ResultMsg。
//
// 这里没有 Code2Session 那样的 XxxErrCodeMap：事件是被动接收的，报文里没有
// errcode / errmsg 返回头，解析失败只可能是报文不完整、不是事件、事件名对不上，
// 这些都用 core/errors 的语义码表达。真正需要查表的是 order_status（配送状态），
// 所以下面给出了 ImmediateDeliveryStatusMap 和 ImmediateDeliveryStatusMsg，用法和
// Code2SessionErrMsg 一致。

// Event names the immediate-delivery callbacks carry in the Event field.
//
// WeChat documents all of them as case-insensitive; the dispatcher lowercases the
// route, so the constants are lowercase too.
const (
	// ImmediateDeliveryEventOrderStatus is "update_waybill_status": the courier
	// updated a delivery order's status. This one is pushed to the merchant (小程序方).
	ImmediateDeliveryEventOrderStatus = "update_waybill_status"
	// ImmediateDeliveryEventMockUpdateOrder is "mock_update_order_status": the
	// merchant simulates a status change while testing a courier integration.
	ImmediateDeliveryEventMockUpdateOrder = "mock_update_order_status"
	// ImmediateDeliveryEventCancelOrder is "transport_cancel_order": the merchant
	// cancelled a delivery order.
	ImmediateDeliveryEventCancelOrder = "transport_cancel_order"
	// ImmediateDeliveryEventPreCancelOrder is "transport_precancel_order": the
	// merchant asks what cancelling would cost before it cancels.
	ImmediateDeliveryEventPreCancelOrder = "transport_precancel_order"
	// ImmediateDeliveryEventConfirmReturn is "transport_confirm_return_to_biz": the
	// merchant confirms it received a parcel that could not be delivered.
	ImmediateDeliveryEventConfirmReturn = "transport_confirm_return_to_biz"
	// ImmediateDeliveryEventQueryOrder is "transport_query_order_status": the
	// merchant asks for a delivery order's current status.
	ImmediateDeliveryEventQueryOrder = "transport_query_order_status"
)

// Delivery order states, as documented for order_status.
const (
	// ImmediateDeliveryStatusWaitingAssign is 101: accepted, no courier yet.
	ImmediateDeliveryStatusWaitingAssign = 101
	// ImmediateDeliveryStatusAssigned is 102: a courier took the order.
	ImmediateDeliveryStatusAssigned = 102
	// ImmediateDeliveryStatusShopCancelled is 103: the merchant cancelled before
	// pickup. Terminal.
	ImmediateDeliveryStatusShopCancelled = 103
	// ImmediateDeliveryStatusPicking is 201: the courier arrived and is collecting.
	ImmediateDeliveryStatusPicking = 201
	// ImmediateDeliveryStatusPicked is 202: the parcel is collected.
	ImmediateDeliveryStatusPicked = 202
	// ImmediateDeliveryStatusPickFailedShopCancelled is 203: pickup failed and the
	// merchant cancelled. Terminal.
	ImmediateDeliveryStatusPickFailedShopCancelled = 203
	// ImmediateDeliveryStatusPickFailedCourierCancelled is 204: pickup failed
	// because of the courier. Terminal.
	ImmediateDeliveryStatusPickFailedCourierCancelled = 204
	// ImmediateDeliveryStatusPickFailedShopReason is 205: pickup failed because of
	// the merchant. Terminal.
	ImmediateDeliveryStatusPickFailedShopReason = 205
	// ImmediateDeliveryStatusDelivering is 301: the courier is on the way.
	ImmediateDeliveryStatusDelivering = 301
	// ImmediateDeliveryStatusDelivered is 302: delivered. Terminal, and the only
	// successful terminal state.
	ImmediateDeliveryStatusDelivered = 302
	// ImmediateDeliveryStatusReturningShopCancelled is 303: the merchant cancelled
	// while the parcel was in transit; it is on its way back.
	ImmediateDeliveryStatusReturningShopCancelled = 303
	// ImmediateDeliveryStatusReturningUnreachable is 304: the recipient could not be
	// reached; the parcel is on its way back.
	ImmediateDeliveryStatusReturningUnreachable = 304
	// ImmediateDeliveryStatusReturningRejected is 305: the recipient refused the
	// parcel; it is on its way back.
	ImmediateDeliveryStatusReturningRejected = 305
	// ImmediateDeliveryStatusReturned is 401: the parcel is back with the merchant.
	// Terminal.
	ImmediateDeliveryStatusReturned = 401
	// ImmediateDeliveryStatusCancelledByCarrier is 501: the courier's own system
	// cancelled the order. Terminal.
	ImmediateDeliveryStatusCancelledByCarrier = 501
	// ImmediateDeliveryStatusCancelledForceMajeure is 502: cancelled for weather,
	// road closure and the like. Terminal.
	ImmediateDeliveryStatusCancelledForceMajeure = 502
)

// ImmediateDeliveryStatusMap holds the order_status descriptions, keyed by the
// value carried in the payload.
//
// It is exported so callers can render a human-readable state without repeating
// the table themselves, the same way Code2SessionErrCodeMap works.
var ImmediateDeliveryStatusMap = map[int]string{
	ImmediateDeliveryStatusWaitingAssign:              "等待分配骑手",
	ImmediateDeliveryStatusAssigned:                   "分配骑手成功",
	ImmediateDeliveryStatusShopCancelled:              "商家取消订单",
	ImmediateDeliveryStatusPicking:                    "骑手到店开始取货",
	ImmediateDeliveryStatusPicked:                     "取货成功",
	ImmediateDeliveryStatusPickFailedShopCancelled:    "取货失败，商家取消订单",
	ImmediateDeliveryStatusPickFailedCourierCancelled: "取货失败，骑手取消订单",
	ImmediateDeliveryStatusPickFailedShopReason:       "取货失败，商家原因取消订单",
	ImmediateDeliveryStatusDelivering:                 "配送中",
	ImmediateDeliveryStatusDelivered:                  "配送成功",
	ImmediateDeliveryStatusReturningShopCancelled:     "配送中商家取消，物品返还商家",
	ImmediateDeliveryStatusReturningUnreachable:       "无法联系收货人，物品返还商家",
	ImmediateDeliveryStatusReturningRejected:          "收货人拒收，物品返还商家",
	ImmediateDeliveryStatusReturned:                   "物品返还商家成功",
	ImmediateDeliveryStatusCancelledByCarrier:         "运力系统原因取消",
	ImmediateDeliveryStatusCancelledForceMajeure:      "不可抗拒因素取消",
}

// ImmediateDeliveryStatusMsg returns the description of an order_status value.
//
// An undocumented value returns "" rather than a made-up label: the caller can
// then decide to log the raw number or reject the payload.
func ImmediateDeliveryStatusMsg(status int) string {
	return ImmediateDeliveryStatusMap[status]
}

// ImmediateDeliveryStatusTerminal reports whether the order is finished.
//
// The documented terminal states are 302 (delivered) plus every failure state;
// the returning states (303/304/305) are not terminal yet because the parcel is
// still moving.
func ImmediateDeliveryStatusTerminal(status int) bool {
	switch status {
	case ImmediateDeliveryStatusDelivered,
		ImmediateDeliveryStatusShopCancelled,
		ImmediateDeliveryStatusPickFailedShopCancelled,
		ImmediateDeliveryStatusPickFailedCourierCancelled,
		ImmediateDeliveryStatusPickFailedShopReason,
		ImmediateDeliveryStatusReturned,
		ImmediateDeliveryStatusCancelledByCarrier,
		ImmediateDeliveryStatusCancelledForceMajeure:
		return true
	}

	return false
}

// ImmediateDeliveryStatusSucceeded reports whether the order ended well.
//
// 302 is the only successful terminal state; every other terminal state is a
// failure.
func ImmediateDeliveryStatusSucceeded(status int) bool {
	return status == ImmediateDeliveryStatusDelivered
}

// ImmediateDeliveryAgent is the courier assigned to an order.
type ImmediateDeliveryAgent struct {
	// Name is the rider's name.
	Name string `xml:"name" json:"name"`
	// Phone is the rider's number.
	Phone string `xml:"phone" json:"phone"`
	// ReachTime is the estimated arrival, as a Unix timestamp. It is only carried
	// while the order is in transit.
	ReachTime int64 `xml:"reach_time" json:"reach_time"`
}

// ImmediateDeliveryReply is the header every immediate-delivery reply echoes.
//
// The field names are lowercase resultcode / resultmsg, unlike the logistics
// helper's ResultCode / ResultMsg; a reply with the wrong casing is rejected.
type ImmediateDeliveryReply struct {
	// XMLName makes the document start with <xml>.
	XMLName xml.Name `xml:"xml" json:"-"`

	// ToUserName is the request's FromUserName.
	ToUserName codec.CDATA `xml:"ToUserName" json:"ToUserName"`
	// FromUserName is the request's ToUserName, the courier mini program.
	FromUserName codec.CDATA `xml:"FromUserName" json:"FromUserName"`
	// CreateTime echoes the request's CreateTime.
	CreateTime int64 `xml:"CreateTime" json:"CreateTime"`
	// MsgType is always "event".
	MsgType string `xml:"MsgType" json:"MsgType"`
	// Event echoes the request's event name.
	Event string `xml:"Event" json:"Event"`
	// ResultCode is 0 when the callback was handled.
	ResultCode int `xml:"resultcode" json:"resultcode"`
	// ResultMsg explains a non-zero ResultCode.
	ResultMsg string `xml:"resultmsg" json:"resultmsg"`

	// DeductFee is the cancellation penalty in yuan, for the cancel and
	// pre-cancel replies only. Cancelling for free is a meaningful answer, so 0
	// is sent when Fee applies.
	DeductFee float64 `xml:"deduct_fee,omitempty" json:"deduct_fee,omitempty"`
	// Desc explains DeductFee.
	Desc string `xml:"desc,omitempty" json:"desc,omitempty"`

	// OrderStatus, ActionMsg and WaybillID answer a status query and are empty in
	// every other reply.
	OrderStatus int    `xml:"order_status,omitempty" json:"order_status,omitempty"`
	ActionMsg   string `xml:"action_msg,omitempty" json:"action_msg,omitempty"`
	WaybillID   string `xml:"waybill_id,omitempty" json:"waybill_id,omitempty"`
}

// Echo fills the reply header from the request.
func (r *ImmediateDeliveryReply) Echo(req *EventNotify) {
	if r == nil || req == nil {
		return
	}

	r.ToUserName = codec.CDATA(req.FromUserName)
	r.FromUserName = codec.CDATA(req.ToUserName)
	r.CreateTime = req.CreateTime
	r.MsgType = EventNotifyMsgType
	r.Event = req.Event
}

// Result sets ResultCode and ResultMsg.
func (r *ImmediateDeliveryReply) Result(code int, msg string) {
	if r == nil {
		return
	}

	r.ResultCode = code
	r.ResultMsg = msg
}

// Succeed marks the reply as handled.
func (r *ImmediateDeliveryReply) Succeed() {
	r.Result(0, "")
}

// OK reports whether the reply says the request was handled.
func (r *ImmediateDeliveryReply) OK() bool {
	return r != nil && r.ResultCode == 0
}

// ImmediateDeliveryEvent holds the fields the courier-side callbacks share.
//
// Four of the six callbacks carry exactly these fields, so the struct keeps them
// in one place instead of repeating them five times.
type ImmediateDeliveryEvent struct {
	EventNotify

	// ShopID is the merchant id the courier assigned.
	ShopID string `xml:"shopid" json:"shopid"`
	// ShopOrderID is the merchant's own order id.
	ShopOrderID string `xml:"shop_order_id" json:"shop_order_id"`
	// ShopNo is the merchant's store number, registered with the courier.
	ShopNo string `xml:"shop_no" json:"shop_no"`
	// WaybillID is the delivery order id.
	WaybillID string `xml:"waybill_id" json:"waybill_id"`
	// DeliverySign is the signature the courier verifies before trusting the push.
	DeliverySign string `xml:"delivery_sign" json:"delivery_sign"`
}

// OK reports whether the header and the order identity are present.
func (e *ImmediateDeliveryEvent) OK() bool {
	return e != nil &&
		e.EventNotify.OK() &&
		e.ShopID != "" &&
		(e.ShopOrderID != "" || e.WaybillID != "")
}
