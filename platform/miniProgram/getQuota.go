package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getQuotaPath returns the remaining electronic waybill quota.
const (
	getQuotaPath          = "/cgi-bin/express/business/quota/get"
	GetQuotaDefaultErrMsg = "系统错误，请稍后再试"
)

// GetQuotaErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/quota/get, keyed by the errcode as it appears in the
// reply.
var GetQuotaErrCodeMap = logisticsErrCodes()

// GetQuotaParams identifies the bound account whose quota to read.
type GetQuotaParams struct {
	// DeliveryID is the courier id.
	DeliveryID string `json:"delivery_id"`
	// BizID is the bound courier account.
	BizID string `json:"biz_id"`
}

// CheckGetQuotaParams validates params before anything is sent to WeChat.
func CheckGetQuotaParams(params GetQuotaParams) error {
	if params.DeliveryID == "" {
		return invalidLogisticsField("delivery_id is required")
	}
	if params.BizID == "" {
		return invalidLogisticsField("biz_id is required")
	}

	return nil
}

// GetQuotaResponse is the reply of /cgi-bin/express/business/quota/get.
type GetQuotaResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	// QuotaNum is the remaining number of waybills.
	QuotaNum int64 `json:"quota_num"`
	// QuotaUpdateTime is when the courier refreshed the quota, as a Unix
	// timestamp. The value is the courier's, so it can lag behind the real quota.
	QuotaUpdateTime int64 `json:"quota_update_time"`
}

// OK reports whether WeChat answered with a quota.
func (r *GetQuotaResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetQuotaErrMsg returns the description of a get quota error.
func GetQuotaErrMsg(code int) string {
	return logisticsErrMsg(GetQuotaErrCodeMap, code, GetQuotaDefaultErrMsg)
}

// GetQuotaError turns the errcode from the reply into an error.
func GetQuotaError(r *GetQuotaResponse) error {
	return errors.New(GetQuotaErrMsg(r.ErrCode))
}

// GetQuota returns the remaining electronic waybill quota of a bound account.
//
// Not every courier supports it: GetAllDelivery reports that as CanGetQuota.
// Couriers returning 0 there answer this call with an error rather than a quota.
func (a *Application) GetQuota(
	ctx context.Context,
	params GetQuotaParams,
) (*GetQuotaResponse, error) {
	if err := CheckGetQuotaParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getQuotaPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result GetQuotaResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
