package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// addOrderPath creates a waybill.
const (
	addOrderPath          = "/cgi-bin/express/business/order/add"
	AddOrderDefaultErrMsg = "系统错误，请稍后再试"
)

// AddOrderErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/order/add, keyed by the errcode as it appears in the
// reply.
var AddOrderErrCodeMap = logisticsErrCodes()

// AddOrderParams describes the waybill to create.
type AddOrderParams struct {
	// OrderID is the merchant order id; the same OrderID must always produce the
	// same waybill, so retrying with it is safe.
	OrderID string `json:"order_id"`
	// OpenID is the buyer; required for LogisticsAddSourceMiniProgram, ignored
	// otherwise.
	OpenID string `json:"openid,omitempty"`
	// DeliveryID is the courier id, see GetAllDelivery.
	DeliveryID string `json:"delivery_id"`
	// BizID is the bound courier account, see GetAllAccount.
	BizID string `json:"biz_id"`
	// CustomRemark is the note printed on the waybill, e.g. 易碎物品.
	CustomRemark string `json:"custom_remark,omitempty"`
	// TagID groups orders for a platform mini program; leave 0 when unused.
	TagID int64 `json:"tagid,omitempty"`
	// AddSource is LogisticsAddSourceMiniProgram or
	// LogisticsAddSourceAppOrH5. The zero value is the mini program source, which
	// is also what an omitted field means, so it is always sent.
	AddSource int `json:"add_source"`
	// WxAppID is the App/H5 appid; required when AddSource is
	// LogisticsAddSourceAppOrH5.
	WxAppID string `json:"wx_appid,omitempty"`
	// Sender is the shipper.
	Sender LogisticsPersonInfo `json:"sender"`
	// Receiver is the consignee.
	Receiver LogisticsPersonInfo `json:"receiver"`
	// Cargo describes the parcel.
	Cargo LogisticsCargo `json:"cargo"`
	// Shop is the product information shown to the buyer.
	Shop *LogisticsShop `json:"shop,omitempty"`
	// Insured is the insurance option; omit it for no insurance.
	Insured *LogisticsInsured `json:"insured,omitempty"`
	// Service selects the courier service level; omit it for the courier default.
	Service *LogisticsService `json:"service,omitempty"`
}

// CheckAddOrderParams validates params before anything is sent to WeChat.
//
// It only rejects what would certainly fail remotely: a missing identifier, an
// unknown add_source, or a contact without a phone number.
func CheckAddOrderParams(params AddOrderParams) error {
	if params.OrderID == "" {
		return invalidLogisticsField("order_id is required")
	}
	if params.DeliveryID == "" {
		return invalidLogisticsField("delivery_id is required")
	}
	if params.BizID == "" {
		return invalidLogisticsField("biz_id is required")
	}

	switch params.AddSource {
	case LogisticsAddSourceMiniProgram:
		if params.OpenID == "" {
			return invalidLogisticsField("openid is required when add_source is 0")
		}
	case LogisticsAddSourceAppOrH5:
		if params.WxAppID == "" {
			return invalidLogisticsField("wx_appid is required when add_source is 2")
		}
	default:
		return invalidLogisticsField("add_source must be 0 or 2")
	}

	if err := params.Sender.Check("sender"); err != nil {
		return err
	}
	if err := params.Receiver.Check("receiver"); err != nil {
		return err
	}
	if params.Shop != nil && params.Shop.GoodsCount <= 0 {
		return invalidLogisticsField("shop.goods_count must be greater than 0")
	}

	return params.Cargo.Check()
}

// AddOrderResponse is the reply of /cgi-bin/express/business/order/add.
type AddOrderResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	OrderID string `json:"order_id"`
	// WaybillID is the waybill number; it is what GetPath and CancelOrder need.
	WaybillID string `json:"waybill_id"`
	// WaybillData is what to print on the waybill.
	WaybillData []LogisticsWaybillData `json:"waybill_data"`
	// DeliveryResultCode is the courier's own result code, 0 on success.
	DeliveryResultCode int `json:"delivery_resultcode"`
	// DeliveryResultMsg explains a non-zero DeliveryResultCode.
	DeliveryResultMsg string `json:"delivery_resultmsg"`
}

// OK reports whether a waybill was actually created.
//
// Both layers must succeed: WeChat can accept the call (errcode 0) while the
// courier refuses the order, in which case there is no waybill_id to print.
func (r *AddOrderResponse) OK() bool {
	return r != nil &&
		r.ErrCode == 0 &&
		r.DeliveryResultCode == 0 &&
		r.WaybillID != ""
}

// CourierError returns the courier's rejection as an error, or nil when the
// courier accepted the order.
//
// It exists because a non-zero DeliveryResultCode arrives with errcode 0, so
// AddOrderError alone would report success.
func (r *AddOrderResponse) CourierError() error {
	if r == nil || r.DeliveryResultCode == 0 {
		return nil
	}

	msg := r.DeliveryResultMsg
	if msg == "" {
		msg = "运力方拒绝下单"
	}

	return errors.New(msg)
}

// AddOrderErrMsg returns the description of an add order error.
func AddOrderErrMsg(code int) string {
	return logisticsErrMsg(AddOrderErrCodeMap, code, AddOrderDefaultErrMsg)
}

// AddOrderError returns nil when the order succeeded, the courier's message when
// the courier refused it, and the errcode description otherwise.
//
// It covers both failure layers so a caller needs one call instead of checking
// ErrCode and CourierError itself.
func AddOrderError(r *AddOrderResponse) error {
	if r == nil {
		return errors.New(AddOrderDefaultErrMsg)
	}
	if r.ErrCode != 0 {
		return errors.New(AddOrderErrMsg(r.ErrCode))
	}

	return r.CourierError()
}

// AddOrder creates a waybill.
//
// The reply has two failure layers, so check OK() rather than only ErrCode; when
// it is false, AddOrderError and CourierError tell which side rejected the
// order.
func (a *Application) AddOrder(
	ctx context.Context,
	params AddOrderParams,
) (*AddOrderResponse, error) {
	if err := CheckAddOrderParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, addOrderPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result AddOrderResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
