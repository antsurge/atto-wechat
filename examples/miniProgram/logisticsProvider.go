package main

import (
	"context"
	"encoding/base64"
	"log"
	"os"
	"time"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// 物流助手（运力方使用）示例。
//
// 这一组接口由事件驱动：微信把 add_waybill 事件推到运力方的回调地址，报文里
// 带 token，之后所有操作都用这个 token 定位运单，而不是商户的 order_id。
// 所以下面的函数都以 token 为入口参数，由事件处理器把值传进来。
//
// 这里的 token、waybill_id、appid 都是占位值，换成真实值才能跑通。

// ApproveProviderAccount 回复商户的入驻审核结果。result_code 为 0 是审核通过，
// 非 0 是驳回，驳回时 result_msg 展示给商户，写清楚缺什么材料。
func ApproveProviderAccount(ctx context.Context, app *miniProgram.Application) error {
	return updateProviderBusiness(ctx, app, 0, "")
}

// RejectProviderAccount 驳回商户的入驻申请并给出原因。
func RejectProviderAccount(ctx context.Context, app *miniProgram.Application) error {
	return updateProviderBusiness(ctx, app, 1001, "营业执照照片不清晰，请重新上传")
}

func updateProviderBusiness(ctx context.Context, app *miniProgram.Application, code int, msg string) error {
	res, err := app.ProviderUpdateBusiness(ctx, miniProgram.ProviderUpdateBusinessParams{
		ShopAppID:  "wx1234567890abcdef",
		BizID:      "biz-1",
		ResultCode: code,
		ResultMsg:  msg,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ProviderUpdateBusinessError(res)
	}

	log.Printf("审核结果已回复 result_code=%d", code)
	return nil
}

// PreviewProviderTemplate 用商户下单时带的 waybill_data 渲染一次面单模板，
// 结果是一张 base64 编码的 HTML，落成文件就能在浏览器里看。
//
// 调试模板时用它，比真的下一单再打印快得多。
func PreviewProviderTemplate(ctx context.Context, app *miniProgram.Application, waybillID string) error {
	// waybill_data 在报文里是 JSON 字符串，用 WaybillDataJSON 拼，不要手写。
	waybillData, err := miniProgram.WaybillDataJSON([]miniProgram.LogisticsWaybillData{
		{Key: "sender_name", Value: "发货人"},
		{Key: "receiver_name", Value: "收货人"},
	})
	if err != nil {
		return err
	}

	template, err := os.ReadFile("waybill_template.html")
	if err != nil {
		return err
	}

	res, err := app.ProviderPreviewTemplate(ctx, miniProgram.ProviderPreviewTemplateParams{
		WaybillID:       waybillID,
		WaybillTemplate: base64.StdEncoding.EncodeToString(template),
		WaybillData:     waybillData,
		// custom 是商户下单时的原始报文，字段与 AddOrderParams 完全一致。
		Custom: miniProgram.AddOrderParams{
			OrderID:    "order-1",
			OpenID:     "o_user",
			DeliveryID: "SF",
			BizID:      "biz-1",
			AddSource:  miniProgram.LogisticsAddSourceMiniProgram,
			Sender: miniProgram.LogisticsPersonInfo{
				Name: "发货人", Mobile: "13800000000",
				Province: "广东省", City: "深圳市", Area: "南山区", Address: "科技园 1 号",
			},
			Receiver: miniProgram.LogisticsPersonInfo{
				Name: "收货人", Mobile: "13900000000",
				Province: "北京市", City: "北京市", Area: "海淀区", Address: "中关村 1 号",
			},
			Cargo: miniProgram.LogisticsCargo{
				Count: 1, Weight: 0.5,
				DetailList: []miniProgram.LogisticsCargoDetail{{Name: "图书", Count: 1}},
			},
		},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ProviderPreviewTemplateError(res)
	}

	rendered, err := base64.StdEncoding.DecodeString(res.RenderedWaybillTemplate)
	if err != nil {
		return err
	}
	if err := os.WriteFile("waybill_preview.html", rendered, 0o600); err != nil {
		return err
	}

	log.Printf("模板已渲染到 waybill_preview.html（%d 字节）", len(rendered))
	return nil
}

// GetProviderContact 取面单上的收寄件人信息。运力方拿到的地址是合并成一行的，
// 没有省市区分段。
func GetProviderContact(ctx context.Context, app *miniProgram.Application, token, waybillID string) error {
	res, err := app.ProviderGetContact(ctx, miniProgram.ProviderGetContactParams{
		Token:     token,
		WaybillID: waybillID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ProviderGetContactError(res)
	}

	log.Printf("寄件人 %s %s %s", res.Sender.Name, res.Sender.Mobile, res.Sender.Address)
	log.Printf("收件人 %s %s %s", res.Receiver.Name, res.Receiver.Mobile, res.Receiver.Address)
	return nil
}

// PushProviderPath 回传物流轨迹，用户在运单页看到的就是这条。
func PushProviderPath(ctx context.Context, app *miniProgram.Application, token, waybillID string) error {
	res, err := app.ProviderUpdatePath(ctx, miniProgram.ProviderUpdatePathParams{
		Token:      token,
		WaybillID:  waybillID,
		ActionTime: time.Now().Unix(),
		ActionType: miniProgram.ProviderActionInTransit,
		ActionMsg:  "快件已到达深圳转运中心",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ProviderUpdatePathError(res)
	}

	log.Printf("轨迹已回传")
	return nil
}

// UpdateProviderOrderStatus 推进揽件前的订单状态（网点接单、分配业务员、支付）。
// 这些阶段还没有面单，waybill_id 可以留空。
func UpdateProviderOrderStatus(ctx context.Context, app *miniProgram.Application, token string) error {
	res, err := app.ProviderUpdateOrderStatus(ctx, miniProgram.ProviderUpdateOrderStatusParams{
		Token:              token,
		ActionTime:         time.Now().Unix(),
		ActionType:         miniProgram.ProviderActionCourierAssigned,
		ActionMsg:          "已分配快递员，电话 13800000000",
		PickupCourierName:  "张三",
		PickupCourierPhone: "13800000000",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ProviderUpdateOrderStatusError(res)
	}

	log.Printf("订单状态已更新")
	return nil
}

// CancelProviderOrder 运力方取消订单，例如收件地址超出派送范围。
// reason 会展示给商户，写清楚原因。
func CancelProviderOrder(ctx context.Context, app *miniProgram.Application, token string) error {
	res, err := app.ProviderCancelOrder(ctx, miniProgram.ProviderCancelOrderParams{
		Token:  token,
		Reason: "收件地址超出派送范围",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ProviderCancelOrderError(res)
	}

	log.Printf("订单已取消")
	return nil
}

// UpdateProviderOrderFee 更新买家需要支付的运费，金额单位是分。
// original_fee 必须等于 base_fee + insured_fee + other_fee。
func UpdateProviderOrderFee(ctx context.Context, app *miniProgram.Application, token, waybillID string) error {
	res, err := app.ProviderUpdateOrderFee(ctx, miniProgram.ProviderUpdateOrderFeeParams{
		Token:        token,
		WaybillID:    waybillID,
		NeedPay:      miniProgram.ProviderNeedPayWechat,
		Fee:          1500,
		OriginalFee:  1500,
		BaseFee:      1500,
		PayGoodsName: "运费",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ProviderUpdateOrderFeeError(res)
	}

	log.Printf("运费已更新")
	return nil
}

// RefundProviderOrder 退款，金额单位是分。
func RefundProviderOrder(ctx context.Context, app *miniProgram.Application, token string) error {
	res, err := app.ProviderRefundOrder(ctx, miniProgram.ProviderRefundOrderParams{
		Token: token,
		Fee:   500,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ProviderRefundOrderError(res)
	}

	log.Printf("退款已提交")
	return nil
}

// DownloadProviderBill 下载对账单。微信每天生成一次，当天通常还没有内容，
// 所以取昨天的日期。成功时返回的就是文件本身，失败才是 JSON。
func DownloadProviderBill(ctx context.Context, app *miniProgram.Application) error {
	day := time.Now().AddDate(0, 0, -1).Format("20060102")

	res, err := app.ProviderGetBill(ctx, miniProgram.ProviderGetBillParams{
		Date: day,
		Type: miniProgram.ProviderBillTypeAll,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ProviderGetBillError(res)
	}

	name := "bill_" + day + ".csv"
	if err := os.WriteFile(name, res.Content, 0o600); err != nil {
		return err
	}

	log.Printf("对账单已保存到 %s（%d 字节，%s）", name, len(res.Content), res.ContentType)
	return nil
}

// ReplyProviderComplaint 回复用户投诉的处理结果。desc 是给买家看的，
// 别写内部备注。
func ReplyProviderComplaint(ctx context.Context, app *miniProgram.Application, token, waybillID string) error {
	res, err := app.ProviderUpdateComplaintResult(ctx, miniProgram.ProviderUpdateComplaintResultParams{
		Token:     token,
		WaybillID: waybillID,
		Result:    "已赔付",
		Desc:      "已按保价金额赔付，3 个工作日内到账",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ProviderUpdateComplaintResultError(res)
	}

	log.Printf("投诉处理结果已回复")
	return nil
}
