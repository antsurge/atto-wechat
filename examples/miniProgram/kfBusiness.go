package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// avatarPNG 是一张 1x1 的透明 PNG，只为让示例可编译可运行；真实场景换成商户头像。
var avatarPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
	0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
	0x42, 0x60, 0x82,
}

// KFBusiness 演示客服子商户与微信客服绑定。
//
// 两条线解决的问题不同：
//   - 客服子商户（/cgi-bin/business/*）把一个电商平台小程序拆成多个独立会话主体，
//     每个子商户有自己的客服账号与消息；
//   - 微信客服绑定（/customservice/work/*）把整个小程序接到企业微信的微信客服。
func KFBusiness(ctx context.Context, app *miniProgram.Application) {
	// 头像必须先通过 UploadTempMedia 上传，拿到临时素材 media_id。
	avatar, err := app.UploadTempMedia(ctx, miniProgram.UploadTempMediaParams{
		MediaType: miniProgram.TempMediaTypeImage,
		Filename:  "avatar.png",
		Data:      avatarPNG,
	})
	if err != nil || !avatar.OK() {
		log.Printf("上传头像失败: %v", err)

		return
	}

	created, err := app.RegisterBusiness(ctx, miniProgram.RegisterBusinessParams{
		AccountName:      "apple_store",
		Nickname:         "苹果旗舰店",
		IconMediaID:      avatar.MediaID,
		TransferToCommKF: false,
	})
	if err != nil {
		log.Printf("创建子商户失败: %v", err)

		return
	}

	if !created.OK() {
		log.Printf("创建子商户失败: %s",
			miniProgram.RegisterBusinessErrMsg(created.ErrCode))

		return
	}

	log.Printf("子商户 business_id = %d", created.BusinessID)

	// 子商户的客服账号要带 business_id，会话才是独立的。
	if _, err := app.AddKFAccount(ctx, miniProgram.AddKFAccountParams{
		KFWx:       "kf_wechat_id",
		BusinessID: created.BusinessID,
	}); err != nil {
		log.Printf("绑定子商户客服账号失败: %v", err)
	}

	// account_name 唯一，重复创建返回 45070，所以它要落库。
	got, err := app.GetBusiness(ctx, miniProgram.GetBusinessParams{
		BusinessID: created.BusinessID,
	})
	if err != nil {
		log.Printf("拉取子商户失败: %v", err)
	} else if got.OK() {
		log.Printf("子商户 %s（%s）", got.BusinessInfo.Nickname, got.BusinessInfo.AccountName)
	}

	// 分页拉全量：一页不满 count 就是最后一页。
	for offset := 0; ; offset += 100 {
		page, err := app.ListBusiness(ctx, miniProgram.ListBusinessParams{
			Offset: offset,
			Count:  100,
		})
		if err != nil {
			log.Printf("拉取子商户列表失败: %v", err)

			return
		}

		if !page.OK() {
			log.Printf("拉取子商户列表失败: %s",
				miniProgram.ListBusinessErrMsg(page.ErrCode))

			return
		}

		for _, b := range page.List {
			log.Printf("- %d %s %s", b.BusinessID, b.AccountName, b.Nickname)
		}

		if page.Done(100) {
			break
		}
	}

	// 微信客服绑定：先看有没有绑，再决定绑还是解绑。
	bound, err := app.GetKFWorkBound(ctx)
	if err != nil {
		log.Printf("查询微信客服绑定失败: %v", err)

		return
	}

	switch {
	case !bound.OK():
		log.Printf("查询微信客服绑定失败: %s",
			miniProgram.GetKFWorkBoundErrMsg(bound.ErrCode))
	case bound.Bound():
		log.Printf("已绑定 %s（%s），绑定于 %d",
			bound.EntityName, bound.CorpID, bound.BindTime)
	default:
		log.Print("未绑定微信客服")
	}

	// corpid 必须与小程序主体一致，否则 10130602。
	if _, err := app.BindKFWork(ctx, miniProgram.BindKFWorkParams{
		CorpID: "ww1234567890",
	}); err != nil {
		log.Printf("绑定微信客服失败: %v", err)
	}

	// 解绑要传绑定的那个 corpid，所以先读再加解绑。
	if bound.Bound() {
		if _, err := app.UnbindKFWork(ctx, miniProgram.UnbindKFWorkParams{
			CorpID: bound.CorpID,
		}); err != nil {
			log.Printf("解绑微信客服失败: %v", err)
		}
	}
}
