package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getVisitPagePath reads per-page visit statistics of one day.
const getVisitPagePath = "/datacube/getweanalysisappidvisitpage"

// DatacubeVisitPage is the statistics of one page.
type DatacubeVisitPage struct {
	// PagePath is the page path, e.g. "pages/main/main.html".
	PagePath string `json:"page_path"`
	// PageVisitPV is the number of visits to the page.
	PageVisitPV int `json:"page_visit_pv"`
	// PageVisitUV is the number of visitors to the page.
	PageVisitUV int `json:"page_visit_uv"`
	// PageStayTimePV is the average stay per visit, in seconds.
	PageStayTimePV float64 `json:"page_staytime_pv"`
	// EntryPagePV is how often the page was the entry page.
	EntryPagePV int `json:"entrypage_pv"`
	// ExitPagePV is how often the page was the exit page.
	ExitPagePV int `json:"exitpage_pv"`
	// PageSharePV is how often the page was shared.
	PageSharePV int `json:"page_share_pv"`
	// PageShareUV is how many users shared the page.
	PageShareUV int `json:"page_share_uv"`
}

// GetVisitPageResponse is the reply of the page statistics endpoint.
type GetVisitPageResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// RefDate is the day the figures belong to, yyyymmdd.
	RefDate string `json:"ref_date"`
	// List is one entry per page.
	List []DatacubeVisitPage `json:"list"`
}

// OK reports whether the call succeeded.
//
// An empty List is a normal answer, not a failure.
func (r *GetVisitPageResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetVisitPageError turns the errcode from the reply into an error.
func GetVisitPageError(r *GetVisitPageResponse) error {
	return datacubeError(r.ErrCode)
}

// GetVisitPage reads per-page visit statistics of one day.
//
// begin_date and end_date must be the same day. The list is not sorted in any
// documented order, so sort by the metric you need before showing it.
func (a *Application) GetVisitPage(
	ctx context.Context,
	date DatacubeDateRange,
) (*GetVisitPageResponse, error) {
	if err := CheckDatacubeDayRange(date); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getVisitPagePath, nil, date.body())
	if err != nil {
		return nil, err
	}

	var result GetVisitPageResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
