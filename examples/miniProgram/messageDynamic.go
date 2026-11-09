package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// CreateActivityID 创建动态消息的 activity_id。
//
// 动态消息是一张可以持续更新的卡片：分享到群聊后，成员数、房间上限、开始状态
// 都会随卡片重绘。activity_id 有效期 24 小时，必须在分享前创建。
//
// 私密消息需要把 openID 或 unionID 传进来，声明唯一的分享者；普通动态消息两者
// 都留空即可。
func CreateActivityID(
	ctx context.Context,
	app *miniProgram.Application,
	openID string,
	unionID string,
) error {
	res, err := app.CreateActivityID(ctx, miniProgram.CreateActivityIDParams{
		OpenID:  openID,
		UnionID: unionID,
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.CreateActivityIDError(res)
	}

	log.Printf("CreateActivityID: %s, expires at %d", res.ActivityID, res.ExpirationTime)

	return nil
}

// SetUpdatableMsg 修改已分享的动态消息。
//
// 卡片只有两套模板，target_state 决定用哪一套：
//   - 未开始（0）读 member_count、room_limit，渲染“成员正在加入，当前 x/y 人”；
//   - 已开始（1）读 path、version_type，渲染“已开始”，点击进入指定页面。
//
// 状态只能向前走，且 activity_id 过期后微信返回 47504，需要重新创建并分享。
func SetUpdatableMsg(
	ctx context.Context,
	app *miniProgram.Application,
	activityID string,
	memberCount string,
	roomLimit string,
) error {
	res, err := app.SetUpdatableMsg(ctx, miniProgram.SetUpdatableMsgParams{
		ActivityID:  activityID,
		TargetState: miniProgram.UpdatableMsgStateNotStarted,
		ParameterList: []miniProgram.UpdatableMsgParameter{
			{Name: miniProgram.UpdatableMsgParamMemberCount, Value: memberCount},
			{Name: miniProgram.UpdatableMsgParamRoomLimit, Value: roomLimit},
		},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SetUpdatableMsgError(res)
	}

	log.Println("SetUpdatableMsg: success")

	return nil
}

// SetChatToolMsg 修改聊天工具分享的动态卡片。
//
// 与普通动态消息不同，卡片由聊天工具模板生成，所以每次更新都要带 template_id。
// participator_info_list 决定卡片上展示谁参与了，target_state 为 3 时卡片进入
// 已结束状态。
func SetChatToolMsg(
	ctx context.Context,
	app *miniProgram.Application,
	activityID string,
	templateID string,
	groupOpenID string,
) error {
	res, err := app.SetChatToolMsg(ctx, miniProgram.SetChatToolMsgParams{
		ActivityID:  activityID,
		TargetState: miniProgram.ChatToolMsgStateRunning,
		TemplateID:  templateID,
		VersionType: miniProgram.ChatToolMsgVersionRelease,
		ParticipatorList: []miniProgram.ChatToolMsgParticipator{
			{
				GroupOpenID: groupOpenID,
				State:       miniProgram.ChatToolMsgParticipantJoined,
			},
		},
	})
	if err != nil {
		return err
	}
	if !res.OK() {
		return miniProgram.SetChatToolMsgError(res)
	}

	log.Println("SetChatToolMsg: success")

	return nil
}
