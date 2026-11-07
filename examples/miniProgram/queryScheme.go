package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// QueryScheme 查询 scheme 码（接口），可查加密 scheme 和明文 scheme。
//
// scheme 是之前用生成接口拿到的链接，例如 "weixin://dl/business/?t=xxxx"。
// 查询结果在 res.SchemeInfo 中：appid、path、query、env_version，以及
// create_time / expire_time（expire_time 为 0 表示永久有效）。加密 scheme 的
// query 已由微信解码，可直接使用。
//
// 只想查当天剩余访问次数时，把 QueryType 设为 1 并且不必传 Scheme，结果在
// res.QuotaInfo.RemainVisitQuota 中。查询不存在的 scheme 会返回 85403。
func QueryScheme(
	ctx context.Context,
	app *miniProgram.Application,
	scheme string,
) error {
	res, err := app.QueryScheme(ctx, miniProgram.QuerySchemeParams{
		Scheme: scheme,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.QuerySchemeError(res)
	}

	log.Printf(
		"QueryScheme: appid=%s path=%q query=%q env=%s create=%d expire=%d",
		res.SchemeInfo.AppID,
		res.SchemeInfo.Path,
		res.SchemeInfo.Query,
		res.SchemeInfo.EnvVersion,
		res.SchemeInfo.CreateTime,
		res.SchemeInfo.ExpireTime,
	)

	return nil
}
