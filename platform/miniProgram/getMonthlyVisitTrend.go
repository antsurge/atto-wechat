package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getMonthlyVisitTrendPath reads visit statistics of one natural month.
const getMonthlyVisitTrendPath = "/datacube/getweanalysisappidmonthlyvisittrend"

// GetMonthlyVisitTrendResponse is the reply of the monthly trend endpoint.
type GetMonthlyVisitTrendResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// List is the statistics of the requested month, with RefDate in yyyymm.
	List []DatacubeVisitTrend `json:"list"`
}

// OK reports whether the call succeeded.
//
// An empty List is a normal answer, not a failure.
func (r *GetMonthlyVisitTrendResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetMonthlyVisitTrendError turns the errcode from the reply into an error.
func GetMonthlyVisitTrendError(r *GetMonthlyVisitTrendResponse) error {
	return datacubeError(r.ErrCode)
}

// GetMonthlyVisitTrend reads the visit statistics of one natural month.
//
// begin_date must be the first day of a month and end_date its last day.
func (a *Application) GetMonthlyVisitTrend(
	ctx context.Context,
	date DatacubeDateRange,
) (*GetMonthlyVisitTrendResponse, error) {
	if err := CheckDatacubeMonthRange(date); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getMonthlyVisitTrendPath, nil, date.body())
	if err != nil {
		return nil, err
	}

	var result GetMonthlyVisitTrendResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
