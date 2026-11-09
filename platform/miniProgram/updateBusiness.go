package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// updateBusinessPath updates a 客服子商户.
const updateBusinessPath = "/cgi-bin/business/update"

// UpdateBusinessParams describes what to change.
//
// An empty field is left untouched, which is why both fields are optional: WeChat
// reads an absent value as "keep the current one", not as "clear it".
type UpdateBusinessParams struct {
	// BusinessID is the merchant to update.
	BusinessID int64

	// Nickname replaces the displayed name when set.
	Nickname string

	// IconMediaID replaces the avatar when set, uploaded through UploadTempMedia.
	IconMediaID string
}

// CheckUpdateBusinessParams validates params before anything is sent to WeChat.
func CheckUpdateBusinessParams(params UpdateBusinessParams) error {
	if params.BusinessID == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: UpdateBusiness business_id is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
//
// The two optional strings are sent as empty rather than dropped: WeChat documents
// an empty value as "do not update", so both spellings behave the same, and
// keeping them makes the body match the documented example.
func (p UpdateBusinessParams) body() *updateBusinessBody {
	return &updateBusinessBody{
		BusinessID:  p.BusinessID,
		Nickname:    p.Nickname,
		IconMediaID: p.IconMediaID,
	}
}

// updateBusinessBody is the JSON body of /cgi-bin/business/update.
type updateBusinessBody struct {
	BusinessID  int64  `json:"business_id"`
	Nickname    string `json:"nickname"`
	IconMediaID string `json:"icon_media_id"`
}

// UpdateBusinessResponse is the reply of /cgi-bin/business/update.
//
// It carries no business payload: success is reported by errcode alone.
type UpdateBusinessResponse struct {
	response.WechatResponse
}

// OK reports whether the merchant was updated.
func (r *UpdateBusinessResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// UpdateBusinessErrMsg returns the description of an updateBusiness error.
func UpdateBusinessErrMsg(code int) string {
	return BusinessErrMsg(code)
}

// UpdateBusinessError turns the errcode from the reply into an error.
func UpdateBusinessError(r *UpdateBusinessResponse) error {
	return errors.New(UpdateBusinessErrMsg(r.ErrCode))
}

// UpdateBusiness changes the nickname or avatar of a 客服子商户.
//
// Leaving both fields empty is accepted and changes nothing; that is WeChat's
// documented behaviour rather than an error, so it is not rejected here.
func (a *Application) UpdateBusiness(
	ctx context.Context,
	params UpdateBusinessParams,
) (*UpdateBusinessResponse, error) {
	if err := CheckUpdateBusinessParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, updateBusinessPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result UpdateBusinessResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
