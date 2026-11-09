package miniProgram

import (
	"context"
	"errors"

	"github.com/antsurge/atto-wechat/core/response"
)

// getKFWorkBoundPath reads the 微信客服 binding of the mini program.
const getKFWorkBoundPath = "/customservice/work/get"

// GetKFWorkBoundResponse is the reply of /customservice/work/get.
//
// The three business fields are absent while the mini program is not bound, so an
// errcode of 0 with an empty CorpID means "not bound", not "bound to nothing".
type GetKFWorkBoundResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	// EntityName is the mini program's subject name.
	EntityName string `json:"entityName"`
	// CorpID is the bound enterprise.
	CorpID string `json:"corpid"`
	// BindTime is when the binding happened, as a Unix timestamp in seconds.
	BindTime int64 `json:"bindTime"`
}

// OK reports whether the call succeeded.
func (r *GetKFWorkBoundResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// Bound reports whether the mini program is bound to a 微信客服 enterprise.
func (r *GetKFWorkBoundResponse) Bound() bool {
	return r.OK() && r.CorpID != ""
}

// GetKFWorkBoundErrMsg returns the description of a getKfWorkBound error.
func GetKFWorkBoundErrMsg(code int) string {
	return KFWorkErrMsg(code)
}

// GetKFWorkBoundError turns the errcode from the reply into an error.
func GetKFWorkBoundError(r *GetKFWorkBoundResponse) error {
	return errors.New(GetKFWorkBoundErrMsg(r.ErrCode))
}

// GetKFWorkBound reads which 微信客服 enterprise the mini program is bound to.
//
// Only non-individual mini programs may bind (10130030), and the enterprise must
// belong to the same subject as the mini program.
func (a *Application) GetKFWorkBound(
	ctx context.Context,
) (*GetKFWorkBoundResponse, error) {
	resp, err := a.api.Get(ctx, getKFWorkBoundPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetKFWorkBoundResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
