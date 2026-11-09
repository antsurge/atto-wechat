package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// 物流助手（小程序使用）示例。
//
// 调用顺序有依赖：先用 getAllDelivery 取快递公司 id，再用 getall 取已绑定的
// biz_id，有了这两个才能下单。下面的 biz_id / waybill_id 都是占位值，换成
// 自己账号里的真实值才能跑通。

// ListLogisticsDelivery 列出可用的快递公司，结果里的 DeliveryID 是后续所有
// 接口的 delivery_id 来源。
func ListLogisticsDelivery(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetAllDelivery(ctx)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetAllDeliveryError(res)
	}

	for _, company := range res.Data {
		log.Printf("delivery_id=%s name=%s canUseCash=%d canGetQuota=%d",
			company.DeliveryID, company.DeliveryName, company.CanUseCash, company.CanGetQuota)
	}

	return nil
}

// ListLogisticsAccount 列出已绑定的月结账号。
func ListLogisticsAccount(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetAllAccount(ctx)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetAllAccountError(res)
	}

	for _, account := range res.List {
		// 审核未通过的账号 statusCode 非 0，用它区分能不能直接下单。
		log.Printf("biz_id=%s delivery_id=%s status=%d quota=%d",
			account.BizID, account.DeliveryID, account.StatusCode, account.QuotaNum)
	}

	return nil
}

// BindLogisticsAccount 绑定月结账号；解绑把 Type 换成
// miniProgram.LogisticsBindTypeUnbind。
func BindLogisticsAccount(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.BindAccount(ctx, miniProgram.BindAccountParams{
		Type:       miniProgram.LogisticsBindTypeBind,
		BizID:      "biz-1",
		DeliveryID: "SF",
		Password:   "customer-password",
		// EMS 审核时要求填联系人、电话、服务类型、发货地址。
		RemarkContent: "电话/联系人/服务类型/发货地址",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.BindAccountError(res)
	}

	log.Println("BindAccount: ok")

	return nil
}

// GetLogisticsQuota 查电子面单余额；CanGetQuota 为 0 的快递公司不支持。
func GetLogisticsQuota(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetQuota(ctx, miniProgram.GetQuotaParams{
		DeliveryID: "SF",
		BizID:      "biz-1",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetQuotaError(res)
	}

	log.Printf("GetQuota: quota=%d updatedAt=%d", res.QuotaNum, res.QuotaUpdateTime)

	return nil
}

// AddLogisticsOrder 生成电子面单。
//
// order_id 是幂等键：同一个订单重复调用返回同一张运单，所以超时后可以放心重试。
// 失败有两层——微信 errcode 和运力方 delivery_resultcode，OK() 会把两层都判掉，
// 出错时 AddOrderError 会给出发起方或运力方的原因。
func AddLogisticsOrder(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.AddOrder(ctx, miniProgram.AddOrderParams{
		OrderID:    "order-1",
		OpenID:     "o_user_openid",
		DeliveryID: "SF",
		BizID:      "biz-1",
		AddSource:  miniProgram.LogisticsAddSourceMiniProgram,
		Sender: miniProgram.LogisticsPersonInfo{
			Name:     "发货人",
			Mobile:   "13800000000",
			Province: "广东省",
			City:     "深圳市",
			Area:     "南山区",
			Address:  "科技园 1 号",
		},
		Receiver: miniProgram.LogisticsPersonInfo{
			Name:     "收货人",
			Mobile:   "13900000000",
			Province: "北京市",
			City:     "北京市",
			Area:     "海淀区",
			Address:  "中关村 1 号",
		},
		Cargo: miniProgram.LogisticsCargo{
			Count:      1,
			Weight:     0.5,
			SpaceX:     20,
			SpaceY:     15,
			SpaceZ:     10,
			DetailList: []miniProgram.LogisticsCargoDetail{{Name: "图书", Count: 1}},
		},
		// shop 决定物流服务通知里显示的商品信息，不传就没有商品卡片。
		Shop: &miniProgram.LogisticsShop{
			WxaPath:    "pages/order/detail?id=order-1",
			ImgURL:     "https://example.com/goods.png",
			GoodsName:  "图书",
			GoodsCount: 1,
		},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.AddOrderError(res)
	}

	log.Printf("AddOrder: waybill_id=%s", res.WaybillID)

	return nil
}

// PrintLogisticsOrder 取面单打印数据；GetOrder 返回的 print_html 交给云打印。
func PrintLogisticsOrder(ctx context.Context, app *miniProgram.Application, waybillID string) error {
	res, err := app.GetOrder(ctx, miniProgram.GetOrderParams{
		OrderID:    "order-1",
		DeliveryID: "SF",
		WaybillID:  waybillID,
		PrintType:  miniProgram.LogisticsPrintTypeOnePart,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetOrderError(res)
	}

	log.Printf("GetOrder: waybill=%s status=%d printHtml=%d bytes",
		res.WaybillID, res.OrderStatus, len(res.PrintHTML))

	return nil
}

// BatchPrintLogisticsOrder 批量取面单，单次最多 100 条，适合每天集中打印。
func BatchPrintLogisticsOrder(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.BatchGetOrder(ctx, miniProgram.BatchGetOrderParams{
		OrderList: []miniProgram.LogisticsOrderQuery{
			{OrderID: "order-1", DeliveryID: "SF", WaybillID: "SF123"},
			{OrderID: "order-2", DeliveryID: "SF", WaybillID: "SF124"},
		},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.BatchGetOrderError(res)
	}

	for _, order := range res.OrderList {
		log.Printf("BatchGetOrder: waybill=%s printHtml=%d bytes", order.WaybillID, len(order.PrintHTML))
	}

	return nil
}

// GetLogisticsPath 查运单轨迹；刚下单还没有揽件记录时返回空列表，不算失败。
func GetLogisticsPath(ctx context.Context, app *miniProgram.Application, waybillID string) error {
	res, err := app.GetPath(ctx, miniProgram.GetPathParams{
		DeliveryID: "SF",
		WaybillID:  waybillID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetPathError(res)
	}

	if latest := res.Latest(); latest != nil {
		log.Printf("GetPath: %s at %d", latest.ActionMsg, latest.ActionTime)
	} else {
		log.Println("GetPath: 暂无轨迹")
	}

	return nil
}

// CancelLogisticsOrder 取消运单；已揽收的包裹运力方可能拒绝，此时 OK() 为 false。
func CancelLogisticsOrder(ctx context.Context, app *miniProgram.Application, waybillID string) error {
	res, err := app.CancelOrder(ctx, miniProgram.CancelOrderParams{
		OrderID:    "order-1",
		OpenID:     "o_user_openid",
		DeliveryID: "SF",
		WaybillID:  waybillID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.CancelOrderError(res)
	}

	log.Println("CancelOrder: ok")

	return nil
}

// UpdateLogisticsPrinter 给员工开面单打印权限；撤销用
// miniProgram.LogisticsPrinterUpdateUnbind。
func UpdateLogisticsPrinter(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.UpdatePrinter(ctx, miniProgram.UpdatePrinterParams{
		OpenID:     "o_employee_openid",
		TagIDList:  "123,456",
		UpdateType: miniProgram.LogisticsPrinterUpdateBind,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.UpdatePrinterError(res)
	}

	log.Println("UpdatePrinter: ok")

	return nil
}

// ListLogisticsPrinter 列出有打印权限的员工。
func ListLogisticsPrinter(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetAllPrinter(ctx)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetAllPrinterError(res)
	}

	log.Printf("GetAllPrinter: %d printers %v", res.Count, res.OpenID)

	return nil
}

// TestUpdateLogisticsOrder 注入一条假轨迹，只用于联调；biz_id 必须是
// test_biz_id、delivery_id 必须是 TEST，买家看到的轨迹和真实的一样，别在线上调。
func TestUpdateLogisticsOrder(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.TestUpdateOrder(ctx, miniProgram.TestUpdateOrderParams{
		BizID:      "test_biz_id",
		OrderID:    "order-1",
		DeliveryID: "TEST",
		WaybillID:  "TEST123",
		ActionTime: 1735689600,
		ActionType: 100001,
		ActionMsg:  "已揽件",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.TestUpdateOrderError(res)
	}

	log.Println("TestUpdateOrder: ok")

	return nil
}
