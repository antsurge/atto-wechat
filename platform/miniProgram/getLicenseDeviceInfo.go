package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getLicenseDeviceInfoPath reads the licence state of devices.
const getLicenseDeviceInfoPath = "/wxa/business/license/getdeviceinfo"

// GetLicenseDeviceInfoResponse is the reply of the device licence endpoint.
type GetLicenseDeviceInfoResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// DeviceList is the licence state of each requested device.
	DeviceList []HardwareDeviceLicense `json:"device_list"`
}

// OK reports whether the call succeeded.
//
// A device without a licence still comes back in the list, with no expire_time, so
// an empty list means an empty request rather than unlicensed devices.
func (r *GetLicenseDeviceInfoResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetLicenseDeviceInfoError turns the errcode from the reply into an error.
func GetLicenseDeviceInfoError(r *GetLicenseDeviceInfoResponse) error {
	return hardwareDeviceError(r.ErrCode)
}

// GetLicenseDeviceInfo reads when the licences of the given devices expire.
//
// It is the read side of ActiveLicenseDevice: use it to find out which devices are
// licensed before deciding what to activate.
func (a *Application) GetLicenseDeviceInfo(
	ctx context.Context,
	devices []HardwareDevice,
) (*GetLicenseDeviceInfoResponse, error) {
	if err := CheckHardwareDeviceList("GetLicenseDeviceInfo", devices); err != nil {
		return nil, err
	}

	body := &struct {
		DeviceList []HardwareDevice `json:"device_list"`
	}{DeviceList: devices}

	resp, err := a.api.PostJSON(ctx, getLicenseDeviceInfoPath, nil, body)
	if err != nil {
		return nil, err
	}

	var result GetLicenseDeviceInfoResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
