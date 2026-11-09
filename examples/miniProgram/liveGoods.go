package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// AddLiveGoods 添加商品并提审。
//
// 商品要先审核通过才能导入直播间。priceType 决定 price/price2 的含义：
// 一口价只用 price，价格区间和折扣价要同时给 price 和 price2。
// 返回的 errcode 300021 表示商品已入库但审核没过，goodsId 仍可用于修改后重提。
func AddLiveGoods(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.AddLiveGoods(ctx, miniProgram.AddLiveGoodsParams{
		GoodsInfo: miniProgram.LiveGoodsInfo{
			CoverImgURL: "media-goods-cover",
			Name:        "春季新品",
			PriceType:   miniProgram.LiveGoodsPriceTypeFixed,
			Price:       99.00,
			URL:         "pages/goods/detail?id=1",
		},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.AddLiveGoodsError(res)
	}

	log.Printf("AddLiveGoods: goodsId=%d auditId=%d", res.GoodsID, res.AuditID)

	return nil
}

// UpdateLiveGoods 修改商品。
//
// 审核中和已通过的商品能改的字段不同：已通过的商品只允许改价格类型和价格，
// 其它字段要等审核被拒后改再重提。
func UpdateLiveGoods(
	ctx context.Context,
	app *miniProgram.Application,
	goodsID int64,
) error {
	res, err := app.UpdateLiveGoods(ctx, miniProgram.UpdateLiveGoodsParams{
		GoodsInfo: miniProgram.LiveGoodsInfo{
			GoodsID:     goodsID,
			CoverImgURL: "media-goods-cover",
			Name:        "春季新品（调价）",
			PriceType:   miniProgram.LiveGoodsPriceTypeDiscount,
			Price:       129.00,
			Price2:      99.00,
			URL:         "pages/goods/detail?id=1",
		},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.UpdateLiveGoodsError(res)
	}

	return nil
}

// ResubmitLiveGoodsAudit 重新提交商品审核。
//
// 只对审核失败的商品有效，且要先 UpdateLiveGoods 改掉被拒的原因再重提。
func ResubmitLiveGoodsAudit(
	ctx context.Context,
	app *miniProgram.Application,
	goodsID int64,
) error {
	res, err := app.ResubmitLiveGoodsAudit(ctx, miniProgram.ResubmitLiveGoodsAuditParams{
		GoodsID: goodsID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ResubmitLiveGoodsAuditError(res)
	}

	log.Printf("ResubmitLiveGoodsAudit: auditId=%d", res.AuditID)

	return nil
}

// ResetLiveGoodsAudit 撤回商品审核。
//
// 撤回后商品回到可编辑状态，改完可以重新提审；只对审核中的商品有效。
func ResetLiveGoodsAudit(
	ctx context.Context,
	app *miniProgram.Application,
	goodsID int64,
	auditID int64,
) error {
	res, err := app.ResetLiveGoodsAudit(ctx, miniProgram.ResetLiveGoodsAuditParams{
		GoodsID: goodsID,
		AuditID: auditID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ResetLiveGoodsAuditError(res)
	}

	return nil
}

// GetLiveGoodsAuditInfo 查询商品审核状态。
//
// 只有 auditStatus 为 LiveGoodsAuditApproved 的商品才能导入直播间，
// 被拒时 auditMsg 是原因。
func GetLiveGoodsAuditInfo(
	ctx context.Context,
	app *miniProgram.Application,
	goodsIDs []int64,
) error {
	res, err := app.GetLiveGoodsAuditInfo(ctx, miniProgram.GetLiveGoodsAuditInfoParams{
		GoodsIDs: goodsIDs,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetLiveGoodsAuditInfoError(res)
	}

	for _, goods := range res.Goods {
		log.Printf(
			"GetLiveGoodsAuditInfo: %d %s status=%d msg=%s",
			goods.GoodsID,
			goods.Name,
			goods.AuditStatus,
			goods.AuditMsg,
		)
	}

	return nil
}

// GetLiveGoodsList 按审核状态分页拉取商品库。
//
// 这是商品库，不是某个直播间的商品架；房间里的商品看 GetLiveRoomList。
func GetLiveGoodsList(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetLiveGoodsList(ctx, miniProgram.GetLiveGoodsListParams{
		Offset: 0,
		Limit:  20,
		Status: miniProgram.LiveGoodsAuditApproved,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetLiveGoodsListError(res)
	}

	for _, goods := range res.Goods {
		log.Printf("GetLiveGoodsList: %d %s %.2f", goods.GoodsID, goods.Name, goods.Price)
	}

	return nil
}

// DeleteLiveGoods 删除商品。
//
// 商品会同时从所有房间的商品架上消失，且不可恢复。
func DeleteLiveGoods(
	ctx context.Context,
	app *miniProgram.Application,
	goodsID int64,
) error {
	res, err := app.DeleteLiveGoods(ctx, miniProgram.DeleteLiveGoodsParams{GoodsID: goodsID})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.DeleteLiveGoodsError(res)
	}

	return nil
}
