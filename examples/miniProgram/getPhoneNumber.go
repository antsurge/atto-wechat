package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// GetPhoneNumber 用手机号获取凭证换取用户手机号。
//
// code 来自小程序 button open-type="getPhoneNumber" 的 bindgetphonenumber
// 回调，不是 wx.login 的 code；单次有效、5 分钟过期，且与获取它的 AppID 绑定。
func GetPhoneNumber(ctx context.Context, app *miniProgram.Application, code string) error {
	res, err := app.GetPhoneNumber(ctx, code)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetPhoneNumberError(res)
	}

	log.Printf(
		"GetPhoneNumber phone: %s, country code: %s",
		res.PurePhoneNumber,
		res.CountryCode,
	)

	return nil
}
