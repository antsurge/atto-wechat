package miniProgram

import (
	"context"
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// activeLicenseDevicePath activates licences for devices.
const activeLicenseDevicePath = "/wxa/business/license/activedevice"

// LicenseActivation is one device to activate.
type LicenseActivation struct {
	// ModelID is the device model id.
	ModelID string
	// SN is the device serial number.
	SN string
	// ActiveNumber is the activation sequence number: any uint32, never repeated
	// for the same device.
	//
	// It is an idempotency key, not an index: WeChat uses it to tell a retried
	// request from a new one, so a retry has to send the same number again and a
	// new activation a different one. Reusing it comes back as 9800037.
	ActiveNumber uint32
}

// ActiveLicenseDeviceParams is the batch to activate.
type ActiveLicenseDeviceParams struct {
	// PkgType is the device class to spend the licence on, one of the
	// LicensePkgType* values.
	PkgType int
	// DeviceList is the devices to activate.
	DeviceList []LicenseActivation
}

// CheckActiveLicenseDeviceParams validates params before anything is sent.
func CheckActiveLicenseDeviceParams(params ActiveLicenseDeviceParams) error {
	if params.PkgType < LicensePkgTypeTest || params.PkgType > LicensePkgTypeE {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: ActiveLicenseDevice pkg_type must be 0-5",
		)
	}

	if len(params.DeviceList) == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: ActiveLicenseDevice device_list is required",
		)
	}

	// Two devices with the same number in one request means the caller reused a
	// constant: WeChat then rejects the second with 9800037 and the reply has to
	// be read per device to find out which one it was.
	seen := make(map[uint32]struct{}, len(params.DeviceList))

	for i, device := range params.DeviceList {
		if err := CheckHardwareDevice(HardwareDevice{ModelID: device.ModelID, SN: device.SN}); err != nil {
			return cerrors.Wrap(
				cerrors.CodeInvalidArgument,
				"miniprogram: ActiveLicenseDevice device_list["+strconv.Itoa(i)+"] is invalid",
				err,
			)
		}

		if _, duplicated := seen[device.ActiveNumber]; duplicated {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: ActiveLicenseDevice active_number "+strconv.FormatUint(uint64(device.ActiveNumber), 10)+" is used twice",
			)
		}

		seen[device.ActiveNumber] = struct{}{}
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p ActiveLicenseDeviceParams) body() *activeLicenseDeviceBody {
	devices := make([]activeLicenseDevice, 0, len(p.DeviceList))
	for _, device := range p.DeviceList {
		devices = append(devices, activeLicenseDevice{
			ModelID:      device.ModelID,
			SN:           device.SN,
			ActiveNumber: device.ActiveNumber,
		})
	}

	return &activeLicenseDeviceBody{PkgType: p.PkgType, DeviceList: devices}
}

// activeLicenseDeviceBody is the JSON body of an activation.
type activeLicenseDeviceBody struct {
	DeviceList []activeLicenseDevice `json:"device_list"`
	PkgType    int                   `json:"pkg_type"`
}

// activeLicenseDevice is one device in the JSON body.
type activeLicenseDevice struct {
	ModelID      string `json:"model_id"`
	SN           string `json:"sn"`
	ActiveNumber uint32 `json:"active_number"`
}

// ActiveLicenseDeviceResponse is the reply of the activation endpoint.
type ActiveLicenseDeviceResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// DeviceList is the per-device outcome, with the licence expiry on success.
	DeviceList []HardwareDeviceResult `json:"device_list"`
}

// OK reports whether the call itself succeeded.
//
// It says nothing about the devices: errcode 0 with every device rejected is a
// normal reply. Use AllSucceeded for the batch result.
func (r *ActiveLicenseDeviceResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// AllSucceeded reports whether the call succeeded and every device was activated.
func (r *ActiveLicenseDeviceResponse) AllSucceeded() bool {
	return r.OK() && len(hardwareDeviceFailures(r.DeviceList)) == 0
}

// FailedDevices returns the devices WeChat refused, e.g. for an exhausted package
// (9800039) or a mismatched device class (9800040).
func (r *ActiveLicenseDeviceResponse) FailedDevices() []HardwareDeviceResult {
	if r == nil {
		return nil
	}

	return hardwareDeviceFailures(r.DeviceList)
}

// ActiveLicenseDeviceError turns the errcode from the reply into an error.
func ActiveLicenseDeviceError(r *ActiveLicenseDeviceResponse) error {
	return hardwareDeviceError(r.ErrCode)
}

// ActiveLicenseDevice activates licences for a batch of devices.
//
// The batch is not atomic: some devices get activated and others fail. A device
// that was already activated under the same active_number is the case to retry;
// one that failed for a spent package is not.
func (a *Application) ActiveLicenseDevice(
	ctx context.Context,
	params ActiveLicenseDeviceParams,
) (*ActiveLicenseDeviceResponse, error) {
	if err := CheckActiveLicenseDeviceParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, activeLicenseDevicePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result ActiveLicenseDeviceResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
