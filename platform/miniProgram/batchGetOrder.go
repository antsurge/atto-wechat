package miniProgram

import (
	"context"
	"errors"
	"strconv"

	"github.com/antsurge/atto-wechat/core/response"
)

// batchGetOrderPath returns up to 100 waybills in one call.
const (
	batchGetOrderPath          = "/cgi-bin/express/business/order/batchget"
	BatchGetOrderDefaultErrMsg = "系统错误，请稍后再试"
)

// BatchGetOrderErrCodeMap holds the errcode descriptions WeChat documents for
// /cgi-bin/express/business/order/batchget, keyed by the errcode as it appears in
// the reply.
var BatchGetOrderErrCodeMap = logisticsErrCodes()

// BatchGetOrderMaxSize is the number of orders one call accepts.
const BatchGetOrderMaxSize = 100

// LogisticsOrderQuery identifies one waybill inside a batch request.
type LogisticsOrderQuery struct {
	// OrderID is the merchant order id.
	OrderID string `json:"order_id"`
	// OpenID is the buyer, as in GetOrderParams.
	OpenID string `json:"openid,omitempty"`
	// DeliveryID is the courier id.
	DeliveryID string `json:"delivery_id"`
	// WaybillID is the waybill number; omit it to look the waybill up by order id.
	WaybillID string `json:"waybill_id,omitempty"`
}

// BatchGetOrderParams is the waybill list to fetch.
type BatchGetOrderParams struct {
	// OrderList is the waybills to fetch; at most BatchGetOrderMaxSize items.
	OrderList []LogisticsOrderQuery `json:"order_list"`
}

// CheckBatchGetOrderParams validates params before anything is sent to WeChat.
func CheckBatchGetOrderParams(params BatchGetOrderParams) error {
	if len(params.OrderList) == 0 {
		return invalidLogisticsField("order_list must not be empty")
	}
	if len(params.OrderList) > BatchGetOrderMaxSize {
		return invalidLogisticsField("order_list must not contain more than 100 orders")
	}

	for i, order := range params.OrderList {
		if order.OrderID == "" {
			return invalidLogisticsField("order_list[" + strconv.Itoa(i) + "].order_id is required")
		}
		if order.DeliveryID == "" {
			return invalidLogisticsField("order_list[" + strconv.Itoa(i) + "].delivery_id is required")
		}
	}

	return nil
}

// BatchGetOrderResponse is the reply of
// /cgi-bin/express/business/order/batchget.
type BatchGetOrderResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Business reply parameters.
	OrderList []LogisticsOrderData `json:"order_list"`
}

// OK reports whether WeChat answered with at least one waybill.
//
// A well-formed empty list is not treated as success: the caller asked for
// specific orders, so no order back means nothing could be printed.
func (r *BatchGetOrderResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && len(r.OrderList) > 0
}

// BatchGetOrderErrMsg returns the description of a batch get order error.
func BatchGetOrderErrMsg(code int) string {
	return logisticsErrMsg(BatchGetOrderErrCodeMap, code, BatchGetOrderDefaultErrMsg)
}

// BatchGetOrderError turns the errcode from the reply into an error.
func BatchGetOrderError(r *BatchGetOrderResponse) error {
	return errors.New(BatchGetOrderErrMsg(r.ErrCode))
}

// BatchGetOrder fetches several waybills in one call.
//
// It is the same data as GetOrder, batched: use it when printing a day's parcels
// instead of looping over GetOrder, which would burn the rate limit.
func (a *Application) BatchGetOrder(
	ctx context.Context,
	params BatchGetOrderParams,
) (*BatchGetOrderResponse, error) {
	if err := CheckBatchGetOrderParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, batchGetOrderPath, nil, params)
	if err != nil {
		return nil, err
	}

	var result BatchGetOrderResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
