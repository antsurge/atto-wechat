package miniProgram

import (
	cerrors "github.com/antsurge/atto-wechat/core/errors"
)

// BusinessDefaultErrMsg is returned for an errcode the 客服子商户 endpoints do
// not document.
const BusinessDefaultErrMsg = "系统错误，请稍后再试"

// BusinessErrCodeMap holds the errcode descriptions WeChat documents for the
// /cgi-bin/business endpoints, keyed by the errcode as it appears in the reply.
//
// The 450xx range is shared by the whole family: creating, updating and reading a
// sub-merchant all answer with the same account-level failures, so the table
// lives here instead of being repeated in four files.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var BusinessErrCodeMap = map[int]string{
	40004: "icon_media_id 类型不对，应为图片类型",
	40007: "icon_media_id 不是合法的 media_id",
	45070: "account_name 对应的商户已被创建",
	45071: "business_id 对应的商户不存在",
	45077: "子商户数量已达到上限",
	45078: "昵称不合法，请检查是否满足 nickname 规则",
	45079: "昵称含有违规词汇",
	45091: "account_name 不合法，请检查是否满足 account_name 规则",
}

// BusinessErrMsg returns the description of a 客服子商户 error.
func BusinessErrMsg(code int) string {
	if msg, ok := BusinessErrCodeMap[code]; ok {
		return msg
	}

	return BusinessDefaultErrMsg
}

// BusinessInfo is one 客服子商户.
type BusinessInfo struct {
	// BusinessID is the id WeChat assigned at creation.
	//
	// WeChat's parameter table calls it a string, but every example in the same
	// document sends and returns a bare number, and so does the rest of this
	// package (see kfAccountQuery). int64 follows the examples.
	BusinessID int64 `json:"business_id"`

	// AccountName is the name given at creation, unique per mini program.
	AccountName string `json:"account_name"`

	// Nickname is what the merchant's session shows to users.
	Nickname string `json:"nickname"`

	// IconMediaID is the avatar's media id. The reply examples return icon_url
	// instead, so this may stay empty on a read.
	IconMediaID string `json:"icon_media_id"`

	// IconURL is the avatar's URL, as it appears in the reply examples.
	IconURL string `json:"icon_url"`
}

// businessIDBody carries the business_id every endpoint of the family but
// register takes.
type businessIDBody struct {
	BusinessID int64 `json:"business_id"`
}

// checkAccountName validates the account_name rule WeChat documents.
//
// The rule is plain ASCII (6-30 characters, letters/digits/underscore), so it is
// checked here: 45091 costs a round trip and says nothing about which part of the
// name is wrong.
func checkAccountName(operation, accountName string) error {
	if len(accountName) < 6 || len(accountName) > 30 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: "+operation+" account_name must be 6-30 characters",
		)
	}

	for i := 0; i < len(accountName); i++ {
		c := accountName[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_':
		default:
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: "+operation+" account_name accepts letters, digits and underscores only",
			)
		}
	}

	return nil
}
