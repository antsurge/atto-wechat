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
	openID := "oKGug4tqqGYFCB46wZLoMBC5XN9Q"
	sessionKey := "iriBgaEJgBlAj4PXBP3txA=="
	err = CheckSessionKey(ctx, app, openID, miniProgram.CheckSessionSignature(sessionKey))
	if err != nil {
		log.Fatalf("检验登录态失败 %s", err.Error())
	} else {
		log.Println("checksession success")
	}
	return

	// 重置登录态，旧 session_key 会失效
	//err = ResetUserSessionKey(ctx, app, openID, miniProgram.CheckSessionSignature(sessionKey))
	//if err != nil {
	//	log.Fatalf("重置登录态失败 %s", err.Error())
	//} else {
	//	log.Println("resetusersessionkey success")
	//}

}
