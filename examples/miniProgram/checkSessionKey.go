package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// CheckSessionKey 检验登录态是否仍然有效。
//
// signature 由服务端持有的 session_key 通过
// miniProgram.CheckSessionSignature(sessionKey) 计算，无需客户端传入。
func CheckSessionKey(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
	signature string,
) error {
	res, err := app.CheckSessionKey(ctx, openID, signature)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.CheckSessionKeyError(res)
	}

	log.Printf("CheckSessionKey response: %+v", res)

	return nil
}
