package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getMonthlyRetainPath reads month-by-month retention of one natural month.
const getMonthlyRetainPath = "/datacube/getweanalysisappidmonthlyretaininfo"

// GetMonthlyRetainResponse is the reply of the monthly retention endpoint.
type GetMonthlyRetainResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// DatacubeRetention is the single period the call asked about.
	DatacubeRetention
}

// OK reports whether the call returned a period.
func (r *GetMonthlyRetainResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetMonthlyRetainError turns the errcode from the reply into an error.
func GetMonthlyRetainError(r *GetMonthlyRetainResponse) error {
	return datacubeError(r.ErrCode)
}

// GetMonthlyRetain reads how many users came back in the month after one natural
// month.
//
// The curve only has two points: key 0 (the month itself) and key 1 (the month
// after). begin_date must be the first day of a month and end_date its last day.
func (a *Application) GetMonthlyRetain(
	ctx context.Context,
	date DatacubeDateRange,
) (*GetMonthlyRetainResponse, error) {
	if err := CheckDatacubeMonthRange(date); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getMonthlyRetainPath, nil, date.body())
	if err != nil {
		return nil, err
	}

	var result GetMonthlyRetainResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
