package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// OpenAPIMgnt 演示接口调用凭证之外的一类运维接口：额度、重置、排障。
//
// 这些接口解决的是同一个问题——线上调用失败。先用 GetAPIQuota 看是不是额度用完了，
// 用 GetRIDInfo 看微信侧到底收到了什么，再用 ClearAPIQuota 把额度清掉重试。
func OpenAPIMgnt(ctx context.Context, app *miniProgram.Application) {
	// 稳定版凭据：本包自己的 api 调用用的是 AccessToken（/cgi-bin/token），
	// 两者互相隔离。只有需要把 token 交给包外使用时才调这个。
	stable, err := app.GetStableAccessToken(ctx, false)
	if err != nil {
		log.Printf("获取稳定版凭据失败: %v", err)

		return
	}

	if !stable.OK() {
		log.Printf("获取稳定版凭据失败: %s",
			miniProgram.StableAccessTokenErrMsg(stable.ErrCode))

		return
	}

	log.Printf("稳定版凭据有效期 %d 秒", stable.ExpiresIn)

	// 查额度：cgi_path 只写路径，不要带 https://api.weixin.qq.com。
	quota, err := app.GetAPIQuota(ctx, miniProgram.GetAPIQuotaParams{
		CGIPath: "/cgi-bin/message/custom/send",
	})
	if err != nil {
		log.Printf("查询额度失败: %v", err)

		return
	}

	if !quota.OK() {
		log.Printf("查询额度失败: %s", miniProgram.GetAPIQuotaErrMsg(quota.ErrCode))

		return
	}

	log.Printf("每日上限 %d，已用 %d，剩余 %d；频率上限 %d 次 / %d 秒",
		quota.Quota.DailyLimit, quota.Quota.Used, quota.Quota.Remain,
		quota.RateLimit.CallCount, quota.RateLimit.RefreshSecond)

	// 额度确实用完了，才重置；重置本身也有次数上限（48006）。
	if quota.Quota.Remain == 0 {
		reset, err := app.ClearAPIQuota(ctx, miniProgram.ClearAPIQuotaParams{
			CGIPath: "/cgi-bin/message/custom/send",
		})
		if err != nil {
			log.Printf("重置额度失败: %v", err)

			return
		}

		if !reset.OK() {
			log.Printf("重置额度失败: %s", miniProgram.ClearAPIQuotaErrMsg(reset.ErrCode))
		}
	}

	// 报错里带 rid 时，用它看微信侧的请求与响应原文。
	rid, err := app.GetRIDInfo(ctx, miniProgram.GetRIDInfoParams{
		RID: "61725984-6126f6f9-040f19c4",
	})
	if err != nil {
		log.Printf("查询 rid 失败: %v", err)
	} else if rid.OK() {
		log.Printf("rid 请求 %s，耗时 %dms，返回 %s",
			rid.Request.RequestURL, rid.Request.CostInMS, rid.Request.ResponseBody)
	}

	// 回调地址收不到推送时，先跑一次网络检测：DNS 解析不出来是域名问题，
	// ping 全丢是网络问题。
	check, err := app.CallbackCheck(ctx, miniProgram.CallbackCheckParams{
		Action:        miniProgram.CallbackCheckActionAll,
		CheckOperator: miniProgram.CallbackCheckOperatorDefault,
	})
	if err != nil {
		log.Printf("网络检测失败: %v", err)
	} else if check.OK() {
		for _, dns := range check.DNS {
			log.Printf("解析到 %s（%s）", dns.IP, dns.RealOperator)
		}

		for _, ping := range check.Ping {
			log.Printf("ping %s 丢包 %s", ping.IP, ping.PackageLoss)
		}
	}

	// 出方向与入方向的 IP 列表，供防火墙/安全组使用；列表会变，别写死。
	apiIPs, err := app.GetAPIDomainIP(ctx)
	if err != nil {
		log.Printf("获取 API 服务器 IP 失败: %v", err)
	} else if apiIPs.OK() {
		log.Printf("API 服务器 IP: %v", apiIPs.IPList)
	}

	callbackIPs, err := app.GetCallbackIP(ctx)
	if err != nil {
		log.Printf("获取推送服务器 IP 失败: %v", err)
	} else if callbackIPs.OK() {
		log.Printf("推送服务器 IP: %v", callbackIPs.IPList)
	}
}
