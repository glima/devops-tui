package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/samuelenocsson/devops-tui/pkg/browser"
)

const (
	// Visual Studio client ID (public client registered for Azure DevOps)
	DefaultClientID = "872cd9fa-d31f-45e0-9eab-6e460a02d1f1"

	// Azure DevOps scope with offline_access for refresh tokens
	AzureDevOpsScope = "499b84ac-1321-427f-aa17-267ca6975798/user_impersonation offline_access"

	// Microsoft OAuth2 endpoints (using "common" for both work/school and personal accounts)
	DeviceCodeEndpoint = "https://login.microsoftonline.com/common/oauth2/v2.0/devicecode"
	TokenEndpoint      = "https://login.microsoftonline.com/common/oauth2/v2.0/token"
)

// DeviceCodeResponse is the response from the device code endpoint
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

// TokenResponse is the response from the token endpoint
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// TokenCache stores tokens on disk for reuse
type TokenCache struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// TokenError represents an OAuth error response
type TokenError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// DeviceFlowAuthenticator handles OAuth2 device flow authentication
type DeviceFlowAuthenticator struct {
	clientID   string
	httpClient *http.Client
	cacheFile  string
}

// NewDeviceFlowAuthenticator creates a new device flow authenticator
func NewDeviceFlowAuthenticator() *DeviceFlowAuthenticator {
	cacheDir := getCacheDir()
	return &DeviceFlowAuthenticator{
		clientID: DefaultClientID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheFile: filepath.Join(cacheDir, "token.json"),
	}
}

// getCacheDir returns the cache directory for storing tokens
func getCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config", "devops-tui")
}

// GetToken attempts to get a valid access token, using cache or device flow
func (a *DeviceFlowAuthenticator) GetToken() (string, error) {
	// Try to load cached token
	if token, err := a.loadCachedToken(); err == nil {
		// Check if token is still valid (with 5 minute buffer)
		if time.Now().Add(5 * time.Minute).Before(token.ExpiresAt) {
			return token.AccessToken, nil
		}

		// Try to refresh the token
		if token.RefreshToken != "" {
			if newToken, err := a.refreshToken(token.RefreshToken); err == nil {
				return newToken, nil
			}
		}
	}

	// Fall back to device flow
	return a.authenticateWithDeviceFlow()
}

// loadCachedToken loads the token from the cache file
func (a *DeviceFlowAuthenticator) loadCachedToken() (*TokenCache, error) {
	data, err := os.ReadFile(a.cacheFile)
	if err != nil {
		return nil, err
	}

	var cache TokenCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// saveTokenCache saves the token to the cache file
func (a *DeviceFlowAuthenticator) saveTokenCache(tokenResp *TokenResponse) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(a.cacheFile), 0700); err != nil {
		return err
	}

	cache := TokenCache{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(a.cacheFile, data, 0600)
}

// refreshToken attempts to refresh an expired access token
func (a *DeviceFlowAuthenticator) refreshToken(refreshToken string) (string, error) {
	data := url.Values{
		"client_id":     {a.clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {AzureDevOpsScope},
	}

	resp, err := a.httpClient.PostForm(TokenEndpoint, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("failed to refresh token")
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	// Save the new token
	if err := a.saveTokenCache(&tokenResp); err != nil {
		// Log but don't fail - we still have a valid token
		fmt.Fprintf(os.Stderr, "Warning: failed to cache token: %v\n", err)
	}

	return tokenResp.AccessToken, nil
}

// authenticateWithDeviceFlow performs the device flow authentication
func (a *DeviceFlowAuthenticator) authenticateWithDeviceFlow() (string, error) {
	// Step 1: Request device code
	deviceCode, err := a.requestDeviceCode()
	if err != nil {
		return "", fmt.Errorf("failed to request device code: %w", err)
	}

	// Step 2: Display instructions and open browser
	fmt.Println()
	fmt.Println("╭───────────────────────────────────────────────────────────────╮")
	fmt.Println("│                  Azure DevOps Authentication                  │")
	fmt.Println("├───────────────────────────────────────────────────────────────┤")
	fmt.Println("│  To sign in, use a web browser to open:                       │")
	fmt.Printf("│  %-61s│\n", deviceCode.VerificationURI)
	fmt.Println("│                                                               │")
	fmt.Printf("│  And enter the code: %-41s│\n", deviceCode.UserCode)
	fmt.Println("╰───────────────────────────────────────────────────────────────╯")
	fmt.Println()

	// Try to open the browser automatically
	if err := browser.Open(deviceCode.VerificationURI); err != nil {
		fmt.Printf("Could not open browser automatically. Please open the URL manually.\n")
	} else {
		fmt.Printf("Browser opened. Waiting for authentication...\n")
	}

	// Step 3: Poll for token
	token, err := a.pollForToken(deviceCode)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Println("✓ Authentication successful!")
	fmt.Println()

	return token, nil
}

// requestDeviceCode requests a device code from Azure AD
func (a *DeviceFlowAuthenticator) requestDeviceCode() (*DeviceCodeResponse, error) {
	data := url.Values{
		"client_id": {a.clientID},
		"scope":     {AzureDevOpsScope},
	}

	resp, err := a.httpClient.PostForm(DeviceCodeEndpoint, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device code request failed: %s", string(body))
	}

	var deviceCode DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceCode); err != nil {
		return nil, err
	}

	return &deviceCode, nil
}

// pollForToken polls the token endpoint until authentication completes
func (a *DeviceFlowAuthenticator) pollForToken(deviceCode *DeviceCodeResponse) (string, error) {
	interval := time.Duration(deviceCode.Interval) * time.Second
	if interval == 0 {
		interval = 5 * time.Second
	}

	expiration := time.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)

	for time.Now().Before(expiration) {
		time.Sleep(interval)

		data := url.Values{
			"client_id":   {a.clientID},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {deviceCode.DeviceCode},
		}

		resp, err := a.httpClient.PostForm(TokenEndpoint, data)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		// Check for errors
		if resp.StatusCode != http.StatusOK {
			var tokenErr TokenError
			if err := json.Unmarshal(body, &tokenErr); err != nil {
				continue
			}

			switch tokenErr.Error {
			case "authorization_pending":
				// User hasn't authenticated yet, keep polling
				continue
			case "slow_down":
				// We're polling too fast, increase interval
				interval += 5 * time.Second
				continue
			case "expired_token":
				return "", errors.New("device code expired - please try again")
			case "authorization_declined":
				return "", errors.New("user declined authorization")
			default:
				return "", fmt.Errorf("authentication error: %s - %s", tokenErr.Error, tokenErr.ErrorDescription)
			}
		}

		// Success! Parse the token
		var tokenResp TokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			return "", fmt.Errorf("failed to parse token response: %w", err)
		}

		// Save token to cache
		if err := a.saveTokenCache(&tokenResp); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cache token: %v\n", err)
		}

		return tokenResp.AccessToken, nil
	}

	return "", errors.New("authentication timed out")
}

// ClearCache removes the cached token
func (a *DeviceFlowAuthenticator) ClearCache() error {
	err := os.Remove(a.cacheFile)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// HasCachedToken returns true if there's a cached token (may be expired)
func (a *DeviceFlowAuthenticator) HasCachedToken() bool {
	_, err := os.Stat(a.cacheFile)
	return err == nil
}

// GetAuthHeader returns the appropriate authorization header value
// For OAuth tokens, this is "Bearer <token>"
// For PAT, this is "Basic <base64(:PAT)>"
func GetAuthHeader(token string, isPAT bool) string {
	if isPAT {
		encoded := base64.StdEncoding.EncodeToString([]byte(":" + token))
		return "Basic " + encoded
	}
	return "Bearer " + token
}
