package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getUserPortraitPath reads the user portrait of a recent 1/7/30 days.
const getUserPortraitPath = "/datacube/getweanalysisappiduserportrait"

// DatacubePortraitItem is one attribute value of a portrait.
type DatacubePortraitItem struct {
	// ID is the attribute value id.
	//
	// It is absent from the devices list, which is keyed by model name only, so 0
	// means "not provided" rather than a real id.
	ID int `json:"id,omitempty"`
	// Name is the attribute value, e.g. "广东省", "男", "17岁以下".
	Name string `json:"name"`
	// Value is the number of users with this attribute value.
	Value int `json:"value"`
}

// DatacubeUserPortrait is a portrait of a group of users.
type DatacubeUserPortrait struct {
	Province  []DatacubePortraitItem `json:"province"`
	City      []DatacubePortraitItem `json:"city"`
	Genders   []DatacubePortraitItem `json:"genders"`
	Platforms []DatacubePortraitItem `json:"platforms"`
	Devices   []DatacubePortraitItem `json:"devices"`
	Ages      []DatacubePortraitItem `json:"ages"`
}

// GetUserPortraitResponse is the reply of the portrait endpoint.
type GetUserPortraitResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// RefDate is the range the portrait was computed over.
	//
	// A single day comes back as "20170611", a longer range as
	// "20170611-20170617".
	RefDate string `json:"ref_date"`
	// VisitUVNew is the portrait of first-time users.
	VisitUVNew *DatacubeUserPortrait `json:"visit_uv_new"`
	// VisitUV is the portrait of active users.
	VisitUV *DatacubeUserPortrait `json:"visit_uv"`
}

// OK reports whether the call returned a portrait.
func (r *GetUserPortraitResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetUserPortraitError turns the errcode from the reply into an error.
func GetUserPortraitError(r *GetUserPortraitResponse) error {
	return datacubeError(r.ErrCode)
}

// GetUserPortrait reads the portrait -- province, city, gender, platform, device,
// age -- of new and of active users.
//
// The range is restricted: begin_date and end_date have to be 1, 7 or 30 days
// apart, so the data is always "the last N days up to end_date". A range of any
// other length is rejected before it is sent.
func (a *Application) GetUserPortrait(
	ctx context.Context,
	date DatacubeDateRange,
) (*GetUserPortraitResponse, error) {
	if err := CheckDatacubePortraitRange(date); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getUserPortraitPath, nil, date.body())
	if err != nil {
		return nil, err
	}

	var result GetUserPortraitResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
