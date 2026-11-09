package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getKFListPath lists every customer service account of the mini program.
const getKFListPath = "/cgi-bin/customservice/getkflist"

// GetKFListParams selects the sub-merchant to read.
type GetKFListParams struct {
	// BusinessID is the sub-merchant whose accounts to list. Leave it zero for
	// the mini program itself.
	BusinessID int64
}

// GetKFListResponse is the reply of /cgi-bin/customservice/getkflist.
type GetKFListResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	KFList []KFAccount `json:"kf_list"`
}

// GetKFListErrMsg returns the description of a getKFList error.
func GetKFListErrMsg(code int) string {
	return KFAccountErrMsg(code)
}

// GetKFListError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes from
// KFAccountErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func GetKFListError(r *GetKFListResponse) error {
	return errors.New(GetKFListErrMsg(r.ErrCode))
}

// GetKFList lists every customer service account of the mini program.
//
// The reply carries the kf_openid of each account, which is what
// DeleteKFAccount and SetKFAdmin take: the WeChat id bound at creation is not
// enough once the account exists.
func (a *Application) GetKFList(
	ctx context.Context,
	params GetKFListParams,
) (*GetKFListResponse, error) {
	query := kfAccountQuery("", params.BusinessID)

	resp, err := a.api.Get(ctx, getKFListPath, query)
	if err != nil {
		return nil, err
	}

	var result GetKFListResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
