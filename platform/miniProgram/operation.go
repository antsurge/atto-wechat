package miniProgram

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/antsurge/atto-wechat/core/codec"
	cerrors "github.com/antsurge/atto-wechat/core/errors"
)

// The operation endpoints (域名、性能、日志、反馈、发布) share an errcode space and
// a few quirks: numbers that arrive quoted, dates as "2006-01-02" strings. The
// shared parts live here.

// OperationErrDefaultMsg is returned for an errcode these endpoints do not
// document.
const OperationErrDefaultMsg = "运维接口返回了未记录的错误码"

// OperationAll is the value WeChat uses to mean "不筛选" in the filter parameters
// of the log endpoints.
//
// The parameter table writes it as "@_all:" with a trailing colon, but every
// documented example sends "@_all".
const OperationAll = "@_all"

// OperationErrCodeMap holds the errcode descriptions the operation endpoints
// document, keyed by the errcode as it appears in the reply.
var OperationErrCodeMap = map[int]string{
	-3:     "系统繁忙，请稍后再试",
	-1:     "系统繁忙，此时请开发者稍候再试",
	20011:  "频率限制，请稍后再试",
	40001:  "access_token 无效或不是最新的",
	46001:  "不存在 media_id 对应的数据",
	507010: "没有发布的小程序不能使用告警功能",
	200002: "参数错误：date、begintime、endtime 必填，且 endtime 必须大于 begintime",
	200007: "无权限",
	200010: "操作过于频繁，目前限制每分钟 50 次",
}

// OperationErrMsg returns the description of an errcode, or the default when the
// errcode is undocumented.
func OperationErrMsg(code int) string {
	if msg, ok := OperationErrCodeMap[code]; ok {
		return msg
	}

	return OperationErrDefaultMsg
}

// operationError turns an errcode into an error.
func operationError(code int) error {
	return errors.New(OperationErrMsg(code))
}

// FlexString is a string that also accepts a bare JSON number.
//
// Several operation replies mix the two in the same field: the feedback phone is
// documented as a string but shown as 18800000000, and a scene value is "@_all" in
// one entry and 1 in the next. Decoding those into string or int fails on half the
// replies, so the value is kept in its textual form -- which is exactly what the
// filter parameters of the other endpoints take back.
type FlexString string

// String returns the value as a plain string.
func (s FlexString) String() string {
	return string(s)
}

// IsEmpty reports whether the value was absent or blank.
func (s FlexString) IsEmpty() bool {
	return strings.TrimSpace(string(s)) == ""
}

// UnmarshalJSON accepts a JSON string, a number or null.
func (s *FlexString) UnmarshalJSON(data []byte) error {
	var value string
	if err := codec.Unmarshal(data, &value); err == nil {
		*s = FlexString(value)

		return nil
	}

	// A number is kept verbatim rather than through float64: 18800000000 is a
	// phone number, and a float round-trip is how such values lose their tail.
	var number json.Number
	if err := codec.Unmarshal(data, &number); err == nil {
		*s = FlexString(number.String())

		return nil
	}

	if string(data) == "null" {
		*s = ""

		return nil
	}

	return errors.New("miniprogram: expected a string or a number, got " + string(data))
}

// OperationDateLayout is the date format the log endpoints take: "2021-01-25".
const OperationDateLayout = "2006-01-02"

// CheckOperationDate validates a "xxxx-xx-xx" date.
//
// WeChat answers a malformed date with 200002, which lists three parameters and
// does not say which one is wrong.
func CheckOperationDate(name, value string) error {
	if _, err := time.Parse(OperationDateLayout, value); err != nil {
		return cerrors.Wrap(
			cerrors.CodeInvalidArgument,
			"miniprogram: "+name+" must be a date in the form 2006-01-02, got "+strconv.Quote(value),
			err,
		)
	}

	return nil
}

// CheckOperationDateRange validates two "xxxx-xx-xx" dates and their order.
func CheckOperationDateRange(name, begin, end string) error {
	if err := CheckOperationDate(name+" start time", begin); err != nil {
		return err
	}

	if err := CheckOperationDate(name+" end time", end); err != nil {
		return err
	}

	beginTime, err := time.Parse(OperationDateLayout, begin)
	if err != nil {
		return err
	}

	endTime, err := time.Parse(OperationDateLayout, end)
	if err != nil {
		return err
	}

	// Same day is a valid one-day range.
	if endTime.Before(beginTime) {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: "+name+" end time "+end+" is before start time "+begin,
		)
	}

	return nil
}
