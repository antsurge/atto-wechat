package miniProgram

import (
	"context"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getIotGroupInfoPath reads one device group.
const getIotGroupInfoPath = "/wxa/business/group/getinfo"

// GetIotGroupInfoResponse is the reply of the group info endpoint.
type GetIotGroupInfoResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// GroupName is the name the group was created with.
	GroupName string `json:"group_name"`
	// ModelID is the model of the group.
	ModelID string `json:"model_id"`
	// ModelType is the model's device type, e.g. "门禁机".
	ModelType string `json:"model_type"`
	// DeviceList is every device in the group.
	DeviceList []HardwareDevice `json:"device_list"`
}

// OK reports whether the group was read.
func (r *GetIotGroupInfoResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetIotGroupInfoError turns the errcode from the reply into an error.
func GetIotGroupInfoError(r *GetIotGroupInfoResponse) error {
	return hardwareDeviceError(r.ErrCode)
}

// GetIotGroupInfo reads the name, model and members of one device group.
//
// All members come back in one reply; the endpoint has no paging, so a group at
// its device limit returns a correspondingly large payload.
func (a *Application) GetIotGroupInfo(
	ctx context.Context,
	groupID string,
) (*GetIotGroupInfoResponse, error) {
	if groupID == "" {
		return nil, cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetIotGroupInfo group_id is required",
		)
	}

	body := &struct {
		GroupID string `json:"group_id"`
	}{GroupID: groupID}

	resp, err := a.api.PostJSON(ctx, getIotGroupInfoPath, nil, body)
	if err != nil {
		return nil, err
	}

	var result GetIotGroupInfoResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
