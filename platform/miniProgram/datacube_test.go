package miniProgram

import (
	"context"
	"strings"
	"testing"
)

func TestDatacubeDayRangeValidation(t *testing.T) {
	valid := DatacubeDateRange{BeginDate: "20170313", EndDate: "20170313"}
	if err := CheckDatacubeDayRange(valid); err != nil {
		t.Fatalf("valid range rejected: %v", err)
	}

	cases := []struct {
		name string
		r    DatacubeDateRange
	}{
		{name: "two days", r: DatacubeDateRange{BeginDate: "20170313", EndDate: "20170314"}},
		{name: "empty begin", r: DatacubeDateRange{EndDate: "20170313"}},
		{name: "empty end", r: DatacubeDateRange{BeginDate: "20170313"}},
		{name: "dashed format", r: DatacubeDateRange{BeginDate: "2017-03-13", EndDate: "2017-03-13"}},
		// 2017-02-30 does not exist; a wrong date would otherwise come back as an
		// empty list and read as "no data".
		{name: "impossible date", r: DatacubeDateRange{BeginDate: "20170230", EndDate: "20170230"}},
		{name: "begin after end", r: DatacubeDateRange{BeginDate: "20170314", EndDate: "20170313"}},
	}

	for _, tc := range cases {
		if err := CheckDatacubeDayRange(tc.r); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestDatacubeWeekRangeValidation(t *testing.T) {
	valid := DatacubeDateRange{BeginDate: "20170306", EndDate: "20170312"}
	if err := CheckDatacubeWeekRange(valid); err != nil {
		t.Fatalf("valid week rejected: %v", err)
	}

	cases := []struct {
		name string
		r    DatacubeDateRange
	}{
		{name: "starts sunday", r: DatacubeDateRange{BeginDate: "20170305", EndDate: "20170311"}},
		{name: "six days", r: DatacubeDateRange{BeginDate: "20170306", EndDate: "20170311"}},
		{name: "eight days", r: DatacubeDateRange{BeginDate: "20170306", EndDate: "20170313"}},
		{name: "one day", r: DatacubeDateRange{BeginDate: "20170306", EndDate: "20170306"}},
	}

	for _, tc := range cases {
		if err := CheckDatacubeWeekRange(tc.r); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestDatacubeMonthRangeValidation(t *testing.T) {
	cases := []struct {
		name string
		r    DatacubeDateRange
	}{
		{name: "february", r: DatacubeDateRange{BeginDate: "20170201", EndDate: "20170228"}},
		{name: "leap february", r: DatacubeDateRange{BeginDate: "20160201", EndDate: "20160229"}},
		{name: "30-day month", r: DatacubeDateRange{BeginDate: "20170401", EndDate: "20170430"}},
		{name: "31-day month", r: DatacubeDateRange{BeginDate: "20170301", EndDate: "20170331"}},
	}

	for _, tc := range cases {
		if err := CheckDatacubeMonthRange(tc.r); err != nil {
			t.Fatalf("%s: valid month rejected: %v", tc.name, err)
		}
	}

	invalid := []struct {
		name string
		r    DatacubeDateRange
	}{
		{name: "starts mid month", r: DatacubeDateRange{BeginDate: "20170302", EndDate: "20170331"}},
		{name: "leap year with 28 days", r: DatacubeDateRange{BeginDate: "20160201", EndDate: "20160228"}},
		{name: "two months", r: DatacubeDateRange{BeginDate: "20170301", EndDate: "20170430"}},
		{name: "ends early", r: DatacubeDateRange{BeginDate: "20170301", EndDate: "20170330"}},
	}

	for _, tc := range invalid {
		if err := CheckDatacubeMonthRange(tc.r); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestDatacubePortraitRangeValidation(t *testing.T) {
	valid := []DatacubeDateRange{
		{BeginDate: "20170611", EndDate: "20170611"},
		{BeginDate: "20170611", EndDate: "20170617"},
		{BeginDate: "20170611", EndDate: "20170710"},
	}

	for _, r := range valid {
		if err := CheckDatacubePortraitRange(r); err != nil {
			t.Fatalf("%v rejected: %v", r, err)
		}
	}

	invalid := []DatacubeDateRange{
		{BeginDate: "20170611", EndDate: "20170612"},
		{BeginDate: "20170611", EndDate: "20170618"},
		{BeginDate: "20170611", EndDate: "20170711"},
	}

	for _, r := range invalid {
		if err := CheckDatacubePortraitRange(r); err == nil {
			t.Fatalf("%v accepted", r)
		}
	}
}

func TestDatacubeVisitTrends(t *testing.T) {
	const reply = `{"list":[{"ref_date":"20170313","session_cnt":142549,"visit_pv":472351,` +
		`"visit_uv":55500,"visit_uv_new":5464,"stay_time_uv":12.5,` +
		`"stay_time_session":0,"visit_depth":1.9838}]}`

	cases := []struct {
		name string
		path string
		call func(app *Application, date DatacubeDateRange) error
		date DatacubeDateRange
	}{
		{
			name: "daily",
			path: getDailyVisitTrendPath,
			date: DatacubeDateRange{BeginDate: "20170313", EndDate: "20170313"},
			call: func(app *Application, date DatacubeDateRange) error {
				_, err := app.GetDailyVisitTrend(context.Background(), date)

				return err
			},
		},
		{
			name: "weekly",
			path: getWeeklyVisitTrendPath,
			date: DatacubeDateRange{BeginDate: "20170306", EndDate: "20170312"},
			call: func(app *Application, date DatacubeDateRange) error {
				_, err := app.GetWeeklyVisitTrend(context.Background(), date)

				return err
			},
		},
		{
			name: "monthly",
			path: getMonthlyVisitTrendPath,
			date: DatacubeDateRange{BeginDate: "20170301", EndDate: "20170331"},
			call: func(app *Application, date DatacubeDateRange) error {
				_, err := app.GetMonthlyVisitTrend(context.Background(), date)

				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, call := newKFServer(t, reply)
			app := newApp(t, srv.URL, nil)

			if err := tc.call(app, tc.date); err != nil {
				t.Fatal(err)
			}

			if call.method != "POST" || call.path != tc.path {
				t.Fatalf("%s %s", call.method, call.path)
			}

			var body struct {
				BeginDate string `json:"begin_date"`
				EndDate   string `json:"end_date"`
			}
			call.decodeBody(t, &body)

			if body.BeginDate != tc.date.BeginDate || body.EndDate != tc.date.EndDate {
				t.Fatalf("body = %+v", body)
			}
		})
	}
}

func TestDatacubeVisitTrendDecodes(t *testing.T) {
	srv, _ := newKFServer(t, `{"list":[{"ref_date":"20170313","session_cnt":142549,`+
		`"visit_pv":472351,"visit_uv":55500,"visit_uv_new":5464,`+
		`"stay_time_uv":12.5,"stay_time_session":0,"visit_depth":1.9838}]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetDailyVisitTrend(context.Background(), DatacubeDateRange{
		BeginDate: "20170313", EndDate: "20170313",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if len(resp.List) != 1 {
		t.Fatalf("list = %+v", resp.List)
	}

	trend := resp.List[0]
	if trend.RefDate != "20170313" || trend.SessionCnt != 142549 || trend.VisitUV != 55500 {
		t.Fatalf("trend = %+v", trend)
	}
	// The averages are floats even when WeChat sends an integer-looking 0.
	if trend.StayTimeUV != 12.5 || trend.StayTimeSession != 0 || trend.VisitDepth != 1.9838 {
		t.Fatalf("trend = %+v", trend)
	}
}

func TestDatacubeRetentions(t *testing.T) {
	const reply = `{"ref_date":"20170306-20170312",` +
		`"visit_uv_new":[{"key":0,"value":0},{"key":1,"value":16853}],` +
		`"visit_uv":[{"key":0,"value":0},{"key":1,"value":99310}]}`

	cases := []struct {
		name string
		path string
		date DatacubeDateRange
		call func(app *Application, date DatacubeDateRange) (*DatacubeRetention, error)
	}{
		{
			name: "daily",
			path: getDailyRetainPath,
			date: DatacubeDateRange{BeginDate: "20170313", EndDate: "20170313"},
			call: func(app *Application, date DatacubeDateRange) (*DatacubeRetention, error) {
				resp, err := app.GetDailyRetain(context.Background(), date)
				if err != nil {
					return nil, err
				}

				return &resp.DatacubeRetention, nil
			},
		},
		{
			name: "weekly",
			path: getWeeklyRetainPath,
			date: DatacubeDateRange{BeginDate: "20170306", EndDate: "20170312"},
			call: func(app *Application, date DatacubeDateRange) (*DatacubeRetention, error) {
				resp, err := app.GetWeeklyRetain(context.Background(), date)
				if err != nil {
					return nil, err
				}

				return &resp.DatacubeRetention, nil
			},
		},
		{
			name: "monthly",
			path: getMonthlyRetainPath,
			date: DatacubeDateRange{BeginDate: "20170301", EndDate: "20170331"},
			call: func(app *Application, date DatacubeDateRange) (*DatacubeRetention, error) {
				resp, err := app.GetMonthlyRetain(context.Background(), date)
				if err != nil {
					return nil, err
				}

				return &resp.DatacubeRetention, nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, call := newKFServer(t, reply)
			app := newApp(t, srv.URL, nil)

			retention, err := tc.call(app, tc.date)
			if err != nil {
				t.Fatal(err)
			}

			if call.path != tc.path {
				t.Fatalf("path = %q", call.path)
			}
			if retention.RefDate != "20170306-20170312" {
				t.Fatalf("ref_date = %q", retention.RefDate)
			}
			if len(retention.VisitUVNew) != 2 || retention.VisitUVNew[1].Value != 16853 {
				t.Fatalf("visit_uv_new = %+v", retention.VisitUVNew)
			}
			if len(retention.VisitUV) != 2 || retention.VisitUV[1].Key != 1 {
				t.Fatalf("visit_uv = %+v", retention.VisitUV)
			}
		})
	}
}

func TestGetDailySummary(t *testing.T) {
	srv, call := newKFServer(t, `{"list":[{"ref_date":"20170313","visit_total":391,`+
		`"share_pv":572,"share_uv":383}]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetDailySummary(context.Background(), DatacubeDateRange{
		BeginDate: "20170313", EndDate: "20170313",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if len(resp.List) != 1 || resp.List[0].VisitTotal != 391 || resp.List[0].ShareUV != 383 {
		t.Fatalf("list = %+v", resp.List)
	}

	if call.path != getDailySummaryPath {
		t.Fatalf("path = %q", call.path)
	}
}

func TestGetVisitPage(t *testing.T) {
	srv, call := newKFServer(t, `{"ref_date":"20170313","list":[{"page_path":"pages/main/main.html",`+
		`"page_visit_pv":213429,"page_visit_uv":55423,"page_staytime_pv":8.139198,`+
		`"entrypage_pv":117922,"exitpage_pv":61304,"page_share_pv":180,"page_share_uv":166}]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetVisitPage(context.Background(), DatacubeDateRange{
		BeginDate: "20170313", EndDate: "20170313",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || resp.RefDate != "20170313" {
		t.Fatalf("resp = %+v", resp)
	}
	if len(resp.List) != 1 {
		t.Fatalf("list = %+v", resp.List)
	}

	page := resp.List[0]
	if page.PagePath != "pages/main/main.html" || page.PageVisitPV != 213429 {
		t.Fatalf("page = %+v", page)
	}
	if page.PageStayTimePV != 8.139198 || page.EntryPagePV != 117922 {
		t.Fatalf("page = %+v", page)
	}
	if call.path != getVisitPagePath {
		t.Fatalf("path = %q", call.path)
	}
}

func TestGetVisitDistribution(t *testing.T) {
	srv, call := newKFServer(t, `{"ref_date":"20170313","list":[`+
		`{"index":"access_source_session_cnt","item_list":[{"key":3,"value":687},{"key":10,"value":5}]},`+
		`{"index":"access_staytime_info","item_list":[{"key":1,"value":100}]}]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetVisitDistribution(context.Background(), DatacubeDateRange{
		BeginDate: "20170313", EndDate: "20170313",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}

	sources := resp.Items(VisitDistributionIndexSource)
	if len(sources) != 2 || sources[0].Key != 3 {
		t.Fatalf("sources = %+v", sources)
	}

	// The key space is per index: 3 is 会话 as a source, 1 is 0-2s as a stay time
	// and 1 页 as a depth.
	if got := DistributionBucketMsg(VisitDistributionIndexSource, 3); got != "会话" {
		t.Fatalf("source 3 = %q", got)
	}
	if got := DistributionBucketMsg(VisitDistributionIndexDepth, 1); got != "1 页" {
		t.Fatalf("depth 1 = %q", got)
	}
	if got := DistributionBucketMsg(VisitDistributionIndexStayTime, 1); got != "0-2s" {
		t.Fatalf("stay time 1 = %q", got)
	}
	if got := DistributionBucketMsg(VisitDistributionIndexDepth, 7); got != ">10 页" {
		t.Fatalf("depth 7 = %q", got)
	}
	if got := DistributionBucketMsg("unknown", 1); got != "" {
		t.Fatalf("unknown index = %q", got)
	}

	// A distribution that was not returned comes back as nil, not as an error.
	if items := resp.Items(VisitDistributionIndexDepth); items != nil {
		t.Fatalf("depth items = %+v", items)
	}

	if call.path != getVisitDistributionPath {
		t.Fatalf("path = %q", call.path)
	}
}

func TestGetUserPortrait(t *testing.T) {
	srv, call := newKFServer(t, `{"ref_date":"20170611-20170617",`+
		`"visit_uv_new":{"province":[{"id":31,"name":"广东省","value":215}],`+
		`"devices":[{"name":"OPPO R9","value":61}]},`+
		`"visit_uv":{"genders":[{"id":1,"name":"男","value":14534}]}}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetUserPortrait(context.Background(), DatacubeDateRange{
		BeginDate: "20170611", EndDate: "20170617",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || resp.RefDate != "20170611-20170617" {
		t.Fatalf("resp = %+v", resp)
	}

	if resp.VisitUVNew == nil || len(resp.VisitUVNew.Province) != 1 {
		t.Fatalf("visit_uv_new = %+v", resp.VisitUVNew)
	}
	if resp.VisitUVNew.Province[0].Name != "广东省" || resp.VisitUVNew.Province[0].Value != 215 {
		t.Fatalf("province = %+v", resp.VisitUVNew.Province)
	}
	// devices carry no id.
	if len(resp.VisitUVNew.Devices) != 1 || resp.VisitUVNew.Devices[0].ID != 0 {
		t.Fatalf("devices = %+v", resp.VisitUVNew.Devices)
	}
	if resp.VisitUV == nil || len(resp.VisitUV.Genders) != 1 {
		t.Fatalf("visit_uv = %+v", resp.VisitUV)
	}

	if call.path != getUserPortraitPath {
		t.Fatalf("path = %q", call.path)
	}
}

func TestGetPerformanceData(t *testing.T) {
	reply := `{"errcode":0,"errmsg":"ok","data":{"body":{"tables":[` +
		`{"id":"memorydiff","zh":"内存增长均值","lines":[{"fields":[` +
		`{"refdate":"20210103","value":"70.7778"},{"refdate":"20210104","value":"72.0446"}]}]}],` +
		`"count":1}}}`

	srv, call := newKFServer(t, reply)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetPerformanceData(context.Background(), GetPerformanceDataParams{
		Module:         PerfModuleMemory,
		BeginTimestamp: 1609603200,
		EndTimestamp:   1609689600,
		Filters: []GetPerformanceDataFilter{
			{Field: PerfFieldNetworkType, Value: "wifi"},
			{Field: PerfFieldDeviceLevel, Value: "1"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}

	tables := resp.Data.Body.Tables
	if len(tables) != 1 || tables[0].ID != "memorydiff" || tables[0].Zh != "内存增长均值" {
		t.Fatalf("tables = %+v", tables)
	}
	if resp.Data.Body.Count != 1 {
		t.Fatalf("count = %d", resp.Data.Body.Count)
	}

	fields := tables[0].Lines[0].Fields
	if len(fields) != 2 || fields[0].RefDate != "20210103" {
		t.Fatalf("fields = %+v", fields)
	}

	value, err := fields[0].Float()
	if err != nil {
		t.Fatal(err)
	}
	if value != 70.7778 {
		t.Fatalf("float = %v", value)
	}

	if call.path != getPerformanceDataPath {
		t.Fatalf("path = %q", call.path)
	}

	var body struct {
		Module string `json:"module"`
		Time   struct {
			BeginTimestamp int64 `json:"begin_timestamp"`
			EndTimestamp   int64 `json:"end_timestamp"`
		} `json:"time"`
		Params []struct {
			Field string `json:"field"`
			Value string `json:"value"`
		} `json:"params"`
	}
	call.decodeBody(t, &body)

	// module travels as a string, as in WeChat's own examples.
	if body.Module != "10022" {
		t.Fatalf("module = %v", body.Module)
	}
	if body.Time.BeginTimestamp != 1609603200 || body.Time.EndTimestamp != 1609689600 {
		t.Fatalf("time = %+v", body.Time)
	}
	if len(body.Params) != 2 || body.Params[0].Field != "networktype" ||
		body.Params[0].Value != "wifi" {
		t.Fatalf("params = %+v", body.Params)
	}
}

func TestGetPerformanceDataValidation(t *testing.T) {
	valid := GetPerformanceDataParams{
		Module:         PerfModuleOpenRate,
		BeginTimestamp: 1609603200,
		EndTimestamp:   1609689600,
		Filters:        []GetPerformanceDataFilter{{Field: PerfFieldDevice, Value: "-1"}},
	}

	if err := CheckGetPerformanceDataParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	// Exactly 30 days is still allowed; one second more is not.
	boundary := valid
	boundary.EndTimestamp = boundary.BeginTimestamp + PerfMaxRangeSeconds
	if err := CheckGetPerformanceDataParams(boundary); err != nil {
		t.Fatalf("30 days rejected: %v", err)
	}

	boundary.EndTimestamp++
	if err := CheckGetPerformanceDataParams(boundary); err == nil {
		t.Fatal("30 days plus one second accepted")
	}

	cases := []struct {
		name   string
		mutate func(*GetPerformanceDataParams)
	}{
		{name: "no module", mutate: func(p *GetPerformanceDataParams) { p.Module = "" }},
		{name: "no begin", mutate: func(p *GetPerformanceDataParams) { p.BeginTimestamp = 0 }},
		{name: "no end", mutate: func(p *GetPerformanceDataParams) { p.EndTimestamp = 0 }},
		{name: "reversed", mutate: func(p *GetPerformanceDataParams) {
			p.BeginTimestamp, p.EndTimestamp = p.EndTimestamp, p.BeginTimestamp
		}},
		{name: "no filters", mutate: func(p *GetPerformanceDataParams) { p.Filters = nil }},
		{name: "empty field", mutate: func(p *GetPerformanceDataParams) {
			p.Filters = []GetPerformanceDataFilter{{Value: "wifi"}}
		}},
		{name: "empty value", mutate: func(p *GetPerformanceDataParams) {
			p.Filters = []GetPerformanceDataFilter{{Field: PerfFieldNetworkType}}
		}},
	}

	for _, tc := range cases {
		params := valid
		tc.mutate(&params)

		if err := CheckGetPerformanceDataParams(params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestPerformanceFieldFloatInvalid(t *testing.T) {
	if _, err := (PerformanceField{Value: "n/a"}).Float(); err == nil {
		t.Fatal("non-numeric value accepted")
	}

	value, err := (PerformanceField{Value: " 302.3218 "}).Float()
	if err != nil {
		t.Fatal(err)
	}
	if value != 302.3218 {
		t.Fatalf("float = %v", value)
	}
}

func TestDatacubeEmptyList(t *testing.T) {
	// No data for the day is a successful call, not a failure: a caller that
	// treats it as an error would retry forever.
	srv, _ := newKFServer(t, `{"list":[]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetDailyVisitTrend(context.Background(), DatacubeDateRange{
		BeginDate: "20170313", EndDate: "20170313",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false for an empty list, resp = %+v", resp)
	}
	if len(resp.List) != 0 {
		t.Fatalf("list = %+v", resp.List)
	}
}

func TestDatacubeErrMsg(t *testing.T) {
	if got := DatacubeErrMsg(-1); got != "系统繁忙，此时请开发者稍候再试" {
		t.Fatalf("errmsg = %q", got)
	}
	if got := DatacubeErrMsg(999999); got != DatacubeDefaultErrMsg {
		t.Fatalf("errmsg = %q", got)
	}
	if got := PerfModuleMsg(PerfModuleMemoryAbnormal); got != "内存异常" {
		t.Fatalf("module msg = %q", got)
	}
	if got := VisitDistributionIndexMsg(VisitDistributionIndexStayTime); !strings.Contains(got, "时长") {
		t.Fatalf("index msg = %q", got)
	}
}

func TestDatacubeErrorWrapping(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":40001,"errmsg":"invalid credential"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetDailySummary(context.Background(), DatacubeDateRange{
		BeginDate: "20170313", EndDate: "20170313",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 40001")
	}
	if err := GetDailySummaryError(resp); err == nil || !strings.Contains(err.Error(), "access_token") {
		t.Fatalf("err = %v", err)
	}
}
