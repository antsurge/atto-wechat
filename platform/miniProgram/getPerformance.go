package miniProgram

import (
	"context"
	"strconv"
	"strings"

	"github.com/antsurge/atto-wechat/core/codec"
	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getPerformancePath reads the startup performance of the mini program.
const getPerformancePath = "/wxaapi/log/get_performance"

// Values of cost_time_type: which stage to measure.
const (
	// PerfCostTimeStartup is the total startup time, 1.
	PerfCostTimeStartup = 1
	// PerfCostTimeDownload is the package download time, 2.
	PerfCostTimeDownload = 2
	// PerfCostTimeFirstRender is the first render time, 3.
	PerfCostTimeFirstRender = 3
)

// Values of device.
const (
	// PerfDeviceIOS is iOS, "1".
	PerfDeviceIOS = "1"
	// PerfDeviceAndroid is Android, "2".
	PerfDeviceAndroid = "2"
)

// Values of is_download_code.
const (
	// PerfDownloadCodeYes means the package was downloaded, "1".
	PerfDownloadCodeYes = "1"
	// PerfDownloadCodeNo means it was cached, "2".
	PerfDownloadCodeNo = "2"
)

// Values of networktype.
const (
	PerfNetworkWifi = "wifi"
	PerfNetwork4G   = "4g"
	PerfNetwork3G   = "3g"
	PerfNetwork2G   = "2g"
)

// GetPerformanceParams selects the range and the slice of traffic to measure.
type GetPerformanceParams struct {
	// CostTimeType is one of the PerfCostTime* values.
	CostTimeType int
	// DefaultStartTime is the beginning of the range, Unix seconds.
	DefaultStartTime int64
	// DefaultEndTime is the end of the range, Unix seconds.
	DefaultEndTime int64

	// Device is one of the PerfDevice* values; empty means all devices.
	Device string
	// IsDownloadCode is one of the PerfDownloadCode* values; empty means both.
	// It only applies when CostTimeType is PerfCostTimeStartup.
	IsDownloadCode string
	// Scene is a scene value from GetSceneList, e.g. "1011"; empty means all.
	Scene string
	// NetworkType is one of the PerfNetwork* values; empty means all. It only
	// applies when CostTimeType is PerfCostTimeDownload.
	NetworkType string
}

// CheckGetPerformanceParams validates params before anything is sent.
func CheckGetPerformanceParams(params GetPerformanceParams) error {
	switch params.CostTimeType {
	case PerfCostTimeStartup, PerfCostTimeDownload, PerfCostTimeFirstRender:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetPerformance cost_time_type must be 1, 2 or 3",
		)
	}

	if params.DefaultStartTime <= 0 || params.DefaultEndTime <= 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetPerformance start and end time are required",
		)
	}

	if params.DefaultEndTime <= params.DefaultStartTime {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetPerformance end time must be after start time",
		)
	}

	switch params.Device {
	case "", OperationAll, PerfDeviceIOS, PerfDeviceAndroid:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetPerformance device must be 1 (iOS) or 2 (Android)",
		)
	}

	switch params.IsDownloadCode {
	case "", OperationAll, PerfDownloadCodeYes, PerfDownloadCodeNo:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetPerformance is_download_code must be 1 or 2",
		)
	}

	switch params.NetworkType {
	case "", OperationAll, PerfNetworkWifi, PerfNetwork4G, PerfNetwork3G, PerfNetwork2G:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetPerformance networktype must be wifi, 4g, 3g or 2g",
		)
	}

	// The scene comes from GetSceneList: "@_all" or a numeric id. A scene name
	// ("小程序历史列表") is the mistake worth catching, since WeChat answers it with
	// an empty series rather than an error.
	if params.Scene != "" && params.Scene != OperationAll {
		if _, err := strconv.ParseUint(params.Scene, 10, 32); err != nil {
			return cerrors.New(
				cerrors.CodeInvalidArgument,
				"miniprogram: GetPerformance scene must be a scene value from GetSceneList, got "+strconv.Quote(params.Scene),
			)
		}
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
//
// Every filter is documented as required, and "@_all" is how "no filter" is
// spelled, so an empty filter is sent as "@_all" instead of being dropped: a
// missing field is a parameter error rather than a default.
func (p GetPerformanceParams) body() *getPerformanceBody {
	return &getPerformanceBody{
		CostTimeType:     p.CostTimeType,
		DefaultStartTime: p.DefaultStartTime,
		DefaultEndTime:   p.DefaultEndTime,
		Device:           orOperationAll(p.Device),
		IsDownloadCode:   orOperationAll(p.IsDownloadCode),
		Scene:            orOperationAll(p.Scene),
		NetworkType:      orOperationAll(p.NetworkType),
	}
}

// orOperationAll turns an unset filter into "@_all".
func orOperationAll(value string) string {
	if value == "" {
		return OperationAll
	}

	return value
}

// getPerformanceBody is the JSON body of a performance query.
type getPerformanceBody struct {
	CostTimeType     int    `json:"cost_time_type"`
	DefaultStartTime int64  `json:"default_start_time"`
	DefaultEndTime   int64  `json:"default_end_time"`
	Device           string `json:"device"`
	IsDownloadCode   string `json:"is_download_code"`
	Scene            string `json:"scene"`
	NetworkType      string `json:"networktype"`
}

// PerfSample is one day of the performance series.
type PerfSample struct {
	// RefDate is the date, "20060102".
	RefDate string `json:"ref_date"`
	// CostTimeType is the stage measured, one of the PerfCostTime* values.
	CostTimeType int `json:"cost_time_type"`
	// CostTime is the duration in milliseconds.
	CostTime int `json:"cost_time"`
}

// perfTimeData is the nested object inside the JSON string WeChat replies with.
type perfTimeData struct {
	List []PerfSample `json:"list"`
}

// GetPerformanceResponse is the reply of /wxaapi/log/get_performance.
//
// The series arrive twice-encoded: the fields are JSON documents in their own
// right, so the series must be parsed before it can be read. Samples and
// CompareSamples do that.
type GetPerformanceResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// DefaultTimeData is the series over the requested range, as a JSON string.
	DefaultTimeData string `json:"default_time_data"`
	// CompareTimeData is the series over the comparison range, as a JSON string.
	// It is empty when there is nothing to compare against.
	CompareTimeData string `json:"compare_time_data"`
}

// OK reports whether the call succeeded.
//
// An empty series is a valid reply -- a range with no traffic -- so the presence
// of data is not part of success.
func (r *GetPerformanceResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// Samples parses the series over the requested range.
func (r *GetPerformanceResponse) Samples() ([]PerfSample, error) {
	return parsePerfTimeData(r.DefaultTimeData)
}

// CompareSamples parses the series over the comparison range.
func (r *GetPerformanceResponse) CompareSamples() ([]PerfSample, error) {
	return parsePerfTimeData(r.CompareTimeData)
}

// HasComparison reports whether WeChat returned a comparison range at all.
func (r *GetPerformanceResponse) HasComparison() bool {
	return r != nil && strings.TrimSpace(r.CompareTimeData) != ""
}

// parsePerfTimeData decodes one of the two nested JSON strings.
//
// The string is kept verbatim on the response, so a parse failure is reported
// here instead of during decoding: a reply with an unexpected shape still yields
// an errcode the caller can act on.
func parsePerfTimeData(data string) ([]PerfSample, error) {
	if strings.TrimSpace(data) == "" {
		return nil, nil
	}

	var parsed perfTimeData
	if err := codec.UnmarshalString(data, &parsed); err != nil {
		return nil, cerrors.Wrap(
			cerrors.CodeDecodeFailed,
			"miniprogram: GetPerformance reply carries malformed time data",
			err,
		)
	}

	return parsed.List, nil
}

// GetPerformanceError turns the errcode from the reply into an error.
func GetPerformanceError(r *GetPerformanceResponse) error {
	return operationError(r.ErrCode)
}

// GetPerformance reads the startup performance series of the mini program.
//
// Only the stage named by CostTimeType is measured, and the filters that do not
// apply to that stage are ignored by WeChat rather than rejected.
func (a *Application) GetPerformance(
	ctx context.Context,
	params GetPerformanceParams,
) (*GetPerformanceResponse, error) {
	if err := CheckGetPerformanceParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getPerformancePath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GetPerformanceResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
