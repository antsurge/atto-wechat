package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// addKFAccountPath binds a WeChat user as a customer service account.
const addKFAccountPath = "/customservice/kfaccount/add"

// AddKFAccountParams identifies the WeChat user to bind.
type AddKFAccountParams struct {
	// KFWx is the WeChat id of the user to bind. WeChat resolves it to the
	// kf_openid the other endpoints take.
	KFWx string

	// BusinessID is the sub-merchant to add the account to. Leave it zero for
	// the mini program itself.
	BusinessID int64
}

// CheckAddKFAccountParams validates params before anything is sent to WeChat.
func CheckAddKFAccountParams(params AddKFAccountParams) error {
	if params.KFWx == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: kf_wx is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p AddKFAccountParams) body() *addKFAccountBody {
	return &addKFAccountBody{KFWx: p.KFWx, BusinessID: p.BusinessID}
}

// addKFAccountBody is the JSON body of /customservice/kfaccount/add.
type addKFAccountBody struct {
	KFWx       string `json:"kf_wx"`
	BusinessID int64  `json:"business_id,omitempty"`
}

// AddKFAccountResponse is the reply of /customservice/kfaccount/add.
//
// It carries no business payload: success is reported by errcode alone.
type AddKFAccountResponse struct {
	response.WechatResponse
}

// AddKFAccountErrMsg returns the description of an addKFAccount error.
func AddKFAccountErrMsg(code int) string {
	return KFAccountErrMsg(code)
}

// AddKFAccountError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes from
// KFAccountErrCodeMap. An undocumented errcode has no agreed meaning, so it
// falls back to a generic message rather than exposing WeChat's raw errmsg.
func AddKFAccountError(r *AddKFAccountResponse) error {
	return errors.New(AddKFAccountErrMsg(r.ErrCode))
}

// AddKFAccount binds a WeChat user as a customer service account.
//
// The user must follow the mini program's official account, and one WeChat id
// may serve only one account. A mini program holds a limited number of accounts,
// so 65406 means an old one has to be deleted first.
//
// Binding is not a message permission: the account still answers through
// SendCustomMessage within the 48-hour window.
func (a *Application) AddKFAccount(
	ctx context.Context,
	params AddKFAccountParams,
) (*AddKFAccountResponse, error) {
	if err := CheckAddKFAccountParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, addKFAccountPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result AddKFAccountResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
