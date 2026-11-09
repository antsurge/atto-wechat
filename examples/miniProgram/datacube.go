package main

import (
	"context"
	"log"
	"sort"
	"time"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// Datacube 演示数据分析接口：概况、趋势、留存、分布、画像与性能。
//
// 所有日期都是 yyyymmdd，而且各接口只接受一种区间形态：单日、自然周、自然月，
// 画像固定为最近 1/7/30 天。区间不对请求会在本地被拒，不必等线上报错。
func Datacube(ctx context.Context, app *miniProgram.Application) {
	yesterday := time.Now().AddDate(0, 0, -1)
	day := miniProgram.DatacubeDateRange{
		BeginDate: yesterday.Format(miniProgram.DatacubeDateLayout),
		EndDate:   yesterday.Format(miniProgram.DatacubeDateLayout),
	}

	summary, err := app.GetDailySummary(ctx, day)
	if err != nil {
		log.Printf("拉取概况失败: %v", err)

		return
	}

	for _, item := range summary.List {
		log.Printf("%s 累计用户 %d，转发 %d 次", item.RefDate, item.VisitTotal, item.SharePV)
	}

	trend, err := app.GetDailyVisitTrend(ctx, day)
	if err != nil {
		log.Printf("拉取日趋势失败: %v", err)

		return
	}

	for _, item := range trend.List {
		log.Printf("%s 打开 %d 次，访客 %d 人，人均停留 %.1fs",
			item.RefDate, item.SessionCnt, item.VisitUV, item.StayTimeUV)
	}

	// 单日留存：key 0 是当天，1-7、14、30 是之后第 N 天回访的人数。
	retention, err := app.GetDailyRetain(ctx, day)
	if err != nil {
		log.Printf("拉取日留存失败: %v", err)
	} else {
		for _, point := range retention.VisitUV {
			log.Printf("第 %d 天回访 %d 人", point.Key, point.Value)
		}
	}

	// 访问来源：key 的含义由 index 决定，用 DistributionBucketMsg 翻译。
	distribution, err := app.GetVisitDistribution(ctx, day)
	if err != nil {
		log.Printf("拉取访问分布失败: %v", err)
	} else {
		sources := distribution.Items(miniProgram.VisitDistributionIndexSource)
		sort.Slice(sources, func(i, j int) bool { return sources[i].Value > sources[j].Value })

		for _, item := range sources {
			log.Printf("来源 %s：%d 次",
				miniProgram.DistributionBucketMsg(miniProgram.VisitDistributionIndexSource, item.Key),
				item.Value)
		}
	}

	// 画像只支持最近 1/7/30 天，这里取最近 7 天。
	portrait, err := app.GetUserPortrait(ctx, miniProgram.DatacubeDateRange{
		BeginDate: yesterday.AddDate(0, 0, -6).Format(miniProgram.DatacubeDateLayout),
		EndDate:   day.EndDate,
	})
	if err != nil {
		log.Printf("拉取用户画像失败: %v", err)
	} else if portrait.VisitUV != nil {
		for _, province := range portrait.VisitUV.Province {
			log.Printf("活跃用户 %s：%d 人", province.Name, province.Value)
		}
	}

	// 性能数据的时间戳是 Unix 秒，跨度上限 30 天。
	performance, err := app.GetPerformanceData(ctx, miniProgram.GetPerformanceDataParams{
		Module:         miniProgram.PerfModuleMemory,
		BeginTimestamp: yesterday.AddDate(0, 0, -6).Unix(),
		EndTimestamp:   yesterday.Unix(),
		Filters: []miniProgram.GetPerformanceDataFilter{
			{Field: miniProgram.PerfFieldNetworkType, Value: "wifi"},
			{Field: miniProgram.PerfFieldDeviceLevel, Value: "-1"},
		},
	})
	if err != nil {
		log.Printf("拉取性能数据失败: %v", err)

		return
	}

	for _, table := range performance.Data.Body.Tables {
		for _, line := range table.Lines {
			for _, field := range line.Fields {
				value, err := field.Float()
				if err != nil {
					log.Printf("%s %s 解析失败: %v", table.Zh, field.RefDate, err)

					continue
				}

				log.Printf("%s %s：%.4f", table.Zh, field.RefDate, value)
			}
		}
	}
}
