package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// CheckEncryptedData 检查一段加密数据是否由微信生成。
//
// 入参 encryptedData 是前端拿到的加密数据原文（例如手机号组件返回的
// encryptedData），SDK 会先算 sha256 的 hex 再传给微信；只支持手机号
// 加密数据，且只能校验最近 3 天生成的数据。
//
// vaild 为 false 时 errcode 仍是 0，表示微信明确否认这段数据，不是调用失败。
func CheckEncryptedData(
	ctx context.Context,
	app *miniProgram.Application,
	encryptedData string,
) error {
	res, err := app.CheckEncryptedData(ctx, miniProgram.EncryptedMsgHash([]byte(encryptedData)))
	if err != nil {
		return err
	}
	if res.ErrCode != 0 {
		return miniProgram.CheckEncryptedDataError(res)
	}

	if !res.Valid {
		log.Printf("CheckEncryptedData: 数据不是微信生成的合法数据")

		return nil
	}

	log.Printf("CheckEncryptedData valid, create time: %d", res.CreateTime)

	return nil
}
