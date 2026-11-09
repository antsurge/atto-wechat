package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getGrayReleasePlanPath reads the staged release plan of the mini program.
const getGrayReleasePlanPath = "/wxa/getgrayreleaseplan"

// Values of the status of a release plan.
const (
	// GrayReleaseStatusInitial means no run has started, 0.
	GrayReleaseStatusInitial = 0
	// GrayReleaseStatusRunning means the rollout is in progress, 1.
	GrayReleaseStatusRunning = 1
	// GrayReleaseStatusPaused means the rollout is paused, 2.
	GrayReleaseStatusPaused = 2
	// GrayReleaseStatusFinished means the rollout reached its percentage, 3.
	GrayReleaseStatusFinished = 3
	// GrayReleaseStatusDeleted means the plan was removed, 4.
	GrayReleaseStatusDeleted = 4
)

// GrayReleaseStatusMsg returns the Chinese name of a release plan status.
func GrayReleaseStatusMsg(status int) string {
	switch status {
	case GrayReleaseStatusInitial:
		return "初始状态"
	case GrayReleaseStatusRunning:
		return "执行中"
	case GrayReleaseStatusPaused:
		return "暂停中"
	case GrayReleaseStatusFinished:
		return "执行完毕"
	case GrayReleaseStatusDeleted:
		return "被删除"
	default:
		return "未知状态"
	}
}

// GrayReleasePlan is the current staged release plan.
type GrayReleasePlan struct {
	// Status is one of the GrayReleaseStatus* values.
	Status int `json:"status"`
	// CreateTimestamp is when the plan was created, Unix seconds.
	CreateTimestamp int64 `json:"create_timestamp"`
	// GrayPercentage is the share of users the release reaches, 0-100.
	GrayPercentage int `json:"gray_percentage"`
	// SupportDebuggerFirst reports whether project members get the release
	// before the rollout starts.
	//
	// The field on the wire is support_debuger_first, with the typo, and the tag
	// keeps it verbatim.
	SupportDebuggerFirst bool `json:"support_debuger_first"`
	// SupportExperiencerFirst reports whether experience members get the release
	// first.
	SupportExperiencerFirst bool `json:"support_experiencer_first"`
}

// Running reports whether the rollout is currently in progress.
func (p GrayReleasePlan) Running() bool {
	return p.Status == GrayReleaseStatusRunning
}

// GetGrayReleasePlanResponse is the reply of /wxa/getgrayreleaseplan.
type GetGrayReleasePlanResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// GrayReleasePlan is the plan, or nil when none exists.
	//
	// It is a pointer because "no plan" and "a plan of zeros" must be told
	// apart: status 0 is a real status, and a mini program that was never
	// staged has no plan object at all.
	GrayReleasePlan *GrayReleasePlan `json:"gray_release_plan"`
}

// OK reports whether the plan was read.
func (r *GetGrayReleasePlanResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// HasPlan reports whether a plan exists.
func (r *GetGrayReleasePlanResponse) HasPlan() bool {
	return r != nil && r.GrayReleasePlan != nil
}

// Running reports whether a rollout is in progress right now.
//
// It is the question worth asking before releasing again: WeChat refuses a new
// release while one is running.
func (r *GetGrayReleasePlanResponse) Running() bool {
	return r.HasPlan() && r.GrayReleasePlan.Running()
}

// GetGrayReleasePlanError turns the errcode from the reply into an error.
func GetGrayReleasePlanError(r *GetGrayReleasePlanResponse) error {
	return operationError(r.ErrCode)
}

// GetGrayReleasePlan reads the staged release plan.
//
// The reply is the plan, not a paged list: to see who received it, the client
// side is the only source.
func (a *Application) GetGrayReleasePlan(ctx context.Context) (*GetGrayReleasePlanResponse, error) {
	resp, err := a.api.Get(ctx, getGrayReleasePlanPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetGrayReleasePlanResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
