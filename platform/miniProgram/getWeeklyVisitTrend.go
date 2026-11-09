package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getWeeklyVisitTrendPath reads visit statistics of one natural week.
const getWeeklyVisitTrendPath = "/datacube/getweanalysisappidweeklyvisittrend"

// GetWeeklyVisitTrendResponse is the reply of the weekly trend endpoint.
type GetWeeklyVisitTrendResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// List is the statistics of the requested week.
	List []DatacubeVisitTrend `json:"list"`
}

// OK reports whether the call succeeded.
//
// An empty List is a normal answer, not a failure.
func (r *GetWeeklyVisitTrendResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetWeeklyVisitTrendError turns the errcode from the reply into an error.
func GetWeeklyVisitTrendError(r *GetWeeklyVisitTrendResponse) error {
	return datacubeError(r.ErrCode)
}

// GetWeeklyVisitTrend reads the visit statistics of one natural week.
//
// begin_date must be a Monday and end_date the Sunday of the same week; the
// figures are summed over the week, with visit_uv de-duplicated across it.
func (a *Application) GetWeeklyVisitTrend(
	ctx context.Context,
	date DatacubeDateRange,
) (*GetWeeklyVisitTrendResponse, error) {
	if err := CheckDatacubeWeekRange(date); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getWeeklyVisitTrendPath, nil, date.body())
	if err != nil {
		return nil, err
	}

	var result GetWeeklyVisitTrendResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
