package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// CreateLiveRoom 创建直播间。
//
// 封面、分享图、购物频道封面都是 UploadTempMedia 返回的 mediaID，三天后失效，
// 所以开播时间超过三天的房间，要么晚点创建，要么在开播前 EditLiveRoom 换图。
func CreateLiveRoom(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.CreateLiveRoom(ctx, miniProgram.CreateLiveRoomParams{
		Name:          "春季新品直播",
		CoverImg:      "media-cover",
		StartTime:     1735689600,
		EndTime:       1735693200,
		AnchorName:    "主播昵称",
		AnchorWechat:  "anchor_wechat",
		ShareImg:      "media-share",
		FeedsImg:      "media-feeds",
		LiveType:      miniProgram.LiveRoomTypePhone,
		IsFeedsPublic: miniProgram.IntPtr(1),
		CloseKf:       miniProgram.IntPtr(0),
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.CreateLiveRoomError(res)
	}

	log.Printf("CreateLiveRoom: roomId=%d qrcode=%s", res.RoomID, res.QRCodeURL)

	return nil
}

// EditLiveRoom 修改直播间。
//
// 微信是用请求体整体替换房间，所以每个必填字段都要带上当前值，哪怕只改一项。
func EditLiveRoom(ctx context.Context, app *miniProgram.Application, roomID int64) error {
	res, err := app.EditLiveRoom(ctx, miniProgram.EditLiveRoomParams{
		RoomID:       roomID,
		Name:         "春季新品直播（改期）",
		CoverImg:     "media-cover",
		StartTime:    1735693200,
		EndTime:      1735696800,
		AnchorName:   "主播昵称",
		AnchorWechat: "anchor_wechat",
		ShareImg:     "media-share",
		FeedsImg:     "media-feeds",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.EditLiveRoomError(res)
	}

	return nil
}

// GetLiveRoomList 分页拉取直播间列表。
//
// Action 设为 get_replay 时改为拉某个房间的回放片段，RoomID 必填。
func GetLiveRoomList(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetLiveRoomList(ctx, miniProgram.GetLiveRoomListParams{
		Start: 0,
		Limit: 20,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetLiveRoomListError(res)
	}

	for _, room := range res.RoomInfo {
		log.Printf("GetLiveRoomList: %d %s status=%d", room.RoomID, room.Name, room.LiveStatus)
	}

	return nil
}

// GetLiveRoomReplay 拉取某个直播间的回放片段。
//
// 回放一般在直播结束十分钟内生成，media_url 有有效期，过期后要重新拉。
func GetLiveRoomReplay(ctx context.Context, app *miniProgram.Application, roomID int64) error {
	res, err := app.GetLiveRoomList(ctx, miniProgram.GetLiveRoomListParams{
		Start:  0,
		Limit:  20,
		Action: miniProgram.LiveRoomActionGetReplay,
		RoomID: roomID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetLiveRoomListError(res)
	}

	for _, clip := range res.LiveReplay {
		log.Printf("GetLiveRoomReplay: %s 过期于 %s", clip.MediaURL, clip.ExpireTime)
	}

	return nil
}

// DeleteLiveRoom 删除直播间。
//
// 只删房间，商品仍然留在商品库里。
func DeleteLiveRoom(ctx context.Context, app *miniProgram.Application, roomID int64) error {
	res, err := app.DeleteLiveRoom(ctx, miniProgram.DeleteLiveRoomParams{RoomID: roomID})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.DeleteLiveRoomError(res)
	}

	return nil
}

// GetLiveRoomPushURL 获取推流地址。
//
// 只对 LiveRoomTypePush 的房间有意义：手机开播用的是小程序直播助手，不需要推流地址。
func GetLiveRoomPushURL(ctx context.Context, app *miniProgram.Application, roomID int64) error {
	res, err := app.GetLiveRoomPushURL(ctx, miniProgram.GetLiveRoomPushURLParams{RoomID: roomID})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetLiveRoomPushURLError(res)
	}

	log.Printf("GetLiveRoomPushURL: %s", res.PushAddr)

	return nil
}

// GetLiveRoomSharedCode 获取直播间分享码。
//
// Params 会原样带进房间，用来把访客归因到分享来源。
func GetLiveRoomSharedCode(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
) error {
	res, err := app.GetLiveRoomSharedCode(ctx, miniProgram.GetLiveRoomSharedCodeParams{
		RoomID: roomID,
		Params: "source=wechat",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetLiveRoomSharedCodeError(res)
	}

	log.Printf("GetLiveRoomSharedCode: cdn=%s page=%s", res.CDNURL, res.PagePath)

	return nil
}

// GetSubAnchor 查询主播副号。
//
// 一个房间最多一个副号，未设置时用户名返回空字符串而不是报错。
func GetSubAnchor(ctx context.Context, app *miniProgram.Application, roomID int64) error {
	res, err := app.GetSubAnchor(ctx, miniProgram.GetSubAnchorParams{RoomID: roomID})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetSubAnchorError(res)
	}

	log.Printf("GetSubAnchor: %s", res.Username)

	return nil
}

// AddSubAnchor 添加主播副号。
//
// 副号需要完成实名认证，且不能与主号或已有副号重复。
func AddSubAnchor(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	username string,
) error {
	res, err := app.AddSubAnchor(ctx, miniProgram.AddSubAnchorParams{
		RoomID:   roomID,
		Username: username,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.AddSubAnchorError(res)
	}

	return nil
}

// ModifySubAnchor 修改主播副号。
//
// 直接换成新账号，不需要先删除。
func ModifySubAnchor(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	username string,
) error {
	res, err := app.ModifySubAnchor(ctx, miniProgram.ModifySubAnchorParams{
		RoomID:   roomID,
		Username: username,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ModifySubAnchorError(res)
	}

	return nil
}

// DeleteSubAnchor 删除主播副号。
func DeleteSubAnchor(ctx context.Context, app *miniProgram.Application, roomID int64) error {
	res, err := app.DeleteSubAnchor(ctx, miniProgram.DeleteSubAnchorParams{RoomID: roomID})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.DeleteSubAnchorError(res)
	}

	return nil
}

// ImportLiveRoomGoods 把商品导入直播间。
//
// 只有审核通过、且通过接口添加的商品能导入；直播间有商品数量上限。
func ImportLiveRoomGoods(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	goodsIDs []int64,
) error {
	res, err := app.ImportLiveRoomGoods(ctx, miniProgram.ImportLiveRoomGoodsParams{
		RoomID:   roomID,
		GoodsIDs: goodsIDs,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ImportLiveRoomGoodsError(res)
	}

	return nil
}

// DeleteLiveRoomGoods 把商品移出直播间。
//
// 只影响这个房间的商品架，商品本身仍留在商品库里。
func DeleteLiveRoomGoods(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	goodsID int64,
) error {
	res, err := app.DeleteLiveRoomGoods(ctx, miniProgram.DeleteLiveRoomGoodsParams{
		RoomID:  roomID,
		GoodsID: goodsID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.DeleteLiveRoomGoodsError(res)
	}

	return nil
}

// PushLiveRoomGoods 推送商品讲解。
//
// 推送后商品卡片会弹到观众端；同一房间同时只能有一个商品在推送中。
func PushLiveRoomGoods(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	goodsID int64,
) error {
	res, err := app.PushLiveRoomGoods(ctx, miniProgram.PushLiveRoomGoodsParams{
		RoomID:  roomID,
		GoodsID: goodsID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.PushLiveRoomGoodsError(res)
	}

	return nil
}

// SaleLiveRoomGoods 商品上下架。
//
// 上下架是按房间生效的，且在推送过程中不能切换。
func SaleLiveRoomGoods(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	goodsID int64,
	onSale bool,
) error {
	onSaleFlag := 0
	if onSale {
		onSaleFlag = 1
	}

	res, err := app.SaleLiveRoomGoods(ctx, miniProgram.SaleLiveRoomGoodsParams{
		RoomID:  roomID,
		GoodsID: goodsID,
		OnSale:  onSaleFlag,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SaleLiveRoomGoodsError(res)
	}

	return nil
}

// SortLiveRoomGoods 调整直播间商品顺序。
//
// 传的是调整后完整的商品列表，商品 id 要与房间里现有的商品一致。
func SortLiveRoomGoods(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	goodsIDs []int64,
) error {
	goods := make([]miniProgram.LiveGoodsOrder, 0, len(goodsIDs))
	for _, goodsID := range goodsIDs {
		goods = append(goods, miniProgram.LiveGoodsOrder{GoodsID: goodsID})
	}

	res, err := app.SortLiveRoomGoods(ctx, miniProgram.SortLiveRoomGoodsParams{
		RoomID: roomID,
		Goods:  goods,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SortLiveRoomGoodsError(res)
	}

	return nil
}

// AddLiveRoomAssistant 添加直播小助手。
//
// 小助手是房间级的，可以管理评论和商品，但不是主播。
func AddLiveRoomAssistant(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	username string,
	nickname string,
) error {
	res, err := app.AddLiveRoomAssistant(ctx, miniProgram.AddLiveRoomAssistantParams{
		RoomID: roomID,
		Users: []miniProgram.LiveRoomAssistant{
			{Username: username, Nickname: nickname},
		},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.AddLiveRoomAssistantError(res)
	}

	return nil
}

// ModifyLiveRoomAssistant 修改小助手昵称。
func ModifyLiveRoomAssistant(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	username string,
	nickname string,
) error {
	res, err := app.ModifyLiveRoomAssistant(ctx, miniProgram.ModifyLiveRoomAssistantParams{
		RoomID:   roomID,
		Username: username,
		Nickname: nickname,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.ModifyLiveRoomAssistantError(res)
	}

	return nil
}

// GetLiveRoomAssistantList 查询直播间的小助手列表。
func GetLiveRoomAssistantList(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
) error {
	res, err := app.GetLiveRoomAssistantList(
		ctx,
		miniProgram.GetLiveRoomAssistantListParams{RoomID: roomID},
	)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetLiveRoomAssistantListError(res)
	}

	for _, assistant := range res.List {
		log.Printf("GetLiveRoomAssistantList: %s %s", assistant.Username, assistant.Nickname)
	}

	return nil
}

// RemoveLiveRoomAssistant 移除直播小助手。
func RemoveLiveRoomAssistant(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	username string,
) error {
	res, err := app.RemoveLiveRoomAssistant(ctx, miniProgram.RemoveLiveRoomAssistantParams{
		RoomID:   roomID,
		Username: username,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.RemoveLiveRoomAssistantError(res)
	}

	return nil
}

// UpdateLiveRoomComment 开启或关闭评论。
//
// banComment=1 关闭，0 开启；已经在直播中改也生效。
func UpdateLiveRoomComment(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	ban bool,
) error {
	banFlag := 0
	if ban {
		banFlag = 1
	}

	res, err := app.UpdateLiveRoomComment(ctx, miniProgram.UpdateLiveRoomCommentParams{
		RoomID:     roomID,
		BanComment: banFlag,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.UpdateLiveRoomCommentError(res)
	}

	return nil
}

// UpdateLiveRoomFeedPublic 开启或关闭官方收录。
//
// 开启后直播间会出现在购物直播频道，能带来分享之外的观众。
func UpdateLiveRoomFeedPublic(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	public bool,
) error {
	publicFlag := 0
	if public {
		publicFlag = 1
	}

	res, err := app.UpdateLiveRoomFeedPublic(ctx, miniProgram.UpdateLiveRoomFeedPublicParams{
		RoomID:        roomID,
		IsFeedsPublic: publicFlag,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.UpdateLiveRoomFeedPublicError(res)
	}

	return nil
}

// UpdateLiveRoomKF 开启或关闭客服入口。
//
// 开启前要在小程序管理后台配置好客服，否则微信返回 300038。
func UpdateLiveRoomKF(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	closeKF bool,
) error {
	closeFlag := 0
	if closeKF {
		closeFlag = 1
	}

	res, err := app.UpdateLiveRoomKF(ctx, miniProgram.UpdateLiveRoomKFParams{
		RoomID:  roomID,
		CloseKf: closeFlag,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.UpdateLiveRoomKFError(res)
	}

	return nil
}

// UpdateLiveRoomReplay 开启或关闭回放。
func UpdateLiveRoomReplay(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	closeReplay bool,
) error {
	closeFlag := 0
	if closeReplay {
		closeFlag = 1
	}

	res, err := app.UpdateLiveRoomReplay(ctx, miniProgram.UpdateLiveRoomReplayParams{
		RoomID:      roomID,
		CloseReplay: closeFlag,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.UpdateLiveRoomReplayError(res)
	}

	return nil
}

// DownloadLiveGoodsVideo 获取商品讲解视频地址。
//
// 视频是推送那一刻开始录的，刚推送完会返回 300045（未生成），要过一会儿再取。
func DownloadLiveGoodsVideo(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	goodsID int64,
) error {
	res, err := app.DownloadLiveGoodsVideo(ctx, miniProgram.DownloadLiveGoodsVideoParams{
		RoomID:  roomID,
		GoodsID: goodsID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.DownloadLiveGoodsVideoError(res)
	}

	log.Printf("DownloadLiveGoodsVideo: %s", res.URL)

	return nil
}

// SetLiveGoodsKey 设置直播挂件的商品 key。
//
// key 是账号级的，设置一次对所有房间生效，直播挂件用它去服务端拉商品。
func SetLiveGoodsKey(
	ctx context.Context,
	app *miniProgram.Application,
	keys []string,
) error {
	res, err := app.SetLiveGoodsKey(ctx, miniProgram.SetLiveGoodsKeyParams{GoodsKey: keys})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SetLiveGoodsKeyError(res)
	}

	return nil
}

// GetLiveGoodsKey 读取直播挂件的商品 key。
//
// errcode 为 0 但列表为空，表示还没设置过，不是错误。
func GetLiveGoodsKey(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetLiveGoodsKey(ctx)
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetLiveGoodsKeyError(res)
	}

	log.Printf("GetLiveGoodsKey: %v", res.VendorGoodsKey)

	return nil
}
