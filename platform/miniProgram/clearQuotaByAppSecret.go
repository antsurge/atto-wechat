package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// clearQuotaByAppSecretPath resets the daily call count without a token.
const clearQuotaByAppSecretPath = "/cgi-bin/clear_quota/v2"

// ClearQuotaByAppSecretParams identifies the account whose quota is reset.
//
// It takes the appid and secret rather than a token, which is the point of the
// endpoint: it works when the token itself is what stopped working.
type ClearQuotaByAppSecretParams struct {
	// AppID is the account to clear.
	AppID string
	// AppSecret is its secret. Leave it empty to use the one from the config.
	AppSecret string
}

// CheckClearQuotaByAppSecretParams validates params before anything is sent to
// WeChat.
func CheckClearQuotaByAppSecretParams(params ClearQuotaByAppSecretParams) error {
	if params.AppID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: ClearQuotaByAppSecret appid is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p ClearQuotaByAppSecretParams) body(appSecret string) *clearQuotaByAppSecretBody {
	return &clearQuotaByAppSecretBody{AppID: p.AppID, AppSecret: appSecret}
}

// clearQuotaByAppSecretBody is the JSON body of /cgi-bin/clear_quota/v2.
type clearQuotaByAppSecretBody struct {
	AppID     string `json:"appid"`
	AppSecret string `json:"appsecret"`
}

// ClearQuotaByAppSecretResponse is the reply of /cgi-bin/clear_quota/v2.
//
// It carries no business payload: success is reported by errcode alone.
type ClearQuotaByAppSecretResponse struct {
	response.WechatResponse
}

// OK reports whether the quota was reset.
func (r *ClearQuotaByAppSecretResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// ClearQuotaByAppSecretErrMsg returns the description of a
// clearQuotaByAppSecret error.
func ClearQuotaByAppSecretErrMsg(code int) string {
	return OpenAPIErrMsg(code)
}

// ClearQuotaByAppSecretError turns the errcode from the reply into an error.
func ClearQuotaByAppSecretError(r *ClearQuotaByAppSecretResponse) error {
	return errors.New(ClearQuotaByAppSecretErrMsg(r.ErrCode))
}

// ClearQuotaByAppSecret resets the daily call count using the app secret.
//
// It is the recovery path for a service that can no longer obtain a token: the
// call neither needs nor returns one. When params.AppSecret is empty the
// configured secret is used, which covers the ordinary single-account case.
func (a *Application) ClearQuotaByAppSecret(
	ctx context.Context,
	params ClearQuotaByAppSecretParams,
) (*ClearQuotaByAppSecretResponse, error) {
	if err := CheckClearQuotaByAppSecretParams(params); err != nil {
		return nil, err
	}

	secret := params.AppSecret
	if secret == "" {
		secret = a.cfg.AppSecret
	}

	// The endpoint authenticates in the body, so it goes through the client that
	// does not attach a token.
	resp, err := a.base.PostJSON(ctx, clearQuotaByAppSecretPath, nil, params.body(secret))
	if err != nil {
		return nil, err
	}

	var result ClearQuotaByAppSecretResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
