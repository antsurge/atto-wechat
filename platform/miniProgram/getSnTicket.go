package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getSnTicketPath issues the ticket that binds a device to a user.
const getSnTicketPath = "/wxa/getsnticket"

// GetSnTicketParams identifies the device to issue a ticket for.
type GetSnTicketParams struct {
	// ModelID is the device model id, obtained when registering the device.
	ModelID string
	// SN is the serial number assigned by the vendor.
	SN string
}

// GetSnTicketResponse is the reply of the sn ticket endpoint.
type GetSnTicketResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// SnTicket is the device ticket, valid for 5 minutes.
	SnTicket string `json:"sn_ticket"`
}

// OK reports whether a ticket was issued.
//
// A reply without sn_ticket cannot be used: the ticket is what the device
// component hands to openBluetoothDevice, and it expires in 5 minutes, so treating
// an empty one as success only moves the failure to the client.
func (r *GetSnTicketResponse) OK() bool {
	return r != nil && r.ErrCode == 0 && r.SnTicket != ""
}

// GetSnTicketError turns the errcode from the reply into an error.
func GetSnTicketError(r *GetSnTicketResponse) error {
	return hardwareDeviceError(r.ErrCode)
}

// GetSnTicket issues a ticket for one device.
//
// The ticket is short-lived: request it when the device is about to be bound, not
// when the page loads.
func (a *Application) GetSnTicket(
	ctx context.Context,
	params GetSnTicketParams,
) (*GetSnTicketResponse, error) {
	if err := CheckHardwareDevice(HardwareDevice{ModelID: params.ModelID, SN: params.SN}); err != nil {
		return nil, err
	}

	body := &struct {
		SN      string `json:"sn"`
		ModelID string `json:"model_id"`
	}{SN: params.SN, ModelID: params.ModelID}

	resp, err := a.api.PostJSON(ctx, getSnTicketPath, nil, body)
	if err != nil {
		return nil, err
	}

	var result GetSnTicketResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
