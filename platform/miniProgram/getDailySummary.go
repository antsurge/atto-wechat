package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getDailySummaryPath reads the overview figures of one day.
const getDailySummaryPath = "/datacube/getweanalysisappiddailysummarytrend"

// DatacubeDailySummary is the overview of one day.
type DatacubeDailySummary struct {
	// RefDate is the day, yyyymmdd.
	RefDate string `json:"ref_date"`
	// VisitTotal is the cumulative number of users.
	VisitTotal int `json:"visit_total"`
	// SharePV is the number of times the mini program was forwarded.
	SharePV int `json:"share_pv"`
	// ShareUV is the number of users who forwarded it.
	ShareUV int `json:"share_uv"`
}

// GetDailySummaryResponse is the reply of the summary endpoint.
type GetDailySummaryResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// List is the overview of the requested day.
	List []DatacubeDailySummary `json:"list"`
}

// OK reports whether the call succeeded.
//
// An empty List is a normal answer, not a failure.
func (r *GetDailySummaryResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetDailySummaryError turns the errcode from the reply into an error.
func GetDailySummaryError(r *GetDailySummaryResponse) error {
	return datacubeError(r.ErrCode)
}

// GetDailySummary reads the overview of one day: cumulative users and how much
// the mini program was shared.
//
// begin_date and end_date must be the same day. The three figures here overlap
// with the trend endpoint's, which is the one to use when the per-day breakdown
// matters.
func (a *Application) GetDailySummary(
	ctx context.Context,
	date DatacubeDateRange,
) (*GetDailySummaryResponse, error) {
	if err := CheckDatacubeDayRange(date); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getDailySummaryPath, nil, date.body())
	if err != nil {
		return nil, err
	}

	var result GetDailySummaryResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
