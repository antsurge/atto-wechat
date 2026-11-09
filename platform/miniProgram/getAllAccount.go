package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getAllAccountPath lists the courier accounts bound to the mini program.
const (
	getAllAccountPath          = "/cgi-bin/express/business/account/getall"
	GetAllAccountDefaultErrMsg = "系统错误，请稍后再试"
)

// GetAllAccountErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/account/getall, keyed by the errcode as it appears in
// the reply.
var GetAllAccountErrCodeMap = logisticsErrCodes()

// BoundLogisticsAccount is one bound courier account.
type BoundLogisticsAccount struct {
	// BizID is the merchant's customer code at the courier; this is the value
	// AddOrderParams.BizID must repeat.
	BizID string `json:"biz_id"`
	// DeliveryID is the courier id.
	DeliveryID string `json:"delivery_id"`
	// CreateTime is when the account was bound, as a Unix timestamp in seconds.
	CreateTime int64 `json:"create_time"`
	// UpdateTime is when the account last changed, as a Unix timestamp.
	UpdateTime int64 `json:"update_time"`
	// StatusCode is the courier's review state; 0 means usable.
	StatusCode int `json:"status_code"`
	// Alias is the account name shown to the merchant.
	Alias string `json:"alias"`
	// RemarkWrongMsg explains a rejected review.
	RemarkWrongMsg string `json:"remark_wrong_msg"`
	// RemarkContent is the note submitted with the account.
	RemarkContent string `json:"remark_content"`
	// QuotaNum is the remaining electronic waybill quota.
	QuotaNum int64 `json:"quota_num"`
	// QuotaUpdateTime is when QuotaNum was refreshed, as a Unix timestamp.
	QuotaUpdateTime int64 `json:"quota_update_time"`
	// ServiceType lists the service levels the account supports.
	ServiceType []LogisticsService `json:"service_type"`
}

// OK reports whether the account is bound and usable.
func (a *BoundLogisticsAccount) OK() bool {
	return a != nil && a.BizID != "" && a.DeliveryID != "" && a.StatusCode == 0
}

// GetAllAccountResponse is the reply of
// /cgi-bin/express/business/account/getall.
type GetAllAccountResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Count int                     `json:"count"`
	List  []BoundLogisticsAccount `json:"list"`
}

// OK reports whether WeChat answered with a usable list.
func (r *GetAllAccountResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetAllAccountErrMsg returns the description of a getall account error.
func GetAllAccountErrMsg(code int) string {
	return logisticsErrMsg(GetAllAccountErrCodeMap, code, GetAllAccountDefaultErrMsg)
}

// GetAllAccountError turns the errcode from the reply into an error.
func GetAllAccountError(r *GetAllAccountResponse) error {
	return errors.New(GetAllAccountErrMsg(r.ErrCode))
}

// GetAllAccount lists the courier accounts bound to the mini program.
//
// An empty list is a valid reply: it means no account has been bound yet, not
// that the call failed. Use OK() to tell the two apart.
func (a *Application) GetAllAccount(ctx context.Context) (*GetAllAccountResponse, error) {
	resp, err := a.api.Get(ctx, getAllAccountPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetAllAccountResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
