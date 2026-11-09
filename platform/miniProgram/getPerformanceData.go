package miniProgram

import (
	"context"
	"strconv"
	"strings"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getPerformanceDataPath reads the performance metrics of the mini program.
const getPerformanceDataPath = "/wxa/business/performance/boot"

// Values of module: which metric to read.
//
// They are strings on the wire: the parameter table says number, but both the
// HTTP and the cloud-call examples send "10022", and the examples are what the
// service is exercised against.
const (
	// PerfModuleOpenRate is 10016: cold-start open rate.
	PerfModuleOpenRate = "10016"
	// PerfModuleStartupStages is 10017: time spent in each startup stage.
	PerfModuleStartupStages = "10017"
	// PerfModulePageSwitch is 10021: time spent switching pages.
	PerfModulePageSwitch = "10021"
	// PerfModuleMemory is 10022: memory usage.
	PerfModuleMemory = "10022"
	// PerfModuleMemoryAbnormal is 10023: abnormal memory usage.
	PerfModuleMemoryAbnormal = "10023"
)

// PerfModuleMap holds the descriptions of the module values.
var PerfModuleMap = map[string]string{
	PerfModuleOpenRate:       "打开率",
	PerfModuleStartupStages:  "启动各阶段耗时",
	PerfModulePageSwitch:     "页面切换耗时",
	PerfModuleMemory:         "内存指标",
	PerfModuleMemoryAbnormal: "内存异常",
}

// PerfModuleMsg returns the description of a module, or "" when the value is
// undocumented.
func PerfModuleMsg(module string) string {
	return PerfModuleMap[module]
}

// Values of field: the filter to narrow a module by.
const (
	// PerfFieldNetworkType filters by network type; value is "-1", "3g", "4g" or
	// "wifi", where -1 means all of them.
	PerfFieldNetworkType = "networktype"
	// PerfFieldDeviceLevel filters by device class; value is "-1", "1", "2" or
	// "3", where -1 means all classes.
	PerfFieldDeviceLevel = "device_level"
	// PerfFieldDevice filters by platform; value is "-1", "1" or "2", where -1
	// means all platforms.
	PerfFieldDevice = "device"
)

// PerfMaxRangeSeconds is the documented limit on the time span: 30 days.
const PerfMaxRangeSeconds = 30 * 24 * 60 * 60

// GetPerformanceDataParams selects the metric and the period to read.
type GetPerformanceDataParams struct {
	// Module is one of the PerfModule* values.
	Module string

	// BeginTimestamp and EndTimestamp are Unix seconds, and must span at most 30
	// days.
	BeginTimestamp int64
	EndTimestamp   int64

	// Filters narrow the data down, e.g. by network type or device class.
	//
	// At least one is required. Omitting a field means "all values of it", so
	// there is nothing to gain by sending an empty list.
	Filters []GetPerformanceDataFilter
}

// GetPerformanceDataFilter is one filter of a performance query.
type GetPerformanceDataFilter struct {
	// Field is one of the PerfField* values.
	Field string
	// Value is the value to select, e.g. "wifi" or "1".
	Value string
}

// CheckGetPerformanceDataParams validates params before anything is sent to
// WeChat.
func CheckGetPerformanceDataParams(params GetPerformanceDataParams) error {
	if params.Module == "" {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetPerformanceData module is required",
		)
	}

	if params.BeginTimestamp <= 0 || params.EndTimestamp <= 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetPerformanceData begin_timestamp and end_timestamp are required",
		)
	}

	if params.EndTimestamp < params.BeginTimestamp {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetPerformanceData begin_timestamp must not be after end_timestamp",
		)
	}

	// The span is checked here because WeChat answers a generic 参数错误 for it,
	// which is hard to tell apart from a wrong module or a wrong filter.
	if params.EndTimestamp-params.BeginTimestamp > PerfMaxRangeSeconds {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetPerformanceData time span must not exceed 30 days",
		)
	}

	if len(params.Filters) == 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetPerformanceData params is required",
		)
	}

	for i, filter := range params.Filters {
		// Reported with the index: a filter list is short, but "one of them is
		// empty" still does not say which.
		if filter.Field == "" || filter.Value == "" {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: GetPerformanceData params["+strconv.Itoa(i)+"] needs both field and value",
			)
		}
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p GetPerformanceDataParams) body() *getPerformanceDataBody {
	filters := make([]getPerformanceDataFilter, 0, len(p.Filters))
	for _, filter := range p.Filters {
		filters = append(filters, getPerformanceDataFilter{
			Field: filter.Field,
			Value: filter.Value,
		})
	}

	return &getPerformanceDataBody{
		Module: p.Module,
		Time: getPerformanceDataTime{
			BeginTimestamp: p.BeginTimestamp,
			EndTimestamp:   p.EndTimestamp,
		},
		Params: filters,
	}
}

// getPerformanceDataBody is the JSON body of the performance endpoint.
type getPerformanceDataBody struct {
	Module string                     `json:"module"`
	Time   getPerformanceDataTime     `json:"time"`
	Params []getPerformanceDataFilter `json:"params"`
}

// getPerformanceDataTime is the time range of a performance query.
type getPerformanceDataTime struct {
	BeginTimestamp int64 `json:"begin_timestamp"`
	EndTimestamp   int64 `json:"end_timestamp"`
}

// getPerformanceDataFilter is one filter of the JSON body.
type getPerformanceDataFilter struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

// PerformanceField is one data point: a metric value on one day.
type PerformanceField struct {
	// RefDate is the day, yyyymmdd.
	RefDate string `json:"refdate"`
	// Value is the metric value, as a string.
	//
	// WeChat sends numbers as strings here (e.g. "70.7778"); use Float to read it
	// as a number.
	Value string `json:"value"`
}

// Float returns Value as a number.
func (f PerformanceField) Float() (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(f.Value), 64)
	if err != nil {
		return 0, cerrors.Wrap(
			cerrors.CodeInvalidArgument,
			"miniprogram: performance value is not a number: "+f.Value,
			err,
		)
	}

	return value, nil
}

// PerformanceLine is a series of data points.
//
// The series is wrapped in an array of lines with a single element each, which is
// how WeChat sends it; the shape is mirrored rather than flattened so decoding
// stays faithful to the reply.
type PerformanceLine struct {
	Fields []PerformanceField `json:"fields"`
}

// PerformanceTable is one metric over time.
type PerformanceTable struct {
	// ID is the metric id, e.g. "memorydiff".
	ID string `json:"id"`
	// Zh is the metric name in Chinese, e.g. "内存增长均值".
	Zh string `json:"zh"`
	// Lines holds the data points.
	Lines []PerformanceLine `json:"lines"`
}

// PerformanceDataBody is the payload of a performance reply.
type PerformanceDataBody struct {
	Tables []PerformanceTable `json:"tables"`
	Count  int                `json:"count"`
}

// PerformanceData wraps the payload.
type PerformanceData struct {
	Body PerformanceDataBody `json:"body"`
}

// GetPerformanceDataResponse is the reply of the performance endpoint.
type GetPerformanceDataResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Data holds the metrics.
	Data *PerformanceData `json:"data"`
}

// OK reports whether the call returned metrics.
func (r *GetPerformanceDataResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.Data != nil
}

// GetPerformanceDataError turns the errcode from the reply into an error.
func GetPerformanceDataError(r *GetPerformanceDataResponse) error {
	return datacubeError(r.ErrCode)
}

// GetPerformanceData reads performance metrics -- startup stages, page switches,
// memory -- over a period of at most 30 days.
//
// The metrics are grouped by physical device behaviour, so the filters matter:
// a result for one network type says nothing about the others.
func (a *Application) GetPerformanceData(
	ctx context.Context,
	params GetPerformanceDataParams,
) (*GetPerformanceDataResponse, error) {
	if err := CheckGetPerformanceDataParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getPerformanceDataPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GetPerformanceDataResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
