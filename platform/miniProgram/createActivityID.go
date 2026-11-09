package miniProgram

import (
	"context"
	"errors"
	"net/url"

	"github.com/antsurge/atto-wechat/core/response"
)

// createActivityIDPath issues the ID that a dynamic message is later updated by.
const (
	createActivityIDPath          = "/cgi-bin/message/wxopen/activityid/create"
	CreateActivityIDDefaultErrMsg = "系统错误，请稍后再试"
)

// CreateActivityIDErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/message/wxopen/activityid/create, keyed by the errcode as it appears
// in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var CreateActivityIDErrCodeMap = map[int]string{
	-1:    "系统繁忙，请稍后再试",
	40001: "access_token 无效",
	42001: "access_token 已过期",
}

// CreateActivityIDParams describes who may share the activity.
//
// Both fields are optional and only matter for a private message: they name the
// sharer, so no other user can share the activity_id. Leave both empty for an
// ordinary dynamic message.
type CreateActivityIDParams struct {
	// OpenID is the sharer's openid. It and UnionID are alternatives: pass at
	// most one.
	OpenID string

	// UnionID is the sharer's unionid. It and OpenID are alternatives: pass at
	// most one.
	UnionID string
}

// query renders params as the query string of the endpoint.
//
// An empty value is dropped rather than sent as "openid=": WeChat treats the
// presence of the parameter as an attempt to name a sharer and rejects the
// empty one.
func (p CreateActivityIDParams) query() url.Values {
	q := url.Values{}
	if p.OpenID != "" {
		q.Set("openid", p.OpenID)
	}
	if p.UnionID != "" {
		q.Set("unionid", p.UnionID)
	}

	return q
}

// CreateActivityIDResponse is the reply of
// /cgi-bin/message/wxopen/activityid/create.
type CreateActivityIDResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	ActivityID     string `json:"activity_id"`
	ExpirationTime int64  `json:"expiration_time"`
}

// OK reports whether the reply carries an activity_id.
func (r *CreateActivityIDResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.ActivityID != ""
}

// CreateActivityIDErrMsg returns the description of a createActivityId error.
func CreateActivityIDErrMsg(code int) string {
	if msg, ok := CreateActivityIDErrCodeMap[code]; ok {
		return msg
	}
	return CreateActivityIDDefaultErrMsg
}

// CreateActivityIDError turns the errcode from the reply into an error.
//
// The errcode itself is the error code, and its description comes straight from
// CreateActivityIDErrCodeMap. An undocumented errcode has no agreed meaning, so
// it falls back to a generic message rather than exposing WeChat's raw errmsg.
func CreateActivityIDError(r *CreateActivityIDResponse) error {
	return errors.New(CreateActivityIDErrMsg(r.ErrCode))
}

// CreateActivityID creates the activity_id of a dynamic message.
//
// A dynamic message is a card shared into a chat that the sender keeps updating
// afterwards: the joined member count of a group, the score of a challenge, and
// so on. The card content is fixed at share time, so this ID must be created
// before the message is shared, and it expires 24 hours later.
//
// The same call also backs private messages: passing OpenID or UnionID names the
// only user allowed to share the resulting activity_id.
func (a *Application) CreateActivityID(
	ctx context.Context,
	params CreateActivityIDParams,
) (*CreateActivityIDResponse, error) {
	resp, err := a.api.Get(ctx, createActivityIDPath, params.query())
	if err != nil {
		return nil, err
	}

	var result CreateActivityIDResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
