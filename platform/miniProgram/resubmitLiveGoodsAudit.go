package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// resubmitLiveGoodsAuditPath submits rejected goods for review again.
const (
	resubmitLiveGoodsAuditPath          = "/wxaapi/broadcast/goods/audit"
	ResubmitLiveGoodsAuditDefaultErrMsg = "系统错误，请稍后再试"
)

// ResubmitLiveGoodsAuditErrCodeMap holds the errcode descriptions WeChat
// documents for /wxaapi/broadcast/goods/audit, keyed by the errcode as it
// appears in the reply.
var ResubmitLiveGoodsAuditErrCodeMap = liveErrCodes(
	200002,
	300003,
	300004,
	300007,
	300010,
	300011,
	300018,
	300024,
	40001,
)

// ResubmitLiveGoodsAuditParams resubmits one goods for review.
type ResubmitLiveGoodsAuditParams struct {
	// GoodsID is the goods to review again, after the reason of the rejection
	// was fixed with UpdateLiveGoods.
	GoodsID int64 `json:"goodsId"`
}

// CheckResubmitLiveGoodsAuditParams validates params before anything is sent to
// WeChat.
func CheckResubmitLiveGoodsAuditParams(params ResubmitLiveGoodsAuditParams) error {
	if params.GoodsID <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: goodsId is required")
	}

	return nil
}

// ResubmitLiveGoodsAuditResponse is the reply of
// /wxaapi/broadcast/goods/audit.
type ResubmitLiveGoodsAuditResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	AuditID int64 `json:"auditId"`
}

// OK reports whether WeChat queued the review.
func (r *ResubmitLiveGoodsAuditResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.AuditID != 0
}

// ResubmitLiveGoodsAuditErrMsg returns the description of an audit error.
func ResubmitLiveGoodsAuditErrMsg(code int) string {
	if msg, ok := ResubmitLiveGoodsAuditErrCodeMap[code]; ok {
		return msg
	}
	return ResubmitLiveGoodsAuditDefaultErrMsg
}

// ResubmitLiveGoodsAuditError turns the errcode from the reply into an error.
func ResubmitLiveGoodsAuditError(r *ResubmitLiveGoodsAuditResponse) error {
	return errors.New(ResubmitLiveGoodsAuditErrMsg(r.ErrCode))
}

// ResubmitLiveGoodsAudit submits rejected goods for review again.
//
// Only goods whose review failed can be resubmitted, and the changes have to be
// saved with UpdateLiveGoods first: a resubmission of unchanged goods just
// fails again. The new AuditID supersedes the one returned by AddLiveGoods.
func (a *Application) ResubmitLiveGoodsAudit(
	ctx context.Context,
	params ResubmitLiveGoodsAuditParams,
) (*ResubmitLiveGoodsAuditResponse, error) {
	if err := CheckResubmitLiveGoodsAuditParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, resubmitLiveGoodsAuditPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result ResubmitLiveGoodsAuditResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
