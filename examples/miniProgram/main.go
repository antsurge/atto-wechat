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

}
