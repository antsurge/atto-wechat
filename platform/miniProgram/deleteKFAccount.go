package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// deleteKFAccountPath unbinds a customer service account.
const deleteKFAccountPath = "/customservice/kfaccount/del"

// DeleteKFAccountParams identifies the account to unbind.
type DeleteKFAccountParams struct {
	// KFOpenID is the openid of the bound WeChat user, as returned by
	// GetKFList. It is not the kf_account name.
	KFOpenID string

	// BusinessID is the sub-merchant the account belongs to. Leave it zero for
	// the mini program itself.
	BusinessID int64
}

// CheckDeleteKFAccountParams validates params before anything is sent to WeChat.
func CheckDeleteKFAccountParams(params DeleteKFAccountParams) error {
	if params.KFOpenID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: kf_openid is required",
		)
	}

	return nil
}

// DeleteKFAccountResponse is the reply of /customservice/kfaccount/del.
//
// It carries no business payload: success is reported by errcode alone.
type DeleteKFAccountResponse struct {
	response.WechatResponse
}

// DeleteKFAccountErrMsg returns the description of a deleteKFAccount error.
func DeleteKFAccountErrMsg(code int) string {
	return KFAccountErrMsg(code)
}

// DeleteKFAccountError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes from
// KFAccountErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func DeleteKFAccountError(r *DeleteKFAccountResponse) error {
	return errors.New(DeleteKFAccountErrMsg(r.ErrCode))
}

// DeleteKFAccount unbinds a customer service account.
//
// The conversations the account already took stay in the message history, but
// new ones are no longer routed to it. Deleting an account also frees a slot
// against the account limit, so this is the counterpart of AddKFAccount's 65406.
func (a *Application) DeleteKFAccount(
	ctx context.Context,
	params DeleteKFAccountParams,
) (*DeleteKFAccountResponse, error) {
	if err := CheckDeleteKFAccountParams(params); err != nil {
		return nil, err
	}

	query := kfAccountQuery(params.KFOpenID, params.BusinessID)

	resp, err := a.api.Get(ctx, deleteKFAccountPath, query)
	if err != nil {
		return nil, err
	}

	var result DeleteKFAccountResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
