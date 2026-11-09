package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/examples"
	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

func main() {
	app, err := miniProgram.New(&miniProgram.Config{
		AppID:     examples.WECHAT_APPID,
		AppSecret: examples.WECHAT_APPSECRET,
	})
	if err != nil {
		log.Fatalf("build application: %v", err)
	}
	ctx := context.Background()

	// 小程序登录凭证校验
	//code := "0d3d1NFa1YlqKM0koXFa1yGSd22d1NFt"
	//err = Code2Session(ctx, app, code)
	//if err != nil {
	//	log.Fatalf("登录失败 %s", err.Error())
	//} else {
	//	log.Println("code2session success")
	//}
	//return

	// 检验登录态是否有效
	//openID := "oKGug4tqqGYFCB46wZLoMBC5XN9Q"
	//sessionKey := "iriBgaEJgBlAj4PXBP3txA=="
	//err = CheckSessionKey(ctx, app, openID, miniProgram.CheckSessionSignature(sessionKey))
	//if err != nil {
	//	log.Fatalf("检验登录态失败 %s", err.Error())
	//} else {
	//	log.Println("checksession success")
	//}

	// 重置登录态，旧 session_key 会失效
	//err = ResetUserSessionKey(ctx, app, openID, miniProgram.CheckSessionSignature(sessionKey))
	//if err != nil {
	//	log.Fatalf("重置登录态失败 %s", err.Error())
	//} else {
	//	log.Println("resetusersessionkey success")
	//}
	//return

	// 获取手机号，code 来自 button open-type="getPhoneNumber" 的回调
	//phoneCode := "069922a96289ce57798bd3a382246228cc19a1e3764148d0bbaf15892dd0d5ad"
	//err = GetPhoneNumber(ctx, app, phoneCode)
	//if err != nil {
	//	log.Fatalf("获取手机号失败 %s", err.Error())
	//} else {
	//	log.Println("getphonenumber success")
	//}

	// 获取插件用户 openpid，code 由插件内的 wx.pluginLogin 获取
	pluginCode := "XXXXXXXX"
	err = GetPluginOpenPID(ctx, app, pluginCode)
	if err != nil {
		log.Fatalf("获取插件 openpid 失败 %s", err.Error())
	} else {
		log.Println("getpluginopenpid success")
	}
	//return

	// 检查加密信息，encryptedData 是前端拿到的加密数据原文
	//encryptedData := "XXXXXXXX"
	//err = CheckEncryptedData(ctx, app, encryptedData)
	//if err != nil {
	//	log.Fatalf("检查加密信息失败 %s", err.Error())
	//} else {
	//	log.Println("checkencrypteddata success")
	//}

	// 支付后获取 UnionID，需在支付完成后 5 分钟内调用
	//paidOpenID := "XXXXXXXX"
	//transactionID := "XXXXXXXX"
	//err = GetPaidUnionID(ctx, app, paidOpenID, transactionID)
	//if err != nil {
	//	log.Fatalf("获取 UnionID 失败 %s", err.Error())
	//} else {
	//	log.Println("getpaidunionid success")
	//}

	// 获取用户 encrypt key，sessionKey 来自 Code2Session
	//encryptOpenID := "oKGug4tqqGYFCB46wZLoMBC5XN9Q"
	//encryptSessionKey := "iriBgaEJgBlAj4PXBP3txA=="
	//err = GetUserEncryptKey(ctx, app, encryptOpenID, encryptSessionKey)
	//if err != nil {
	//	log.Fatalf("获取 encrypt key 失败 %s", err.Error())
	//} else {
	//	log.Println("getuserencryptkey success")
	//}

	// 获取小程序码，返回图片二进制，path 必须是已发布小程序中的页面
	//qrPath := "pages/index/index"
	//err = GetQRCode(ctx, app, qrPath)
	//if err != nil {
	//	log.Fatalf("获取小程序码失败 %s", err.Error())
	//} else {
	//	log.Println("getqrcode success")
	//}

	// 获取不限制的小程序码，scene 扫码后作为 query 传给页面，需自行编码为合法字符
	//qrScene := "a=1"
	//err = GetUnlimitedQRCode(ctx, app, qrScene)
	//if err != nil {
	//	log.Fatalf("获取不限制的小程序码失败 %s", err.Error())
	//} else {
	//	log.Println("getunlimitedqrcode success")
	//}

	// 获取小程序二维码，返回图片二进制，path 必须是已发布小程序中的页面
	//qrCodePath := "pages/index/index"
	//err = CreateQRCode(ctx, app, qrCodePath)
	//if err != nil {
	//	log.Fatalf("获取小程序二维码失败 %s", err.Error())
	//} else {
	//	log.Println("createqrcode success")
	//}

	// 查询 scheme 码，scheme 由生成接口返回，query_type 为 1 时查剩余访问次数
	//scheme := "weixin://dl/business/?t=XXXXXXXX"
	//err = QueryScheme(ctx, app, scheme)
	//if err != nil {
	//	log.Fatalf("查询 scheme 码失败 %s", err.Error())
	//} else {
	//	log.Println("queryscheme success")
	//}

	// 获取加密 scheme 码，path 必须是已发布小程序中的页面且不能带 query
	//schemePath := "/pages/index/index"
	//err = GenerateScheme(ctx, app, schemePath, "")
	//if err != nil {
	//	log.Fatalf("获取加密 scheme 码失败 %s", err.Error())
	//} else {
	//	log.Println("generatescheme success")
	//}

	// 获取 NFC 的小程序 scheme，modelID 为 NFC 设备型号，一机一码时需额外传 sn
	//nfcPath := "/pages/index/index"
	//modelID := "xxx"
	//sn := ""
	//err = GenerateNFCScheme(ctx, app, nfcPath, modelID, sn)
	//if err != nil {
	//	log.Fatalf("获取 NFC 的小程序 scheme 失败 %s", err.Error())
	//} else {
	//	log.Println("generatenfcscheme success")
	//}

	// 获取加密 URLLink，适用于短信、邮件、网页等外部场景，path 不能带 query
	//urlLinkPath := "/pages/index/index"
	//err = GenerateURLLink(ctx, app, urlLinkPath, "")
	//if err != nil {
	//	log.Fatalf("获取加密 URLLink 失败 %s", err.Error())
	//} else {
	//	log.Println("generateurllink success")
	//}

	// 查询加密 URLLink，queryType 为 1 时查询当天剩余访问次数，无需传 urlLink
	//urlLink := "https://wxaurl.cn/XXXXXXXX"
	//queryType := 0
	//err = QueryURLLink(ctx, app, urlLink, queryType)
	//if err != nil {
	//	log.Fatalf("查询加密 URLLink 失败 %s", err.Error())
	//} else {
	//	log.Println("queryurllink success")
	//}

	// 获取 Short Link，适用于微信内拉起小程序，pageURL 可携带 query 且不加前导斜杠
	//shortLinkPageURL := "pages/index/index?query1=q1"
	//pageTitle := "Homework title"
	//isPermanent := false
	//err = GenerateShortLink(ctx, app, shortLinkPageURL, pageTitle, isPermanent)
	//if err != nil {
	//	log.Fatalf("获取 Short Link 失败 %s", err.Error())
	//} else {
	//	log.Println("generateshortlink success")
	//}

	// ---- 消息相关：动态消息 ----

	// 创建动态消息的 activity_id，私密消息才需要 openID/unionID
	//err = CreateActivityID(ctx, app, "", "")
	//if err != nil {
	//	log.Fatalf("创建 activity_id 失败 %s", err.Error())
	//} else {
	//	log.Println("createactivityid success")
	//}

	// 修改动态消息，更新已加入人数与房间上限
	//activityID := "XXXXXXXX"
	//err = SetUpdatableMsg(ctx, app, activityID, "3", "10")
	//if err != nil {
	//	log.Fatalf("修改动态消息失败 %s", err.Error())
	//} else {
	//	log.Println("setupdatablemsg success")
	//}

	// 修改聊天工具动态卡片，需带上模板 id
	//chatToolTemplateID := "XXXXXXXX"
	//groupOpenID := "XXXXXXXX"
	//err = SetChatToolMsg(ctx, app, activityID, chatToolTemplateID, groupOpenID)
	//if err != nil {
	//	log.Fatalf("修改聊天工具动态卡片失败 %s", err.Error())
	//} else {
	//	log.Println("setchattoolmsg success")
	//}

	// ---- 消息相关：订阅消息 ----

	// 查询小程序类目，订阅消息只能从这些类目的公共模板里选
	//err = GetCategory(ctx, app)
	//if err != nil {
	//	log.Fatalf("查询类目失败 %s", err.Error())
	//} else {
	//	log.Println("getcategory success")
	//}

	// 分页拉取类目下的公共模板
	//categoryID := int64(616)
	//err = GetPubTemplateTitles(ctx, app, categoryID)
	//if err != nil {
	//	log.Fatalf("拉取公共模板失败 %s", err.Error())
	//} else {
	//	log.Println("getpubtemplatetitles success")
	//}

	// 查询公共模板的关键词，rule 决定发送时 data 的取值规则
	//titleID := "99"
	//err = GetPubTemplateKeywords(ctx, app, titleID)
	//if err != nil {
	//	log.Fatalf("查询模板关键词失败 %s", err.Error())
	//} else {
	//	log.Println("getpubtemplatekeywords success")
	//}

	// 选用公共模板，返回的 priTmplId 才是发送时用的 template_id
	//keywordIDs := []int64{1, 2}
	//sceneDesc := "下单成功后提醒用户取餐"
	//err = AddMessageTemplate(ctx, app, titleID, keywordIDs, sceneDesc)
	//if err != nil {
	//	log.Fatalf("选用模板失败 %s", err.Error())
	//} else {
	//	log.Println("addmessagetemplate success")
	//}

	// 查询小程序已选用的模板
	//err = GetMessageTemplateList(ctx, app)
	//if err != nil {
	//	log.Fatalf("查询模板列表失败 %s", err.Error())
	//} else {
	//	log.Println("getmessagetemplatelist success")
	//}

	// 删除已选用的模板
	//priTemplateID := "XXXXXXXX"
	//err = DeleteMessageTemplate(ctx, app, priTemplateID)
	//if err != nil {
	//	log.Fatalf("删除模板失败 %s", err.Error())
	//} else {
	//	log.Println("deletemessagetemplate success")
	//}

	// 发送订阅消息，一次性订阅的授权只能成功发送一条
	//subscribeOpenID := "oKGug4tqqGYFCB46wZLoMBC5XN9Q"
	//err = SendSubscribeMessage(ctx, app, subscribeOpenID, priTemplateID, "黄焖鸡", "2026年11月9日")
	//if err != nil {
	//	log.Fatalf("发送订阅消息失败 %s", err.Error())
	//} else {
	//	log.Println("sendsubscribemessage success")
	//}

	// 激活或更新服务卡片，仅电商类目可用
	//notifyCode := "XXXXXXXX"
	//contentJSON := `{"character_string1":{"value":"20261109001"}}`
	//err = SetUserNotify(ctx, app, subscribeOpenID, notifyCode, contentJSON)
	//if err != nil {
	//	log.Fatalf("更新服务卡片失败 %s", err.Error())
	//} else {
	//	log.Println("setusernotify success")
	//}

	// 更新服务卡片的扩展信息
	//extJSON := `{"shipping_list":[]}`
	//err = SetUserNotifyExt(ctx, app, subscribeOpenID, notifyCode, extJSON)
	//if err != nil {
	//	log.Fatalf("更新服务卡片扩展信息失败 %s", err.Error())
	//} else {
	//	log.Println("setusernotifyext success")
	//}

	// 查询服务卡片状态
	//err = GetUserNotify(ctx, app, subscribeOpenID, notifyCode)
	//if err != nil {
	//	log.Fatalf("查询服务卡片状态失败 %s", err.Error())
	//} else {
	//	log.Println("getusernotify success")
	//}

	// ---- 消息相关：客服消息 ----

	// 在客服会话里回复文本，受用户最后一条消息后的 48 小时窗口限制
	//err = SendCustomTextMessage(ctx, app, subscribeOpenID, "您好，请问有什么可以帮您？")
	//if err != nil {
	//	log.Fatalf("发送客服文本消息失败 %s", err.Error())
	//} else {
	//	log.Println("sendcustommessage(text) success")
	//}

	// 上传临时素材，返回的 media_id 可用于发送图片，有效期 3 天
	//imageData, _ := os.ReadFile("cover.png")
	//err = UploadTempMedia(ctx, app, "cover.png", imageData)
	//if err != nil {
	//	log.Fatalf("上传临时素材失败 %s", err.Error())
	//} else {
	//	log.Println("uploadtempmedia success")
	//}

	// 在客服会话里回复图片，mediaID 来自上传临时素材
	//mediaID := "XXXXXXXX"
	//err = SendCustomImageMessage(ctx, app, subscribeOpenID, mediaID)
	//if err != nil {
	//	log.Fatalf("发送客服图片消息失败 %s", err.Error())
	//} else {
	//	log.Println("sendcustommessage(image) success")
	//}

	// 显示“正在输入”，30 秒后自动消失
	//err = SetTyping(ctx, app, subscribeOpenID)
	//if err != nil {
	//	log.Fatalf("设置客服输入状态失败 %s", err.Error())
	//} else {
	//	log.Println("settyping success")
	//}

	// 下载客服消息里的临时素材
	//err = GetTempMedia(ctx, app, mediaID)
	//if err != nil {
	//	log.Fatalf("下载临时素材失败 %s", err.Error())
	//} else {
	//	log.Println("gettempmedia success")
	//}

	// 把微信号绑定为客服账号
	//kfWx := "kf_wx"
	//err = AddKFAccount(ctx, app, kfWx)
	//if err != nil {
	//	log.Fatalf("添加客服账号失败 %s", err.Error())
	//} else {
	//	log.Println("addkfaccount success")
	//}

	// 查询全部客服账号，返回的 kf_openid 用于删除与管理员设置
	//err = GetKFList(ctx, app)
	//if err != nil {
	//	log.Fatalf("查询客服账号失败 %s", err.Error())
	//} else {
	//	log.Println("getkflist success")
	//}

	// 查询在线客服账号
	//err = GetOnlineKFList(ctx, app)
	//if err != nil {
	//	log.Fatalf("查询在线客服失败 %s", err.Error())
	//} else {
	//	log.Println("getonlinekflist success")
	//}

	// 设置客服管理员
	//kfOpenID := "XXXXXXXX"
	//err = SetKFAdmin(ctx, app, kfOpenID)
	//if err != nil {
	//	log.Fatalf("设置客服管理员失败 %s", err.Error())
	//} else {
	//	log.Println("setkfadmin success")
	//}

	// 取消客服管理员
	//err = CancelKFAdmin(ctx, app, kfOpenID)
	//if err != nil {
	//	log.Fatalf("取消客服管理员失败 %s", err.Error())
	//} else {
	//	log.Println("cancelkfadmin success")
	//}

	// 解绑客服账号
	//err = DeleteKFAccount(ctx, app, kfOpenID)
	//if err != nil {
	//	log.Fatalf("删除客服账号失败 %s", err.Error())
	//} else {
	//	log.Println("deletekfaccount success")
	//}

}
