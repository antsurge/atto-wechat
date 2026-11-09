package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// AddLiveRole 给账号授予直播成员角色。
//
// 角色是账号级的：主播能在所有房间开播，管理员管房间，运营管评论和商品。
// 主播未实名时会返回 codeurl，让主播打开它完成认证，而不是直接失败。
func AddLiveRole(
	ctx context.Context,
	app *miniProgram.Application,
	username string,
	role int,
) error {
	res, err := app.AddLiveRole(ctx, miniProgram.AddLiveRoleParams{
		Username: username,
		Role:     role,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.AddLiveRoleError(res)
	}

	if res.CodeURL != "" {
		log.Printf("AddLiveRole: 主播需先实名认证 %s", res.CodeURL)
	}

	return nil
}

// DeleteLiveRole 移除账号的直播成员角色。
//
// 主播名下还有未开播的直播间时，移除主播角色会失败（400005），
// 需要先删掉或移交这些房间。
func DeleteLiveRole(
	ctx context.Context,
	app *miniProgram.Application,
	username string,
	role int,
) error {
	res, err := app.DeleteLiveRole(ctx, miniProgram.DeleteLiveRoleParams{
		Username: username,
		Role:     role,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.DeleteLiveRoleError(res)
	}

	return nil
}

// GetLiveRoleList 分页查询直播成员。
//
// keyword 按微信号或昵称模糊匹配，传空表示不筛选。
func GetLiveRoleList(ctx context.Context, app *miniProgram.Application) error {
	res, err := app.GetLiveRoleList(ctx, miniProgram.GetLiveRoleListParams{
		Offset:  0,
		Limit:   20,
		Keyword: "",
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.GetLiveRoleListError(res)
	}

	for _, member := range res.List {
		log.Printf("GetLiveRoleList: %s %s role=%d", member.Username, member.Nickname, member.Role)
	}

	return nil
}

// GetLiveFollowers 分页拉取长期订阅用户。
//
// 按游标翻页：把上一次返回的 page_break 回传，直到它为空；
// 拿到的 openid 就是 PushLiveMessage 的提醒对象。
func GetLiveFollowers(ctx context.Context, app *miniProgram.Application) error {
	pageBreak := ""

	for {
		res, err := app.GetLiveFollowers(ctx, miniProgram.GetLiveFollowersParams{
			Limit:     100,
			PageBreak: pageBreak,
		})
		if err != nil {
			return err
		}
		if !res.OK() {
			return miniProgram.GetLiveFollowersError(res)
		}

		for _, follower := range res.Followers {
			log.Printf("GetLiveFollowers: %s", follower.OpenID)
		}

		if res.PageBreak == "" {
			return nil
		}
		pageBreak = res.PageBreak
	}
}

// PushLiveMessage 给长期订阅用户推送开播提醒。
//
// 只有直播中的房间能推（9410006），且每天只能推一个房间（9410005）。
func PushLiveMessage(
	ctx context.Context,
	app *miniProgram.Application,
	roomID int64,
	openIDs []string,
) error {
	res, err := app.PushLiveMessage(ctx, miniProgram.PushLiveMessageParams{
		RoomID:      roomID,
		UserOpenIDs: openIDs,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.PushLiveMessageError(res)
	}

	log.Printf("PushLiveMessage: messageId=%s", res.MessageID)

	return nil
}
