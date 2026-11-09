package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// resetLiveGoodsAuditPath withdraws goods from review.
const (
	resetLiveGoodsAuditPath          = "/wxaapi/broadcast/goods/resetaudit"
	ResetLiveGoodsAuditDefaultErrMsg = "系统错误，请稍后再试"
)

// ResetLiveGoodsAuditErrCodeMap holds the errcode descriptions WeChat documents
// for /wxaapi/broadcast/goods/resetaudit, keyed by the errcode as it appears in
// the reply.
var ResetLiveGoodsAuditErrCodeMap = liveErrCodes(
	200002,
	300009,
	300010,
	300011,
	300024,
	40001,
)

// ResetLiveGoodsAuditParams withdraws one goods from review.
type ResetLiveGoodsAuditParams struct {
	// GoodsID is the goods being reviewed.
	GoodsID int64 `json:"goodsId"`

	// AuditID is the id returned by AddLiveGoods or ResubmitLiveGoodsAudit.
	AuditID int64 `json:"auditId"`
}

// CheckResetLiveGoodsAuditParams validates params before anything is sent to
// WeChat.
func CheckResetLiveGoodsAuditParams(params ResetLiveGoodsAuditParams) error {
	switch {
	case params.GoodsID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: goodsId is required")
	case params.AuditID <= 0:
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: auditId is required")
	}

	return nil
}

// ResetLiveGoodsAuditResponse is the reply of
// /wxaapi/broadcast/goods/resetaudit.
//
// It carries no business payload: success is reported by errcode alone.
type ResetLiveGoodsAuditResponse struct {
	// Common reply parameters.
	response.WechatResponse
}

// OK reports whether WeChat withdrew the goods from review.
func (r *ResetLiveGoodsAuditResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ResetLiveGoodsAuditErrMsg returns the description of a resetaudit error.
func ResetLiveGoodsAuditErrMsg(code int) string {
	if msg, ok := ResetLiveGoodsAuditErrCodeMap[code]; ok {
		return msg
	}
	return ResetLiveGoodsAuditDefaultErrMsg
}

// ResetLiveGoodsAuditError turns the errcode from the reply into an error.
func ResetLiveGoodsAuditError(r *ResetLiveGoodsAuditResponse) error {
	return errors.New(ResetLiveGoodsAuditErrMsg(r.ErrCode))
}

// ResetLiveGoodsAudit withdraws goods from review.
//
// Withdrawing is the way back to an editable goods: once the review is
// cancelled the goods can be updated and submitted again. It only works while
// the review is still pending, and it fails (300009) if the withdrawal itself
// does not go through.
func (a *Application) ResetLiveGoodsAudit(
	ctx context.Context,
	params ResetLiveGoodsAuditParams,
) (*ResetLiveGoodsAuditResponse, error) {
	if err := CheckResetLiveGoodsAuditParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, resetLiveGoodsAuditPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ResetLiveGoodsAuditResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
