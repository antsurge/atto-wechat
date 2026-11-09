package miniProgram

import (
	"context"
	"encoding/xml"
	"reflect"

	"github.com/antsurge/atto-wechat/core/codec"
	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/event"
)

// 物流助手（运力方使用）的事件通知。
//
// 和公众号事件不同，这些推送是「一问一答」的：微信推一个 JSON 或 XML 报文过来，
// 运力方必须回一个结构相同的报文，里面带 ResultCode。回 "success" 会导致微信把
// 这次推送判为失败。
//
// 所以这一组事件的处理函数返回一个回复结构体，由框架用 ReplyWith 写回；请求是什么
// 格式（XML 或 JSON），回复就是什么格式。
//
// 注意：不要给这些回调开启 Dispatcher 的去重（event.WithDedup）。去重命中时处理函数
// 不会执行，响应退回默认的 "success"，而微信只认带 ResultCode 的报文——它会把这次
// 推送判为失败并重试，重试又被去重挡掉，于是永远失败。幂等要在自己的业务里做
// （例如 add_waybill 按 BizID + OrderID 返回同一个运单号），不能靠回调层去重。

// Event names the courier (运力方) callbacks carry in the Event field.
const (
	// ProviderEventCancelWaybill is "cancel_waybill": the merchant cancelled an
	// order before pickup.
	ProviderEventCancelWaybill = "cancel_waybill"
	// ProviderEventCheckBiz is "check_biz": WeChat asks whether a merchant's
	// account is known and how much balance it has.
	ProviderEventCheckBiz = "check_biz"
	// ProviderEventAddWaybill is "add_waybill": WeChat asks the courier to create
	// a waybill for an order.
	ProviderEventAddWaybill = "add_waybill"
	// ProviderEventGetQuota is "get_quota": WeChat asks for the merchant's
	// remaining balance.
	ProviderEventGetQuota = "get_quota"
	// ProviderEventWaybillNotifyQuery is "waybill_notify_query": WeChat asks for
	// the tracking nodes of a waybill.
	ProviderEventWaybillNotifyQuery = "waybill_notify_query"
	// ProviderEventRemindWaybill is "remind_waybill": the user asked where the
	// parcel is.
	ProviderEventRemindWaybill = "remind_waybill"
	// ProviderEventAddSingleWaybill is "add_single_waybill": a single-parcel order
	// is placed.
	ProviderEventAddSingleWaybill = "add_single_waybill"
	// ProviderEventCancelSingleWaybill is "cancel_single_waybill": a single-parcel
	// order is cancelled.
	ProviderEventCancelSingleWaybill = "cancel_single_waybill"
	// ProviderEventQuerySingleWaybillFee is "query_single_waybill_fee": WeChat asks
	// for the price of shipping between two addresses.
	ProviderEventQuerySingleWaybillFee = "query_single_waybill_fee"
	// ProviderEventQuerySingleWaybill is "query_single_waybill": WeChat asks for
	// everything it knows about a waybill.
	ProviderEventQuerySingleWaybill = "query_single_waybill"
	// ProviderEventQueryExpectTimeRange is "waybill_query_expect_time_range":
	// WeChat asks which pickup slots are still bookable in a city.
	ProviderEventQueryExpectTimeRange = "waybill_query_expect_time_range"
	// ProviderEventAddExpressPath is "add_express_path": a batch waybill got new
	// tracking nodes.
	ProviderEventAddExpressPath = "add_express_path"
	// ProviderEventNotifySingleWaybillPay is "notify_single_waybill_pay": the buyer
	// paid for a single-parcel order.
	ProviderEventNotifySingleWaybillPay = "notify_single_waybill_pay"
	// ProviderEventEvaluateAgent is "waybill_evaluate_agent": the user rated the
	// courier who handled the parcel.
	ProviderEventEvaluateAgent = "waybill_evaluate_agent"
	// ProviderEventComplaint is "waybill_complaint": the user filed a complaint.
	ProviderEventComplaint = "waybill_complaint"
)

// 物流助手（小程序方使用）的事件通知。
//
// 这一组和上面相反：处理函数只回 "success" 或空串，不需要结构化报文，所以签名是
// 返回 error 而不是返回 reply。
const (
	// ProviderEventBindResult is "update_business_bind_result": the audit result of
	// a bound courier account changed.
	ProviderEventBindResult = "update_business_bind_result"
	// ProviderEventClaimResult is "wxainsurance_claim_result": a 无忧退 claim
	// reached a final state.
	ProviderEventClaimResult = "wxainsurance_claim_result"
)

// ProviderResultCodeMap holds the ResultCode descriptions the logistics helper
// documents. The courier is free to invent its own values above these, so an
// unknown code is not an error.
var ProviderResultCodeMap = map[int]string{
	0:     "成功",
	10001: "客户编码或者月结账户不存在",
	10002: "客户密码不正确",
	20001: "运单 ID 不正确",
	20002: "发件人信息不完整",
	20003: "发件人地址不可达或者发货地址不在服务范围",
	20004: "收件人信息不完整",
	20005: "收件人地址不可达或者收货地址不在服务范围",
	20006: "货物数量、重量或尺寸不正确或者不合理",
	20007: "商户余额不足，需要充值后再进行下单",
	20008: "保价信息不正确",
	20009: "服务信息不正确",
	30001: "参数错误（BizID、OrderID、WayBillID 不存在）",
	30002: "已经揽收，不可取消",
	30003: "无效单（已经取消过，或者签收超过一定时间）",
	30004: "快递不支持取消运单",
}

// ProviderResultMsg returns the description of a courier ResultCode, or the empty
// string when the courier defined its own code.
func ProviderResultMsg(code int) string {
	return ProviderResultCodeMap[code]
}

// ProviderResultOK is the ResultCode that means the callback was handled.
const ProviderResultOK = 0

// ProviderReply is the header every courier reply echoes back.
//
// WeChat matches a reply to its request by ToUserName/FromUserName/Event, so the
// values must be swapped, not copied: what the reply addresses is the account
// that pushed the request.
type ProviderReply struct {
	// XMLName makes the document start with <xml>, which WeChat requires.
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
	// ResultCode is what the courier made of the request; 0 means handled.
	ResultCode int `xml:"ResultCode" json:"ResultCode"`
	// ResultMsg explains a non-zero ResultCode.
	ResultMsg string `xml:"ResultMsg" json:"ResultMsg"`
}

// Echo fills the reply header from the request.
//
// It must be called before the reply is returned: an empty header makes WeChat
// treat the reply as unrelated to the request.
func (r *ProviderReply) Echo(req *EventNotify) {
	if r == nil || req == nil {
		return
	}

	r.ToUserName = codec.CDATA(req.FromUserName)
	r.FromUserName = codec.CDATA(req.ToUserName)
	r.CreateTime = req.CreateTime
	r.MsgType = EventNotifyMsgType
	r.Event = req.Event
}

// Result sets ResultCode and, when msg is empty, its documented description.
func (r *ProviderReply) Result(code int, msg string) {
	if r == nil {
		return
	}

	r.ResultCode = code
	if msg == "" {
		msg = ProviderResultMsg(code)
	}
	r.ResultMsg = msg
}

// Succeed marks the reply as handled.
func (r *ProviderReply) Succeed() {
	r.Result(ProviderResultOK, "")
}

// OK reports whether the reply says the request was handled.
func (r *ProviderReply) OK() bool {
	return r != nil && r.ResultCode == ProviderResultOK
}

// isNilReply reports whether v is nil, including the typed nil pointer a handler
// produces with `return nil, nil`.
//
// That case is not caught by v == nil: the interface holds a type, so the check
// fails and the value reaches ReplyWith, which encodes it as "null" (JSON) or an
// empty <xml/> (XML). WeChat accepts neither, and the callback would fail on every
// retry without saying why.
func isNilReply(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func:
		return rv.IsNil()
	}

	return false
}

// onProviderEvent wires a typed handler that answers with a structured reply.
//
// newEvent and the type assertion in handle are kept together in each event file,
// so the pair cannot drift apart.
func (a *Application) onProviderEvent(
	name string,
	newEvent func() eventPayload,
	handle func(ctx context.Context, evt eventPayload) (any, error),
) *Application {
	if newEvent == nil || handle == nil {
		return a
	}

	a.dispatcher.On(
		EventNotifyMsgType+"."+name,
		event.Handler(func(ctx context.Context, msg *event.Message) error {
			evt := newEvent()

			if err := decodeDispatched(msg, []string{name}, evt); err != nil {
				return err
			}

			reply, err := handle(ctx, evt)
			if err != nil {
				return err
			}
			if isNilReply(reply) {
				return cerrors.New(
					cerrors.CodeInvalidArgument,
					"miniprogram: "+name+" handler returned no reply",
				)
			}

			return msg.ReplyWith(reply)
		}),
	)

	return a
}

// ProviderAddress is the address shape the single-parcel callbacks use.
//
// It is flatter than ProviderWaybillPerson: the logistics helper sends the
// province, city and district separately plus a street, and no company or country.
type ProviderAddress struct {
	// Name is the person's name.
	Name string `xml:"Name" json:"Name"`
	// Tel is the landline number.
	Tel string `xml:"Tel" json:"Tel"`
	// Mobile is the mobile number.
	Mobile string `xml:"Mobile" json:"Mobile"`
	// Province is the province, e.g. 广东省.
	Province string `xml:"Province" json:"Province"`
	// City is the city, e.g. 广州市.
	City string `xml:"City" json:"City"`
	// Area is the district, e.g. 海珠区.
	Area string `xml:"Area" json:"Area"`
	// Street is the street, e.g. 新港中路.
	Street string `xml:"Street" json:"Street"`
	// Address is the rest of the address.
	Address string `xml:"Address" json:"Address"`
	// ID is WeChat's address id, when the user picked a saved address.
	ID string `xml:"Id" json:"Id"`
}

// ProviderGoods describes the parcel in the single-parcel callbacks.
type ProviderGoods struct {
	// Weight is the total weight in kilograms.
	Weight float64 `xml:"Weight" json:"Weight"`
	// Special is the item category, as listed in the helper's documentation.
	Special int `xml:"Special" json:"Special"`
}

// ProviderPathNode is one tracking node in snake_case.
//
// The single-parcel callbacks spell the tracking fields in snake_case while the
// rest of the document is PascalCase; the tags follow the document.
type ProviderPathNode struct {
	// ActionTime is when the node happened, as a Unix timestamp.
	ActionTime int64 `xml:"action_time" json:"action_time"`
	// ActionType is the node type, one of the ProviderAction* constants.
	ActionType int `xml:"action_type" json:"action_type"`
	// ActionMsg is the text shown on the tracking page.
	ActionMsg string `xml:"action_msg" json:"action_msg"`
	// PickupCourierName is set once a pickup courier is assigned.
	PickupCourierName string `xml:"pickup_courier_name" json:"pickup_courier_name"`
	// PickupCourierPhone is the pickup courier's number.
	PickupCourierPhone string `xml:"pickup_courier_phone" json:"pickup_courier_phone"`
	// DeliveryCourierName is set once a delivery courier is assigned.
	DeliveryCourierName string `xml:"delivery_courier_name" json:"delivery_courier_name"`
	// DeliveryCourierPhone is the delivery courier's number.
	DeliveryCourierPhone string `xml:"delivery_courier_phone" json:"delivery_courier_phone"`
}

// ProviderPathInfo wraps the tracking nodes, because the helper nests them one
// level deeper than the rest of the reply.
type ProviderPathInfo struct {
	// Nodes are the tracking nodes, oldest first.
	Nodes []ProviderPathNode `xml:"path_info" json:"path_info"`
}
