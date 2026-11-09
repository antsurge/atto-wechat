package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getOnlineKFListPath lists the customer service accounts currently online.
const getOnlineKFListPath = "/cgi-bin/customservice/getonlinekflist"

// GetOnlineKFListParams selects the sub-merchant to read.
type GetOnlineKFListParams struct {
	// BusinessID is the sub-merchant whose accounts to list. Leave it zero for
	// the mini program itself.
	BusinessID int64
}

// GetOnlineKFListResponse is the reply of
// /cgi-bin/customservice/getonlinekflist.
type GetOnlineKFListResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	KFOnlineList []KFOnlineAccount `json:"kf_online_list"`
}

// GetOnlineKFListErrMsg returns the description of a getOnlineKFList error.
func GetOnlineKFListErrMsg(code int) string {
	return KFAccountErrMsg(code)
}

// GetOnlineKFListError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes from
// KFAccountErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func GetOnlineKFListError(r *GetOnlineKFListResponse) error {
	return errors.New(GetOnlineKFListErrMsg(r.ErrCode))
}

// GetOnlineKFList lists the customer service accounts currently online.
//
// WeChat routes a user to an online account, so this is the read used to decide
// whether a conversation can be answered at all. An account that is bound but
// has not opened the customer service tool does not appear here.
func (a *Application) GetOnlineKFList(
	ctx context.Context,
	params GetOnlineKFListParams,
) (*GetOnlineKFListResponse, error) {
	query := kfAccountQuery("", params.BusinessID)

	resp, err := a.api.Get(ctx, getOnlineKFListPath, query)
	if err != nil {
		return nil, err
	}

	var result GetOnlineKFListResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
