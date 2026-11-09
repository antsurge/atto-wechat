package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getWeeklyRetainPath reads week-by-week retention of one natural week.
const getWeeklyRetainPath = "/datacube/getweanalysisappidweeklyretaininfo"

// GetWeeklyRetainResponse is the reply of the weekly retention endpoint.
type GetWeeklyRetainResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// DatacubeRetention is the single period the call asked about.
	DatacubeRetention
}

// OK reports whether the call returned a period.
func (r *GetWeeklyRetainResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetWeeklyRetainError turns the errcode from the reply into an error.
func GetWeeklyRetainError(r *GetWeeklyRetainResponse) error {
	return datacubeError(r.ErrCode)
}

// GetWeeklyRetain reads how many users came back in the weeks after one natural
// week.
//
// The curve is key 0-4: the week itself plus four weeks later. begin_date must be
// a Monday and end_date the Sunday of the same week.
func (a *Application) GetWeeklyRetain(
	ctx context.Context,
	date DatacubeDateRange,
) (*GetWeeklyRetainResponse, error) {
	if err := CheckDatacubeWeekRange(date); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getWeeklyRetainPath, nil, date.body())
	if err != nil {
		return nil, err
	}

	var result GetWeeklyRetainResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
