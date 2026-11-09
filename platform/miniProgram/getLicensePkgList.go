package miniProgram

import (
	"context"

	cerrors "github.com/antsurge/atto-wechat/core/errors"
	"github.com/antsurge/atto-wechat/core/response"
)

// getLicensePkgListPath lists the licence packages of one device class.
const getLicensePkgListPath = "/wxa/business/license/getpkglist"

// Values of pkg_type: the device class a licence package covers.
const (
	// LicensePkgTypeTest is the trial package, 0.
	LicensePkgTypeTest = 0
	// LicensePkgTypeA is class A, 1.
	LicensePkgTypeA = 1
	// LicensePkgTypeB is class B, 2.
	LicensePkgTypeB = 2
	// LicensePkgTypeC is class C, 3.
	LicensePkgTypeC = 3
	// LicensePkgTypeD is class D, 4.
	LicensePkgTypeD = 4
	// LicensePkgTypeE is class E, 5.
	LicensePkgTypeE = 5
)

// Values of pkg_status.
const (
	// LicensePkgStatusActive means the package is in effect.
	LicensePkgStatusActive = 1
	// LicensePkgStatusInactive means it is not in effect yet.
	LicensePkgStatusInactive = 2
	// LicensePkgStatusExpired means it has expired.
	LicensePkgStatusExpired = 3
)

// LicensePkg is one licence package.
type LicensePkg struct {
	// ID is the package id, e.g. "ZY100000000".
	ID string `json:"pkg_id"`
	// Type is the device class, one of the LicensePkgType* values.
	Type int `json:"pkg_type"`
	// StartTime is when the package took effect, Unix seconds.
	StartTime int64 `json:"start_time"`
	// EndTime is when it expires, Unix seconds.
	EndTime int64 `json:"end_time"`
	// Status is one of the LicensePkgStatus* values.
	Status int `json:"pkg_status"`
	// Used is how many activations were spent.
	Used int `json:"used"`
	// All is how many the package holds.
	All int `json:"all"`
}

// Remaining is how many activations the package still has.
//
// It is derived rather than read: WeChat only reports used and all, and the
// subtraction is the number that decides whether an activation can succeed.
func (p LicensePkg) Remaining() int {
	return p.All - p.Used
}

// Usable reports whether the package can activate a device right now.
func (p LicensePkg) Usable() bool {
	return p.Status == LicensePkgStatusActive && p.Remaining() > 0
}

// GetLicensePkgListParams selects the device class to list.
type GetLicensePkgListParams struct {
	// PkgType is one of the LicensePkgType* values.
	PkgType int
}

// CheckGetLicensePkgListParams validates params before anything is sent.
func CheckGetLicensePkgListParams(params GetLicensePkgListParams) error {
	if params.PkgType < LicensePkgTypeTest || params.PkgType > LicensePkgTypeE {
		return cerrors.New(
			cerrors.CodeInvalidArgument,
			"miniprogram: GetLicensePkgList pkg_type must be 0-5",
		)
	}

	return nil
}

// body renders params as the JSON body the endpoint expects.
func (p GetLicensePkgListParams) body() *getLicensePkgListBody {
	return &getLicensePkgListBody{PkgType: p.PkgType}
}

// getLicensePkgListBody is the JSON body of a package list request.
type getLicensePkgListBody struct {
	PkgType int `json:"pkg_type"`
}

// GetLicensePkgListResponse is the reply of the package list endpoint.
type GetLicensePkgListResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// PkgList is one entry per package of the requested class.
	PkgList []LicensePkg `json:"pkg_list"`
	// MaxActiveNumber is deprecated by WeChat and carries no meaning; do not use
	// it to pick an active_number.
	MaxActiveNumber int `json:"max_active_number"`
}

// OK reports whether the list was read.
func (r *GetLicensePkgListResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// TotalRemaining is how many activations are left across every usable package.
func (r *GetLicensePkgListResponse) TotalRemaining() int {
	if r == nil {
		return 0
	}

	total := 0
	for _, pkg := range r.PkgList {
		if pkg.Usable() {
			total += pkg.Remaining()
		}
	}

	return total
}

// GetLicensePkgListError turns the errcode from the reply into an error.
func GetLicensePkgListError(r *GetLicensePkgListResponse) error {
	return hardwareDeviceError(r.ErrCode)
}

// GetLicensePkgList lists the licence packages of one device class.
//
// Check TotalRemaining before activating a batch: a package that runs out fails
// the activation with 9800039, per device.
func (a *Application) GetLicensePkgList(
	ctx context.Context,
	params GetLicensePkgListParams,
) (*GetLicensePkgListResponse, error) {
	if err := CheckGetLicensePkgListParams(params); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getLicensePkgListPath, nil, params.body())
	if err != nil {
		return nil, err
	}

	var result GetLicensePkgListResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
