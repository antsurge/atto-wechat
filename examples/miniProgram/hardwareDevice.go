package main

import (
	"context"
	"log"

	"github.com/antsurge/atto-wechat/platform/miniProgram"
)

// HardwareDevice 演示硬件设备接口：设备票据、订阅消息、设备组与 license 激活。
//
// 设备在所有接口里都由 (model_id, sn) 标识。sn 只接受数字、字母、下划线和连字符，
// 且不超过 128 字节，本地就会校验，不必等微信回 9800001/9800002。
func HardwareDevice(ctx context.Context, app *miniProgram.Application) {
	device := miniProgram.HardwareDevice{ModelID: "MODEL_ID1", SN: "SN1"}

	// 1. 设备票据：5 分钟有效，绑定时现取，不要缓存。
	ticket, err := app.GetSnTicket(ctx, miniProgram.GetSnTicketParams{
		ModelID: device.ModelID,
		SN:      device.SN,
	})
	if err != nil {
		log.Printf("获取设备票据失败: %v", err)

		return
	}

	if !ticket.OK() {
		log.Printf("没有拿到 sn_ticket: %s", miniProgram.GetSnTicketError(ticket))

		return
	}

	log.Printf("sn_ticket: %s", ticket.SnTicket)

	// 2. 推订阅消息：data 是对象，不是 JSON 字符串。
	msg, err := app.SendHardwareDeviceMessage(ctx, miniProgram.SendHardwareDeviceMessageParams{
		TemplateID:   "TEMPLATE_ID",
		ModelID:      device.ModelID,
		SN:           device.SN,
		ToOpenIDList: []string{"OPENID1"},
		Data: map[string]miniProgram.HardwareDeviceMsgValue{
			"character_string1": {Value: "SN1"},
			"thing2":            {Value: "开门成功"},
		},
		Page: "pages/index/index",
	})
	if err != nil {
		log.Printf("推送设备消息失败: %v", err)

		return
	}

	// 43101 是用户自己取消了订阅，重试没有意义。
	if !msg.OK() {
		log.Printf("推送被拒: %s", miniProgram.SendHardwareDeviceMessageError(msg))
	}

	// 3. 设备组：创建时定名，之后改不了。
	group, err := app.CreateIotGroupId(ctx, miniProgram.CreateIotGroupIdParams{
		ModelID:   device.ModelID,
		GroupName: "一号门禁组",
	})
	if err != nil || !group.OK() {
		log.Printf("创建设备组失败: %v", err)

		return
	}

	added, err := app.AddIotGroupDevice(ctx, miniProgram.AddIotGroupDeviceParams{
		GroupID:    group.GroupID,
		DeviceList: []miniProgram.HardwareDevice{device},
	})
	if err != nil {
		log.Printf("添加设备失败: %v", err)

		return
	}

	// 顶层 errcode 为 0 不代表每个设备都成功，要按设备看。
	for _, failed := range added.FailedDevices() {
		log.Printf("设备 %s 添加失败: %s",
			failed.SN, miniProgram.HardwareDeviceErrMsg(failed.ErrCode))
	}

	info, err := app.GetIotGroupInfo(ctx, group.GroupID)
	if err != nil || !info.OK() {
		log.Printf("查询设备组失败: %v", err)

		return
	}

	log.Printf("设备组 %s（%s）共 %d 台", info.GroupName, info.ModelType, len(info.DeviceList))

	// 4. license：先看余额，再激活。
	pkgs, err := app.GetLicensePkgList(ctx, miniProgram.GetLicensePkgListParams{
		PkgType: miniProgram.LicensePkgTypeA,
	})
	if err != nil || !pkgs.OK() {
		log.Printf("查询资源包失败: %v", err)

		return
	}

	if pkgs.TotalRemaining() == 0 {
		log.Print("A 类资源包已无余额，无法激活")

		return
	}

	// active_number 是幂等键：重试要用同一个号，新激活要换号。
	activated, err := app.ActiveLicenseDevice(ctx, miniProgram.ActiveLicenseDeviceParams{
		PkgType: miniProgram.LicensePkgTypeA,
		DeviceList: []miniProgram.LicenseActivation{
			{ModelID: device.ModelID, SN: device.SN, ActiveNumber: 1},
		},
	})
	if err != nil {
		log.Printf("激活 license 失败: %v", err)

		return
	}

	for _, failed := range activated.FailedDevices() {
		log.Printf("设备 %s 激活失败: %s",
			failed.SN, miniProgram.HardwareDeviceErrMsg(failed.ErrCode))
	}

	licenses, err := app.GetLicenseDeviceInfo(ctx, []miniProgram.HardwareDevice{device})
	if err != nil || !licenses.OK() {
		log.Printf("查询激活详情失败: %v", err)

		return
	}

	for _, license := range licenses.DeviceList {
		if !license.Active() {
			log.Printf("设备 %s 尚未激活", license.SN)

			continue
		}

		log.Printf("设备 %s 有效期至 %d", license.SN, license.ExpireTime)
	}
}
