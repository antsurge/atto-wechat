package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getAllDeliveryPath lists the couriers the mini program can use.
const (
	getAllDeliveryPath          = "/cgi-bin/express/business/delivery/getall"
	GetAllDeliveryDefaultErrMsg = "系统错误，请稍后再试"
)

// GetAllDeliveryErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/delivery/getall, keyed by the errcode as it appears
// in the reply.
var GetAllDeliveryErrCodeMap = logisticsErrCodes()

// LogisticsDeliveryCompany is one supported courier.
type LogisticsDeliveryCompany struct {
	// DeliveryID is the courier id; every other endpoint refers to it.
	DeliveryID string `json:"delivery_id"`
	// DeliveryName is the courier name.
	DeliveryName string `json:"delivery_name"`
	// CanUseCash is 1 when the courier accepts cash-on-delivery orders.
	CanUseCash int `json:"can_use_cash"`
	// CanGetQuota is 1 when GetQuota works for the courier.
	CanGetQuota int `json:"can_get_quota"`
	// CashBizID is the account used for cash-on-delivery orders.
	CashBizID string `json:"cash_biz_id"`
	// ServiceType lists the service levels the courier offers.
	ServiceType []LogisticsService `json:"service_type"`
}

// GetAllDeliveryResponse is the reply of
// /cgi-bin/express/business/delivery/getall.
type GetAllDeliveryResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Count int                        `json:"count"`
	Data  []LogisticsDeliveryCompany `json:"data"`
}

// OK reports whether WeChat answered with a usable list.
func (r *GetAllDeliveryResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// Find returns the courier with the given id, or nil when it is not supported.
func (r *GetAllDeliveryResponse) Find(deliveryID string) *LogisticsDeliveryCompany {
	if r == nil || deliveryID == "" {
		return nil
	}

	for i := range r.Data {
		if r.Data[i].DeliveryID == deliveryID {
			return &r.Data[i]
		}
	}

	return nil
}

// GetAllDeliveryErrMsg returns the description of a getall delivery error.
func GetAllDeliveryErrMsg(code int) string {
	return logisticsErrMsg(GetAllDeliveryErrCodeMap, code, GetAllDeliveryDefaultErrMsg)
}

// GetAllDeliveryError turns the errcode from the reply into an error.
func GetAllDeliveryError(r *GetAllDeliveryResponse) error {
	return errors.New(GetAllDeliveryErrMsg(r.ErrCode))
}

// GetAllDelivery lists the couriers the mini program can create waybills with.
//
// The list is cached by WeChat, so a courier added to the account may take a
// while to show up. It is still the authority for DeliveryID and ServiceType
// values, so call it once instead of hard-coding courier ids.
func (a *Application) GetAllDelivery(ctx context.Context) (*GetAllDeliveryResponse, error) {
	resp, err := a.api.Get(ctx, getAllDeliveryPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetAllDeliveryResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
