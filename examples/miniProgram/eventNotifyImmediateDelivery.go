package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// ImmediateDeliveryEventNotify 演示即时配送的事件接入。
//
// 和物流助手是同一套「一问一答」的模型：配送公司推报文过来，必须回带 resultcode
// 的报文。区别只有字段名——回复里是小写的 resultcode / resultmsg，不是
// ResultCode / ResultMsg，写错了微信直接判失败。
//
// 六个事件里五个是运力方（配送公司）要实现的回调：微信（或商户）问什么，配送公司
// 答什么。剩下一个是商户侧接收配送状态变更（OnImmediateDeliveryOrderStatus），
// 方向和物流助手（小程序方使用）的那两个一致，但同样要回结构化报文。
//
// 下面的实现都是占位逻辑，注册完这些处理器之后，起服务时挂到同一个回调地址即可。
func ImmediateDeliveryEventNotify(app *miniProgram.Application) {
	// 模拟更新订单状态：商户联调时用它造一条状态变更，走的代码路径要和真实推送一致。
	app.OnImmediateDeliveryMockUpdateOrder(func(
		ctx context.Context, evt *miniProgram.ImmediateDeliveryMockUpdateOrderEvent,
	) (*miniProgram.ImmediateDeliveryReply, error) {
		log.Printf("[mock] 配送单 %s 状态变为 %d（%s）",
			evt.WaybillID, evt.OrderStatus, evt.StatusMsg())

		applyDeliveryStatus(ctx, evt.ShopOrderID, evt.WaybillID, evt.OrderStatus)

		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	// 查询订单状态：商户来问当前状态，必须把 order_status 和 waybill_id 都带回去，
	// 否则商户对不上自己的单子。
	app.OnImmediateDeliveryQueryOrder(func(
		ctx context.Context, evt *miniProgram.ImmediateDeliveryQueryOrderEvent,
	) (*miniProgram.ImmediateDeliveryReply, error) {
		status, ok := lookupDeliveryStatus(ctx, evt.ShopOrderID, evt.WaybillID)
		if !ok {
			reply := evt.NewReply()
			// 查不到单子属于业务拒绝：报文正常返回，resultcode 非 0。
			reply.Result(1, "配送单不存在")

			return reply, nil
		}

		return evt.StatusReply(status, ""), nil
	})

	// 预取消：先告诉商户取消费用，再让商户决定要不要取消。微信说这一步非必须，
	// 但不实现的话商户没法在取消前提示用户违约金。
	app.OnImmediateDeliveryPreCancelOrder(func(
		ctx context.Context, evt *miniProgram.ImmediateDeliveryPreCancelOrderEvent,
	) (*miniProgram.ImmediateDeliveryReply, error) {
		fee, desc := estimateCancelFee(ctx, evt.ShopOrderID, evt.WaybillID)

		reply := evt.NewReply()
		reply.Succeed()
		reply.DeductFee = fee
		reply.Desc = desc

		return reply, nil
	})

	// 取消订单：真正执行取消。这里同样要回违约金，免费也要回（DeductFee 为 0）。
	app.OnImmediateDeliveryCancelOrder(func(
		ctx context.Context, evt *miniProgram.ImmediateDeliveryCancelOrderEvent,
	) (*miniProgram.ImmediateDeliveryReply, error) {
		fee, desc := cancelDeliveryOrder(ctx, evt.ShopOrderID, evt.WaybillID)

		log.Printf("配送单 %s 取消，原因 %d %q，扣费 %.2f 元",
			evt.WaybillID, evt.CancelReasonID, evt.CancelReason, fee)

		reply := evt.NewReply()
		reply.Succeed()
		reply.DeductFee = fee
		reply.Desc = desc

		return reply, nil
	})

	// 异常妥投确认：骑手送不到、货已经退回商家，商户确认收货。
	// 只有达达、闪送、人人快送支持这个回调。
	app.OnImmediateDeliveryConfirmReturn(func(
		ctx context.Context, evt *miniProgram.ImmediateDeliveryConfirmReturnEvent,
	) (*miniProgram.ImmediateDeliveryReply, error) {
		log.Printf("配送单 %s 退回商家，商户确认收货", evt.WaybillID)

		confirmReturnReceived(ctx, evt.ShopOrderID, evt.WaybillID)

		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})

	// ---- 下面是商户侧接收的事件 ----

	// 配送状态变更：配送公司更新状态时推过来。终态（302 成功，或各种失败态）之后
	// 不会再有更新，可以据此关单。
	app.OnImmediateDeliveryOrderStatus(func(
		ctx context.Context, evt *miniProgram.ImmediateDeliveryOrderStatusEvent,
	) (*miniProgram.ImmediateDeliveryReply, error) {
		log.Printf("配送单 %s 状态 %d（%s）%s",
			evt.WaybillID, evt.OrderStatus, evt.StatusMsg(), evt.ActionMsg)

		if evt.Agent != nil {
			log.Printf("骑手 %s %s", evt.Agent.Name, evt.Agent.Phone)
		}

		applyDeliveryStatus(ctx, evt.ShopOrderID, evt.WaybillID, evt.OrderStatus)

		if evt.Terminal() {
			closeDeliveryOrder(ctx, evt.ShopOrderID, evt.WaybillID,
				miniProgram.ImmediateDeliveryStatusSucceeded(evt.OrderStatus))
		}

		reply := evt.NewReply()
		reply.Succeed()

		return reply, nil
	})
}

// 占位实现：真实接入时换成自己的数据访问层。
func applyDeliveryStatus(_ context.Context, _ string, _ string, _ int) {}

func lookupDeliveryStatus(_ context.Context, _ string, _ string) (int, bool) {
	return miniProgram.ImmediateDeliveryStatusDelivered, true
}

func estimateCancelFee(_ context.Context, _ string, _ string) (float64, string) {
	return 0, "骑手未接单，不收费"
}

func cancelDeliveryOrder(_ context.Context, _ string, _ string) (float64, string) {
	return 2.5, "骑手已取货"
}

func confirmReturnReceived(_ context.Context, _ string, _ string) {}

func closeDeliveryOrder(_ context.Context, _ string, _ string, _ bool) {}
