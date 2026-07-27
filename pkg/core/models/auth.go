package models

// DeviceFlowCode represents the GitHub device authorization flow initiation response.
// Returned by StartDeviceFlow and used to display the user code and poll for the token.
type DeviceFlowCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
}
