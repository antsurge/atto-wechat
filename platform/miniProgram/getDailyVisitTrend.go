package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getDailyVisitTrendPath reads visit statistics of one day.
const getDailyVisitTrendPath = "/datacube/getweanalysisappiddailyvisittrend"

// GetDailyVisitTrendResponse is the reply of the daily trend endpoint.
type GetDailyVisitTrendResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// List is the statistics of the requested day.
	List []DatacubeVisitTrend `json:"list"`
}

// OK reports whether the call succeeded.
//
// An empty List is a normal answer: a day that has not been aggregated yet, or a
// day without visits, both come back as no data rather than an error.
func (r *GetDailyVisitTrendResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetDailyVisitTrendError turns the errcode from the reply into an error.
func GetDailyVisitTrendError(r *GetDailyVisitTrendResponse) error {
	return datacubeError(r.ErrCode)
}

// GetDailyVisitTrend reads the visit statistics of one day.
//
// begin_date and end_date must be the same day, and that day must be in the past:
// today's data is not aggregated yet.
func (a *Application) GetDailyVisitTrend(
	ctx context.Context,
	date DatacubeDateRange,
) (*GetDailyVisitTrendResponse, error) {
	if err := CheckDatacubeDayRange(date); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getDailyVisitTrendPath, nil, date.body())
	if err != nil {
		return nil, err
	}

	var result GetDailyVisitTrendResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
