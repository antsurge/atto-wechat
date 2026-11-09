package miniProgram

import (
	"context"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// removeIotGroupDevicePath removes devices from a device group.
const removeIotGroupDevicePath = "/wxa/business/group/removedevice"

// RemoveIotGroupDeviceParams is the batch of devices to remove.
type RemoveIotGroupDeviceParams struct {
	// GroupID is the group to remove from.
	GroupID string
	// DeviceList is the devices to remove.
	DeviceList []HardwareDevice
}

// CheckRemoveIotGroupDeviceParams validates params before anything is sent.
func CheckRemoveIotGroupDeviceParams(params RemoveIotGroupDeviceParams) error {
	if params.GroupID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: RemoveIotGroupDevice group_id is required",
		)
	}

	return CheckHardwareDeviceList("RemoveIotGroupDevice", params.DeviceList)
}

// body renders params as the JSON body the endpoint expects.
func (p RemoveIotGroupDeviceParams) body() *removeIotGroupDeviceBody {
	devices := make([]HardwareDevice, 0, len(p.DeviceList))
	devices = append(devices, p.DeviceList...)

	return &removeIotGroupDeviceBody{GroupID: p.GroupID, DeviceList: devices}
}

// removeIotGroupDeviceBody is the JSON body of a removal.
type removeIotGroupDeviceBody struct {
	GroupID    string           `json:"group_id"`
	DeviceList []HardwareDevice `json:"device_list"`
}

// RemoveIotGroupDeviceResponse is the reply of the remove endpoint.
type RemoveIotGroupDeviceResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// DeviceList is the per-device outcome.
	DeviceList []HardwareDeviceResult `json:"device_list"`
}

// OK reports whether the call itself succeeded.
//
// It says nothing about the devices: errcode 0 with every device failed is a
// normal reply. Use AllSucceeded for the batch result.
func (r *RemoveIotGroupDeviceResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// AllSucceeded reports whether the call succeeded and every device was removed.
func (r *RemoveIotGroupDeviceResponse) AllSucceeded() bool {
	return r.OK() && len(hardwareDeviceFailures(r.DeviceList)) == 0
}

// FailedDevices returns the devices WeChat refused, e.g. because they were not in
// the group to begin with (9800017).
func (r *RemoveIotGroupDeviceResponse) FailedDevices() []HardwareDeviceResult {
	if r == nil {
		return nil
	}

	return hardwareDeviceFailures(r.DeviceList)
}

// RemoveIotGroupDeviceError turns the errcode from the reply into an error.
func RemoveIotGroupDeviceError(r *RemoveIotGroupDeviceResponse) error {
	return hardwareDeviceError(r.ErrCode)
}

// RemoveIotGroupDevice removes devices from a group.
//
// The reply is per device, like the add endpoint. Removing a device that is not in
// the group fails for that device alone; that is the case worth tolerating when
// reconciling, since the wanted state already holds.
func (a *Application) RemoveIotGroupDevice(
	ctx context.Context,
	params RemoveIotGroupDeviceParams,
) (*RemoveIotGroupDeviceResponse, error) {
	if err := CheckRemoveIotGroupDeviceParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, removeIotGroupDevicePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result RemoveIotGroupDeviceResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
