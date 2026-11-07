package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// GenerateURLLink 获取加密 URLLink，适用于短信、邮件、网页等外部场景拉起小程序。
//
// 与加密 scheme 不同，页面参数直接放在请求体顶层（path/query/env_version），
// 且返回字段是 url_link。path 必须是已发布小程序中的页面且不能带 query，
// 业务参数全部放 query。
//
// 默认生成永久有效的链接；把 isExpire 改为 true 并给出 expireInterval（天数，
// 最长 30 天）或 expireTime（Unix 时间戳），链接即到期失效。
//
// 结果在 res.UrlLink 中。生成量每天 50 万，与 URL Scheme 共享。
func GenerateURLLink(
	ctx context.Context,
	app *miniProgram.Application,
	path string,
	query string,
) error {
	res, err := app.GenerateURLLink(ctx, miniProgram.GenerateURLLinkParams{
		Path:  path,
		Query: query,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GenerateURLLinkError(res)
	}

	log.Printf("GenerateURLLink: %s", res.UrlLink)

	return nil
}
