package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// GenerateShortLink 获取小程序 Short Link，适用于微信内拉起小程序的场景。
//
// 与 URL Link 不同，short link 只在微信内有效，但更短、更适合放进卡片或聊天
// 消息。pageURL 直接写页面路径并可携带 query，如
// "pages/index/index?foo=bar"，不要加前导斜杠，且该页面必须存在于已发布的小程序。
//
// pageTitle 超过 20 字符会被微信用“…”截断。isPermanent 为 false（默认）时链接
// 30 天有效；为 true 时永久有效，但单个小程序累计上限 10 万条，每日生成总量
// 100 万条（短期与永久合计）。
//
// 该能力目前只对电商类目开放（电商平台、商家自营、跨境电商），其余主体微信
// 返回 43104；页面不存在返回 40066。
func GenerateShortLink(
	ctx context.Context,
	app *miniProgram.Application,
	pageURL string,
	pageTitle string,
	isPermanent bool,
) error {
	res, err := app.GenerateShortLink(ctx, miniProgram.GenerateShortLinkParams{
		PageURL:     pageURL,
		PageTitle:   pageTitle,
		IsPermanent: isPermanent,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GenerateShortLinkError(res)
	}

	log.Printf("GenerateShortLink: %s", res.Link)

	return nil
}
