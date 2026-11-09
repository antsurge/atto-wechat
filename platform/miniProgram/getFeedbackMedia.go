package miniProgram

import (
	"context"
	"net/url"
	"strconv"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getFeedbackMediaPath downloads one image attached to a piece of feedback.
const getFeedbackMediaPath = "/cgi-bin/media/getfeedbackmedia"

// GetFeedbackMediaParams identifies the image to download.
type GetFeedbackMediaParams struct {
	// RecordID is the feedback the image belongs to.
	//
	// It is required even though media_id already identifies the image: the
	// pair is what WeChat checks, so a media_id from one record cannot be pulled
	// through another.
	RecordID int64
	// MediaID is one of the MediaIDs of the feedback, from GetFeedback.
	MediaID string
}

// CheckGetFeedbackMediaParams validates params before anything is sent.
func CheckGetFeedbackMediaParams(params GetFeedbackMediaParams) error {
	if params.RecordID <= 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetFeedbackMedia record_id is required",
		)
	}

	if params.MediaID == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetFeedbackMedia media_id is required",
		)
	}

	return nil
}

// query renders params as the query string the endpoint expects.
func (p GetFeedbackMediaParams) query() url.Values {
	return url.Values{
		"record_id": {strconv.FormatInt(p.RecordID, 10)},
		"media_id":  {p.MediaID},
	}
}

// GetFeedbackMediaResponse is the reply of /cgi-bin/media/getfeedbackmedia.
//
// The endpoint answers with the image itself, not a JSON document: Image holds
// the bytes and an errcode is only set when the request failed.
type GetFeedbackMediaResponse struct {
	// Common reply parameters, set only on failure.
	response.WechatResponse

	// Image is the image content.
	Image []byte
	// ContentType is the media type WeChat reported, e.g. "image/jpeg". It may be
	// empty, in which case the bytes still have to be sniffed.
	ContentType string
}

// OK reports whether an image was returned.
func (r *GetFeedbackMediaResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && len(r.Image) > 0
}

// GetFeedbackMediaError turns the errcode from the reply into an error.
//
// 46001 means the media_id does not exist, which for feedback means the image
// was never stored rather than that the id was mistyped.
func GetFeedbackMediaError(r *GetFeedbackMediaResponse) error {
	return operationError(r.ErrCode)
}

// GetFeedbackMedia downloads one image attached to a piece of feedback.
//
// The bytes are returned rather than written anywhere: where feedback images end
// up is the caller's storage decision.
func (a *Application) GetFeedbackMedia(
	ctx context.Context,
	params GetFeedbackMediaParams,
) (*GetFeedbackMediaResponse, error) {
	if err := CheckGetFeedbackMediaParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, getFeedbackMediaPath, params.query())
	if err != nil {
		return nil, err
	}

	image, contentType, errResp := decodeImageReply(resp)
	if errResp.ErrCode != 0 {
		return &GetFeedbackMediaResponse{WechatResponse: errResp, ContentType: contentType}, nil
	}

	return &GetFeedbackMediaResponse{Image: image, ContentType: contentType}, nil
}
