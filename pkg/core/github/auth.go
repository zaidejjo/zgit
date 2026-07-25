package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DeviceFlowToken represents the GitHub device flow response.
type DeviceFlowToken struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
}

// DeviceFlowStatus is the result of polling for device flow authorization.
type DeviceFlowStatus struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

// StartDeviceFlow initiates GitHub's device authorization flow.
// Returns a URL the user visits to authorize, and a polling function to get the token.
func StartDeviceFlow(ctx context.Context) (*DeviceFlowToken, error) {
	// POST https://github.com/login/device/code
	// with client_id = "Iv1.8a2b3c4d5e6f7g8h" (this is GitButler's public app ID,
	// we should register our own)
	return nil, fmt.Errorf("device flow not yet implemented; use PAT-based auth for now")
	// TODO: Implement device flow with GitHub OAuth App
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
