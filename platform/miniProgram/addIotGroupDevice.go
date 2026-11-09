package miniProgram

import (
	"context"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// addIotGroupDevicePath adds devices to a device group.
const addIotGroupDevicePath = "/wxa/business/group/adddevice"

// AddIotGroupDeviceParams is the batch of devices to add.
type AddIotGroupDeviceParams struct {
	// GroupID is the group to add to.
	GroupID string
	// DeviceList is the devices to add.
	DeviceList []HardwareDevice

	// ForceAdd takes devices away from their current group and puts them in this
	// one.
	//
	// A device belongs to at most one group: without ForceAdd, adding a device
	// that already sits in another group fails with 9800018. The flag is not a
	// retry switch, it silently moves somebody else's device.
	ForceAdd bool
}

// CheckAddIotGroupDeviceParams validates params before anything is sent.
func CheckAddIotGroupDeviceParams(params AddIotGroupDeviceParams) error {
	if params.GroupID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: AddIotGroupDevice group_id is required",
		)
	}

	return CheckHardwareDeviceList("AddIotGroupDevice", params.DeviceList)
}

// body renders params as the JSON body the endpoint expects.
func (p AddIotGroupDeviceParams) body() *addIotGroupDeviceBody {
	devices := make([]HardwareDevice, 0, len(p.DeviceList))
	devices = append(devices, p.DeviceList...)

	return &addIotGroupDeviceBody{
		GroupID:    p.GroupID,
		DeviceList: devices,
		ForceAdd:   p.ForceAdd,
	}
}

// addIotGroupDeviceBody is the JSON body of an add request.
type addIotGroupDeviceBody struct {
	GroupID    string           `json:"group_id"`
	DeviceList []HardwareDevice `json:"device_list"`
	ForceAdd   bool             `json:"force_add,omitempty"`
}

// AddIotGroupDeviceResponse is the reply of the add endpoint.
type AddIotGroupDeviceResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// DeviceList is the per-device outcome.
	DeviceList []HardwareDeviceResult `json:"device_list"`
}

// OK reports whether the call itself succeeded.
//
// It says nothing about the devices: errcode 0 with every device failed is a
// normal reply. Use AllSucceeded for the batch result.
func (r *AddIotGroupDeviceResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// AllSucceeded reports whether the call succeeded and every device was added.
func (r *AddIotGroupDeviceResponse) AllSucceeded() bool {
	return r.OK() && len(hardwareDeviceFailures(r.DeviceList)) == 0
}

// FailedDevices returns the devices WeChat refused, e.g. because they already
// belong to another group.
func (r *AddIotGroupDeviceResponse) FailedDevices() []HardwareDeviceResult {
	if r == nil {
		return nil
	}

	return hardwareDeviceFailures(r.DeviceList)
}

// AddIotGroupDeviceError turns the errcode from the reply into an error.
func AddIotGroupDeviceError(r *AddIotGroupDeviceResponse) error {
	return hardwareDeviceError(r.ErrCode)
}

// AddIotGroupDevice adds devices to a group.
//
// The reply is per device: one device already belonging to another group fails
// with its own errcode while the rest are added.
func (a *Application) AddIotGroupDevice(
	ctx context.Context,
	params AddIotGroupDeviceParams,
) (*AddIotGroupDeviceResponse, error) {
	if err := CheckAddIotGroupDeviceParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, addIotGroupDevicePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result AddIotGroupDeviceResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
