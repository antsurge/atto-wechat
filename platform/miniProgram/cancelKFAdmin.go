package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// cancelKFAdminPath demotes a customer service administrator.
const cancelKFAdminPath = "/customservice/kfaccount/canceladmin"

// CancelKFAdminParams identifies the administrator to demote.
type CancelKFAdminParams struct {
	// KFOpenID is the openid of the bound WeChat user, as returned by
	// GetKFList.
	KFOpenID string

	// BusinessID is the sub-merchant the account belongs to. Leave it zero for
	// the mini program itself.
	BusinessID int64
}

// CheckCancelKFAdminParams validates params before anything is sent to WeChat.
func CheckCancelKFAdminParams(params CancelKFAdminParams) error {
	if params.KFOpenID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: kf_openid is required",
		)
	}

	return nil
}

// CancelKFAdminResponse is the reply of /customservice/kfaccount/canceladmin.
//
// It carries no business payload: success is reported by errcode alone.
type CancelKFAdminResponse struct {
	response.WechatResponse
}

// CancelKFAdminErrMsg returns the description of a cancelKFAdmin error.
func CancelKFAdminErrMsg(code int) string {
	return KFAccountErrMsg(code)
}

// CancelKFAdminError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes from
// KFAccountErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func CancelKFAdminError(r *CancelKFAdminResponse) error {
	return errors.New(CancelKFAdminErrMsg(r.ErrCode))
}

// CancelKFAdmin demotes a customer service administrator.
//
// The account keeps answering conversations, it only loses the right to manage
// the other accounts. WeChat answers 65414 when the account was not an
// administrator to begin with.
func (a *Application) CancelKFAdmin(
	ctx context.Context,
	params CancelKFAdminParams,
) (*CancelKFAdminResponse, error) {
	if err := CheckCancelKFAdminParams(params); err != nil {
		return nil, err
	}

	query := kfAccountQuery(params.KFOpenID, params.BusinessID)

	resp, err := a.api.Get(ctx, cancelKFAdminPath, query)
	if err != nil {
		return nil, err
	}

	var result CancelKFAdminResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
