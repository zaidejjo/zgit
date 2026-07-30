package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	// DefaultDeviceFlowInterval is the fallback poll interval (seconds) if GitHub doesn't specify one.
	DefaultDeviceFlowInterval = 5
	// MinDeviceFlowInterval is the minimum allowed poll interval.
	MinDeviceFlowInterval = 5
	// MaxDeviceFlowDuration is how long we poll before giving up.
	MaxDeviceFlowDuration = 10 * time.Minute
	// SlowDownBackoff is added to the interval on each slow_down response.
	SlowDownBackoff = 5
	// deviceFlowHTTPTimeout is the per-request HTTP timeout for device flow API calls.
	deviceFlowHTTPTimeout = 30 * time.Second
)

// DefaultDeviceFlowClientID is the default GitHub OAuth App client_id for device flow.
// Priority: env var ZGIT_GITHUB_CLIENT_ID > config key github.device_flow_client_id > this default.
// Register your own OAuth App at https://github.com/settings/developers
// (must have "Device Flow" grant type enabled).
var DefaultDeviceFlowClientID = "Ov23liqkTja60fCr98aD"

func init() {
	if env := os.Getenv("ZGIT_GITHUB_CLIENT_ID"); env != "" {
		DefaultDeviceFlowClientID = env
	}
}

// DeviceFlowToken represents the GitHub device flow initiation response.
type DeviceFlowToken struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	Interval        int    `json:"interval"`
}

// githubErrorBody is used to parse error responses from GitHub.
type githubErrorBody struct {
	Error     string `json:"error"`
	ErrorDesc string `json:"error_description"`
	ErrorURI  string `json:"error_uri"`
}

// readBodyAsError reads the response body and returns a descriptive error.
// On failure to read, falls back to the HTTP status.
func readBodyAsError(resp *http.Response, fallbackFmt string, args ...interface{}) error {
	fallback := fmt.Sprintf(fallbackFmt, args...)
	raw, err := io.ReadAll(resp.Body)
	if err != nil || len(raw) == 0 {
		return fmt.Errorf("%s (empty body)", fallback)
	}

	// Try JSON error envelope
	var ghErr githubErrorBody
	if json.Unmarshal(raw, &ghErr) == nil && ghErr.Error != "" {
		if ghErr.ErrorDesc != "" {
			return fmt.Errorf("%s: %s (%s)", fallback, ghErr.ErrorDesc, ghErr.Error)
		}
		return fmt.Errorf("%s: %s", fallback, ghErr.Error)
	}

	// Return raw body as fallback
	bodyStr := string(bytes.TrimSpace(raw))
	if len(bodyStr) > 200 {
		bodyStr = bodyStr[:200] + "..."
	}
	return fmt.Errorf("%s: %s", fallback, bodyStr)
}

// DeviceFlowPollResult is the result of polling for device flow authorization.
type DeviceFlowPollResult struct {
	AccessToken string `json:"access_token,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
	Scope       string `json:"scope,omitempty"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

// deviceFlowClient is a dedicated HTTP client for device flow API calls.
// It has a per-request timeout to prevent hanging when GitHub is unreachable.
var deviceFlowClient = &http.Client{
	Timeout: deviceFlowHTTPTimeout,
}

// deviceFlowRequest sends a JSON POST to the given GitHub OAuth endpoint
// with the provided JSON body, and returns the response.
func deviceFlowRequest(ctx context.Context, url string, body any) (*http.Response, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := deviceFlowClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return resp, nil
}

// StartDeviceFlow initiates GitHub's device authorization flow.
// Returns the device code, user code, verification URI, and polling interval.
func StartDeviceFlow(ctx context.Context, clientID string) (*DeviceFlowToken, error) {
	if clientID == "" {
		clientID = DefaultDeviceFlowClientID
	}

	body := map[string]string{
		"client_id": clientID,
		"scope":     "repo read:user user:email",
	}

	resp, err := deviceFlowRequest(ctx, "https://github.com/login/device/code", body)
	if err != nil {
		return nil, fmt.Errorf("device code request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, readBodyAsError(resp, "GitHub device code request failed (HTTP %d)", resp.StatusCode)
	}

	var token DeviceFlowToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("parse device code response: %w", err)
	}

	if token.DeviceCode == "" {
		return nil, fmt.Errorf("device code response missing device_code — check client_id is valid and device flow is enabled")
	}

	return &token, nil
}

// PollDeviceFlow polls GitHub for device flow authorization.
// Returns the access token if authorized.
// Returns empty string if still pending (caller should retry after interval).
func PollDeviceFlow(ctx context.Context, clientID, deviceCode string) (string, error) {
	if clientID == "" {
		clientID = DefaultDeviceFlowClientID
	}

	body := map[string]string{
		"client_id":   clientID,
		"device_code": deviceCode,
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
	}

	resp, err := deviceFlowRequest(ctx, "https://github.com/login/oauth/access_token", body)
	if err != nil {
		return "", fmt.Errorf("poll request: %w", err)
	}
	defer resp.Body.Close()

	// Read raw body first so we can inspect it on error
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read poll response: %w", err)
	}

	var result DeviceFlowPollResult
	if err := json.Unmarshal(raw, &result); err != nil {
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
		// "bad_verification_code" = device code invalid/expired
		if result.Error == "authorization_pending" {
			return "", nil
		}
		desc := result.ErrorDesc
		if desc == "" {
			desc = result.Error
		}
		return "", fmt.Errorf("%s", desc)
	}

	// Non-200 with no structured error
	if resp.StatusCode != http.StatusOK {
		bodyStr := string(bytes.TrimSpace(raw))
		if len(bodyStr) > 200 {
			bodyStr = bodyStr[:200] + "..."
		}
		return "", fmt.Errorf("GitHub poll returned HTTP %d: %s", resp.StatusCode, bodyStr)
	}

	return "", fmt.Errorf("unexpected poll response: no token and no error")
}

// PollDeviceFlowWithRetry polls GitHub for device flow authorization with automatic retry.
// It respects the poll interval from StartDeviceFlow, handles slow_down by increasing the
// interval, and gives up after MaxDeviceFlowDuration (10 minutes).
// The caller should pass the device_code and interval returned by StartDeviceFlow.
func PollDeviceFlowWithRetry(ctx context.Context, clientID, deviceCode string, initialInterval int) (string, error) {
	if clientID == "" {
		clientID = DefaultDeviceFlowClientID
	}

	interval := initialInterval
	if interval < MinDeviceFlowInterval {
		interval = MinDeviceFlowInterval
	}

	deadline := time.Now().Add(MaxDeviceFlowDuration)

	for {
		// Check context cancellation before sleeping
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Check global timeout
		if time.Now().After(deadline) {
			return "", fmt.Errorf("authorization timed out after 10 minutes — please try again")
		}

		// Sleep for the current interval between polls
		timer := time.NewTimer(time.Duration(interval) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}

		body := map[string]string{
			"client_id":   clientID,
			"device_code": deviceCode,
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		}

		resp, err := deviceFlowRequest(ctx, "https://github.com/login/oauth/access_token", body)
		if err != nil {
			return "", fmt.Errorf("poll request: %w", err)
		}

		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return "", fmt.Errorf("read poll response: %w", readErr)
		}

		var result DeviceFlowPollResult
		if err := json.Unmarshal(raw, &result); err != nil {
			return "", fmt.Errorf("parse poll response: %w", err)
		}

		if result.AccessToken != "" {
			return result.AccessToken, nil
		}

		if result.Error != "" {
			switch result.Error {
			case "authorization_pending":
				// Normal — user hasn't approved yet, keep polling at current interval
				continue
			case "slow_down":
				// GitHub says we're polling too fast — increase interval by 5s
				interval += SlowDownBackoff
				continue
			case "expired_token":
				return "", fmt.Errorf("device code expired — click Start again to get a new code")
			case "access_denied":
				return "", fmt.Errorf("authorization declined — you cancelled the flow on GitHub")
			default:
				desc := result.ErrorDesc
				if desc == "" {
					desc = result.Error
				}
				return "", fmt.Errorf("%s", desc)
			}
		}

		// Non-200 with no structured error
		if resp.StatusCode != http.StatusOK {
			bodyStr := string(bytes.TrimSpace(raw))
			if len(bodyStr) > 200 {
				bodyStr = bodyStr[:200] + "..."
			}
			return "", fmt.Errorf("GitHub poll returned HTTP %d: %s", resp.StatusCode, bodyStr)
		}
	}
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
		return "", readBodyAsError(resp, "token validation failed (HTTP %d)", resp.StatusCode)
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("parse user response: %w", err)
	}

	return user.Login, nil
}
