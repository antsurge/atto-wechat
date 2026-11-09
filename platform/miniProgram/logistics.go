package miniProgram

import (
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
)

// Logistics helper (mini program side) shares a few types across endpoints:
// the sender/receiver pair, the parcel description and the insurance/service
// options of an order are the same no matter which endpoint consumes them.
//
// Unlike the other business APIs, a logistics call can fail twice over:
//
//   - errcode/errmsg is WeChat rejecting the call itself (bad token, bad JSON,
//     rate limit);
//   - delivery_resultcode/delivery_resultmsg is the courier rejecting the
//     business operation. It can be non-zero while errcode is 0, because WeChat
//     successfully forwarded a request the courier refused.
//
// The response structs therefore keep both, and OK() requires both to succeed.

// LogisticsAddSource values carried in AddOrderParams.AddSource.
const (
	// LogisticsAddSourceMiniProgram is an order placed inside the mini program.
	// It triggers the logistics service notification, so OpenID is required.
	LogisticsAddSourceMiniProgram = 0
	// LogisticsAddSourceAppOrH5 is an order placed in an App or H5 page. It does
	// not trigger the notification, so OpenID may be empty but WxAppID is needed.
	LogisticsAddSourceAppOrH5 = 2
)

// LogisticsPrintType values carried in GetOrderParams.PrintType.
const (
	// LogisticsPrintTypeTwoPart prints the two-part waybill.
	LogisticsPrintTypeTwoPart = 0
	// LogisticsPrintTypeOnePart prints the one-part waybill.
	LogisticsPrintTypeOnePart = 1
)

// LogisticsOrderStatus values carried in LogisticsOrderData.OrderStatus.
const (
	// LogisticsOrderStatusNormal means the waybill is still valid.
	LogisticsOrderStatusNormal = 0
	// LogisticsOrderStatusCancelled means the waybill was cancelled.
	LogisticsOrderStatusCancelled = 1
)

// logisticsErrCodeMap holds the errcode descriptions that apply to every
// logistics endpoint.
//
// Courier-specific failures are not listed here: they arrive as
// delivery_resultcode, which the courier defines.
var logisticsErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后重试",
	40001: "access_token 无效或 AppSecret 错误",
	40013: "不合法的 AppID",
	42001: "access_token 已过期",
	45009: "接口调用超过限额",
	47001: "解析 JSON 内容错误",
}

// logisticsErrCodes returns a copy of the shared logistics errcode table.
//
// Each endpoint exposes its own map built from this one, so a caller can look up
// a message without knowing which table it came from, and one endpoint gaining a
// code cannot silently change another's.
func logisticsErrCodes() map[int]string {
	table := make(map[int]string, len(logisticsErrCodeMap))
	for code, msg := range logisticsErrCodeMap {
		table[code] = msg
	}

	return table
}

// logisticsErrMsg resolves an errcode against a table, falling back to fallback.
func logisticsErrMsg(table map[int]string, code int, fallback string) string {
	if msg, ok := table[code]; ok {
		return msg
	}

	return fallback
}

// invalidLogisticsField reports a caller-side parameter problem.
//
// It is used instead of plain errors.New so every logistics endpoint answers
// with the same semantic code for the same mistake.
func invalidLogisticsField(msg string) error {
	return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: "+msg)
}

// LogisticsPersonInfo is a sender or receiver.
//
// Tel is a landline and Mobile is a cellphone; at least one of them must be
// filled in, which is what Check enforces.
type LogisticsPersonInfo struct {
	// Name is the contact name.
	Name string `json:"name,omitempty"`
	// Tel is the landline number.
	Tel string `json:"tel,omitempty"`
	// Mobile is the cellphone number.
	Mobile string `json:"mobile,omitempty"`
	// Company is the company name.
	Company string `json:"company,omitempty"`
	// PostCode is the postal code.
	PostCode string `json:"post_code,omitempty"`
	// Country is the country, usually 中国.
	Country string `json:"country,omitempty"`
	// Province is the province.
	Province string `json:"province,omitempty"`
	// City is the city.
	City string `json:"city,omitempty"`
	// Area is the district.
	Area string `json:"area,omitempty"`
	// Address is the street address.
	Address string `json:"address,omitempty"`
}

// Check reports whether the contact has enough information for a waybill.
func (p LogisticsPersonInfo) Check(field string) error {
	if p.Name == "" {
		return invalidLogisticsField(field + ".name is required")
	}
	if p.Mobile == "" && p.Tel == "" {
		return invalidLogisticsField(field + ".mobile or " + field + ".tel is required")
	}

	return nil
}

// LogisticsCargoDetail is one line item inside a parcel.
type LogisticsCargoDetail struct {
	// Name is the item name.
	Name string `json:"name"`
	// Count is how many of the item are in the parcel.
	Count int `json:"count"`
}

// LogisticsCargo describes the parcel.
type LogisticsCargo struct {
	// Count is the number of parcels.
	Count int `json:"count"`
	// Weight is the total weight in kilograms.
	Weight float64 `json:"weight"`
	// SpaceX is the length in centimetres.
	SpaceX float64 `json:"space_x"`
	// SpaceY is the width in centimetres.
	SpaceY float64 `json:"space_y"`
	// SpaceZ is the height in centimetres.
	SpaceZ float64 `json:"space_z"`
	// DetailList is what the parcel contains.
	DetailList []LogisticsCargoDetail `json:"detail_list"`
}

// Check reports whether the parcel description is usable.
func (c LogisticsCargo) Check() error {
	if c.Count <= 0 {
		return invalidLogisticsField("cargo.count must be greater than 0")
	}
	if len(c.DetailList) == 0 {
		return invalidLogisticsField("cargo.detail_list must not be empty")
	}

	for i, item := range c.DetailList {
		if item.Name == "" {
			return invalidLogisticsField("cargo.detail_list[" + strconv.Itoa(i) + "].name is required")
		}
		if item.Count <= 0 {
			return invalidLogisticsField("cargo.detail_list[" + strconv.Itoa(i) + "].count must be greater than 0")
		}
	}

	return nil
}

// LogisticsShopGoodsDetail is a product shown on the service notification.
type LogisticsShopGoodsDetail struct {
	// GoodsName is the product name.
	GoodsName string `json:"goods_name"`
	// GoodsImgURL is the product image.
	GoodsImgURL string `json:"goods_img_url"`
	// GoodsDesc is the product description.
	GoodsDesc string `json:"goods_desc"`
}

// LogisticsShop is the product information WeChat shows on the waybill and in
// the logistics service notification.
type LogisticsShop struct {
	// WxaPath is the mini program path opened from the notification.
	WxaPath string `json:"wxa_path,omitempty"`
	// ImgURL is the product image.
	ImgURL string `json:"img_url,omitempty"`
	// GoodsName is the product name.
	GoodsName string `json:"goods_name,omitempty"`
	// GoodsCount is how many products the order contains.
	GoodsCount int `json:"goods_count,omitempty"`
	// DetailList describes several products in one order.
	DetailList []LogisticsShopGoodsDetail `json:"detail_list,omitempty"`
}

// LogisticsInsured is the insurance option.
type LogisticsInsured struct {
	// UseInsured is 0 for no insurance and 1 for insurance.
	UseInsured int `json:"use_insured"`
	// InsuredValue is the insured amount in cents.
	InsuredValue int64 `json:"insured_value"`
}

// LogisticsService is the courier service level.
type LogisticsService struct {
	// ServiceType is the courier's service type id; 0 is the standard service of
	// most couriers.
	ServiceType int `json:"service_type"`
	// ServiceName is the service name, e.g. 标准快递.
	ServiceName string `json:"service_name"`
}

// LogisticsWaybillData is one key/value pair printed on the waybill.
type LogisticsWaybillData struct {
	// Key is the field name.
	Key string `json:"key"`
	// Value is the field value.
	Value string `json:"value"`
}

// LogisticsOrderData is the waybill payload shared by the order query endpoints.
type LogisticsOrderData struct {
	// PrintHTML is the printable waybill markup.
	PrintHTML string `json:"print_html"`
	// WaybillData is what to print on the waybill.
	WaybillData []LogisticsWaybillData `json:"waybill_data"`
	// OrderID is the merchant order id.
	OrderID string `json:"order_id"`
	// DeliveryID is the courier id.
	DeliveryID string `json:"delivery_id"`
	// WaybillID is the waybill number.
	WaybillID string `json:"waybill_id"`
	// OrderStatus is one of the LogisticsOrderStatus constants.
	OrderStatus int `json:"order_status"`
}

// OK reports whether the payload identifies a waybill.
func (d *LogisticsOrderData) OK() bool {
	return d != nil && d.WaybillID != ""
}
