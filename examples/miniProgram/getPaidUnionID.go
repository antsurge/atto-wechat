package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// GetPaidUnionID 获取刚完成支付的用户的 UnionID。
//
// openID 是用户在本小程序的 openid，transactionID 是微信支付订单号；
// 也可以在调用处改用 miniProgram.PaidUnionIDParams 的 MchID/OutTradeNo 组合。
//
// 只能在支付完成后 5 分钟内调用，且小程序需已绑定开放平台账号，
// 否则微信会返回 89300（订单无效）或 89002（未绑定开放平台账号）。
func GetPaidUnionID(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
	transactionID string,
) error {
	res, err := app.GetPaidUnionID(ctx, miniProgram.PaidUnionIDParams{
		OpenID:        openID,
		TransactionID: transactionID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetPaidUnionIDError(res)
	}

	log.Printf("GetPaidUnionID unionid: %s", res.UnionID)

	return nil
}
