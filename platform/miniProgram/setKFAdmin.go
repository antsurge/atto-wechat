package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// setKFAdminPath promotes a customer service account to administrator.
const setKFAdminPath = "/customservice/kfaccount/setadmin"

// SetKFAdminParams identifies the account to promote.
type SetKFAdminParams struct {
	// KFOpenID is the openid of the bound WeChat user, as returned by
	// GetKFList.
	KFOpenID string

	// BusinessID is the sub-merchant the account belongs to. Leave it zero for
	// the mini program itself.
	BusinessID int64
}

// CheckSetKFAdminParams validates params before anything is sent to WeChat.
func CheckSetKFAdminParams(params SetKFAdminParams) error {
	if params.KFOpenID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: kf_openid is required",
		)
	}

	return nil
}

// SetKFAdminResponse is the reply of /customservice/kfaccount/setadmin.
//
// It carries no business payload: success is reported by errcode alone.
type SetKFAdminResponse struct {
	response.WechatResponse
}

// SetKFAdminErrMsg returns the description of a setKFAdmin error.
func SetKFAdminErrMsg(code int) string {
	return KFAccountErrMsg(code)
}

// SetKFAdminError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes from
// KFAccountErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func SetKFAdminError(r *SetKFAdminResponse) error {
	return errors.New(SetKFAdminErrMsg(r.ErrCode))
}

// SetKFAdmin promotes a customer service account to administrator.
//
// An administrator may bind and unbind the other accounts of the mini program
// from the customer service tool, which is how a team avoids routing every
// staffing change through the backend. An account holds one administrator, so
// 65413 means it already has one.
func (a *Application) SetKFAdmin(
	ctx context.Context,
	params SetKFAdminParams,
) (*SetKFAdminResponse, error) {
	if err := CheckSetKFAdminParams(params); err != nil {
		return nil, err
	}

	query := kfAccountQuery(params.KFOpenID, params.BusinessID)

	resp, err := a.api.Get(ctx, setKFAdminPath, query)
	if err != nil {
		return nil, err
	}

	var result SetKFAdminResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
