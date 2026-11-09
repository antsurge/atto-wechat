package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// registerBusinessPath creates a 客服子商户.
const registerBusinessPath = "/cgi-bin/business/register"

// RegisterBusinessParams describes the sub-merchant to create.
type RegisterBusinessParams struct {
	// AccountName is the account name, unique within the mini program.
	//
	// One account_name can be created only once (45070), so it is the field the
	// caller has to keep: the business_id it maps to comes back in the reply.
	AccountName string

	// Nickname is shown in the merchant's session. WeChat accepts 4-30
	// characters, counting a Chinese character as two.
	Nickname string

	// IconMediaID is the avatar, uploaded first through UploadTempMedia. WeChat
	// rejects a permanent media id or a non-image asset (40004/40007).
	IconMediaID string

	// TransferToCommKF forwards messages to the mini program's own customer
	// service accounts. With it on, the 客服管理 endpoints take the resulting
	// business_id.
	TransferToCommKF bool
}

// CheckRegisterBusinessParams validates params before anything is sent to WeChat.
//
// Only account_name is checked in full: its rule is plain ASCII. nickname is left
// to WeChat (45078) because the "a Chinese character counts as two" rule leaves
// room for judgement at the boundaries, and a wrong local guess would block the
// creation of an otherwise valid merchant.
func CheckRegisterBusinessParams(params RegisterBusinessParams) error {
	if err := checkAccountName("RegisterBusiness", params.AccountName); err != nil {
		return err
	}

	if params.Nickname == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: RegisterBusiness nickname is required",
		)
	}

	if params.IconMediaID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: RegisterBusiness icon_media_id is required",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p RegisterBusinessParams) body() *registerBusinessBody {
	return &registerBusinessBody{
		AccountName:      p.AccountName,
		Nickname:         p.Nickname,
		IconMediaID:      p.IconMediaID,
		TransferToCommKF: p.TransferToCommKF,
	}
}

// registerBusinessBody is the JSON body of /cgi-bin/business/register.
type registerBusinessBody struct {
	AccountName      string `json:"account_name"`
	Nickname         string `json:"nickname"`
	IconMediaID      string `json:"icon_media_id"`
	TransferToCommKF bool   `json:"transfer_to_commkf,omitempty"`
}

// RegisterBusinessResponse is the reply of /cgi-bin/business/register.
type RegisterBusinessResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	BusinessID int64 `json:"business_id"`
}

// OK reports whether the merchant was created.
func (r *RegisterBusinessResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.BusinessID != 0
}

// RegisterBusinessErrMsg returns the description of a registerBusiness error.
func RegisterBusinessErrMsg(code int) string {
	return BusinessErrMsg(code)
}

// RegisterBusinessError turns the errcode from the reply into an error.
func RegisterBusinessError(r *RegisterBusinessResponse) error {
	return errors.New(RegisterBusinessErrMsg(r.ErrCode))
}

// RegisterBusiness creates a 客服子商户 under the mini program.
//
// The merchant gets its own session in the 小程序客服 component, and its messages
// are answered by the accounts added with AddKFAccount(business_id). Only mini
// programs in the 电商平台 category may call this, which WeChat enforces rather
// than advertises in the reply.
func (a *Application) RegisterBusiness(
	ctx context.Context,
	params RegisterBusinessParams,
) (*RegisterBusinessResponse, error) {
	if err := CheckRegisterBusinessParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, registerBusinessPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result RegisterBusinessResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
