package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// LaborRelation 演示用工关系：向已绑定关系的用户推送消息，以及批量解绑。
//
// 用工关系本身由用户在商家助手里建立，这个包只负责在关系已存在之后推送与解绑。
func LaborRelation(ctx context.Context, app *miniProgram.Application) {
	// 模板内容必须是「JSON 编码后的字符串」，用辅助函数生成，避免手写转义。
	data, err := miniProgram.BuildEmployeeRelationMsgData(map[string]string{
		"character_string1": "ORDER-20240101",
		"amount1":           "222",
	})
	if err != nil {
		log.Printf("构造消息内容失败: %v", err)

		return
	}

	sent, err := app.SendEmployeeRelationMsg(ctx, miniProgram.SendEmployeeRelationMsgParams{
		TemplateID: "bV8Jk-XXXXXX",
		Page:       "pages/order/detail?id=1",
		ToUser:     "oL7T268t3zlOb64IvrN64-XXXXXX",
		Data:       data,
	})
	if err != nil {
		log.Printf("推送用工消息失败: %v", err)

		return
	}

	if !sent.OK() {
		// 40256 表示该用户没有绑定用工关系，50001 表示用户未订阅模板。
		log.Printf("推送用工消息失败: %s",
			miniProgram.SendEmployeeRelationMsgErrMsg(sent.ErrCode))

		return
	}

	log.Print("推送用工消息成功")

	// 解绑是整批成功的：列表里有一个非法 openid，整批都不生效。
	unbound, err := app.UnbindUserB2CAuthInfo(ctx, miniProgram.UnbindUserB2CAuthInfoParams{
		OpenIDList: []string{"oL7T268t3zlOb64IvrN64-XXXXXX"},
	})
	if err != nil {
		log.Printf("解绑用工关系失败: %v", err)

		return
	}

	if !unbound.OK() {
		log.Printf("解绑用工关系失败: %s",
			miniProgram.UnbindUserB2CAuthInfoErrMsg(unbound.ErrCode))

		return
	}

	log.Print("解绑用工关系成功")
}
