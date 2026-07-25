package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// DefaultDeviceFlowClientID is the default GitHub OAuth App client_id for device flow.
// Users can override this by setting `github.device_flow_client_id` in config.
// To use your own, register an OAuth App at https://github.com/settings/developers
// and set the client_id. The app must have the "Device Flow" grant type enabled.
const DefaultDeviceFlowClientID = "Iv1.8a2b3c4d5e6f7g8h" // placeholder — register your own

// DeviceFlowToken represents the GitHub device flow initiation response.
type DeviceFlowToken struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
}

// DeviceFlowPollResult is the result of polling for device flow authorization.
type DeviceFlowPollResult struct {
	AccessToken string `json:"access_token,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

// StartDeviceFlow initiates GitHub's device authorization flow.
// Returns the device code, user code, verification URI, and polling interval.
func StartDeviceFlow(ctx context.Context, clientID string) (*DeviceFlowToken, error) {
	if clientID == "" {
		clientID = DefaultDeviceFlowClientID
	}

	v := url.Values{
		"client_id": {clientID},
		"scope":     {"repo,read:user"},
	}
	body := strings.NewReader(v.Encode())

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/device/code", body)
	if err != nil {
		return nil, fmt.Errorf("create device code request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device code request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request got HTTP %d", resp.StatusCode)
	}

	var token DeviceFlowToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("parse device code response: %w", err)
	}

	if token.DeviceCode == "" {
		return nil, fmt.Errorf("device code response missing device_code")
	}

	return &token, nil
}

// PollDeviceFlow polls GitHub for device flow authorization.
// Returns the access token if authorized, or an error if still pending or failed.
// The caller should respect the interval from StartDeviceFlow between calls.
func PollDeviceFlow(ctx context.Context, clientID, deviceCode string) (string, error) {
	if clientID == "" {
		clientID = DefaultDeviceFlowClientID
	}

	v := url.Values{
		"client_id":   {clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}
	body := strings.NewReader(v.Encode())

	req, err := http.NewRequestWithContext(ctx, "POST", "https://github.com/login/oauth/access_token", body)
	if err != nil {
		return "", fmt.Errorf("create poll request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("poll request failed: %w", err)
	}
	defer resp.Body.Close()

	var result DeviceFlowPollResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse poll response: %w", err)
	}

	if result.AccessToken != "" {
		return result.AccessToken, nil
	}

	if result.Error != "" {
		// "authorization_pending" = not yet approved (keep polling)
		// "slow_down" = polling too fast (increase interval by 5s)
		// "expired_token" = user took too long
		// "access_denied" = user declined
		return "", fmt.Errorf("%s: %s", result.Error, result.ErrorDesc)
	}

	return "", fmt.Errorf("unexpected poll response: no token and no error")
}

// ValidateToken checks whether the given token is a valid GitHub PAT.
func ValidateToken(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token validation failed (HTTP %d)", resp.StatusCode)
	}

	// Extract username from response
	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("parse user response: %w", err)
	}

	return user.Login, nil
}
