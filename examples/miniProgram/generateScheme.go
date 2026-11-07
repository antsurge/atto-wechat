package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// GenerateScheme 获取加密 scheme 码（接口），返回的链接可在短信、邮件、外部网页
// 等场景拉起小程序。
//
// path 必须是已发布小程序中的页面，且不能带 query，业务参数放 query 字段：
// query 最大 1024 字符，只支持数字、大小写英文及部分特殊字符。这里不设置
// IsExpire，生成的是永久有效的链接；需要定期失效时用 IsExpire + ExpireType
// 选择「到期时间」（ExpireTime，Unix 时间戳）或「有效天数」（ExpireInterval，
// 最长 30 天），两者只能选一个。
//
// 结果在 res.OpenLink 中，形如 "weixin://dl/business/?t=xxxx"。只有已发布的
// 非个人主体小程序才能生成，且每天 50 万条额度与 URL Link 共享。
func GenerateScheme(
	ctx context.Context,
	app *miniProgram.Application,
	path string,
	query string,
) error {
	res, err := app.GenerateScheme(ctx, miniProgram.GenerateSchemeParams{
		JumpWXA: &miniProgram.JumpWXA{
			Path:  path,
			Query: query,
		},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GenerateSchemeError(res)
	}

	log.Printf("GenerateScheme: %s", res.OpenLink)

	return nil
}
