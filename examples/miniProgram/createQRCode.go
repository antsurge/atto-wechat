package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// CreateQRCode 获取小程序二维码（接口），成功时返回图片二进制。
//
// 这是最基础的小程序码接口：只能生成黑白二维码样式，不支持 auto_color、
// line_color、is_hyaline 等美化参数，也没有 env_version。它与
// GetQRCode 共享 10 万张的总额度，额度用尽不会重置，因此只适合码数量较少且
// 可以提前生成的场景；需要按用户动态生成时应改用 GetUnlimitedQRCode。
//
// path 最大 128 字节，且必须是已发布小程序中的页面；小游戏可以只传 query，
// 例如 "?foo=bar"。width 默认 430，范围 280~1280。返回的 Image 是原始
// 图片字节（JPEG），直接落盘或作为 HTTP 响应输出即可。
func CreateQRCode(
	ctx context.Context,
	app *miniProgram.Application,
	path string,
) error {
	res, err := app.CreateQRCode(ctx, miniProgram.CreateQRCodeParams{
		Path: path,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.CreateQRCodeError(res)
	}

	log.Printf("CreateQRCode: %d bytes, content type %s", len(res.Image), res.ContentType)

	return nil
}
