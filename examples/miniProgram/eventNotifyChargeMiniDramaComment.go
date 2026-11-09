package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// ChargeMiniDramaCommentEventNotify 演示付费管理、短剧媒资、小程序评价三个方向的
// 事件接入。
//
// 这几个都是纯推送：处理函数返回 error，不返回 reply，框架回默认的 "success"，
// 正是官方要求的应答。
//
// 需要注意付费管理的两个报文没有 ToUserName / MsgType / Event 头，只有 event_type，
// 框架按 event_type 路由，所以注册方式和别的事件看起来一样，但别指望在报文里读到
// 发送方。
func ChargeMiniDramaCommentEventNotify(app *miniProgram.Application) {
	// 用量告警：余量到 20%、10%、5%、0% 各推一次，是四条报文而不是一条。
	app.OnChargeQuotaNotify(func(ctx context.Context, evt *miniProgram.ChargeQuotaNotifyEvent) error {
		if !evt.OK() {
			log.Printf("用量告警字段不完整: %+v", evt)
			return nil
		}

		log.Printf("%s（SPU %d）余量 %d/%d，剩余 %.1f%%",
			evt.SPUName, evt.SPUID, evt.RemainingQuota(), evt.TotalQuota,
			evt.RemainingPercent())

		recordQuotaAlert(ctx, evt.SPUID, evt.RemainingPercent())

		return nil
	})

	// 有效期告警：到期前 15、7、3、1 天各推一次。
	app.OnChargeValidityNotify(func(ctx context.Context, evt *miniProgram.ChargeValidityNotifyEvent) error {
		if !evt.OK() {
			log.Printf("有效期告警字段不完整: %+v", evt)
			return nil
		}

		// 官方没规定 validity_end_time 的格式，这里原样记录，不要直接 time.Parse。
		log.Printf("%s（SPU %d）将于 %s 到期",
			evt.SPUName, evt.SPUID, evt.ValidityEndTime)

		recordValidityAlert(ctx, evt.SPUID, evt.ValidityEndTime)

		return nil
	})

	// 媒资上传完成：media_id 是后续查/播的入口，失败时 errcode 非 0。
	app.OnMiniDramaAssetUpload(func(ctx context.Context, evt *miniProgram.MiniDramaAssetUploadEvent) error {
		if !evt.OK() {
			log.Printf("媒资上传事件字段不完整: %+v", evt)
			return nil
		}

		upload := evt.Upload()

		// source_context 是上传时自己传的值，用它把推送对回本地记录，
		// media_id 只是微信侧的编号。
		if !upload.Succeeded() {
			log.Printf("媒资上传失败 media_id=%d context=%s: %s",
				upload.MediaID, upload.SourceContext, upload.ErrMsg)
			markMediaUploadFailed(ctx, upload.MediaID, upload.SourceContext, upload.ErrMsg)

			return nil
		}

		log.Printf("媒资上传成功 media_id=%d context=%s", upload.MediaID, upload.SourceContext)
		markMediaUploaded(ctx, upload.MediaID, upload.SourceContext)

		return nil
	})

	// 审核状态：只有 2、3、4 是结论，1（审核中）和 0（无效值）都要继续等。
	app.OnMiniDramaReviewStatus(func(ctx context.Context, evt *miniProgram.MiniDramaReviewStatusEvent) error {
		if !evt.OK() {
			log.Printf("审核状态事件字段不完整: %+v", evt)
			return nil
		}

		audit := evt.Audit()
		log.Printf("剧目 %d 审核%s（%s）",
			evt.DramaID(), audit.StatusMsg(), audit.AuditTypeMsg())

		if !audit.Settled() {
			return nil
		}

		saveDramaReview(ctx, evt.DramaID(), audit.Status, audit.AuditType)

		return nil
	})

	// 差评通知：拿到 comment_id 后才能重置客服 quota 并主动联系用户。
	app.OnCommentBadScore(func(ctx context.Context, evt *miniProgram.CommentBadScoreEvent) error {
		if !evt.OK() {
			log.Printf("差评事件字段不完整: %+v", evt)
			return nil
		}

		log.Printf("收到差评 %s", evt.CommentID())
		queueBadReviewFollowUp(ctx, evt.CommentID())

		return nil
	})
}

// 占位实现：真实接入时换成自己的数据访问层。
func recordQuotaAlert(_ context.Context, _ int64, _ float64) {}

func recordValidityAlert(_ context.Context, _ int64, _ string) {}

func markMediaUploaded(_ context.Context, _ int64, _ string) {}

func markMediaUploadFailed(_ context.Context, _ int64, _ string, _ string) {}

func saveDramaReview(_ context.Context, _ int64, _ int, _ int) {}

func queueBadReviewFollowUp(_ context.Context, _ string) {}
