package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// GetQRCode 获取小程序码（限量接口），成功时返回图片二进制。
//
// path 必须是已发布小程序中存在的页面，且不能以 "/" 开头，否则返回 41030。
// 如果需要在运行时为大量用户生成带参码，应改用 /wxa/getwxacodeunlimit。
//
// 返回的 Image 是原始图片字节（默认 JPEG，IsHyaline 为 true 时是 PNG），
// 直接落盘或作为 HTTP 响应输出即可，不要当 JSON 解析。
func GetQRCode(
	ctx context.Context,
	app *miniProgram.Application,
	path string,
) error {
	res, err := app.GetQRCode(ctx, miniProgram.GetQRCodeParams{
		Path: path,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetQRCodeError(res)
	}

	log.Printf("GetQRCode: %d bytes, content type %s", len(res.Image), res.ContentType)

	return nil
}
