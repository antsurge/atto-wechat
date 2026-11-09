package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getSceneListPath lists the scene values the log endpoints accept.
const getSceneListPath = "/wxaapi/log/get_scene"

// LogScene is one scene: where the user came from.
type LogScene struct {
	// Name is the scene name in Chinese, e.g. "小程序历史列表".
	Name string `json:"name"`
	// Value is the scene id, used as the scene filter of GetPerformance.
	//
	// It is a string in one entry and a number in the next -- the first entry is
	// literally "@_all" -- so FlexString keeps both readable and hands the value
	// back as the text the filter parameter wants.
	Value FlexString `json:"value"`
}

// GetSceneListResponse is the reply of /wxaapi/log/get_scene.
type GetSceneListResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Scene is the list of scenes, starting with the "全部" entry.
	Scene []LogScene `json:"scene"`
}

// OK reports whether the list was read.
func (r *GetSceneListResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// Find returns the scene with the given value, if present.
//
// The list is a reference table, not a paged reply: a value missing from it is a
// scene this mini program cannot be opened from.
func (r *GetSceneListResponse) Find(value string) (LogScene, bool) {
	if r == nil {
		return LogScene{}, false
	}

	for _, scene := range r.Scene {
		if scene.Value.String() == value {
			return scene, true
		}
	}

	return LogScene{}, false
}

// GetSceneListError turns the errcode from the reply into an error.
func GetSceneListError(r *GetSceneListResponse) error {
	return operationError(r.ErrCode)
}

// GetSceneList lists the scenes the performance and log filters accept.
//
// The values are stable ids, so the reply can be cached: fetch it once and keep
// the ids, rather than calling this before every query.
func (a *Application) GetSceneList(ctx context.Context) (*GetSceneListResponse, error) {
	resp, err := a.api.Get(ctx, getSceneListPath, nil)
	if err != nil {
		return nil, err
	}

	var result GetSceneListResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
