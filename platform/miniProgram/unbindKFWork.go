package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// unbindKFWorkPath removes the 微信客服 binding of the mini program.
const unbindKFWorkPath = "/customservice/work/unbind"

// UnbindKFWorkParams names the enterprise to unbind.
type UnbindKFWorkParams struct {
	// CorpID is the bound 微信客服 enterprise id.
	CorpID string
}

// CheckUnbindKFWorkParams validates params before anything is sent to WeChat.
func CheckUnbindKFWorkParams(params UnbindKFWorkParams) error {
	if params.CorpID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: UnbindKFWork corpid is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p UnbindKFWorkParams) body() *kfWorkCorpIDBody {
	return &kfWorkCorpIDBody{CorpID: p.CorpID}
}

// UnbindKFWorkResponse is the reply of /customservice/work/unbind.
//
// It carries no business payload: success is reported by errcode alone.
type UnbindKFWorkResponse struct {
	response.WechatResponse
}

// OK reports whether the binding was removed.
func (r *UnbindKFWorkResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// UnbindKFWorkErrMsg returns the description of an unbindKfWork error.
func UnbindKFWorkErrMsg(code int) string {
	return KFWorkErrMsg(code)
}

// UnbindKFWorkError turns the errcode from the reply into an error.
func UnbindKFWorkError(r *UnbindKFWorkResponse) error {
	return errors.New(UnbindKFWorkErrMsg(r.ErrCode))
}

// UnbindKFWork removes the mini program's 微信客服 binding.
//
// The corpid has to match the bound enterprise, so read it with GetKFWorkBound
// first rather than reconstructing it from config.
func (a *Application) UnbindKFWork(
	ctx context.Context,
	params UnbindKFWorkParams,
) (*UnbindKFWorkResponse, error) {
	if err := CheckUnbindKFWorkParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, unbindKFWorkPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result UnbindKFWorkResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
