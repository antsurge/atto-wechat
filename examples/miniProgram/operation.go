package main

import (
	"context"
	"log"
	"time"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// Operation 演示运维中心接口：域名、性能、日志、反馈、JS 错误与分阶段发布。
func Operation(ctx context.Context, app *miniProgram.Application) {
	// 1. 域名：客户端请求被拦时，服务端只看到"没请求上来"，先查配置。
	domains, err := app.GetDomainInfo(ctx, miniProgram.GetDomainInfoParams{
		Action: miniProgram.DomainActionServer,
	})
	if err != nil {
		log.Printf("查询域名失败: %v", err)

		return
	}

	for _, domain := range []string{"https://api.example.com", "https://cdn.example.com"} {
		if !domains.ContainsServerDomain(domain) {
			log.Printf("域名未配置: %s", domain)
		}
	}

	// 2. 访问来源与客户端版本是参照表，取一次缓存起来。
	scenes, err := app.GetSceneList(ctx)
	if err != nil || !scenes.OK() {
		log.Printf("查询访问来源失败: %v", err)

		return
	}

	scene, ok := scenes.Find("1011")
	if !ok {
		log.Print("没有 1011 这个场景")

		return
	}

	log.Printf("场景 %s 可用于筛选", scene.Name)

	versions, err := app.GetVersionList(ctx)
	if err != nil || !versions.OK() {
		log.Printf("查询客户端版本失败: %v", err)

		return
	}

	log.Printf("客户端版本: %v", versions.Versions(miniProgram.ClientVersionTypeClient))

	// 3. 性能：两个折线要自己解析，微信把 JSON 又塞进了 JSON 字符串里。
	now := time.Now()
	perf, err := app.GetPerformance(ctx, miniProgram.GetPerformanceParams{
		CostTimeType:     miniProgram.PerfCostTimeStartup,
		DefaultStartTime: now.AddDate(0, 0, -7).Unix(),
		DefaultEndTime:   now.Unix(),
	})
	if err != nil || !perf.OK() {
		log.Printf("查询性能失败: %v", err)

		return
	}

	samples, err := perf.Samples()
	if err != nil {
		log.Printf("解析性能数据失败: %v", err)

		return
	}

	for _, sample := range samples {
		log.Printf("%s 启动耗时 %dms", sample.RefDate, sample.CostTime)
	}

	// 4. 实时日志：先按 level 过滤，再按 traceid 还原一次会话。
	logs, err := app.RealTimeLogSearch(ctx, miniProgram.RealTimeLogSearchParams{
		Date:      now.Format("20060102"),
		BeginTime: now.Add(-2 * time.Hour).Unix(),
		EndTime:   now.Unix(),
		Limit:     20,
		Level:     miniProgram.LogLevelError,
	})
	if err != nil || !logs.OK() {
		log.Printf("查询实时日志失败: %v", err)

		return
	}

	for _, record := range logs.Data.List {
		// 一条记录的 level 是位或，可能是 6（info+warn），不能直接比较。
		for _, message := range record.Messages(miniProgram.LogLevelError) {
			log.Printf("[%s] %s %v", record.TraceID, record.URL, message.Msg)
		}
	}

	// 5. 用户反馈：先翻页，再按 record_id + media_id 取图片。
	feedback, err := app.GetFeedback(ctx, miniProgram.GetFeedbackParams{
		Type: miniProgram.FeedbackTypeCannotOpen,
		Page: 1,
		Num:  20,
	})
	if err != nil || !feedback.OK() {
		log.Printf("查询反馈失败: %v", err)

		return
	}

	for _, item := range feedback.List {
		log.Printf("反馈 %d: %s（%s）",
			item.RecordID, item.Content, miniProgram.FeedbackTypeMsg(item.Type))

		if !item.HasMedia() {
			continue
		}

		media, err := app.GetFeedbackMedia(ctx, miniProgram.GetFeedbackMediaParams{
			RecordID: item.RecordID,
			MediaID:  item.MediaIDs[0],
		})
		if err != nil {
			log.Printf("下载反馈图片失败: %v", err)

			continue
		}

		if !media.OK() {
			log.Printf("下载反馈图片被拒: %s", miniProgram.GetFeedbackMediaError(media))

			continue
		}

		log.Printf("图片 %d 字节，类型 %s", len(media.Image), media.ContentType)
	}

	// 6. JS 错误：列表给 md5，详情按 md5 展开。
	errList, err := app.GetJsErrList(ctx, miniProgram.GetJsErrListParams{
		StartTime: now.AddDate(0, 0, -7).Format("2006-01-02"),
		EndTime:   now.Format("2006-01-02"),
		ErrType:   miniProgram.JsErrTypeBusiness,
	})
	if err != nil || !errList.OK() {
		log.Printf("查询错误列表失败: %v", err)

		return
	}

	log.Printf("共 %d 个错误", errList.TotalCount)

	if len(errList.Data) == 0 {
		return
	}

	top := errList.Data[0]

	detail, err := app.GetJsErrDetail(ctx, miniProgram.GetJsErrDetailParams{
		StartTime:     now.AddDate(0, 0, -7).Format("2006-01-02"),
		EndTime:       now.Format("2006-01-02"),
		ErrorMsgMD5:   top.ErrorMsgMD5,
		ErrorStackMD5: top.ErrorStackMD5,
		OSName:        miniProgram.JsErrOSAndroid,
	})
	if err != nil || !detail.OK() {
		log.Printf("查询错误详情失败: %v", err)

		return
	}

	for _, item := range detail.Data {
		log.Printf("%s %s %s 出现 %s 次", item.TimeStamp, item.OSName, item.AppVersion, item.Count)
	}

	// 7. 分阶段发布：发布前确认没有正在执行的计划。
	plan, err := app.GetGrayReleasePlan(ctx)
	if err != nil || !plan.OK() {
		log.Printf("查询分阶段发布失败: %v", err)

		return
	}

	if !plan.HasPlan() {
		log.Print("没有分阶段发布计划")

		return
	}

	log.Printf("灰度 %d%%，状态 %s",
		plan.GrayReleasePlan.GrayPercentage, miniProgram.GrayReleaseStatusMsg(plan.GrayReleasePlan.Status))
}
