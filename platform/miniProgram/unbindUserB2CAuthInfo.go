package miniProgram

import (
	"context"
	"errors"
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// unbindUserB2CAuthInfoPath removes 用工关系 bindings in bulk.
const unbindUserB2CAuthInfoPath = "/wxa/business/unbinduserb2cauthinfo"

// UnbindUserB2CAuthInfoErrCodeMap holds the errcode descriptions WeChat documents
// for this endpoint, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var UnbindUserB2CAuthInfoErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40003: "openid 无效",
	40097: "参数错误",
}

// UnbindUserB2CAuthInfoDefaultErrMsg is returned for an errcode this endpoint does
// not document.
const UnbindUserB2CAuthInfoDefaultErrMsg = "系统错误，请稍后再试"

// UnbindUserB2CAuthInfoErrMsg returns the description of an
// unbindUserB2CAuthInfo error, or the default when the errcode is undocumented.
func UnbindUserB2CAuthInfoErrMsg(code int) string {
	if msg, ok := UnbindUserB2CAuthInfoErrCodeMap[code]; ok {
		return msg
	}

	return UnbindUserB2CAuthInfoDefaultErrMsg
}

// UnbindUserB2CAuthInfoParams names the users to unbind.
type UnbindUserB2CAuthInfoParams struct {
	// OpenIDList are the users whose 用工关系 is removed.
	//
	// The call is all-or-nothing: a single invalid openid answers 40003 and none
	// of the list is unbound, so a rejected batch has to be cleaned up and resent
	// rather than partially retried.
	OpenIDList []string
}

// CheckUnbindUserB2CAuthInfoParams validates params before anything is sent to
// WeChat.
func CheckUnbindUserB2CAuthInfoParams(params UnbindUserB2CAuthInfoParams) error {
	if len(params.OpenIDList) == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: UnbindUserB2CAuthInfo openid_list is required",
		)
	}

	for i, openID := range params.OpenIDList {
		if openID == "" {
			// Reported with the index: in a batch of hundreds, "one of them is
			// empty" is not actionable on its own.
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: UnbindUserB2CAuthInfo openid_list["+strconv.Itoa(i)+"] is empty",
			)
		}
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p UnbindUserB2CAuthInfoParams) body() *unbindUserB2CAuthInfoBody {
	return &unbindUserB2CAuthInfoBody{OpenIDList: p.OpenIDList}
}

// unbindUserB2CAuthInfoBody is the JSON body of /wxa/business/unbinduserb2cauthinfo.
type unbindUserB2CAuthInfoBody struct {
	OpenIDList []string `json:"openid_list"`
}

// UnbindUserB2CAuthInfoResponse is the reply of the unbind endpoint.
//
// It carries no per-user result: the batch either went through or it did not.
type UnbindUserB2CAuthInfoResponse struct {
	response.WechatResponse
}

// OK reports whether every openid in the batch was unbound.
func (r *UnbindUserB2CAuthInfoResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// UnbindUserB2CAuthInfoError turns the errcode from the reply into an error.
func UnbindUserB2CAuthInfoError(r *UnbindUserB2CAuthInfoResponse) error {
	return errors.New(UnbindUserB2CAuthInfoErrMsg(r.ErrCode))
}

// UnbindUserB2CAuthInfo removes the 用工关系 of a batch of users.
//
// After this the users can no longer receive messages through
// SendEmployeeRelationMsg; the binding has to be established again before that
// works.
func (a *Application) UnbindUserB2CAuthInfo(
	ctx context.Context,
	params UnbindUserB2CAuthInfoParams,
) (*UnbindUserB2CAuthInfoResponse, error) {
	if err := CheckUnbindUserB2CAuthInfoParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, unbindUserB2CAuthInfoPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result UnbindUserB2CAuthInfoResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
