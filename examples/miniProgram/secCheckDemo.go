package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// SecCheckDemo 演示小程序安全：文本同步检测、多媒体异步检测、用户风险等级。
//
// 三个接口的判定结果都不是错误：errcode 为 0 且带回结论，调用方按结论决定放行、
// 拦截还是转人工。
func SecCheckDemo(ctx context.Context, app *miniProgram.Application) {
	const openID = "oKGug4tqqGYFCB46wZLoMBC5XN9Q"

	// 昵称、评论一类短文本：同步拿结论。
	text, err := app.MsgSecCheck(ctx, miniProgram.MsgSecCheckParams{
		Content:  "这是一段待检测的文本",
		Scene:    miniProgram.MsgSecCheckSceneProfile,
		OpenID:   openID,
		Nickname: "用户昵称",
	})
	if err != nil {
		log.Printf("文本检测失败: %v", err)

		return
	}

	if !text.OK() {
		log.Printf("文本检测失败: %s", miniProgram.MsgSecCheckErrMsg(text.ErrCode))

		return
	}

	switch {
	case text.Passed():
		log.Print("文本通过")
	case text.Risky():
		// 结论命中违规，label 说明命中的类别。
		log.Printf("文本违规：%s", text.Verdict.LabelMsg())
	default:
		log.Printf("文本可疑，转人工：%s", text.Verdict.LabelMsg())
	}

	// 多媒体是异步的：调用只拿到 trace_id，结论 30 分钟内推送到事件服务器。
	media, err := app.MediaCheckAsync(ctx, miniProgram.MediaCheckAsyncParams{
		MediaURL:  "https://example.com/avatar.png",
		MediaType: miniProgram.MediaCheckTypeImage,
		Scene:     miniProgram.MsgSecCheckSceneProfile,
		OpenID:    openID,
	})
	if err != nil {
		log.Printf("提交多媒体检测失败: %v", err)

		return
	}

	if !media.OK() {
		log.Printf("提交多媒体检测失败: %s", miniProgram.MediaCheckAsyncErrMsg(media.ErrCode))

		return
	}

	// trace_id 要落库，推送到达时靠它对应回是哪一次提交。
	log.Printf("已提交多媒体检测，trace_id = %s", media.TraceID)

	// 结论推送的处理器，注册一次即可；推送由事件服务器分发（见 eventNotify 示例）。
	app.OnMediaCheckResult(func(ctx context.Context, evt *miniProgram.MediaCheckResultEvent) error {
		switch {
		case evt.DownloadFailed():
			// 微信下载不到素材，等于没检测：不要当成通过。
			log.Printf("多媒体检测未能执行（下载失败），trace_id = %s", evt.TraceID)
		case evt.Passed():
			log.Printf("多媒体通过，trace_id = %s", evt.TraceID)
		case evt.Risky():
			log.Printf("多媒体违规：%s，trace_id = %s", evt.Verdict.LabelMsg(), evt.TraceID)
		default:
			log.Printf("多媒体可疑，转人工：%s，trace_id = %s", evt.Verdict.LabelMsg(), evt.TraceID)
		}

		return nil
	})

	// 风险等级：0 最可信，4 风险最高，数字越大越可疑。
	risk, err := app.GetUserRiskRank(ctx, miniProgram.GetUserRiskRankParams{
		OpenID:   openID,
		Scene:    miniProgram.UserRiskSceneRegister,
		ClientIP: "203.0.113.7",
		// 反代场景取 X-Forwarded-For 的第一段并去掉端口，只传单个 IP。
	})
	if err != nil {
		log.Printf("查询风险等级失败: %v", err)

		return
	}

	if rank, ok := risk.Rank(); ok {
		log.Printf("用户风险等级 = %d", rank)
	} else {
		// 没有结论，不是「等级 0」：按业务决定放行还是拒绝。
		log.Printf("未取得风险等级: %s", miniProgram.GetUserRiskRankErrMsg(risk.ErrCode))
	}
}
