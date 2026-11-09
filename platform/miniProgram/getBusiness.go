package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getBusinessPath reads one 客服子商户.
const getBusinessPath = "/cgi-bin/business/get"

// GetBusinessParams identifies the merchant to read.
//
// WeChat takes either form, not both: business_id if the caller stored the id from
// RegisterBusiness, account_name if it only kept the name it chose.
type GetBusinessParams struct {
	// BusinessID is the merchant id.
	BusinessID int64

	// AccountName is the merchant's account name.
	AccountName string
}

// CheckGetBusinessParams validates params before anything is sent to WeChat.
func CheckGetBusinessParams(params GetBusinessParams) error {
	hasID := params.BusinessID != 0
	hasName := params.AccountName != ""

	switch {
	case !hasID && !hasName:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetBusiness needs either business_id or account_name",
		)
	case hasID && hasName:
		// Both is not a precedence puzzle: WeChat documents the two as
		// alternatives, so sending both would leave which one wins undocumented.
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetBusiness takes business_id or account_name, not both",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p GetBusinessParams) body() *getBusinessBody {
	return &getBusinessBody{BusinessID: p.BusinessID, AccountName: p.AccountName}
}

// getBusinessBody is the JSON body of /cgi-bin/business/get.
type getBusinessBody struct {
	BusinessID  int64  `json:"business_id,omitempty"`
	AccountName string `json:"account_name,omitempty"`
}

// GetBusinessResponse is the reply of /cgi-bin/business/get.
type GetBusinessResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	BusinessInfo *BusinessInfo `json:"business_info"`
}

// OK reports whether the reply carries the merchant.
func (r *GetBusinessResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.BusinessInfo != nil
}

// GetBusinessErrMsg returns the description of a getBusiness error.
func GetBusinessErrMsg(code int) string {
	return BusinessErrMsg(code)
}

// GetBusinessError turns the errcode from the reply into an error.
func GetBusinessError(r *GetBusinessResponse) error {
	return errors.New(GetBusinessErrMsg(r.ErrCode))
}

// GetBusiness reads one 客服子商户, by id or by account name.
func (a *Application) GetBusiness(
	ctx context.Context,
	params GetBusinessParams,
) (*GetBusinessResponse, error) {
	if err := CheckGetBusinessParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getBusinessPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GetBusinessResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
