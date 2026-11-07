package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// GetUserEncryptKey 获取用户最近三次的 encrypt key，用于解密小程序加密网络通道。
//
// sessionKey 是 Code2Session 返回的 session_key；签名由它对空字符串做
// HMAC-SHA256 得到，可直接用 miniProgram.CheckSessionSignature。
//
// key 只有 3600 秒有效期，过期后需重新获取；session_key 失效时微信返回 87007。
func GetUserEncryptKey(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
	sessionKey string,
) error {
	res, err := app.GetUserEncryptKey(ctx, openID, miniProgram.CheckSessionSignature(sessionKey))
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetUserEncryptKeyError(res)
	}

	// key_info_list 里第一个是最新版本；只打印版本与有效期，不打印密钥本身。
	key := res.KeyInfoList[0]

	log.Printf(
		"GetUserEncryptKey version: %d, expire in %d s, created at %d",
		key.Version,
		key.ExpireIn,
		key.CreateTime,
	)

	return nil
}
