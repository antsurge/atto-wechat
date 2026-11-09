package miniProgram

import (
	"context"
	"errors"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// listBusinessPath lists the 客服子商户 of the mini program.
const listBusinessPath = "/cgi-bin/business/list"

// ListBusinessPageSize is the largest page WeChat returns in one call.
const ListBusinessPageSize = 200

// ListBusinessParams is one page of the merchant list.
type ListBusinessParams struct {
	// Offset is how many merchants to skip, starting at 0.
	Offset int

	// Count is how many to return, at most ListBusinessPageSize.
	Count int
}

// CheckListBusinessParams validates params before anything is sent to WeChat.
func CheckListBusinessParams(params ListBusinessParams) error {
	if params.Offset < 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: ListBusiness offset must not be negative",
		)
	}

	if params.Count < 1 || params.Count > ListBusinessPageSize {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: ListBusiness count must be between 1 and 200",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p ListBusinessParams) body() *listBusinessBody {
	return &listBusinessBody{Offset: p.Offset, Count: p.Count}
}

// listBusinessBody is the JSON body of /cgi-bin/business/list.
type listBusinessBody struct {
	Offset int `json:"offset"`
	Count  int `json:"count"`
}

// ListBusinessResponse is the reply of /cgi-bin/business/list.
type ListBusinessResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	List []BusinessInfo `json:"list"`
}

// OK reports whether the call succeeded.
//
// An empty list is a valid answer: it means the offset is past the last merchant.
func (r *ListBusinessResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// Done reports whether this page was the last one.
//
// A page shorter than the requested count means the end has been reached, which is
// how WeChat documents paging for this endpoint.
func (r *ListBusinessResponse) Done(requested int) bool {
	return len(r.List) < requested
}

// ListBusinessErrMsg returns the description of a listBusiness error.
func ListBusinessErrMsg(code int) string {
	return BusinessErrMsg(code)
}

// ListBusinessError turns the errcode from the reply into an error.
func ListBusinessError(r *ListBusinessResponse) error {
	return errors.New(ListBusinessErrMsg(r.ErrCode))
}

// ListBusiness reads a page of 客服子商户.
//
// Paging is offset-based and a short page ends it, so the usual loop advances
// Offset by Count until Done reports true.
func (a *Application) ListBusiness(
	ctx context.Context,
	params ListBusinessParams,
) (*ListBusinessResponse, error) {
	if err := CheckListBusinessParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, listBusinessPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result ListBusinessResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
