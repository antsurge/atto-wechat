package miniProgram

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getLiveRoomAssistantListPath lists the assistants of a room.
const (
	getLiveRoomAssistantListPath          = "/wxaapi/broadcast/room/getassistantlist"
	GetLiveRoomAssistantListDefaultErrMsg = "系统错误，请稍后再试"
)

// GetLiveRoomAssistantListErrCodeMap holds the errcode descriptions WeChat
// documents for /wxaapi/broadcast/room/getassistantlist, keyed by the errcode as
// it appears in the reply.
var GetLiveRoomAssistantListErrCodeMap = liveErrCodes(
	300022,
	40001,
	43001,
)

// GetLiveRoomAssistantListParams identifies the room whose assistants are
// wanted.
type GetLiveRoomAssistantListParams struct {
	// RoomID is the room id returned by CreateLiveRoom.
	RoomID int64
}

// CheckGetLiveRoomAssistantListParams validates params before anything is sent
// to WeChat.
func CheckGetLiveRoomAssistantListParams(params GetLiveRoomAssistantListParams) error {
	if params.RoomID <= 0 {
		return cerrors.New(cerrors.CodeInvalidArgument, "miniprogram: roomId is required")
	}

	return nil
}

// query renders params as the query string of the endpoint.
func (p GetLiveRoomAssistantListParams) query() url.Values {
	return url.Values{"roomId": {strconv.FormatInt(p.RoomID, 10)}}
}

// GetLiveRoomAssistantListResponse is the reply of
// /wxaapi/broadcast/room/getassistantlist.
type GetLiveRoomAssistantListResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	List []LiveRoomAssistant `json:"list"`
}

// OK reports whether WeChat answered with a usable list.
//
// A room without assistants is a valid answer, so only the errcode is checked.
func (r *GetLiveRoomAssistantListResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetLiveRoomAssistantListErrMsg returns the description of a
// getAssistantList error.
func GetLiveRoomAssistantListErrMsg(code int) string {
	if msg, ok := GetLiveRoomAssistantListErrCodeMap[code]; ok {
		return msg
	}
	return GetLiveRoomAssistantListDefaultErrMsg
}

// GetLiveRoomAssistantListError turns the errcode from the reply into an error.
func GetLiveRoomAssistantListError(r *GetLiveRoomAssistantListResponse) error {
	return errors.New(GetLiveRoomAssistantListErrMsg(r.ErrCode))
}

// GetLiveRoomAssistantList lists the assistants of a room.
func (a *Application) GetLiveRoomAssistantList(
	ctx context.Context,
	params GetLiveRoomAssistantListParams,
) (*GetLiveRoomAssistantListResponse, error) {
	if err := CheckGetLiveRoomAssistantListParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getLiveRoomAssistantListPath, params.query())
	if err != nil {
		return nil, err
	}

	var result GetLiveRoomAssistantListResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
