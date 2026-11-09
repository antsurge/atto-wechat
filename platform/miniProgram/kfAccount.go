package miniProgram

import (
	"net/url"
	"strconv"
)

// KFAccountDefaultErrMsg is returned for an errcode the customer service
// account endpoints do not document.
//
// The 654xx range is shared by the whole family: adding, deleting and binding an
// administrator all answer with the same account-level failures, so the table
// lives here instead of being repeated in six files.
const KFAccountDefaultErrMsg = "系统错误，请稍后再试"

// KFAccountErrCodeMap holds the errcode descriptions WeChat documents for the
// /customservice/kfaccount and /cgi-bin/customservice endpoints, keyed by the
// errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var KFAccountErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	40003: "openid 或 kf_openid 不正确",
	42001: "access_token 已过期",
	65400: "客服微信号不存在",
	65401: "该微信号已绑定其他客服账号",
	65402: "客服账号已存在",
	65405: "客服账号不存在",
	65406: "客服账号数量已达上限",
	65413: "客服账号已绑定管理员",
	65414: "客服账号未绑定管理员",
}

// KFAccountErrMsg returns the description of a customer service account error.
func KFAccountErrMsg(code int) string {
	if msg, ok := KFAccountErrCodeMap[code]; ok {
		return msg
	}
	return KFAccountDefaultErrMsg
}

// KFAccount is one customer service account of the mini program.
type KFAccount struct {
	// KFAccount is the account name, in the "prefix@wechat_id" form of the
	// official account platform.
	KFAccount string `json:"kf_account"`

	// KFNick is the nickname shown to users.
	KFNick string `json:"kf_nick"`

	// KFID is the internal id of the account.
	KFID string `json:"kf_id"`

	// KFHeadImgURL is the avatar of the account.
	KFHeadImgURL string `json:"kf_headimgurl"`

	// KFWx is the WeChat id bound to the account.
	KFWx string `json:"kf_wx"`

	// KFOpenID is the openid of the bound WeChat user, which is what the
	// delete and administrator endpoints take.
	KFOpenID string `json:"kf_openid"`
}

// KFOnlineAccount is one customer service account currently online.
type KFOnlineAccount struct {
	// KFAccount is the account name.
	KFAccount string `json:"kf_account"`

	// KFID is the internal id of the account.
	KFID string `json:"kf_id"`

	// KFOpenID is the openid of the bound WeChat user.
	KFOpenID string `json:"kf_openid"`

	// Status is 1 when the account is online.
	Status int `json:"status"`

	// AcceptedCase is the number of conversations the account has taken.
	AcceptedCase int `json:"accepted_case"`
}

// kfAccountQuery renders the optional business id shared by the endpoints that
// address one customer service account.
//
// A zero business id is dropped: it means "the mini program itself", not a
// sub-merchant.
func kfAccountQuery(kfOpenID string, businessID int64) url.Values {
	q := url.Values{}
	if kfOpenID != "" {
		q.Set("kf_openid", kfOpenID)
	}

	if businessID > 0 {
		q.Set("business_id", strconv.FormatInt(businessID, 10))
	}

	return q
}
