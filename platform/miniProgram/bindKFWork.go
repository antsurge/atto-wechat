package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// bindKFWorkPath binds the mini program to a 微信客服 enterprise.
const bindKFWorkPath = "/customservice/work/bind"

// BindKFWorkParams names the enterprise to bind.
type BindKFWorkParams struct {
	// CorpID is the 微信客服 enterprise id.
	//
	// It must belong to the same subject as the mini program (10130602), which is
	// what WeChat checks: binding an unrelated enterprise is refused, not merely
	// discouraged.
	CorpID string
}

// CheckBindKFWorkParams validates params before anything is sent to WeChat.
func CheckBindKFWorkParams(params BindKFWorkParams) error {
	if params.CorpID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: BindKFWork corpid is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p BindKFWorkParams) body() *kfWorkCorpIDBody {
	return &kfWorkCorpIDBody{CorpID: p.CorpID}
}

// BindKFWorkResponse is the reply of /customservice/work/bind.
//
// It carries no business payload: success is reported by errcode alone.
type BindKFWorkResponse struct {
	response.WechatResponse
}

// OK reports whether the binding was created.
func (r *BindKFWorkResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// BindKFWorkErrMsg returns the description of a bindKfWork error.
func BindKFWorkErrMsg(code int) string {
	return KFWorkErrMsg(code)
}

// BindKFWorkError turns the errcode from the reply into an error.
func BindKFWorkError(r *BindKFWorkResponse) error {
	return errors.New(BindKFWorkErrMsg(r.ErrCode))
}

// BindKFWork binds the mini program to a 微信客服 enterprise, so its sessions are
// answered in 企业微信.
//
// A mini program holds one binding at a time: binding again without UnbindKFWork
// answers 10130603. Once bound, GetKFWorkBound reports the corp and the time.
func (a *Application) BindKFWork(
	ctx context.Context,
	params BindKFWorkParams,
) (*BindKFWorkResponse, error) {
	if err := CheckBindKFWorkParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, bindKFWorkPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result BindKFWorkResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
