package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// GetCategory 查询小程序所属类目。
//
// 订阅消息只能从这些类目的公共模板里选，返回的 id 既是后续按类目浏览模板的
// tid，也是判断“这个小程序能不能发某类消息”的依据。
func GetCategory(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetCategory(ctx)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetCategoryError(res)
	}

	for _, category := range res.Data {
		log.Printf("GetCategory: %d %s", category.ID, category.Name)
	}

	return nil
}

// GetPubTemplateTitles 分页拉取某个类目下的公共模板。
//
// 公共模板由微信提供，一条模板就是一个固定句式；count 是总数而不是本页条数。
func GetPubTemplateTitles(
	ctx context.Context,
	app *miniProgram.Application,
	categoryID int64,
) error {
	res, err := app.GetPubTemplateTitles(ctx, miniProgram.GetPubTemplateTitlesParams{
		CategoryIDs: []int64{categoryID},
		Start:       0,
		Limit:       30,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetPubTemplateTitlesError(res)
	}

	for _, title := range res.Data {
		log.Printf("GetPubTemplateTitles: %s %s", title.TitleID, title.Title)
	}

	return nil
}

// GetPubTemplateKeywords 查询公共模板的关键词。
//
// 关键词决定了模板能留下哪些“坑”，rule 是每个坑的取值规则（thing、number、
// date 等），也是发送时 data 必须满足的约束。
func GetPubTemplateKeywords(
	ctx context.Context,
	app *miniProgram.Application,
	titleID string,
) error {
	res, err := app.GetPubTemplateKeywords(ctx, miniProgram.GetPubTemplateKeywordsParams{
		TitleID: titleID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetPubTemplateKeywordsError(res)
	}

	for _, keyword := range res.Data {
		log.Printf("GetPubTemplateKeywords: %d %s %s %s",
			keyword.Kid, keyword.Name, keyword.Rule, keyword.Example)
	}

	return nil
}

// AddMessageTemplate 把公共模板选用到小程序。
//
// kidList 是保留的关键词，sceneDesc 会展示在订阅弹窗里，用来说明消息用途。
// 返回的 priTmplId 才是发送时用的 id，一个小程序最多 25 条模板。
func AddMessageTemplate(
	ctx context.Context,
	app *miniProgram.Application,
	titleID string,
	keywordIDs []int64,
	sceneDesc string,
) error {
	res, err := app.AddMessageTemplate(ctx, miniProgram.AddMessageTemplateParams{
		TitleID:          titleID,
		KeywordIDs:       keywordIDs,
		SceneDescription: sceneDesc,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.AddMessageTemplateError(res)
	}

	log.Printf("AddMessageTemplate: %s", res.PriTemplateID)

	return nil
}

// GetMessageTemplateList 查询小程序已选用的模板。
//
// 返回的 priTmplId 就是发送时要填的 template_id；删除模板后，仍指向它的发送
// 请求会被微信以 43101 拒绝。
func GetMessageTemplateList(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetMessageTemplateList(ctx)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetMessageTemplateListError(res)
	}

	for _, template := range res.Data {
		log.Printf("GetMessageTemplateList: %s %s", template.PriTemplateID, template.Title)
	}

	return nil
}

// DeleteMessageTemplate 删除已选用的模板。
func DeleteMessageTemplate(
	ctx context.Context,
	app *miniProgram.Application,
	priTemplateID string,
) error {
	res, err := app.DeleteMessageTemplate(ctx, miniProgram.DeleteMessageTemplateParams{
		PriTemplateID: priTemplateID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.DeleteMessageTemplateError(res)
	}

	log.Println("DeleteMessageTemplate: success")

	return nil
}

// SendSubscribeMessage 发送订阅消息。
//
// 用户必须先授权：一次性订阅的一次授权只能成功发送一条，发完即失效，再发会被
// 43101 拒绝。data 的键是模板关键词（带规则前缀和序号，如 thing1、date2），
// 值必须满足该关键词的规则，否则 47003 会指出具体字段。
func SendSubscribeMessage(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
	priTemplateID string,
	thing string,
	date string,
) error {
	res, err := app.SendSubscribeMessage(ctx, miniProgram.SendSubscribeMessageParams{
		ToUser:     openID,
		TemplateID: priTemplateID,
		Page:       "pages/index/index",
		Data: map[string]miniProgram.SubscribeMessageValue{
			"thing1": {Value: thing},
			"date2":  {Value: date},
		},
		MiniProgramState: miniProgram.SubscribeMsgStateFormal,
		Lang:             miniProgram.SubscribeMsgLangZhCN,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SendSubscribeMessageError(res)
	}

	log.Println("SendSubscribeMessage: success")

	return nil
}

// SetUserNotify 激活或更新服务卡片。
//
// 服务卡片不在聊天里，而在微信的服务通知面板中长期存在，所以要靠这个接口持续
// 刷新状态。content_json 是卡片内容，按 notify_type 定义结构，序列化成字符串
// 传入。该能力仅对电商类目开放。
func SetUserNotify(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
	notifyCode string,
	contentJSON string,
) error {
	res, err := app.SetUserNotify(ctx, miniProgram.SetUserNotifyParams{
		OpenID:      openID,
		NotifyType:  1003,
		NotifyCode:  notifyCode,
		ContentJSON: contentJSON,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SetUserNotifyError(res)
	}

	log.Println("SetUserNotify: success")

	return nil
}

// SetUserNotifyExt 更新服务卡片的扩展信息，如订单行、物流、门店地址。
func SetUserNotifyExt(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
	notifyCode string,
	extJSON string,
) error {
	res, err := app.SetUserNotifyExt(ctx, miniProgram.SetUserNotifyExtParams{
		OpenID:     openID,
		NotifyType: 1003,
		NotifyCode: notifyCode,
		ExtJSON:    extJSON,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SetUserNotifyExtError(res)
	}

	log.Println("SetUserNotifyExt: success")

	return nil
}

// GetUserNotify 查询服务卡片的当前状态。
//
// notify_code 未知时微信返回 85437，它同时也代表“这张卡片从未激活过”，可以
// 用来区分“用户没看过”和“更新被拒绝”。
func GetUserNotify(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
	notifyCode string,
) error {
	res, err := app.GetUserNotify(ctx, miniProgram.GetUserNotifyParams{
		OpenID:     openID,
		NotifyCode: notifyCode,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetUserNotifyError(res)
	}

	for _, card := range res.Data {
		log.Printf("GetUserNotify: %d %s %s",
			card.NotifyType, card.NotifyCode, card.ContentJSON)
	}

	return nil
}
