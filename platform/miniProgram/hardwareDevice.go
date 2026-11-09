package miniProgram

import (
	"errors"
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
)

// The hardware-device endpoints all identify a device by (model_id, sn), and most
// of them take or return a list of them. The identity, its validation and the
// errcode table live here so the nine endpoints agree on what a device is.

// HardwareDeviceErrDefaultMsg is returned for an errcode these endpoints do not
// document.
const HardwareDeviceErrDefaultMsg = "硬件设备接口返回了未记录的错误码"

// HardwareDeviceErrCodeMap holds the errcode descriptions the hardware-device
// endpoints document, keyed by the errcode as it appears in the reply.
var HardwareDeviceErrCodeMap = map[int]string{
	-1:      "系统繁忙，此时请开发者稍候再试",
	40037:   "不合法的 template_id",
	41030:   "page 路径不正确，需要保证在现网版本小程序中存在，与 app.json 保持一致",
	43002:   "方法调用错误，请用 POST 方法调用",
	43101:   "用户拒绝接受消息：之前订阅过，后来取消了订阅关系",
	44002:   "POST 的数据包为空",
	47001:   "解析 JSON 内容错误，参数需以 JSON 字符串格式写在 POST 请求的 body 中",
	47003:   "模板参数不准确，可能为空或者不满足规则，errmsg 会提示具体是哪个字段出错",
	9800001: "sn 长度不能超过 128 字节",
	9800002: "sn 包含非法字符",
	9800003: "model_id 检查不通过",
	9800012: "group_name 不合法，长度应为 1-32 字节",
	9800013: "group_id 不合法",
	9800014: "设备类型与设备组不符合",
	9800015: "该设备已存在于设备组",
	9800016: "设备组中设备数到达上限",
	9800017: "该设备不属于设备组",
	9800018: "该设备已经属于另一个设备组",
	9800020: "设备数超出限制",
	9800037: "激活码序号已使用",
	9800038: "设备有效期超出限制",
	9800039: "资源包余额不足",
	9800040: "资源包类型和设备类型不匹配",
	9800044: "已扩容至最大设备数量",
}

// HardwareDeviceErrMsg returns the description of an errcode, or the default when
// the errcode is undocumented.
func HardwareDeviceErrMsg(code int) string {
	if msg, ok := HardwareDeviceErrCodeMap[code]; ok {
		return msg
	}

	return HardwareDeviceErrDefaultMsg
}

// HardwareDeviceMaxSNLength is the documented limit on sn: 128 bytes.
const HardwareDeviceMaxSNLength = 128

// HardwareDevice is how every hardware endpoint identifies a device: the model id
// it was registered under, and its serial number.
type HardwareDevice struct {
	// ModelID is the device model id, obtained when registering the device.
	ModelID string `json:"model_id"`
	// SN is the serial number assigned by the vendor.
	SN string `json:"sn"`
}

// CheckHardwareDevice validates one device identity.
//
// WeChat rejects a bad sn with 9800001 or 9800002, but only after the request is
// on the wire and against a list it cannot point into; checking here names the
// exact device.
func CheckHardwareDevice(device HardwareDevice) error {
	if device.ModelID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: hardware device model_id is required",
		)
	}

	if device.SN == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: hardware device sn is required",
		)
	}

	// The limit is in bytes, not runes: an sn is ASCII by the rule below, but the
	// check has to happen before that rule so the message names the real problem.
	if len(device.SN) > HardwareDeviceMaxSNLength {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: hardware device sn must not exceed 128 bytes",
		)
	}

	// Only digits, letters, underscore and hyphen are accepted. Catching it here
	// matters for sn values that come from scanned labels or user input.
	for i := 0; i < len(device.SN); i++ {
		c := device.SN[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c == '_', c == '-':
		default:
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: hardware device sn may only contain digits, letters, '_' and '-', got "+strconv.Quote(device.SN),
			)
		}
	}

	return nil
}

// CheckHardwareDeviceList validates a device list, naming the offending index.
func CheckHardwareDeviceList(name string, devices []HardwareDevice) error {
	if len(devices) == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: "+name+" device_list is required",
		)
	}

	for i, device := range devices {
		if err := CheckHardwareDevice(device); err != nil {
			return cerrors.Wrap(
				cerrors.CodeInvalidArgument,
				"miniprogram: "+name+" device_list["+strconv.Itoa(i)+"] is invalid",
				err,
			)
		}
	}

	return nil
}

// HardwareDeviceResult is the per-device outcome of a batch call.
//
// The batch endpoints answer errcode 0 even when some devices failed: the result
// of each device is in its own errcode. Checking only the top-level errcode
// reports success for a request where nothing happened.
type HardwareDeviceResult struct {
	// ModelID is the device model id.
	ModelID string `json:"model_id"`
	// SN is the device serial number.
	SN string `json:"sn"`
	// ExpireTime is the expiry as Unix seconds; only the licence endpoints set
	// it.
	ExpireTime int64 `json:"expire_time,omitempty"`
	// ErrCode is this device's result, 0 on success.
	ErrCode int `json:"errcode"`
}

// OK reports whether this device was handled successfully.
func (r HardwareDeviceResult) OK() bool {
	return r.ErrCode == 0
}

// HardwareDeviceLicense is the licence state of one device.
type HardwareDeviceLicense struct {
	// ModelID is the device model id.
	ModelID string `json:"model_id"`
	// SN is the device serial number.
	SN string `json:"sn"`
	// ExpireTime is the expiry as Unix seconds.
	//
	// An unlicensed device comes back with no expiry, which reads as 0 -- use
	// Active to tell "expired on 1970" from "never activated".
	ExpireTime int64 `json:"expire_time"`
}

// Active reports whether the device has a licence expiry at all.
func (l HardwareDeviceLicense) Active() bool {
	return l.ExpireTime > 0
}

// hardwareDeviceFailures picks the devices that failed out of a batch result.
func hardwareDeviceFailures(devices []HardwareDeviceResult) []HardwareDeviceResult {
	var failed []HardwareDeviceResult

	for _, device := range devices {
		if !device.OK() {
			failed = append(failed, device)
		}
	}

	return failed
}

// hardwareDeviceError turns an errcode into an error for any hardware endpoint.
func hardwareDeviceError(code int) error {
	return errors.New(HardwareDeviceErrMsg(code))
}
