package miniProgram

import (
	"context"
	"net/url"
	"strconv"
	"time"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// realTimeLogSearchPath searches the realtime logs the mini program uploaded.
const realTimeLogSearchPath = "/wxaapi/userlog/userlog_search"

// Log level bits. A record's level is the OR of the levels of its messages, so
// one record can be both Info and Warn (2|4 = 6).
const (
	// LogLevelInfo is log.info, 2.
	LogLevelInfo = 2
	// LogLevelWarn is log.warn, 4.
	LogLevelWarn = 4
	// LogLevelError is log.error, 8.
	LogLevelError = 8
)

// Values of platform.
const (
	// LogPlatformAndroid is Android, 1.
	LogPlatformAndroid = 1
	// LogPlatformIOS is iOS, 2.
	LogPlatformIOS = 2
)

// RealTimeLogSearchParams is the filter of a realtime log search.
type RealTimeLogSearchParams struct {
	// Date is the day to search, as "20060102".
	//
	// The parameter table says the last 7 days are allowed while errcode 200002
	// says 3, so the window is left to WeChat: it is the only side that knows
	// today's date in its own timezone.
	Date string
	// BeginTime is the start of the window, Unix seconds, on Date.
	BeginTime int64
	// EndTime is the end of the window, Unix seconds, on Date.
	EndTime int64

	// Start is the offset of the first record; empty means 0.
	Start int
	// Limit is how many records to return; empty means 20.
	Limit int

	// TraceID narrows to one launch of the mini program.
	//
	// A trace covers every page the user visited in that launch, which is what
	// makes it useful: it is the only way to see a whole session rather than one
	// page.
	TraceID string
	// URL narrows to one page, e.g. "pages/index/index".
	URL string
	// ID narrows to one user, by 微信号 or openid.
	ID string
	// FilterMsg narrows to the logs tagged by addFilterMsg / setFilterMsg.
	FilterMsg string
	// Level keeps the records at or above a level: LogLevelWarn returns the Warn
	// and Error records. Zero means every level.
	Level int
}

// CheckRealTimeLogSearchParams validates params before anything is sent.
func CheckRealTimeLogSearchParams(params RealTimeLogSearchParams) error {
	if _, err := time.Parse("20060102", params.Date); err != nil {
		return cerrors.Wrap(
			cerrors.CodeInvalidArgument,
			"miniprogram: RealTimeLogSearch date must be in the form 20060102, got "+strconv.Quote(params.Date),
			err,
		)
	}

	if params.BeginTime <= 0 || params.EndTime <= 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: RealTimeLogSearch begin and end time are required",
		)
	}

	if params.EndTime <= params.BeginTime {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: RealTimeLogSearch end time must be after begin time",
		)
	}

	if params.Start < 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: RealTimeLogSearch start must not be negative",
		)
	}

	if params.Limit < 0 {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: RealTimeLogSearch limit must not be negative",
		)
	}

	switch params.Level {
	case 0, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: RealTimeLogSearch level must be 2 (info), 4 (warn) or 8 (error)",
		)
	}

	return nil
}

// query renders params as the query string the endpoint expects.
//
// The documentation lists these as a request body, but the endpoint is a GET and
// the filters travel in the query string: a GET body is dropped by the proxies in
// front of api.weixin.qq.com, which shows up as errcode 200002 "参数必填" for
// parameters that were sent.
func (p RealTimeLogSearchParams) query() url.Values {
	values := url.Values{
		"date":      {p.Date},
		"begintime": {strconv.FormatInt(p.BeginTime, 10)},
		"endtime":   {strconv.FormatInt(p.EndTime, 10)},
	}

	if p.Start > 0 {
		values.Set("start", strconv.Itoa(p.Start))
	}

	if p.Limit > 0 {
		values.Set("limit", strconv.Itoa(p.Limit))
	}

	optional := map[string]string{
		"traceId":   p.TraceID,
		"url":       p.URL,
		"id":        p.ID,
		"filterMsg": p.FilterMsg,
	}

	for key, value := range optional {
		if value != "" {
			values.Set(key, value)
		}
	}

	if p.Level != 0 {
		values.Set("level", strconv.Itoa(p.Level))
	}

	return values
}

// LogMessage is one log call inside a record.
type LogMessage struct {
	// Time is when the call happened, Unix seconds.
	Time int64 `json:"time"`
	// Msg holds the arguments of the call, one per element.
	Msg []string `json:"msg"`
	// Level is one of the LogLevel* values.
	Level int `json:"level"`
}

// LogRecord is one page view worth of logs.
type LogRecord struct {
	// Level is the OR of the levels of Msg: 6 means the record holds both Info
	// and Warn entries. Use HasLevel to test for one level.
	Level int `json:"level"`
	// LibraryVersion is the base library version.
	LibraryVersion string `json:"libraryVersion"`
	// ClientVersion is the WeChat client version.
	ClientVersion string `json:"clientVersion"`
	// ID is the user's openid.
	ID string `json:"id"`
	// Timestamp is when the logs were uploaded, Unix seconds.
	Timestamp int64 `json:"timestamp"`
	// Platform is one of the LogPlatform* values.
	Platform int `json:"platform"`
	// URL is the page, e.g. "pages/chat/chat".
	URL string `json:"url"`
	// Msg is the ordered list of log calls.
	Msg []LogMessage `json:"msg"`
	// TraceID identifies the launch the record belongs to.
	TraceID string `json:"traceid"`
	// FilterMsg is the tag set by addFilterMsg / setFilterMsg.
	FilterMsg string `json:"filterMsg"`
}

// HasLevel reports whether the record contains a message of the given level.
//
// Testing Level for equality is the trap: a record with Info and Warn messages
// has level 6 and matches neither 2 nor 4.
func (r LogRecord) HasLevel(level int) bool {
	return r.Level&level != 0
}

// Messages returns the messages of one level, in call order.
func (r LogRecord) Messages(level int) []LogMessage {
	var messages []LogMessage

	for _, message := range r.Msg {
		if message.Level&level != 0 {
			messages = append(messages, message)
		}
	}

	return messages
}

// LogSearchResult is the page of logs plus the total behind it.
type LogSearchResult struct {
	// List is the page of records.
	List []LogRecord `json:"list"`
	// Total is how many records match, not how many are in List.
	Total int `json:"total"`
}

// RealTimeLogSearchResponse is the reply of /wxaapi/userlog/userlog_search.
type RealTimeLogSearchResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// Data is the page of logs.
	Data LogSearchResult `json:"data"`
}

// OK reports whether the search ran.
//
// An empty list is a valid reply: those logs were never uploaded, which happens
// for a release build before the user upgrades.
func (r *RealTimeLogSearchResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// RealTimeLogSearchError turns the errcode from the reply into an error.
func RealTimeLogSearchError(r *RealTimeLogSearchResponse) error {
	return operationError(r.ErrCode)
}

// RealTimeLogSearch searches the realtime logs uploaded by the log manager.
//
// The endpoint is rate limited to 50 calls per minute (200010); a paging loop
// that ignores Total hits the limit rather than the end.
func (a *Application) RealTimeLogSearch(
	ctx context.Context,
	params RealTimeLogSearchParams,
) (*RealTimeLogSearchResponse, error) {
	if err := CheckRealTimeLogSearchParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.Get(ctx, realTimeLogSearchPath, params.query())
	if err != nil {
		return nil, err
	}

	var result RealTimeLogSearchResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
