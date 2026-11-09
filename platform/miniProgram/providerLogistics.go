package miniProgram

import "encoding/json"

// Logistics helper (provider side) — the endpoints a courier company's own mini
// program calls back into WeChat.
//
// They differ from the merchant side in two ways worth remembering:
//   - the paths live under /cgi-bin/express/delivery, not .../business;
//   - orders are addressed by the token WeChat pushed with the order, not by the
//     merchant's order_id. Keeping the token is the courier's only handle on a
//     waybill.

// action_type values shared by ProviderUpdatePath and ProviderUpdateOrderStatus.
const (
	// ProviderActionSiteAccepted is 揽件前阶段-网点接单.
	ProviderActionSiteAccepted = 90001
	// ProviderActionCourierAssigned is 揽件前阶段-分配业务员.
	ProviderActionCourierAssigned = 90002
	// ProviderActionCourierReassigned is 揽件前阶段-重新分配业务员.
	ProviderActionCourierReassigned = 90003
	// ProviderActionWaitingPayment is 揽件前阶段-待支付.
	ProviderActionWaitingPayment = 90010
	// ProviderActionPaid is 揽件前阶段-已支付.
	ProviderActionPaid = 90011

	// ProviderActionPicked is 揽件阶段-揽件成功.
	ProviderActionPicked = 100001
	// ProviderActionPickFailed is 揽件阶段-揽件失败.
	ProviderActionPickFailed = 100002

	// ProviderActionInTransit is 运输阶段-更新运输轨迹.
	ProviderActionInTransit = 200001

	// ProviderActionDelivering is 派送阶段-派送中.
	ProviderActionDelivering = 300002
	// ProviderActionSigned is 派送阶段-签收成功.
	ProviderActionSigned = 300003
	// ProviderActionSignFailed is 派送阶段-签收失败.
	ProviderActionSignFailed = 300004
	// ProviderActionThirdPartyStored is 派送阶段-第三方代收入库.
	ProviderActionThirdPartyStored = 300005
	// ProviderActionThirdPartyTaken is 派送阶段-第三方代收快递员取出.
	ProviderActionThirdPartyTaken = 300006
	// ProviderActionProxySigned is 派送阶段-代签收.
	ProviderActionProxySigned = 300007

	// ProviderActionCancelled is 异常阶段-订单取消.
	ProviderActionCancelled = 400001
	// ProviderActionStranded is 异常阶段-订单滞留.
	ProviderActionStranded = 400002
)

// ProviderContact is a sender or receiver as WeChat stored it for a waybill.
//
// Address is already merged into one line, unlike the split
// province/city/area of LogisticsPersonInfo.
type ProviderContact struct {
	// Name is the contact name.
	Name string `json:"name"`
	// Tel is the landline number.
	Tel string `json:"tel"`
	// Mobile is the mobile number.
	Mobile string `json:"mobile"`
	// Address is the full address.
	Address string `json:"address"`
}

// ProviderBillType values carried in ProviderGetBillParams.Type.
const (
	// ProviderBillTypeAll downloads every waybill of the day.
	ProviderBillTypeAll = "ALL"
	// ProviderBillTypeSuccess downloads the waybills that were fulfilled.
	ProviderBillTypeSuccess = "SUCCESS"
	// ProviderBillTypeRefund downloads the waybills that were refunded.
	ProviderBillTypeRefund = "REFUND"
)

// WaybillDataJSON encodes waybill data for
// ProviderPreviewTemplateParams.WaybillData.
//
// WeChat wants that field as a JSON string rather than a nested array, which is
// why it cannot simply be []LogisticsWaybillData on the struct.
func WaybillDataJSON(items []LogisticsWaybillData) (string, error) {
	data, err := json.Marshal(items)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
