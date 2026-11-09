package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// SendCustomTextMessage 在客服会话里回复一条文本消息。
//
// 客服消息不是群发：微信只允许在用户最后一条消息之后的 48 小时内回复，且一个
// 窗口内最多 5 条。超出窗口微信返回 45015，此时只能改用订阅消息触达。
func SendCustomTextMessage(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
	content string,
) error {
	res, err := app.SendCustomMessage(ctx, miniProgram.SendCustomMessageParams{
		ToUser:  openID,
		MsgType: miniProgram.CustomMsgTypeText,
		Text:    &miniProgram.CustomText{Content: content},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SendCustomMessageError(res)
	}

	log.Println("SendCustomMessage(text): success")

	return nil
}

// SendCustomImageMessage 在客服会话里回复一张图片。
//
// media_id 来自 UploadTempMedia，临时素材有效期 3 天，不要长期保存。
func SendCustomImageMessage(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
	mediaID string,
) error {
	res, err := app.SendCustomMessage(ctx, miniProgram.SendCustomMessageParams{
		ToUser:  openID,
		MsgType: miniProgram.CustomMsgTypeImage,
		Image:   &miniProgram.CustomImage{MediaID: mediaID},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SendCustomMessageError(res)
	}

	log.Println("SendCustomMessage(image): success")

	return nil
}

// SetTyping 显示或隐藏“正在输入”。
//
// 这个提示 30 秒后自动消失，人工客服回复较慢时需要重复调用维持；回复完成后
// 可以主动取消。它同样受 48 小时窗口限制。
func SetTyping(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
) error {
	res, err := app.SetTyping(ctx, miniProgram.SetTypingParams{
		ToUser:  openID,
		Command: miniProgram.TypingCommandTyping,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SetTypingError(res)
	}

	log.Println("SetTyping: success")

	return nil
}

// AddKFAccount 把一个微信号绑定为客服账号。
//
// 绑定不等于能发消息，账号仍需在 48 小时窗口内通过 SendCustomMessage 回复。
// 一个微信号只能绑定一个账号，数量超限时微信返回 65406。
func AddKFAccount(
	ctx context.Context,
	app *miniProgram.Application,
	kfWx string,
) error {
	res, err := app.AddKFAccount(ctx, miniProgram.AddKFAccountParams{
		KFWx: kfWx,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.AddKFAccountError(res)
	}

	log.Println("AddKFAccount: success")

	return nil
}

// GetKFList 查询全部客服账号。
//
// 返回的 kf_openid 才是删除账号、设置管理员时使用的标识。
func GetKFList(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetKFList(ctx, miniProgram.GetKFListParams{})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetKFListError(res)
	}

	for _, account := range res.KFList {
		log.Printf("GetKFList: %s %s %s",
			account.KFAccount, account.KFNick, account.KFOpenID)
	}

	return nil
}

// GetOnlineKFList 查询当前在线的客服账号。
//
// 只返回已打开客服工具的账号，可用来判断此刻有没有人能接会话。
func GetOnlineKFList(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetOnlineKFList(ctx, miniProgram.GetOnlineKFListParams{})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetOnlineKFListError(res)
	}

	for _, account := range res.KFOnlineList {
		log.Printf("GetOnlineKFList: %s status=%d", account.KFAccount, account.Status)
	}

	return nil
}

// DeleteKFAccount 解绑客服账号。
//
// 历史会话保留，但后续会话不再分配给该账号；同时释放一个账号名额。
func DeleteKFAccount(
	ctx context.Context,
	app *miniProgram.Application,
	kfOpenID string,
) error {
	res, err := app.DeleteKFAccount(ctx, miniProgram.DeleteKFAccountParams{
		KFOpenID: kfOpenID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.DeleteKFAccountError(res)
	}

	log.Println("DeleteKFAccount: success")

	return nil
}

// SetKFAdmin 把客服账号设为管理员，使其能在客服工具里管理其他账号。
func SetKFAdmin(
	ctx context.Context,
	app *miniProgram.Application,
	kfOpenID string,
) error {
	res, err := app.SetKFAdmin(ctx, miniProgram.SetKFAdminParams{
		KFOpenID: kfOpenID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SetKFAdminError(res)
	}

	log.Println("SetKFAdmin: success")

	return nil
}

// CancelKFAdmin 取消客服账号的管理员身份，账号仍可继续接待用户。
func CancelKFAdmin(
	ctx context.Context,
	app *miniProgram.Application,
	kfOpenID string,
) error {
	res, err := app.CancelKFAdmin(ctx, miniProgram.CancelKFAdminParams{
		KFOpenID: kfOpenID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.CancelKFAdminError(res)
	}

	log.Println("CancelKFAdmin: success")

	return nil
}

// UploadTempMedia 上传客服消息使用的临时素材。
//
// 返回的 media_id 用于发送图片或小程序卡片封面，有效期 3 天；图片不超过 10MB，
// 只支持 JPG/PNG。文件名要带正确扩展名，微信靠它识别格式。
func UploadTempMedia(
	ctx context.Context,
	app *miniProgram.Application,
	filename string,
	data []byte,
) error {
	res, err := app.UploadTempMedia(ctx, miniProgram.UploadTempMediaParams{
		MediaType: miniProgram.TempMediaTypeImage,
		Filename:  filename,
		Data:      data,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.UploadTempMediaError(res)
	}

	log.Printf("UploadTempMedia: %s expires at %d", res.MediaID, res.CreatedAt)

	return nil
}

// GetTempMedia 下载客服消息里的临时素材。
//
// 成功时返回文件原始字节，ContentType 是微信给出的类型；media_id 过期后微信
// 返回 40007，此时文件已不可恢复。
func GetTempMedia(
	ctx context.Context,
	app *miniProgram.Application,
	mediaID string,
) error {
	res, err := app.GetTempMedia(ctx, miniProgram.GetTempMediaParams{
		MediaID: mediaID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetTempMediaError(res)
	}

	log.Printf("GetTempMedia: %s %d bytes", res.ContentType, len(res.Data))

	return nil
}
