package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// GetUnlimitedQRCode 获取不限制的小程序码（接口），成功时返回图片二进制。
//
// scene 是扫码后传给小程序页面的 query 参数，页面通过 onLoad(query) 里的
// query.scene 取值（需要 decodeURIComponent）。它只支持数字、大小写英文和
// !#$&'()*+,/:;=?@-._~，中文等字符需要先自行编码为这些字符。
//
// page 默认主页，不能以 "/" 开头、不能带参数；check_path 默认为 true，会校验
// page 是否存在于已发布的小程序。返回的 Image 是原始图片字节（默认 JPEG，
// IsHyaline 为 true 时是 PNG），直接落盘或作为 HTTP 响应输出即可。
func GetUnlimitedQRCode(
	ctx context.Context,
	app *miniProgram.Application,
	scene string,
) error {
	res, err := app.GetUnlimitedQRCode(ctx, miniProgram.GetUnlimitedQRCodeParams{
		Scene: scene,
		Page:  "pages/index/index",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetUnlimitedQRCodeError(res)
	}

	log.Printf("GetUnlimitedQRCode: %d bytes, content type %s", len(res.Image), res.ContentType)

	return nil
}
