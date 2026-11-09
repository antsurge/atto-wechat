package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// 占位实现用到的错误，真实接入时换成数据访问层返回的错误。
var (
	errBizNotFound     = errors.New("客户编码不存在")
	errOrderNotFound   = errors.New("订单不存在")
	errAlreadyPickedUp = errors.New("已经揽收")
)

// ProviderEventNotify 演示物流助手（运力方使用）的事件接入。
//
// 这一组和公众号事件不同：微信推一个报文过来，运力方必须马上回一个结构相同的
// 报文，里面带 ResultCode。回 "success" 会被判为失败，订单会卡住。所以每个处理器
// 都要构造 reply 并返回，由框架按请求的格式（XML 或 JSON）写回。
//
// 结构如下：
//   - 处理函数返回 (reply, err)；
//   - err != nil 表示这次处理失败，微信会重试（或判失败）；
//   - reply.ResultCode != 0 表示业务上拒绝了请求，报文正常返回给微信展示给商户。
//
// 下面的实现都是占位逻辑：真实运力方要落库、调自己的下单系统。注册这些处理器之后，
// 由 main 里的 EventNotifyServer 起服务接收推送。
func ProviderEventNotify(app *miniProgram.Application) {
	// 审核商户：微信在下单前问一次「这个商户的账号在我这儿吗、还有多少余额」。
	// 查不到账号要回 10001，而不是返回 err——err 会让微信重试到放弃。
	app.OnProviderCheckBiz(func(ctx context.Context, evt *miniProgram.ProviderCheckBizEvent) (*miniProgram.ProviderCheckBizReply, error) {
		reply := evt.NewReply()

		quota, err := lookupProviderQuota(ctx, evt.BizID, evt.BizPwd)
		if err != nil {
			reply.Result(10002, "客户密码不正确")
			return reply, nil
		}

		reply.Quota = quota
		reply.Succeed()

		return reply, nil
	})

	// 查询余额：下单前再确认一次。
	app.OnProviderGetQuota(func(ctx context.Context, evt *miniProgram.ProviderGetQuotaEvent) (*miniProgram.ProviderGetQuotaReply, error) {
		reply := evt.NewReply()

		quota, err := lookupProviderQuota(ctx, evt.BizID, evt.BizPwd)
		if err != nil {
			reply.Result(10002, "客户密码不正确")
			return reply, nil
		}

		reply.Quota = quota
		reply.Succeed()

		return reply, nil
	})

	// 请求下单：整个接入里最重要的一环，把商户订单变成真实运单。
	//
	// 同一个 BizID + OrderID 必须返回同一个运单号：微信网络不稳时会重推，重复生成
	// 会让一个订单拆成两个包裹。
	app.OnProviderAddWaybill(func(ctx context.Context, evt *miniProgram.ProviderAddWaybillEvent) (*miniProgram.ProviderAddWaybillReply, error) {
		if !evt.OK() {
			reply := evt.NewReply()
			reply.Result(20002, "发件人信息不完整")
			return reply, nil
		}

		// 幂等：先按 BizID + OrderID 查一次，已下过单就直接返回原运单号。
		waybillID, err := findWaybillByOrder(ctx, evt.BizID, evt.OrderID)
		if err == nil {
			reply := evt.NewReply()
			reply.WayBillID = waybillID
			reply.Succeed()
			return reply, nil
		}

		waybillID, err = createWaybill(ctx, evt)
		if err != nil {
			reply := evt.NewReply()
			reply.Result(20006, "货物数量、重量或尺寸不正确或者不合理")
			return reply, nil
		}

		// 面单上的分拣信息按模板占位符拼，键名要和运力方模板一致。
		waybillData, err := miniProgram.ProviderWaybillData(
			"SF_bagAddr", "深圳南山分拣中心",
			"SF_mark", "888-666-666",
		)
		if err != nil {
			return nil, err
		}

		reply := evt.NewReply()
		reply.WayBillID = waybillID
		reply.WaybillData = waybillData
		reply.Succeed()

		return reply, nil
	})

	// 取消订单：商户在揽收前取消。已经揽收的不能取消，回 30002。
	app.OnProviderCancelWaybill(func(ctx context.Context, evt *miniProgram.ProviderCancelWaybillEvent) (*miniProgram.ProviderCancelWaybillReply, error) {
		reply := evt.NewReply()

		switch err := cancelWaybill(ctx, evt.WayBillID); {
		case err == errAlreadyPickedUp:
			reply.Result(30002, "已经揽收，不可取消")
		case err != nil:
			reply.Result(30003, "无效单")
		default:
			reply.Succeed()
		}

		return reply, nil
	})

	// 轨迹更新：这是纯推送，不需要结构化回复，返回 nil 就是 success。
	// Version 是轨迹版本，比本地旧的重推要丢掉，否则轨迹会回退。
	app.OnProviderAddExpressPath(func(ctx context.Context, evt *miniProgram.ProviderAddExpressPathEvent) error {
		if !evt.OK() {
			log.Printf("轨迹字段不完整，跳过: %+v", evt)
			return nil
		}

		if !isNewerPathVersion(evt.WaybillID, evt.Version) {
			log.Printf("运单 %s 的轨迹版本 %d 已处理过，跳过", evt.WaybillID, evt.Version)
			return nil
		}

		for _, action := range evt.Actions {
			log.Printf("运单 %s 轨迹: %s %s",
				evt.WaybillID,
				time.Unix(action.ActionTime, 0).Format(time.DateTime),
				action.ActionMsg)
			savePathAction(ctx, evt.WaybillID, action)
		}

		return nil
	})

	// 查询订单状态：微信要整条轨迹，用于用户的运单页。返回的是全量，不是增量。
	app.OnProviderWaybillNotifyQuery(func(ctx context.Context, evt *miniProgram.ProviderWaybillNotifyQueryEvent) (*miniProgram.ProviderWaybillNotifyQueryReply, error) {
		reply := evt.NewReply()

		nodes, createdAt, err := loadPath(ctx, evt.WaybillID)
		if err != nil {
			reply.Result(20001, "运单 ID 不正确")
			return reply, nil
		}

		reply.WaybillCreateTime = createdAt
		reply.Path = nodes
		reply.Succeed()

		return reply, nil
	})

	// 查询运单：订单详情页要用的全量信息，缺字段会显示成空白。
	app.OnProviderQuerySingleWaybill(func(ctx context.Context, evt *miniProgram.ProviderQuerySingleWaybillEvent) (*miniProgram.ProviderQuerySingleWaybillReply, error) {
		reply := evt.NewReply()

		order, err := loadSingleOrder(ctx, evt.Token)
		if err != nil {
			reply.Result(20001, "运单 ID 不正确")
			return reply, nil
		}

		reply.Sender = order.Sender
		reply.Receiver = order.Receiver
		reply.GoodDetail = order.Goods
		reply.PathInfo = &miniProgram.ProviderPathInfo{Nodes: order.Path}
		reply.Fee = order.Fee
		reply.OriginalFee = order.OriginalFee
		reply.OnlinePay = 1
		reply.PayResult = miniProgram.ProviderPayResultOnline
		reply.Succeed()

		return reply, nil
	})

	// 用户催单：还没生成运单号时可以只回预计送达时间。
	app.OnProviderRemindWaybill(func(ctx context.Context, evt *miniProgram.ProviderRemindWaybillEvent) (*miniProgram.ProviderRemindWaybillReply, error) {
		reply := evt.NewReply()

		waybillID, eta, err := lookupRemind(ctx, evt.Token)
		if err == nil {
			reply.WayBillID = waybillID
			reply.EstimatedDeliveryTime = eta
		}

		// 即使查不到也回成功：催单失败对用户没有意义，只会多一次投诉。
		reply.Succeed()

		return reply, nil
	})

	// 用户投诉：这里只确认收到，处理结果走 ProviderUpdateComplaintResult 回传。
	app.OnProviderComplaint(func(ctx context.Context, evt *miniProgram.ProviderComplaintEvent) (*miniProgram.ProviderReply, error) {
		log.Printf("运单 %s 收到投诉，类型 %d，内容 %q，图片 %v",
			evt.WayBillID, evt.ComplainType, evt.Content, evt.Pictures())

		createComplaintTicket(ctx, evt)

		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	// 给小哥评价：低分只做记录，微信不提供文字评价，别指望能拿到原因。
	app.OnProviderEvaluateAgent(func(ctx context.Context, evt *miniProgram.ProviderEvaluateAgentEvent) (*miniProgram.ProviderReply, error) {
		log.Printf("运单 %s 评价 %d 分，对象: %s",
			evt.WayBillID, evt.Score, courierTypeName(evt.CourierType))

		saveRating(ctx, evt.WayBillID, evt.CourierType, evt.Score)

		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	// ---- 以下是物流助手（小程序方使用）的事件 ----
	// 和上面相反：这两个只回 "success"，处理函数返回 error 即可。

	// 绑定商户审核结果：微信通知商户账号能不能用。
	// 审核失败时 errcode 非 0，这是正常业务结果，不要当错误返回。
	app.OnProviderBindResult(func(ctx context.Context, evt *miniProgram.ProviderBindResultEvent) error {
		if !evt.OK() {
			log.Printf("绑定审核结果字段不完整: %+v", evt)
			return nil
		}

		if evt.Succeeded() {
			log.Printf("物流账号 %s（%s）审核通过", evt.BizID, evt.DeliveryID)
			markAccountApproved(ctx, evt.DeliveryID, evt.BizID)

			return nil
		}

		log.Printf("物流账号 %s（%s）审核失败: %s", evt.BizID, evt.DeliveryID, evt.ErrMsg)
		markAccountRejected(ctx, evt.DeliveryID, evt.BizID, evt.ErrMsg)

		return nil
	})

	// 理赔结果：无忧退的赔付有结论了。XML 报文把字段放在 upload_event 里，
	// JSON 报文放在顶层，用 Claim 取值就不用关心是哪种格式。
	app.OnProviderClaimResult(func(ctx context.Context, evt *miniProgram.ProviderClaimResultEvent) error {
		if !evt.OK() {
			log.Printf("理赔结果字段不完整: %+v", evt)
			return nil
		}

		claim := evt.Claim()

		if claim.Status == miniProgram.ProviderClaimStatusSucceeded {
			log.Printf("支付单 %s 理赔成功，完成时间 %d", claim.OrderNo, claim.FinishTime)
			markClaimPaid(ctx, claim.OrderNo, claim.FinishTime)

			return nil
		}

		log.Printf("支付单 %s 理赔失败: %s", claim.OrderNo, claim.PayFailReason)
		markClaimFailed(ctx, claim.OrderNo, claim.PayFailReason)

		return nil
	})
}

// courierTypeName 把 CourierType 翻译成可读的名字。
func courierTypeName(courierType int) string {
	if courierType == miniProgram.ProviderCourierTypeDelivery {
		return "派件员"
	}

	return "取件员"
}

// 下面都是占位实现，真实接入时换成自己的数据访问层。

func lookupProviderQuota(_ context.Context, bizID, _ string) (int64, error) {
	if bizID == "" {
		return 0, errBizNotFound
	}

	return 100000, nil
}

func findWaybillByOrder(_ context.Context, _ string, orderID string) (string, error) {
	if orderID == "" {
		return "", errOrderNotFound
	}

	return "", errOrderNotFound
}

func createWaybill(_ context.Context, evt *miniProgram.ProviderAddWaybillEvent) (string, error) {
	log.Printf("为订单 %s 生成运单，发件人 %s，收件人 %s",
		evt.OrderID, evt.Sender.Name, evt.Receiver.Name)

	return "SF" + evt.OrderID, nil
}

func cancelWaybill(_ context.Context, _ string) error { return nil }

func isNewerPathVersion(_ string, _ int) bool { return true }

func savePathAction(_ context.Context, _ string, _ miniProgram.ProviderExpressPathAction) {}

func loadPath(_ context.Context, _ string) ([]miniProgram.ProviderNotifyPathNode, int64, error) {
	return nil, time.Now().Unix(), nil
}

func loadSingleOrder(_ context.Context, _ string) (*providerOrder, error) {
	return &providerOrder{}, nil
}

func lookupRemind(_ context.Context, _ string) (string, string, error) {
	return "", "", errOrderNotFound
}

func createComplaintTicket(_ context.Context, _ *miniProgram.ProviderComplaintEvent) {}

func saveRating(_ context.Context, _ string, _ int, _ int) {}

func markAccountApproved(_ context.Context, _ string, _ string) {}

func markAccountRejected(_ context.Context, _ string, _ string, _ string) {}

func markClaimPaid(_ context.Context, _ string, _ int64) {}

func markClaimFailed(_ context.Context, _ string, _ string) {}

// providerOrder 是占位用的订单结构。
type providerOrder struct {
	Sender      *miniProgram.ProviderAddress
	Receiver    *miniProgram.ProviderAddress
	Goods       *miniProgram.ProviderGoods
	Path        []miniProgram.ProviderPathNode
	Fee         int64
	OriginalFee int64
}
