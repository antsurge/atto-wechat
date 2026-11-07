package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// GenerateNFCScheme 获取 NFC 的小程序 scheme 码，把返回的链接写入 NFC 标签后，
// 手机触碰标签即可打开小程序页面。
//
// 与生成普通 scheme 不同，这里必须传 modelID（NFC 标签对应设备的 model_id），
// 且只有已申请 NFC 能力、已发布的非个人主体小程序才能生成。SN 是否必填取决于
// 该 model 的烧写模式：一机一码必须传，一型一码必须不传，填错微信会以
// 9800008 / 9800009 拒绝。
//
// 结果在 res.OpenLink 中。长期有效的 scheme 与 short link 共享 10 万条额度。
func GenerateNFCScheme(
	ctx context.Context,
	app *miniProgram.Application,
	path string,
	modelID string,
	sn string,
) error {
	res, err := app.GenerateNFCScheme(ctx, miniProgram.GenerateNFCSchemeParams{
		JumpWXA: &miniProgram.JumpWXA{
			Path: path,
		},
		ModelID: modelID,
		SN:      sn,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GenerateNFCSchemeError(res)
	}

	log.Printf("GenerateNFCScheme: %s", res.OpenLink)

	return nil
}
