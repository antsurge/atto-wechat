package miniProgram

import (
	"errors"
	"time"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
)

// The datacube family -- /datacube/getweanalysisappid* -- shares its error codes
// and, more importantly, its date rules: every call is a date range in yyyymmdd,
// and each endpoint accepts only one range shape. The rules live here so the
// eleven endpoints validate the same way.
const (
	// DatacubeDateLayout is the only date format the datacube endpoints accept.
	DatacubeDateLayout = "20060102"

	// DatacubeDefaultErrMsg is returned for an errcode the datacube endpoints do
	// not document.
	DatacubeDefaultErrMsg = "系统错误，请稍后再试"
)

// DatacubeErrCodeMap holds the errcode descriptions WeChat documents for the
// datacube endpoints, keyed by the errcode as it appears in the reply.
//
// It is exported so callers can render a human-readable reason for an errcode
// without repeating the table themselves.
var DatacubeErrCodeMap = map[int]string{
	-1:    "系统繁忙，此时请开发者稍候再试",
	40001: "access_token 无效，或获取 access_token 时 AppSecret 错误",
	61504: "系统繁忙，稍后再试",
}

// DatacubeErrMsg returns the description of a datacube error, or the default when
// the errcode is undocumented.
func DatacubeErrMsg(code int) string {
	if msg, ok := DatacubeErrCodeMap[code]; ok {
		return msg
	}

	return DatacubeDefaultErrMsg
}

// DatacubeDateRange is the date range every datacube call takes, in yyyymmdd.
//
// What a valid range is depends on the endpoint -- one day, one natural week, one
// natural month, or a recent 1/7/30 days -- so each call checks it with the
// matching Check function before sending.
type DatacubeDateRange struct {
	// BeginDate is the first day, yyyymmdd.
	BeginDate string
	// EndDate is the last day, yyyymmdd.
	EndDate string
}

// body renders the range as the JSON body the endpoints expect.
func (r DatacubeDateRange) body() *datacubeDateRangeBody {
	return &datacubeDateRangeBody{BeginDate: r.BeginDate, EndDate: r.EndDate}
}

// datacubeDateRangeBody is the JSON body shared by the datacube endpoints.
type datacubeDateRangeBody struct {
	BeginDate string `json:"begin_date"`
	EndDate   string `json:"end_date"`
}

// parseDatacubeDate validates one end of a range.
//
// time.Parse is what rejects 20170230: a range that looks right but names a day
// that does not exist would otherwise come back as an empty list, which reads as
// "no data" rather than "wrong date".
func parseDatacubeDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: datacube date is required",
		)
	}

	date, err := time.Parse(DatacubeDateLayout, value)
	if err != nil {
		return time.Time{}, cerrors.Wrap(
			cerrors.CodeInvalidArgument,
			"miniprogram: datacube date must be a valid yyyymmdd",
			err,
		)
	}

	return date, nil
}

// parseDatacubeRange validates both ends and their order.
func parseDatacubeRange(r DatacubeDateRange) (time.Time, time.Time, error) {
	begin, err := parseDatacubeDate(r.BeginDate)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	end, err := parseDatacubeDate(r.EndDate)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	if begin.After(end) {
		return time.Time{}, time.Time{}, cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: datacube begin_date must not be after end_date",
		)
	}

	return begin, end, nil
}

// daysBetween counts calendar days from begin to end.
func daysBetween(begin, end time.Time) int {
	return int(end.Sub(begin).Hours() / 24)
}

// CheckDatacubeDayRange validates a one-day range, used by the endpoints whose
// end_date must equal begin_date.
func CheckDatacubeDayRange(r DatacubeDateRange) error {
	begin, end, err := parseDatacubeRange(r)
	if err != nil {
		return err
	}

	if daysBetween(begin, end) != 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: datacube begin_date and end_date must be the same day",
		)
	}

	return nil
}

// CheckDatacubeWeekRange validates a natural-week range: Monday to Sunday, seven
// days.
//
// WeChat only aggregates by natural week, so an arbitrary seven-day range is not
// a shifted week -- it is a range the endpoint cannot answer.
func CheckDatacubeWeekRange(r DatacubeDateRange) error {
	begin, end, err := parseDatacubeRange(r)
	if err != nil {
		return err
	}

	if begin.Weekday() != time.Monday {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: datacube week query begin_date must be a Monday",
		)
	}

	if daysBetween(begin, end) != 6 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: datacube week query must cover 7 days, Monday to Sunday",
		)
	}

	return nil
}

// CheckDatacubeMonthRange validates a natural-month range: the first day of a
// month to its last day.
func CheckDatacubeMonthRange(r DatacubeDateRange) error {
	begin, end, err := parseDatacubeRange(r)
	if err != nil {
		return err
	}

	if begin.Day() != 1 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: datacube month query begin_date must be the first day of a month",
		)
	}

	// The last day of the month is the day before the first day of the next one,
	// which is also the only way to get February right in a leap year.
	nextMonth := time.Date(begin.Year(), begin.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	if !end.Equal(nextMonth.AddDate(0, 0, -1)) {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: datacube month query must cover exactly one calendar month",
		)
	}

	return nil
}

// CheckDatacubePortraitRange validates the 1/7/30-day range the portrait endpoint
// requires: begin_date and end_date have to differ by 0, 6 or 29 days.
func CheckDatacubePortraitRange(r DatacubeDateRange) error {
	begin, end, err := parseDatacubeRange(r)
	if err != nil {
		return err
	}

	switch daysBetween(begin, end) {
	case 0, 6, 29:
		return nil
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: datacube portrait query must span 1, 7 or 30 days",
		)
	}
}

// DatacubeRetentionPoint is one bucket of a retention curve.
type DatacubeRetentionPoint struct {
	// Key is the offset: 0 is the day (week, month) itself, 1 is one period
	// later, and so on.
	//
	// The day curve documents 0-7, 14, 30; the week curve 0-4; the month curve
	// only 0 and 1.
	Key int `json:"key"`
	// Value is the number of users in this bucket: for key 0 it is the base
	// (new or active users), for the others the users who came back.
	Value int `json:"value"`
}

// DatacubeRetention is the reply shape shared by the daily, weekly and monthly
// retention endpoints.
//
// None of them wraps the data in a list: there is exactly one period per call.
type DatacubeRetention struct {
	// RefDate is the period, e.g. "20170313", "20170306-20170312" or "201702".
	RefDate string `json:"ref_date"`
	// VisitUVNew is the retention of users who first came in this period.
	VisitUVNew []DatacubeRetentionPoint `json:"visit_uv_new"`
	// VisitUV is the retention of users active in this period.
	VisitUV []DatacubeRetentionPoint `json:"visit_uv"`
}

// DatacubeVisitTrend is one period of visit statistics, shared by the daily,
// weekly and monthly trend endpoints.
type DatacubeVisitTrend struct {
	// RefDate is the period: yyyymmdd for a day, yyyymmdd-yyyymmdd for a week,
	// yyyymm for a month.
	RefDate string `json:"ref_date"`
	// SessionCnt is the number of sessions opened.
	SessionCnt int `json:"session_cnt"`
	// VisitPV is the number of visits.
	VisitPV int `json:"visit_pv"`
	// VisitUV is the number of visitors, de-duplicated over the period.
	VisitUV int `json:"visit_uv"`
	// VisitUVNew is the number of first-time visitors.
	VisitUVNew int `json:"visit_uv_new"`
	// StayTimeUV is the average stay per visitor, in seconds.
	StayTimeUV float64 `json:"stay_time_uv"`
	// StayTimeSession is the average stay per session, in seconds.
	StayTimeSession float64 `json:"stay_time_session"`
	// VisitDepth is the average number of pages per session.
	VisitDepth float64 `json:"visit_depth"`
}

// datacubeError turns an errcode into an error for any datacube endpoint.
func datacubeError(code int) error {
	return errors.New(DatacubeErrMsg(code))
}
