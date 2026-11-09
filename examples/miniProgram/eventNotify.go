package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/antsurge/atto-wechat/core/event"
	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// EventNotify 演示事件通知（消息推送）的接入方式。
//
// 和 Code2Session 这类主动调用不同，事件是微信推过来的：
//   - 不走 access_token，报文里没有 errcode/errmsg；
//   - 必须在 Config 里配置 Token 或 AESKey（消息加解密），并保证返回 success；
//   - 回调地址要能在公网访问，本地调试用 ngrok 之类的工具映射。
//
// 示例先把四个事件处理器注册好，再由 main 里的 EventNotifyServer 起服务。
func EventNotify(app *miniProgram.Application) {
	// 授权用户信息变更。三个事件（资料清理 / 撤回授权 / 注销）共用一个处理器，
	// 用 IsModified、IsRevoked、IsCancelled 区分。
	app.OnUserInfoChange(func(ctx context.Context, evt *miniProgram.UserInfoChangeEvent) error {
		if !evt.OK() {
			log.Printf("授权事件字段不完整: %+v", evt)
			return nil
		}

		switch {
		case evt.IsRevoked():
			// 用户撤回授权后，授权范围内的数据必须删除。
			log.Printf("用户 %s 撤回授权，范围: %s", evt.OpenID, evt.RevokeInfoName())
		case evt.IsCancelled():
			// 用户注销后，该用户的全部数据都要清理。
			log.Printf("用户 %s 已注销，清理数据", evt.OpenID)
		default:
			log.Printf("用户 %s 的资料已被平台清理", evt.OpenID)
		}

		return nil
	})

	// 违规处罚信息通知。
	app.OnPunish(func(ctx context.Context, evt *miniProgram.PunishEvent) error {
		if !evt.OK() {
			log.Printf("处罚事件字段不完整: %+v", evt)
			return nil
		}

		log.Printf("收到处罚 %s（%s），原因: %s", evt.PunishID, evt.TypeName(), evt.IllegalReason)

		// detail 是 JSON 字符串，结构随 event_type 变化，必须按类型解析。
		detail, err := evt.ParseDetail()
		if err != nil {
			log.Printf("解析处罚详情失败: %v", err)
			return nil
		}

		switch {
		case detail.Warning != nil:
			// 0 天表示永久，不是“没有封禁”。
			log.Printf("%s，整改截止: %s", detail.Warning.TypeName(),
				time.Unix(detail.Warning.RectifyDeadline, 0).Format(time.RFC3339))
		case detail.FunctionBan != nil:
			log.Printf("功能封禁: %v，天数: %v", detail.FunctionBan.BannedFunctionNames,
				detail.FunctionBan.BannedDays)
		case detail.Suspend != nil:
			log.Printf("下架 %d 天", detail.Suspend.SuspendedDays)
		case detail.AccountBan != nil:
			log.Printf("账号封禁 %d 天", detail.AccountBan.BannedDays)
		case detail.PageBan != nil:
			log.Printf("页面封禁: %s", detail.PageBan.Path)
		}

		return nil
	})

	// 长期订阅状态同步：用户关注/取消关注直播。
	app.OnLiveFollow(func(ctx context.Context, evt *miniProgram.LiveFollowEvent) error {
		if !evt.OK() {
			log.Printf("订阅事件字段不完整: %+v", evt)
			return nil
		}

		action := "关注"
		if evt.IsUnfollow() {
			action = "取消关注"
		}

		log.Printf("用户 %s %s了直播间 %d（%s）",
			evt.FollowNotify.UserOpenID, action, evt.FollowNotify.RoomID, evt.LiveStatusName())

		return nil
	})

	// 长期订阅群发结果：PushLiveMessage 只负责受理，投递结果从这里回来。
	app.OnLivePushMessageNotify(func(ctx context.Context, evt *miniProgram.LivePushMessageNotifyEvent) error {
		if !evt.OK() {
			log.Printf("群发结果字段不完整: %+v", evt)
			return nil
		}

		log.Printf("群发 %s 完成：共 %d，成功 %d，失败 %d（openid %d / 未关注 %d / 超限 %d / 系统 %d）",
			evt.Notify.MessageID,
			evt.Notify.TotalCount,
			evt.Notify.SuccessCount,
			evt.FailedCount(),
			evt.Notify.OpenIDErrorCount,
			evt.Notify.RelationErrorCount,
			evt.Notify.UserRecvLimitCount,
			evt.Notify.InternalErrorCount,
		)

		return nil
	})

	// 没有内置结构的事件，用 OnEvent 直接拿原始报文。
	app.OnEvent("some_future_event", func(ctx context.Context, msg *event.Message) error {
		log.Printf("收到未建模事件 %s，原始报文 %d 字节", msg.Event, len(msg.Raw))
		return nil
	})
}

// EventNotifyServer 启动只接收事件回调的 HTTP 服务。
//
// Config 里没有 Token/AESKey 时 Callback() 返回 nil，说明回调服务未启用。
func EventNotifyServer(ctx context.Context, app *miniProgram.Application, addr string) error {
	handler := app.Callback()
	if handler == nil {
		return errors.New("回调服务未启用：请在 miniProgram.Config 中配置 Token 或 AESKey")
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("关闭回调服务失败: %v", err)
		}
	}()

	log.Printf("事件回调服务监听 %s", addr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("监听 %s 失败: %w", addr, err)
	}

	return nil
}
