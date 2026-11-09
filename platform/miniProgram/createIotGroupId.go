package miniProgram

import (
	"context"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// createIotGroupIdPath creates a device group.
const createIotGroupIdPath = "/wxa/business/group/createid"

// IotGroupNameMaxLength is the documented limit on a group name: 32 bytes.
const IotGroupNameMaxLength = 32

// CreateIotGroupIdParams describes the group to create.
type CreateIotGroupIdParams struct {
	// ModelID is the model the group holds devices of.
	ModelID string
	// GroupName is the group name, 1-32 bytes.
	//
	// It cannot be changed afterwards: the only way to rename a group is to create
	// another one.
	GroupName string
}

// CheckCreateIotGroupIdParams validates params before anything is sent.
func CheckCreateIotGroupIdParams(params CreateIotGroupIdParams) error {
	if params.ModelID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: CreateIotGroupId model_id is required",
		)
	}

	// Bytes, not runes: 11 Chinese characters already exceed the 32-byte limit,
	// and the error WeChat returns for it (9800012) does not say by how much.
	if len(params.GroupName) == 0 || len(params.GroupName) > IotGroupNameMaxLength {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: CreateIotGroupId group_name must be 1-32 bytes",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p CreateIotGroupIdParams) body() *createIotGroupIdBody {
	return &createIotGroupIdBody{ModelID: p.ModelID, GroupName: p.GroupName}
}

// createIotGroupIdBody is the JSON body of a group creation.
type createIotGroupIdBody struct {
	ModelID   string `json:"model_id"`
	GroupName string `json:"group_name"`
}

// CreateIotGroupIdResponse is the reply of the group creation endpoint.
type CreateIotGroupIdResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// GroupID identifies the new group.
	GroupID string `json:"group_id"`
}

// OK reports whether a group was created.
func (r *CreateIotGroupIdResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.GroupID != ""
}

// CreateIotGroupIdError turns the errcode from the reply into an error.
func CreateIotGroupIdError(r *CreateIotGroupIdResponse) error {
	return hardwareDeviceError(r.ErrCode)
}

// CreateIotGroupId creates a device group and returns its id.
//
// The id is the only handle on the group: it is not derived from the name, so
// losing the reply means losing track of the group.
func (a *Application) CreateIotGroupId(
	ctx context.Context,
	params CreateIotGroupIdParams,
) (*CreateIotGroupIdResponse, error) {
	if err := CheckCreateIotGroupIdParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, createIotGroupIdPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result CreateIotGroupIdResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
