package miniProgram

import (
	"context"
	"net/url"
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getFeedbackPath pages through the feedback users submitted from the mini
// program.
const getFeedbackPath = "/wxaapi/feedback/list"

// Values of the type parameter: the category of feedback to pull.
const (
	// FeedbackTypeAll pulls every category; it is what an empty type does.
	FeedbackTypeAll = 0
	// FeedbackTypeCannotOpen is 无法打开小程序, 1.
	FeedbackTypeCannotOpen = 1
	// FeedbackTypeCrash is 小程序闪退, 2.
	FeedbackTypeCrash = 2
	// FeedbackTypeLag is 卡顿, 3.
	FeedbackTypeLag = 3
	// FeedbackTypeBlank is 黑屏白屏, 4.
	FeedbackTypeBlank = 4
	// FeedbackTypeFreeze is 死机, 5.
	FeedbackTypeFreeze = 5
	// FeedbackTypeMisaligned is 界面错位, 6.
	FeedbackTypeMisaligned = 6
	// FeedbackTypeSlowLoad is 界面加载慢, 7.
	FeedbackTypeSlowLoad = 7
	// FeedbackTypeOther is 其他异常, 8.
	FeedbackTypeOther = 8
)

// FeedbackTypeMsg returns the Chinese name of a feedback category.
func FeedbackTypeMsg(feedbackType int) string {
	switch feedbackType {
	case FeedbackTypeCannotOpen:
		return "无法打开小程序"
	case FeedbackTypeCrash:
		return "小程序闪退"
	case FeedbackTypeLag:
		return "卡顿"
	case FeedbackTypeBlank:
		return "黑屏白屏"
	case FeedbackTypeFreeze:
		return "死机"
	case FeedbackTypeMisaligned:
		return "界面错位"
	case FeedbackTypeSlowLoad:
		return "界面加载慢"
	case FeedbackTypeOther:
		return "其他异常"
	default:
		return "未知类型"
	}
}

// GetFeedbackParams pages the feedback list.
type GetFeedbackParams struct {
	// Type is one of the FeedbackType* values; zero pulls every category.
	Type int
	// Page is the page number, counting from 1.
	Page int
	// Num is how many records to return per page.
	Num int
}

// CheckGetFeedbackParams validates params before anything is sent.
func CheckGetFeedbackParams(params GetFeedbackParams) error {
	if params.Type < FeedbackTypeAll || params.Type > FeedbackTypeOther {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetFeedback type must be 0-8",
		)
	}

	// The page number is 1-based: 0 is not "the first page", it is a rejected
	// request.
	if params.Page < 1 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetFeedback page must be 1 or greater",
		)
	}

	if params.Num < 1 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetFeedback num must be 1 or greater",
		)
	}

	return nil
}

// query renders params as the query string the endpoint expects.
func (p GetFeedbackParams) query() url.Values {
	values := url.Values{
		"page": {strconv.Itoa(p.Page)},
		"num":  {strconv.Itoa(p.Num)},
	}

	// Zero means "every category", and is left out rather than sent as type=0:
	// the parameter is documented as optional with that exact default.
	if p.Type != FeedbackTypeAll {
		values.Set("type", strconv.Itoa(p.Type))
	}

	return values
}

// Feedback is one piece of feedback a user submitted.
type Feedback struct {
	// RecordID identifies the feedback; it is the id GetFeedbackMedia needs to
	// fetch the attached images.
	RecordID int64 `json:"record_id"`
	// CreateTime is when it was submitted, Unix seconds.
	CreateTime int64 `json:"create_time"`
	// Content is what the user wrote.
	Content string `json:"content"`
	// Phone is the contact the user left: a phone number or an email.
	//
	// The table calls it a string while the example sends 18800000000 unquoted,
	// so both are accepted; a number keeps its digits exactly.
	Phone FlexString `json:"phone"`
	// OpenID is the openid of the user who submitted it.
	OpenID string `json:"openid"`
	// Nickname is the user's nickname.
	Nickname string `json:"nickname"`
	// HeadURL is the user's avatar URL.
	HeadURL string `json:"head_url"`
	// Type is one of the FeedbackType* values.
	Type int `json:"type"`
	// MediaIDs are the images attached to the feedback.
	MediaIDs []string `json:"mediaIds"`
	// SystemInfo is the device info the client attached, as a JSON string.
	//
	// It is not in the parameter table but is present in the example, and it is
	// the only report of what the user was running.
	SystemInfo string `json:"systemInfo"`
}

// HasMedia reports whether the feedback carries images.
func (f Feedback) HasMedia() bool {
	return len(f.MediaIDs) > 0
}

// GetFeedbackResponse is the reply of /wxaapi/feedback/list.
type GetFeedbackResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// List is the page of feedback.
	List []Feedback `json:"list"`
	// TotalNum is how many pieces of feedback match, not how many are in List.
	TotalNum int `json:"total_num"`
}

// OK reports whether the page was read.
//
// An empty list is a valid reply: no feedback of that category has been
// submitted, which is not a failure.
func (r *GetFeedbackResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// GetFeedbackError turns the errcode from the reply into an error.
func GetFeedbackError(r *GetFeedbackResponse) error {
	return operationError(r.ErrCode)
}

// GetFeedback pages through the feedback users submitted.
//
// The endpoint has no "since" cursor, only a page number and a total, so a
// complete pull is a loop over TotalNum/Num pages: with new feedback arriving
// during the loop, a record can shift onto a page that was already read.
func (a *Application) GetFeedback(
	ctx context.Context,
	params GetFeedbackParams,
) (*GetFeedbackResponse, error) {
	if err := CheckGetFeedbackParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getFeedbackPath, params.query())
	if err != nil {
		return nil, err
	}

	var result GetFeedbackResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
