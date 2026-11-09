package miniProgram

import (
	"context"
	"strings"
	"testing"

	"github.com/antsurge/atto-wechat/core/codec"
)

const (
	testMD5Msg   = "53b4825ec4a41d966f88c298c718de80"
	testMD5Stack = "e371cd9cae821969c855f9f461327dac"
)

func TestGetDomainInfo(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok",`+
		`"requestdomain":["https://api.example.com"],"wsrequestdomain":[],`+
		`"uploaddomain":[],"downloaddomain":[],"udpdomain":[],`+
		`"bizdomain":["https://biz.example.com"]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetDomainInfo(context.Background(), GetDomainInfoParams{
		Action: DomainActionAll,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}

	if !resp.ContainsServerDomain("https://api.example.com") {
		t.Fatalf("request domain not found in %+v", resp.ServerDomains())
	}
	if resp.ContainsServerDomain("https://evil.example.com") {
		t.Fatal("unknown domain reported as configured")
	}

	// A list that was never requested is nil; one that came back empty is an
	// empty slice. The difference is "not asked" vs "none configured".
	if resp.RequestDomain == nil || resp.WSRequestDomain == nil {
		t.Fatalf("returned lists are nil: %+v", resp)
	}

	if call.method != "POST" || call.path != getDomainInfoPath {
		t.Fatalf("%s %s", call.method, call.path)
	}

	var body map[string]string
	call.decodeBody(t, &body)

	// An empty action reads everything and is left off the wire.
	if _, present := body["action"]; present {
		t.Fatalf("action sent while empty: %v", body)
	}
}

func TestGetDomainInfoValidation(t *testing.T) {
	for _, action := range []string{DomainActionAll, DomainActionBiz, DomainActionServer} {
		if err := CheckGetDomainInfoParams(GetDomainInfoParams{Action: action}); err != nil {
			t.Fatalf("action %q rejected: %v", action, err)
		}
	}

	if err := CheckGetDomainInfoParams(GetDomainInfoParams{Action: "getalldomain"}); err == nil {
		t.Fatal("unknown action accepted")
	}
}

func TestGetPerformance(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok",`+
		`"default_time_data":"{\"list\":[{\"ref_date\":\"20210125\",\"cost_time_type\":1,\"cost_time\":1200},`+
		`{\"ref_date\":\"20210126\",\"cost_time_type\":1,\"cost_time\":1400}]}",`+
		`"compare_time_data":"{\"list\":[{\"ref_date\":\"20210118\",\"cost_time_type\":1,\"cost_time\":1500}]}"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetPerformance(context.Background(), GetPerformanceParams{
		CostTimeType:     PerfCostTimeStartup,
		DefaultStartTime: 1611504000,
		DefaultEndTime:   1611676800,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}

	samples, err := resp.Samples()
	if err != nil {
		t.Fatal(err)
	}

	if len(samples) != 2 || samples[1].CostTime != 1400 || samples[0].RefDate != "20210125" {
		t.Fatalf("samples = %+v", samples)
	}

	if !resp.HasComparison() {
		t.Fatal("HasComparison() = false with compare_time_data present")
	}

	compare, err := resp.CompareSamples()
	if err != nil {
		t.Fatal(err)
	}

	if len(compare) != 1 || compare[0].CostTime != 1500 {
		t.Fatalf("compare samples = %+v", compare)
	}

	if call.path != getPerformancePath {
		t.Fatalf("path = %q", call.path)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	// cost_time_type is the only numeric field; the rest are strings.
	if body["cost_time_type"] != float64(1) {
		t.Fatalf("cost_time_type = %v", body["cost_time_type"])
	}

	// Unset filters are sent as "@_all" instead of being dropped: they are
	// documented as required.
	for _, key := range []string{"device", "is_download_code", "scene", "networktype"} {
		if body[key] != OperationAll {
			t.Fatalf("%s = %v, want %q", key, body[key], OperationAll)
		}
	}
}

func TestGetPerformanceEmptyComparison(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":0,"errmsg":"ok","default_time_data":"{\"list\":[]}"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetPerformance(context.Background(), GetPerformanceParams{
		CostTimeType:     PerfCostTimeFirstRender,
		DefaultStartTime: 1611504000,
		DefaultEndTime:   1611676800,
	})
	if err != nil {
		t.Fatal(err)
	}

	// An empty series is a range without traffic, not a failure.
	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if resp.HasComparison() {
		t.Fatal("HasComparison() = true without compare_time_data")
	}

	samples, err := resp.Samples()
	if err != nil || len(samples) != 0 {
		t.Fatalf("samples = %+v, err = %v", samples, err)
	}

	compare, err := resp.CompareSamples()
	if err != nil || len(compare) != 0 {
		t.Fatalf("compare = %+v, err = %v", compare, err)
	}
}

func TestGetPerformanceMalformedTimeData(t *testing.T) {
	// The nested document is parsed, not trusted: a shape change has to surface
	// as an error rather than as an empty series.
	srv, _ := newKFServer(t, `{"errcode":0,"errmsg":"ok","default_time_data":"not json"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetPerformance(context.Background(), GetPerformanceParams{
		CostTimeType:     PerfCostTimeStartup,
		DefaultStartTime: 1611504000,
		DefaultEndTime:   1611676800,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := resp.Samples(); err == nil {
		t.Fatal("malformed time data parsed without error")
	}
}

func TestGetPerformanceValidation(t *testing.T) {
	valid := GetPerformanceParams{
		CostTimeType:     PerfCostTimeDownload,
		DefaultStartTime: 1611504000,
		DefaultEndTime:   1611676800,
		Device:           PerfDeviceIOS,
		IsDownloadCode:   PerfDownloadCodeYes,
		Scene:            "1011",
		NetworkType:      PerfNetworkWifi,
	}

	if err := CheckGetPerformanceParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*GetPerformanceParams)
	}{
		{name: "bad cost time type", mutate: func(p *GetPerformanceParams) { p.CostTimeType = 4 }},
		{name: "no start", mutate: func(p *GetPerformanceParams) { p.DefaultStartTime = 0 }},
		{name: "no end", mutate: func(p *GetPerformanceParams) { p.DefaultEndTime = 0 }},
		{name: "end before start", mutate: func(p *GetPerformanceParams) { p.DefaultEndTime = p.DefaultStartTime }},
		{name: "bad device", mutate: func(p *GetPerformanceParams) { p.Device = "ios" }},
		{name: "bad download code", mutate: func(p *GetPerformanceParams) { p.IsDownloadCode = "yes" }},
		{name: "bad network", mutate: func(p *GetPerformanceParams) { p.NetworkType = "5g" }},
		// A scene name instead of a scene value comes back as an empty series.
		{name: "scene name", mutate: func(p *GetPerformanceParams) { p.Scene = "小程序历史列表" }},
	}

	for _, tc := range cases {
		params := valid
		tc.mutate(&params)

		if err := CheckGetPerformanceParams(params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}

	// "@_all" is a valid scene filter.
	all := valid
	all.Scene = OperationAll
	if err := CheckGetPerformanceParams(all); err != nil {
		t.Fatalf("@_all scene rejected: %v", err)
	}
}

func TestGetSceneList(t *testing.T) {
	// One value is a string, the next is a number: both must decode into the
	// same field and both must be usable as the scene filter.
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","scene":[`+
		`{"name":"全部","value":"@_all"},{"name":"小程序历史列表","value":1}]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetSceneList(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || len(resp.Scene) != 2 {
		t.Fatalf("resp = %+v", resp)
	}

	if resp.Scene[0].Value.String() != OperationAll {
		t.Fatalf("value = %q", resp.Scene[0].Value)
	}

	scene, ok := resp.Find("1")
	if !ok || scene.Name != "小程序历史列表" {
		t.Fatalf("find = %+v, %v", scene, ok)
	}

	if call.method != "GET" || call.path != getSceneListPath {
		t.Fatalf("%s %s", call.method, call.path)
	}
}

func TestGetVersionList(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","cvlist":[`+
		`{"type":1,"client_version_list":["7.0.20","7.0.21"]},`+
		`{"type":2,"client_version_list":[]}]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetVersionList(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}

	versions := resp.Versions(ClientVersionTypeClient)
	if len(versions) != 2 || versions[1] != "7.0.21" {
		t.Fatalf("versions = %+v", versions)
	}

	if versions := resp.Versions(ClientVersionTypeServiceDirect); len(versions) != 0 {
		t.Fatalf("service direct versions = %+v", versions)
	}

	if call.path != getVersionListPath {
		t.Fatalf("path = %q", call.path)
	}
}

func TestRealTimeLogSearch(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","data":{"list":[`+
		`{"level":6,"libraryVersion":"2.14.1","clientVersion":"7.0.21","id":"OPENID",`+
		`"timestamp":1611553677,"platform":2,"url":"pages/index/index",`+
		`"msg":[{"time":1611553671,"msg":["boot"],"level":2},`+
		`{"time":1611553672,"msg":["slow","api"],"level":4}],`+
		`"traceid":"TRACE","filterMsg":"tag"}],"total":1}}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.RealTimeLogSearch(context.Background(), RealTimeLogSearchParams{
		Date:      "20210125",
		BeginTime: 1611553600,
		EndTime:   1611557200,
		Limit:     20,
		Level:     LogLevelWarn,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || len(resp.Data.List) != 1 || resp.Data.Total != 1 {
		t.Fatalf("resp = %+v", resp)
	}

	record := resp.Data.List[0]

	// 6 is Info|Warn: testing level == 4 misses it, which is the trap.
	if !record.HasLevel(LogLevelWarn) || !record.HasLevel(LogLevelInfo) {
		t.Fatalf("level = %d", record.Level)
	}
	if record.HasLevel(LogLevelError) {
		t.Fatal("record reports Error level")
	}

	warns := record.Messages(LogLevelWarn)
	if len(warns) != 1 || warns[0].Msg[0] != "slow" {
		t.Fatalf("warn messages = %+v", warns)
	}

	if record.Platform != LogPlatformIOS || record.TraceID != "TRACE" {
		t.Fatalf("record = %+v", record)
	}

	// The filters travel in the query string: this is a GET, and a GET body is
	// dropped before it reaches WeChat.
	if call.method != "GET" || call.path != realTimeLogSearchPath {
		t.Fatalf("%s %s", call.method, call.path)
	}

	if got := call.query.Get("date"); got != "20210125" {
		t.Fatalf("date = %q", got)
	}
	if got := call.query.Get("begintime"); got != "1611553600" {
		t.Fatalf("begintime = %q", got)
	}
	if got := call.query.Get("endtime"); got != "1611557200" {
		t.Fatalf("endtime = %q", got)
	}
	if got := call.query.Get("limit"); got != "20" {
		t.Fatalf("limit = %q", got)
	}
	if got := call.query.Get("level"); got != "4" {
		t.Fatalf("level = %q", got)
	}

	if len(call.body) != 0 {
		t.Fatalf("GET carried a body: %q", call.body)
	}
}

func TestRealTimeLogSearchValidation(t *testing.T) {
	valid := RealTimeLogSearchParams{
		Date:      "20210125",
		BeginTime: 1611553600,
		EndTime:   1611557200,
	}

	if err := CheckRealTimeLogSearchParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*RealTimeLogSearchParams)
	}{
		{name: "date without dashes", mutate: func(p *RealTimeLogSearchParams) { p.Date = "2021-01-25" }},
		{name: "impossible date", mutate: func(p *RealTimeLogSearchParams) { p.Date = "20210230" }},
		{name: "no begin", mutate: func(p *RealTimeLogSearchParams) { p.BeginTime = 0 }},
		{name: "no end", mutate: func(p *RealTimeLogSearchParams) { p.EndTime = 0 }},
		{name: "end before begin", mutate: func(p *RealTimeLogSearchParams) { p.EndTime = p.BeginTime }},
		{name: "negative start", mutate: func(p *RealTimeLogSearchParams) { p.Start = -1 }},
		{name: "negative limit", mutate: func(p *RealTimeLogSearchParams) { p.Limit = -1 }},
		{name: "level not a bit", mutate: func(p *RealTimeLogSearchParams) { p.Level = 3 }},
	}

	for _, tc := range cases {
		params := valid
		tc.mutate(&params)

		if err := CheckRealTimeLogSearchParams(params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestGetFeedback(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","list":[`+
		`{"record_id":123456,"create_time":1611553677,"content":"打不开",`+
		`"phone":18800000000,"openid":"OPENID","nickname":"小明",`+
		`"head_url":"https://avatar.example.com/a.png","type":1,`+
		`"mediaIds":["MEDIA1","MEDIA2"],`+
		`"systemInfo":"{\"brand\":\"iPhone\"}"}],"total_num":37}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetFeedback(context.Background(), GetFeedbackParams{
		Type: FeedbackTypeCannotOpen,
		Page: 1,
		Num:  10,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || len(resp.List) != 1 || resp.TotalNum != 37 {
		t.Fatalf("resp = %+v", resp)
	}

	feedback := resp.List[0]

	// The phone is documented as a string and shown as a number.
	if feedback.Phone.String() != "18800000000" {
		t.Fatalf("phone = %q", feedback.Phone)
	}

	if !feedback.HasMedia() || len(feedback.MediaIDs) != 2 {
		t.Fatalf("media = %+v", feedback.MediaIDs)
	}

	if feedback.Type != FeedbackTypeCannotOpen || FeedbackTypeMsg(feedback.Type) != "无法打开小程序" {
		t.Fatalf("type = %d", feedback.Type)
	}

	if feedback.SystemInfo != `{"brand":"iPhone"}` {
		t.Fatalf("systemInfo = %q", feedback.SystemInfo)
	}

	if call.method != "GET" || call.path != getFeedbackPath {
		t.Fatalf("%s %s", call.method, call.path)
	}

	if got := call.query.Get("type"); got != "1" {
		t.Fatalf("type = %q", got)
	}
	if got := call.query.Get("page"); got != "1" {
		t.Fatalf("page = %q", got)
	}
	if got := call.query.Get("num"); got != "10" {
		t.Fatalf("num = %q", got)
	}
}

func TestGetFeedbackAllTypes(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","list":[],"total_num":0}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetFeedback(context.Background(), GetFeedbackParams{Page: 1, Num: 10})
	if err != nil {
		t.Fatal(err)
	}

	// No feedback yet is an empty page, not a failure.
	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}

	// Zero means every category, which is the documented default: the parameter
	// is left out rather than sent as type=0.
	if _, present := call.query["type"]; present {
		t.Fatalf("type sent while 0: %v", call.query)
	}
}

func TestGetFeedbackValidation(t *testing.T) {
	cases := []struct {
		name   string
		params GetFeedbackParams
	}{
		{name: "negative type", params: GetFeedbackParams{Type: -1, Page: 1, Num: 10}},
		{name: "type too large", params: GetFeedbackParams{Type: 9, Page: 1, Num: 10}},
		{name: "page zero", params: GetFeedbackParams{Page: 0, Num: 10}},
		{name: "num zero", params: GetFeedbackParams{Page: 1, Num: 0}},
	}

	for _, tc := range cases {
		if err := CheckGetFeedbackParams(tc.params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestGetFeedbackMedia(t *testing.T) {
	srv, call := newKFServer(t, "\xff\xd8\xff\xe0binaryjpeg")
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetFeedbackMedia(context.Background(), GetFeedbackMediaParams{
		RecordID: 123456,
		MediaID:  "MEDIA1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if string(resp.Image) != "\xff\xd8\xff\xe0binaryjpeg" {
		t.Fatalf("image = %q", resp.Image)
	}

	if call.path != getFeedbackMediaPath {
		t.Fatalf("path = %q", call.path)
	}

	if got := call.query.Get("record_id"); got != "123456" {
		t.Fatalf("record_id = %q", got)
	}
	if got := call.query.Get("media_id"); got != "MEDIA1" {
		t.Fatalf("media_id = %q", got)
	}
}

func TestGetFeedbackMediaError(t *testing.T) {
	// The same endpoint answers with a JSON document when the image is missing,
	// so the reply has to be split by content, not by status code.
	srv, _ := newKFServer(t, `{"errcode":46001,"errmsg":"invalid media_id"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetFeedbackMedia(context.Background(), GetFeedbackMediaParams{
		RecordID: 123456,
		MediaID:  "MISSING",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatalf("OK() = true, resp = %+v", resp)
	}
	if resp.ErrCode != 46001 {
		t.Fatalf("errcode = %d", resp.ErrCode)
	}
	if err := GetFeedbackMediaError(resp); err == nil || !strings.Contains(err.Error(), "media_id") {
		t.Fatalf("err = %v", err)
	}
}

func TestGetFeedbackMediaValidation(t *testing.T) {
	cases := []struct {
		name   string
		params GetFeedbackMediaParams
	}{
		{name: "no record", params: GetFeedbackMediaParams{MediaID: "MEDIA1"}},
		{name: "no media", params: GetFeedbackMediaParams{RecordID: 1}},
	}

	for _, tc := range cases {
		if err := CheckGetFeedbackMediaParams(tc.params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestGetJsErrList(t *testing.T) {
	srv, call := newKFServer(t, `{"success":true,"openid":"","data":[`+
		`{"errorMsgMd5":"`+testMD5Msg+`","errorMsg":"boom","uv":1,"pv":2,`+
		`"errorStackMd5":"`+testMD5Stack+`","errorStack":"stack",`+
		`"pvPercent":"10","uvPercent":"20"}],"totalCount":739,"errcode":0}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetJsErrList(context.Background(), GetJsErrListParams{
		StartTime: "2021-01-25",
		EndTime:   "2021-01-26",
	})
	if err != nil {
		t.Fatal(err)
	}

	// success is part of the verdict: this endpoint reports in it, not only in
	// errcode.
	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if len(resp.Data) != 1 || resp.TotalCount != 739 {
		t.Fatalf("resp = %+v", resp)
	}

	item := resp.Data[0]
	if item.ErrorMsgMD5 != testMD5Msg || item.UV != 1 || item.PV != 2 {
		t.Fatalf("item = %+v", item)
	}

	if call.path != getJsErrListPath {
		t.Fatalf("path = %q", call.path)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	// camelCase on the wire, unlike most endpoints.
	if body["startTime"] != "2021-01-25" || body["endTime"] != "2021-01-26" {
		t.Fatalf("body = %v", body)
	}

	// Unset filters fall back to the documented defaults instead of being
	// dropped: every field here is required.
	for key, want := range map[string]any{
		"appVersion": "0",
		"errType":    "0",
		"orderby":    "uv",
		"desc":       "1",
		"limit":      float64(JsErrListDefaultLimit),
		"offset":     float64(0),
	} {
		if body[key] != want {
			t.Fatalf("%s = %v, want %v", key, body[key], want)
		}
	}
}

func TestGetJsErrListWithoutSuccess(t *testing.T) {
	srv, _ := newKFServer(t, `{"success":false,"openid":"","data":[],"totalCount":0,"errcode":0}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetJsErrList(context.Background(), GetJsErrListParams{
		StartTime: "2021-01-25",
		EndTime:   "2021-01-26",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true with success=false")
	}
}

func TestGetJsErrListValidation(t *testing.T) {
	valid := GetJsErrListParams{StartTime: "2021-01-25", EndTime: "2021-01-26"}

	if err := CheckGetJsErrListParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*GetJsErrListParams)
	}{
		{name: "end before start", mutate: func(p *GetJsErrListParams) { p.EndTime = "2021-01-24" }},
		{name: "bad date", mutate: func(p *GetJsErrListParams) { p.StartTime = "20210125" }},
		{name: "bad err type", mutate: func(p *GetJsErrListParams) { p.ErrType = "9" }},
		{name: "bad order", mutate: func(p *GetJsErrListParams) { p.OrderBy = "count" }},
		{name: "asc as 0", mutate: func(p *GetJsErrListParams) { p.Desc = JsErrDetailOrderAsc }},
		{name: "negative offset", mutate: func(p *GetJsErrListParams) { p.Offset = -1 }},
		{name: "limit too large", mutate: func(p *GetJsErrListParams) { p.Limit = 31 }},
	}

	for _, tc := range cases {
		params := valid
		tc.mutate(&params)

		if err := CheckGetJsErrListParams(params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}

	// 30 is the documented cap and must pass.
	atCap := valid
	atCap.Limit = JsErrListMaxLimit
	if err := CheckGetJsErrListParams(atCap); err != nil {
		t.Fatalf("limit 30 rejected: %v", err)
	}
}

func TestGetJsErrDetail(t *testing.T) {
	srv, call := newKFServer(t, `{"success":true,"openid":"","data":[`+
		`{"Count":"1","sdkVersion":"2.14.1","ClientVersion":"7.0.21",`+
		`"errorStackMd5":"`+testMD5Stack+`","TimeStamp":"2021-01-25 16:36:39",`+
		`"appVersion":"2.6.16","errorMsgMd5":"`+testMD5Msg+`","errorMsg":"boom",`+
		`"errorStack":"stack","Ds":"2021-01-25","OsName":"1",`+
		`"openId":"OPENID","pluginversion":"0","appId":"wxcff7381e631cf54e",`+
		`"DeviceModel":"Redmi Note 5A","source":"","route":"","Uin":"","nickname":""}],`+
		`"totalCount":1,"errcode":0}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetJsErrDetail(context.Background(), GetJsErrDetailParams{
		StartTime:     "2021-01-25",
		EndTime:       "2021-01-26",
		ErrorMsgMD5:   testMD5Msg,
		ErrorStackMD5: testMD5Stack,
		OSName:        JsErrOSAndroid,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || len(resp.Data) != 1 || resp.TotalCount != 1 {
		t.Fatalf("resp = %+v", resp)
	}

	item := resp.Data[0]

	// Count, TimeStamp, ClientVersion and openId are the capitalised tags: a
	// wrong case here is a silently empty field, not a decode error.
	if item.Count != "1" || item.TimeStamp != "2021-01-25 16:36:39" {
		t.Fatalf("item = %+v", item)
	}
	if item.ClientVersion != "7.0.21" || item.OpenID != "OPENID" {
		t.Fatalf("item = %+v", item)
	}
	if item.OSName != JsErrOSAndroid || item.PluginVersion != "0" {
		t.Fatalf("item = %+v", item)
	}

	if call.path != getJsErrDetailPath {
		t.Fatalf("path = %q", call.path)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	if body["errorMsgMd5"] != testMD5Msg || body["errorStackMd5"] != testMD5Stack {
		t.Fatalf("body = %v", body)
	}

	// Empty version filters become "0", and desc defaults to descending.
	if body["appVersion"] != "0" || body["sdkVersion"] != "0" || body["clientVersion"] != "0" {
		t.Fatalf("body = %v", body)
	}
	if body["desc"] != JsErrDetailOrderDesc {
		t.Fatalf("desc = %v", body["desc"])
	}
	if body["osName"] != JsErrOSAndroid {
		t.Fatalf("osName = %v", body["osName"])
	}
}

func TestGetJsErrDetailValidation(t *testing.T) {
	valid := GetJsErrDetailParams{
		StartTime:     "2021-01-25",
		EndTime:       "2021-01-26",
		ErrorMsgMD5:   testMD5Msg,
		ErrorStackMD5: testMD5Stack,
	}

	if err := CheckGetJsErrDetailParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*GetJsErrDetailParams)
	}{
		{name: "no msg md5", mutate: func(p *GetJsErrDetailParams) { p.ErrorMsgMD5 = "" }},
		{name: "short msg md5", mutate: func(p *GetJsErrDetailParams) { p.ErrorMsgMD5 = "abc" }},
		{name: "non hex stack md5", mutate: func(p *GetJsErrDetailParams) {
			p.ErrorStackMD5 = strings.Repeat("z", 32)
		}},
		{name: "end before start", mutate: func(p *GetJsErrDetailParams) { p.EndTime = "2021-01-01" }},
		{name: "bad os", mutate: func(p *GetJsErrDetailParams) { p.OSName = "4" }},
		{name: "list asc value", mutate: func(p *GetJsErrDetailParams) { p.Desc = JsErrListOrderAsc }},
		{name: "negative offset", mutate: func(p *GetJsErrDetailParams) { p.Offset = -1 }},
	}

	for _, tc := range cases {
		params := valid
		tc.mutate(&params)

		if err := CheckGetJsErrDetailParams(params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}

	// The ascending value of this endpoint is the list endpoint's descending one.
	if JsErrDetailOrderAsc == JsErrListOrderAsc {
		t.Fatal("the two endpoints share a desc value but not its meaning")
	}
}

func TestGetGrayReleasePlan(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","gray_release_plan":{`+
		`"status":1,"create_timestamp":1517553721,"gray_percentage":8,`+
		`"support_experiencer_first":true,"support_debuger_first":true}}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetGrayReleasePlan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || !resp.HasPlan() {
		t.Fatalf("resp = %+v", resp)
	}

	plan := resp.GrayReleasePlan
	if !plan.Running() || plan.GrayPercentage != 8 {
		t.Fatalf("plan = %+v", plan)
	}

	// The bool fields arrive as booleans, not as 0/1.
	if !plan.SupportDebuggerFirst || !plan.SupportExperiencerFirst {
		t.Fatalf("plan = %+v", plan)
	}

	if GrayReleaseStatusMsg(plan.Status) != "执行中" {
		t.Fatalf("status msg = %q", GrayReleaseStatusMsg(plan.Status))
	}

	if call.method != "GET" || call.path != getGrayReleasePlanPath {
		t.Fatalf("%s %s", call.method, call.path)
	}
}

func TestGetGrayReleasePlanAbsent(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetGrayReleasePlan(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// No plan is an empty reply, not an error -- and not a plan of zeros.
	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if resp.HasPlan() || resp.Running() {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.GrayReleasePlan != nil {
		t.Fatalf("plan = %+v", resp.GrayReleasePlan)
	}
}

func TestOperationErrMsg(t *testing.T) {
	if got := OperationErrMsg(46001); !strings.Contains(got, "media_id") {
		t.Fatalf("errmsg = %q", got)
	}
	if got := OperationErrMsg(999999); got != OperationErrDefaultMsg {
		t.Fatalf("errmsg = %q", got)
	}
}

func TestFlexString(t *testing.T) {
	var values struct {
		Quoted FlexString `json:"quoted"`
		Number FlexString `json:"number"`
		Null   FlexString `json:"null"`
	}

	if err := codec.UnmarshalString(`{"quoted":"abc","number":18800000000,"null":null}`, &values); err != nil {
		t.Fatal(err)
	}

	if values.Quoted.String() != "abc" {
		t.Fatalf("quoted = %q", values.Quoted)
	}

	// The digits are kept exactly: a float round-trip is how a phone number
	// loses its tail.
	if values.Number.String() != "18800000000" {
		t.Fatalf("number = %q", values.Number)
	}

	if !values.Null.IsEmpty() {
		t.Fatalf("null = %q", values.Null)
	}

	var rejected FlexString
	if err := codec.UnmarshalString(`{"a":1}`, &rejected); err == nil {
		t.Fatal("an object decoded into FlexString")
	}
}
