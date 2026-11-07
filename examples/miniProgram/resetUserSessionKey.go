package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// ResetUserSessionKey 重置登录态：旧的 session_key 立即失效，微信返回新的。
//
// 新 session_key 继承原有过期时间，SDK 会自动写回本地会话，后续
// checksession、解密数据都能继续使用，不需要用户重新 wx.login。
// signature 由服务端持有的 session_key 计算：
// miniProgram.CheckSessionSignature(sessionKey)。
func ResetUserSessionKey(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
	signature string,
) error {
	res, err := app.ResetUserSessionKey(ctx, openID, signature)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ResetUserSessionKeyError(res)
	}

	log.Printf("ResetUserSessionKey response: %+v", res)

	return nil
}
