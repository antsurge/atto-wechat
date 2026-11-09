package miniProgram

import (
	"context"
	"strings"
	"testing"
)

// validHardwareDevice is the device every test starts from.
var validHardwareDevice = HardwareDevice{ModelID: "MODEL_ID1", SN: "SN1"}

func TestSendHardwareDeviceMessage(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.SendHardwareDeviceMessage(context.Background(), SendHardwareDeviceMessageParams{
		TemplateID:   "TPL",
		ModelID:      "MODEL_ID1",
		SN:           "SN1",
		ToOpenIDList: []string{"OPENID1", "OPENID2"},
		Data:         map[string]HardwareDeviceMsgValue{"character_string1": {Value: "xxx"}},
		Page:         "index?foo=bar",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}

	if call.method != "POST" || call.path != sendHardwareDeviceMessagePath {
		t.Fatalf("%s %s", call.method, call.path)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	// modelId is camelCase on the wire; the rest is snake_case.
	if body["modelId"] != "MODEL_ID1" {
		t.Fatalf("modelId = %v", body["modelId"])
	}
	if body["template_id"] != "TPL" || body["sn"] != "SN1" || body["page"] != "index?foo=bar" {
		t.Fatalf("body = %v", body)
	}

	// data is an object of {value: ...}, not a JSON-encoded string -- the opposite
	// of the employee-relation message.
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("data is not an object: %T", body["data"])
	}

	field, ok := data["character_string1"].(map[string]any)
	if !ok || field["value"] != "xxx" {
		t.Fatalf("data = %v", data)
	}

	openIDs, ok := body["to_openid_list"].([]any)
	if !ok || len(openIDs) != 2 || openIDs[1] != "OPENID2" {
		t.Fatalf("to_openid_list = %v", body["to_openid_list"])
	}

	// Optional fields stay off the wire when empty, so WeChat applies its defaults.
	if _, present := body["miniprogram_state"]; present {
		t.Fatalf("miniprogram_state sent while empty: %v", body["miniprogram_state"])
	}
	if _, present := body["lang"]; present {
		t.Fatalf("lang sent while empty: %v", body["lang"])
	}
}

func TestSendHardwareDeviceMessageValidation(t *testing.T) {
	valid := SendHardwareDeviceMessageParams{
		TemplateID:   "TPL",
		ModelID:      "MODEL_ID1",
		SN:           "SN1",
		ToOpenIDList: []string{"OPENID1"},
		Data:         map[string]HardwareDeviceMsgValue{"k": {Value: "v"}},
	}

	if err := CheckSendHardwareDeviceMessageParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*SendHardwareDeviceMessageParams)
	}{
		{name: "no template", mutate: func(p *SendHardwareDeviceMessageParams) { p.TemplateID = "" }},
		{name: "no model", mutate: func(p *SendHardwareDeviceMessageParams) { p.ModelID = "" }},
		{name: "no sn", mutate: func(p *SendHardwareDeviceMessageParams) { p.SN = "" }},
		{name: "no openid", mutate: func(p *SendHardwareDeviceMessageParams) { p.ToOpenIDList = nil }},
		{name: "no data", mutate: func(p *SendHardwareDeviceMessageParams) { p.Data = nil }},
		{name: "bad state", mutate: func(p *SendHardwareDeviceMessageParams) { p.MiniProgramState = "prod" }},
		{name: "bad lang", mutate: func(p *SendHardwareDeviceMessageParams) { p.Lang = "jp" }},
		{name: "url page", mutate: func(p *SendHardwareDeviceMessageParams) { p.Page = "https://example.com/a" }},
	}

	for _, tc := range cases {
		params := valid
		tc.mutate(&params)

		if err := CheckSendHardwareDeviceMessageParams(params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}
}

func TestHardwareDeviceValidation(t *testing.T) {
	if err := CheckHardwareDevice(validHardwareDevice); err != nil {
		t.Fatalf("valid device rejected: %v", err)
	}

	cases := []struct {
		name   string
		device HardwareDevice
	}{
		{name: "no model", device: HardwareDevice{SN: "SN1"}},
		{name: "no sn", device: HardwareDevice{ModelID: "MODEL_ID1"}},
		{name: "sn with space", device: HardwareDevice{ModelID: "M", SN: "SN 1"}},
		{name: "sn with slash", device: HardwareDevice{ModelID: "M", SN: "SN/1"}},
		{name: "sn too long", device: HardwareDevice{ModelID: "M", SN: strings.Repeat("a", 129)}},
	}

	for _, tc := range cases {
		if err := CheckHardwareDevice(tc.device); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}

	// exactly 128 bytes is still fine
	boundary := HardwareDevice{ModelID: "M", SN: strings.Repeat("a", 128)}
	if err := CheckHardwareDevice(boundary); err != nil {
		t.Fatalf("128-byte sn rejected: %v", err)
	}

	// hyphen and underscore are the two allowed punctuation characters
	if err := CheckHardwareDevice(HardwareDevice{ModelID: "M", SN: "SN_1-A"}); err != nil {
		t.Fatalf("underscore/hyphen rejected: %v", err)
	}
}

func TestGetSnTicket(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","sn_ticket":"TICKET"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetSnTicket(context.Background(), GetSnTicketParams{
		ModelID: "MODEL_ID1",
		SN:      "SN1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || resp.SnTicket != "TICKET" {
		t.Fatalf("resp = %+v", resp)
	}

	if call.path != getSnTicketPath {
		t.Fatalf("path = %q", call.path)
	}

	var body map[string]string
	call.decodeBody(t, &body)

	if body["sn"] != "SN1" || body["model_id"] != "MODEL_ID1" {
		t.Fatalf("body = %v", body)
	}
}

func TestGetSnTicketWithoutTicket(t *testing.T) {
	// errcode 0 without a ticket is not usable: the ticket is the payload.
	srv, _ := newKFServer(t, `{"errcode":0,"errmsg":"ok"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetSnTicket(context.Background(), GetSnTicketParams{
		ModelID: "MODEL_ID1",
		SN:      "SN1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatalf("OK() = true without a ticket, resp = %+v", resp)
	}
}

func TestCreateIotGroupId(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","group_id":"GROUP_ID"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.CreateIotGroupId(context.Background(), CreateIotGroupIdParams{
		ModelID:   "MODEL_ID1",
		GroupName: "一号门禁组",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || resp.GroupID != "GROUP_ID" {
		t.Fatalf("resp = %+v", resp)
	}

	if call.path != createIotGroupIdPath {
		t.Fatalf("path = %q", call.path)
	}

	var body map[string]string
	call.decodeBody(t, &body)

	if body["model_id"] != "MODEL_ID1" || body["group_name"] != "一号门禁组" {
		t.Fatalf("body = %v", body)
	}
}

func TestCreateIotGroupIdValidation(t *testing.T) {
	cases := []struct {
		name   string
		params CreateIotGroupIdParams
	}{
		{name: "no model", params: CreateIotGroupIdParams{GroupName: "g"}},
		{name: "no name", params: CreateIotGroupIdParams{ModelID: "M"}},
		// 32 bytes, not 32 characters: 11 Chinese characters are already over.
		{name: "name too long", params: CreateIotGroupIdParams{ModelID: "M", GroupName: strings.Repeat("组", 11)}},
	}

	for _, tc := range cases {
		if err := CheckCreateIotGroupIdParams(tc.params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}

	// exactly 32 bytes passes
	if err := CheckCreateIotGroupIdParams(CreateIotGroupIdParams{
		ModelID: "M", GroupName: strings.Repeat("a", 32),
	}); err != nil {
		t.Fatalf("32-byte name rejected: %v", err)
	}
}

func TestGetIotGroupInfo(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","group_name":"GROUP_NAME",`+
		`"model_id":"MODEL_ID","model_type":"门禁机","device_list":[`+
		`{"model_id":"MODEL_ID1","sn":"SN1"},{"model_id":"MODEL_ID2","sn":"SN2"}]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetIotGroupInfo(context.Background(), "GROUP_ID")
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}

	if resp.GroupName != "GROUP_NAME" || resp.ModelType != "门禁机" {
		t.Fatalf("resp = %+v", resp)
	}
	if len(resp.DeviceList) != 2 || resp.DeviceList[1].SN != "SN2" {
		t.Fatalf("device_list = %+v", resp.DeviceList)
	}

	if call.path != getIotGroupInfoPath {
		t.Fatalf("path = %q", call.path)
	}

	var body map[string]string
	call.decodeBody(t, &body)

	if body["group_id"] != "GROUP_ID" {
		t.Fatalf("body = %v", body)
	}
}

func TestAddIotGroupDevice(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","device_list":[`+
		`{"model_id":"MODEL_ID1","sn":"SN1","errcode":0},`+
		`{"model_id":"MODEL_ID2","sn":"SN2","errcode":9800018}]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.AddIotGroupDevice(context.Background(), AddIotGroupDeviceParams{
		GroupID:    "GROUP_ID",
		DeviceList: []HardwareDevice{{ModelID: "MODEL_ID1", SN: "SN1"}, {ModelID: "MODEL_ID2", SN: "SN2"}},
		ForceAdd:   true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// errcode 0 with one device refused: the call succeeded, the batch did not.
	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if resp.AllSucceeded() {
		t.Fatal("AllSucceeded() = true while a device failed")
	}

	failed := resp.FailedDevices()
	if len(failed) != 1 || failed[0].SN != "SN2" || failed[0].ErrCode != 9800018 {
		t.Fatalf("failed = %+v", failed)
	}
	if failed[0].OK() {
		t.Fatal("failed device reports OK")
	}

	var body struct {
		GroupID    string           `json:"group_id"`
		DeviceList []HardwareDevice `json:"device_list"`
		ForceAdd   bool             `json:"force_add"`
	}
	call.decodeBody(t, &body)

	if body.GroupID != "GROUP_ID" || len(body.DeviceList) != 2 || !body.ForceAdd {
		t.Fatalf("body = %+v", body)
	}

	if call.path != addIotGroupDevicePath {
		t.Fatalf("path = %q", call.path)
	}
}

func TestRemoveIotGroupDevice(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","device_list":[`+
		`{"model_id":"MODEL_ID1","sn":"SN1","errcode":0}]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.RemoveIotGroupDevice(context.Background(), RemoveIotGroupDeviceParams{
		GroupID:    "GROUP_ID",
		DeviceList: []HardwareDevice{validHardwareDevice},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || !resp.AllSucceeded() {
		t.Fatalf("resp = %+v", resp)
	}
	if failed := resp.FailedDevices(); len(failed) != 0 {
		t.Fatalf("failed = %+v", failed)
	}

	if call.path != removeIotGroupDevicePath {
		t.Fatalf("path = %q", call.path)
	}

	var body map[string]any
	call.decodeBody(t, &body)

	// force_add belongs to the add endpoint only.
	if _, present := body["force_add"]; present {
		t.Fatal("force_add sent to the remove endpoint")
	}
}

func TestGetLicensePkgList(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"success","pkg_list":[`+
		`{"pkg_id":"ZY1","pkg_type":1,"start_time":1629907200,"end_time":1630425600,`+
		`"pkg_status":1,"used":10,"all":100},`+
		`{"pkg_id":"ZY2","pkg_type":1,"start_time":1629907200,"end_time":1630425600,`+
		`"pkg_status":3,"used":5,"all":5}],"max_active_number":300}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetLicensePkgList(context.Background(), GetLicensePkgListParams{
		PkgType: LicensePkgTypeA,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() {
		t.Fatalf("OK() = false, resp = %+v", resp)
	}
	if len(resp.PkgList) != 2 {
		t.Fatalf("pkg_list = %+v", resp.PkgList)
	}

	if remaining := resp.PkgList[0].Remaining(); remaining != 90 {
		t.Fatalf("remaining = %d", remaining)
	}
	if !resp.PkgList[0].Usable() {
		t.Fatal("active package with a balance is not usable")
	}

	// expired with nothing left: neither status nor balance allows activating
	if resp.PkgList[1].Usable() {
		t.Fatal("expired package reports usable")
	}

	// only the usable package counts towards the total
	if total := resp.TotalRemaining(); total != 90 {
		t.Fatalf("total remaining = %d", total)
	}

	if call.path != getLicensePkgListPath {
		t.Fatalf("path = %q", call.path)
	}

	var body map[string]int
	call.decodeBody(t, &body)

	if body["pkg_type"] != LicensePkgTypeA {
		t.Fatalf("body = %v", body)
	}
}

func TestGetLicensePkgListValidation(t *testing.T) {
	if err := CheckGetLicensePkgListParams(GetLicensePkgListParams{PkgType: LicensePkgTypeE}); err != nil {
		t.Fatalf("valid type rejected: %v", err)
	}

	for _, pkgType := range []int{-1, 6} {
		if err := CheckGetLicensePkgListParams(GetLicensePkgListParams{PkgType: pkgType}); err == nil {
			t.Fatalf("pkg_type %d accepted", pkgType)
		}
	}
}

func TestActiveLicenseDevice(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","device_list":[`+
		`{"model_id":"MODEL_ID1","sn":"SN1","errcode":0,"expire_time":1630425600}]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.ActiveLicenseDevice(context.Background(), ActiveLicenseDeviceParams{
		PkgType: LicensePkgTypeTest,
		DeviceList: []LicenseActivation{
			{ModelID: "MODEL_ID1", SN: "SN1", ActiveNumber: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || !resp.AllSucceeded() {
		t.Fatalf("resp = %+v", resp)
	}
	if resp.DeviceList[0].ExpireTime != 1630425600 {
		t.Fatalf("device_list = %+v", resp.DeviceList)
	}

	var body struct {
		PkgType    int                   `json:"pkg_type"`
		DeviceList []activeLicenseDevice `json:"device_list"`
	}
	call.decodeBody(t, &body)

	if body.PkgType != LicensePkgTypeTest || len(body.DeviceList) != 1 {
		t.Fatalf("body = %+v", body)
	}
	if body.DeviceList[0].ActiveNumber != 1 {
		t.Fatalf("active_number = %d", body.DeviceList[0].ActiveNumber)
	}

	if call.path != activeLicenseDevicePath {
		t.Fatalf("path = %q", call.path)
	}
}

func TestActiveLicenseDeviceValidation(t *testing.T) {
	valid := ActiveLicenseDeviceParams{
		PkgType:    LicensePkgTypeTest,
		DeviceList: []LicenseActivation{{ModelID: "M", SN: "SN1", ActiveNumber: 1}},
	}

	if err := CheckActiveLicenseDeviceParams(valid); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}

	cases := []struct {
		name   string
		params ActiveLicenseDeviceParams
	}{
		{name: "bad pkg type", params: ActiveLicenseDeviceParams{PkgType: 6,
			DeviceList: valid.DeviceList}},
		{name: "no devices", params: ActiveLicenseDeviceParams{PkgType: 0}},
		{name: "bad sn", params: ActiveLicenseDeviceParams{PkgType: 0,
			DeviceList: []LicenseActivation{{ModelID: "M", SN: "SN 1", ActiveNumber: 1}}}},
	}

	for _, tc := range cases {
		if err := CheckActiveLicenseDeviceParams(tc.params); err == nil {
			t.Fatalf("%s: expected a validation error", tc.name)
		}
	}

	// The same active_number twice means the caller reused a constant: WeChat
	// rejects the second one with 9800037.
	duplicated := ActiveLicenseDeviceParams{
		PkgType: LicensePkgTypeTest,
		DeviceList: []LicenseActivation{
			{ModelID: "M", SN: "SN1", ActiveNumber: 7},
			{ModelID: "M", SN: "SN2", ActiveNumber: 7},
		},
	}

	err := CheckActiveLicenseDeviceParams(duplicated)
	if err == nil || !strings.Contains(err.Error(), "used twice") {
		t.Fatalf("err = %v", err)
	}
}

func TestGetLicenseDeviceInfo(t *testing.T) {
	srv, call := newKFServer(t, `{"errcode":0,"errmsg":"ok","device_list":[`+
		`{"model_id":"MODEL_ID1","sn":"SN1","expire_time":1630425600},`+
		`{"model_id":"MODEL_ID2","sn":"SN2"}]}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetLicenseDeviceInfo(context.Background(), []HardwareDevice{
		{ModelID: "MODEL_ID1", SN: "SN1"},
		{ModelID: "MODEL_ID2", SN: "SN2"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !resp.OK() || len(resp.DeviceList) != 2 {
		t.Fatalf("resp = %+v", resp)
	}
	if !resp.DeviceList[0].Active() || resp.DeviceList[0].ExpireTime != 1630425600 {
		t.Fatalf("device = %+v", resp.DeviceList[0])
	}

	// No expiry means never activated, not activated in 1970.
	if resp.DeviceList[1].Active() {
		t.Fatalf("unlicensed device reports active: %+v", resp.DeviceList[1])
	}

	if call.path != getLicenseDeviceInfoPath {
		t.Fatalf("path = %q", call.path)
	}

	var body struct {
		DeviceList []HardwareDevice `json:"device_list"`
	}
	call.decodeBody(t, &body)

	if len(body.DeviceList) != 2 || body.DeviceList[1].SN != "SN2" {
		t.Fatalf("body = %+v", body)
	}
}

func TestHardwareDeviceErrMsg(t *testing.T) {
	if got := HardwareDeviceErrMsg(9800013); got != "group_id 不合法" {
		t.Fatalf("errmsg = %q", got)
	}
	if got := HardwareDeviceErrMsg(999999); got != HardwareDeviceErrDefaultMsg {
		t.Fatalf("errmsg = %q", got)
	}
}

func TestHardwareDeviceErrorWrapping(t *testing.T) {
	srv, _ := newKFServer(t, `{"errcode":9800013,"errmsg":"invalid group_id"}`)
	app := newApp(t, srv.URL, nil)

	resp, err := app.GetIotGroupInfo(context.Background(), "GROUP_ID")
	if err != nil {
		t.Fatal(err)
	}

	if resp.OK() {
		t.Fatal("OK() = true for errcode 9800013")
	}
	if err := GetIotGroupInfoError(resp); err == nil || !strings.Contains(err.Error(), "group_id") {
		t.Fatalf("err = %v", err)
	}
}
