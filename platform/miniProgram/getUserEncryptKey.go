package miniProgram

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/antsurge/atto-wechat/core/client"
	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/middleware"
	"github.com/antsurge/atto-wechat/core/response"
)

// getUserEncryptKeyPath is the encrypt key endpoint: it hands back the keys the
// server needs to decrypt the encrypted network channel.
const (
	getUserEncryptKeyPath          = "/wxa/business/getuserencryptkey"
	GetUserEncryptKeyDefaultErrMsg = "系统错误，请稍后再试"
	userEncryptKeySigMethod        = "hmac_sha256"
)

// GetUserEncryptKeyErrCodeMap holds the errcode descriptions WeChat documents
// for /wxa/business/getuserencryptkey, keyed by the errcode as it appears in
// the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var GetUserEncryptKeyErrCodeMap = map[int]string{
	-1:    "系统繁忙",
	40001: "access_token 无效",
	40003: "openid 错误",
	45011: "API 调用太频繁，请稍候再试",
	87007: "加密 key 不存在或已过期（session_key 失效）",
	87008: "无效的签名方法",
	87009: "无效的签名",
}

// UserEncryptKey is one entry of key_info_list.
//
// WeChat returns the three most recent keys and only the newest one is usually
// still alive: the older entries carry expire_in 0 once they are dead.
type UserEncryptKey struct {
	EncryptKey string `json:"encrypt_key"`
	Version    int    `json:"version"`
	ExpireIn   int    `json:"expire_in"`
	IV         string `json:"iv"`
	CreateTime int64  `json:"create_time"`
}

// GetUserEncryptKeyResponse is the reply of /wxa/business/getuserencryptkey.
//
// The structure keeps the two kinds of parameters apart: the embedded common
// header is what every WeChat API returns (errcode/errmsg), while the field
// below is specific to this endpoint.
type GetUserEncryptKeyResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	KeyInfoList []UserEncryptKey `json:"key_info_list"`
}

// OK reports whether the response carries at least one key.
func (r *GetUserEncryptKeyResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && len(r.KeyInfoList) > 0
}

// GetUserEncryptKeyErrMsg returns the description of a getUserEncryptKey error.
func GetUserEncryptKeyErrMsg(code int) string {
	if msg, ok := GetUserEncryptKeyErrCodeMap[code]; ok {
		return msg
	}
	return GetUserEncryptKeyDefaultErrMsg
}

// GetUserEncryptKeyError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// GetUserEncryptKeyErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func GetUserEncryptKeyError(r *GetUserEncryptKeyResponse) error {
	return errors.New(GetUserEncryptKeyErrMsg(r.ErrCode))
}

// GetUserEncryptKey returns the encrypt keys of the user behind openID.
//
// signature is CheckSessionSignature(session_key): an HMAC-SHA256 keyed by the
// user's session_key over an empty message, hex encoded. The session_key comes
// from Code2Session, and once it expires WeChat answers 87007.
//
// The request is a POST, but every parameter travels in the query and the body
// must stay empty: WeChat rejects any body, even {}, with 87009. That is also
// why this call builds its own request instead of going through PostJSON.
//
// Keys live for 3600 seconds and WeChat returns the three most recent ones, so
// callers should pick the entry with the highest version or simply the first
// entry, which is the newest.
func (a *Application) GetUserEncryptKey(
	ctx context.Context,
	openID string,
	signature string,
) (*GetUserEncryptKeyResponse, error) {
	if openID == "" {
		return nil, cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: openid must not be empty",
		)
	}
	if signature == "" {
		return nil, cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: signature must not be empty",
		)
	}

	resp, err := a.api.Do(ctx, &client.Request{
		Method: http.MethodPost,
		Path:   getUserEncryptKeyPath,
		Query: url.Values{
			"openid":     {openID},
			"signature":  {signature},
			"sig_method": {userEncryptKeySigMethod},
		},
		// BodyRaw sends no body and no Content-Type: anything resembling a
		// body makes the endpoint report an invalid signature.
		BodyType: middleware.BodyRaw,
	})
	if err != nil {
		return nil, err
	}

	var result GetUserEncryptKeyResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
