package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getLiveFollowersPath lists the long-term subscribers of the account.
const (
	getLiveFollowersPath          = "/wxa/business/get_wxa_followers"
	GetLiveFollowersDefaultErrMsg = "系统错误，请稍后再试"
)

// GetLiveFollowersErrCodeMap holds the errcode descriptions WeChat documents for
// /wxa/business/get_wxa_followers, keyed by the errcode as it appears in the
// reply.
var GetLiveFollowersErrCodeMap = liveErrCodes(
	200002,
)

// GetLiveFollowersParams selects the subscribers to return.
type GetLiveFollowersParams struct {
	// Limit is the page size; WeChat caps it at 10000.
	Limit int `json:"limit"`

	// PageBreak is the cursor returned by the previous call, empty on the first
	// one.
	PageBreak string `json:"page_break,omitempty"`
}

// CheckGetLiveFollowersParams validates params before anything is sent to
// WeChat.
func CheckGetLiveFollowersParams(params GetLiveFollowersParams) error {
	if params.Limit <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: limit is required")
	}

	return nil
}

// LiveFollower is one long-term subscriber.
type LiveFollower struct {
	OpenID     string `json:"openid"`
	CreateTime int64  `json:"create_time"`
}

// GetLiveFollowersResponse is the reply of /wxa/business/get_wxa_followers.
type GetLiveFollowersResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	Followers []LiveFollower `json:"followers"`

	// PageBreak is the cursor of the next page, empty once the list is
	// exhausted.
	PageBreak string `json:"page_break"`
}

// OK reports whether WeChat answered with a usable page.
func (r *GetLiveFollowersResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetLiveFollowersErrMsg returns the description of a get_wxa_followers error.
func GetLiveFollowersErrMsg(code int) string {
	if msg, ok := GetLiveFollowersErrCodeMap[code]; ok {
		return msg
	}
	return GetLiveFollowersDefaultErrMsg
}

// GetLiveFollowersError turns the errcode from the reply into an error.
func GetLiveFollowersError(r *GetLiveFollowersResponse) error {
	return errors.New(GetLiveFollowersErrMsg(r.ErrCode))
}

// GetLiveFollowers lists the users who subscribed to the live broadcast of the
// account.
//
// The list is paged by cursor, not by offset: the caller keeps passing the
// PageBreak of the previous reply until it comes back empty. Those openids are
// what PushLiveMessage sends the live start reminder to.
func (a *Application) GetLiveFollowers(
	ctx context.Context,
	params GetLiveFollowersParams,
) (*GetLiveFollowersResponse, error) {
	if err := CheckGetLiveFollowersParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getLiveFollowersPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result GetLiveFollowersResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
