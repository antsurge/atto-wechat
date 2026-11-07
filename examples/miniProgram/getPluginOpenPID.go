package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// GetPluginOpenPID 用插件登录凭证换取插件用户的唯一标识 openpid。
//
// code 由插件内的 wx.pluginLogin 获取（不是小程序 wx.login 的 code），
// 5 分钟有效、一次性使用；openpid 是插件维度的用户标识，不是 openid。
func GetPluginOpenPID(ctx context.Context, app *miniProgram.Application, code string) error {
	res, err := app.GetPluginOpenPID(ctx, code)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetPluginOpenPIDError(res)
	}

	log.Printf("GetPluginOpenPID openpid: %s", res.OpenPID)

	return nil
}
