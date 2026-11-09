package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getDailyRetainPath reads day-by-day retention of one day.
const getDailyRetainPath = "/datacube/getweanalysisappiddailyretaininfo"

// GetDailyRetainResponse is the reply of the daily retention endpoint.
type GetDailyRetainResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// DatacubeRetention is the single period the call asked about.
	DatacubeRetention
}

// OK reports whether the call returned a period.
func (r *GetDailyRetainResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetDailyRetainError turns the errcode from the reply into an error.
func GetDailyRetainError(r *GetDailyRetainResponse) error {
	return datacubeError(r.ErrCode)
}

// GetDailyRetain reads how many users came back on the days after one day.
//
// The curve is long: key 0-7, 14 and 30, so one day of data tells you the
// retention a month later. begin_date and end_date must be the same day.
func (a *Application) GetDailyRetain(
	ctx context.Context,
	date DatacubeDateRange,
) (*GetDailyRetainResponse, error) {
	if err := CheckDatacubeDayRange(date); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getDailyRetainPath, nil, date.body())
	if err != nil {
		return nil, err
	}

	var result GetDailyRetainResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
