package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// QueryURLLink 查询加密 URLLink 的配置。
//
// queryType 为 0（默认）时返回链接指向的 appid、页面、query 与版本，可以把
// 一条 https://wxaurl.cn/... 链接映射回它打开的页面；为 1 时返回当天剩余访问
// 次数（与 URL Scheme 共享配额），此时不需要传 urlLink。
//
// 不存在或已过期的链接会返回 85403。该接口只读取此前生成的结果，不会新建链接。
func QueryURLLink(
	ctx context.Context,
	app *miniProgram.Application,
	urlLink string,
	queryType int,
) error {
	res, err := app.QueryURLLink(ctx, miniProgram.QueryURLLinkParams{
		URLLink:   urlLink,
		QueryType: queryType,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.QueryURLLinkError(res)
	}

	if res.URLLinkInfo != nil {
		info := res.URLLinkInfo
		cloud := ""
		if info.CloudBase != nil {
			cloud = info.CloudBase.Env
		}

		log.Printf(
			"QueryURLLink: appid=%s path=%q query=%q env=%s create=%d expire=%d cloud=%q",
			info.AppID,
			info.Path,
			info.Query,
			info.EnvVersion,
			info.CreateTime,
			info.ExpireTime,
			cloud,
		)
	} else {
		log.Printf("QueryURLLink: remain visit quota = %d", res.QuotaInfo.RemainVisitQuota)
	}

	return nil
}
