package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/samuelenocsson/devops-tui/internal/config"
)

const (
	apiVersion        = "7.1"
	apiVersionPreview = "7.1-preview"
)

// Client is the Azure DevOps API client
type Client struct {
	httpClient   *http.Client
	baseURL      string
	teamURL      string
	webURL       string
	authHeader   string
	organization string
	project      string
	team         string
}

// NewClient creates a new Azure DevOps API client
func NewClient(cfg *config.Config) *Client {
	var authHeader string

	if cfg.IsPAT() {
		// Azure DevOps uses Basic auth with empty username and PAT as password
		auth := base64.StdEncoding.EncodeToString([]byte(":" + cfg.PAT))
		authHeader = "Basic " + auth
	} else {
		// OAuth uses Bearer token
		authHeader = "Bearer " + cfg.GetToken()
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:      cfg.BaseURL(),
		teamURL:      cfg.TeamURL(),
		webURL:       cfg.WebURL(),
		authHeader:   authHeader,
		organization: cfg.Organization,
		project:      cfg.Project,
		team:         cfg.Team,
	}
}

// NewClientWithToken creates a new Azure DevOps API client with a specific token
// isPAT indicates whether the token is a Personal Access Token or an OAuth token
func NewClientWithToken(cfg *config.Config, token string, isPAT bool) *Client {
	var authHeader string

	if isPAT {
		// Azure DevOps uses Basic auth with empty username and PAT as password
		auth := base64.StdEncoding.EncodeToString([]byte(":" + token))
		authHeader = "Basic " + auth
	} else {
		// OAuth uses Bearer token
		authHeader = "Bearer " + token
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:      cfg.BaseURL(),
		teamURL:      cfg.TeamURL(),
		webURL:       cfg.WebURL(),
		authHeader:   authHeader,
		organization: cfg.Organization,
		project:      cfg.Project,
		team:         cfg.Team,
	}
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, url string, body io.Reader) (*http.Response, error) {
	return c.doRequestWithContentType(method, url, body, "application/json")
}

// doRequestWithContentType performs an HTTP request with authentication and custom content type
func (c *Client) doRequestWithContentType(method, url string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// get performs a GET request to base URL
func (c *Client) get(endpoint string) (*http.Response, error) {
	return c.getWithBase(c.baseURL, endpoint)
}

// getTeam performs a GET request to team-specific URL
func (c *Client) getTeam(endpoint string) (*http.Response, error) {
	return c.getWithBase(c.teamURL, endpoint)
}

// getWithBase performs a GET request with a specific base URL
func (c *Client) getWithBase(baseURL, endpoint string) (*http.Response, error) {
	return c.getWithBaseAndVersion(baseURL, endpoint, apiVersion)
}

// getPreview performs a GET request using preview API version
func (c *Client) getPreview(endpoint string) (*http.Response, error) {
	return c.getWithBaseAndVersion(c.baseURL, endpoint, apiVersionPreview)
}

// getWithBaseAndVersion performs a GET request with a specific base URL and API version
func (c *Client) getWithBaseAndVersion(baseURL, endpoint, version string) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", baseURL, endpoint)
	if endpoint[0] != '/' {
		url = fmt.Sprintf("%s/%s", baseURL, endpoint)
	}

	// Add API version
	if len(url) > 0 {
		separator := "?"
		if len(url) > 0 && url[len(url)-1] != '?' {
			for _, c := range url {
				if c == '?' {
					separator = "&"
					break
				}
			}
		}
		url = fmt.Sprintf("%s%sapi-version=%s", url, separator, version)
	}

	return c.doRequest("GET", url, nil)
}

// post performs a POST request
func (c *Client) post(endpoint string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	if endpoint[0] != '/' {
		url = fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	}

	// Add API version
	separator := "?"
	for _, ch := range url {
		if ch == '?' {
			separator = "&"
			break
		}
	}
	url = fmt.Sprintf("%s%sapi-version=%s", url, separator, apiVersion)

	return c.doRequest("POST", url, body)
}

// patch performs a PATCH request (for work item updates)
func (c *Client) patch(endpoint string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	if endpoint[0] != '/' {
		url = fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	}

	// Add API version
	separator := "?"
	for _, ch := range url {
		if ch == '?' {
			separator = "&"
			break
		}
	}
	url = fmt.Sprintf("%s%sapi-version=%s", url, separator, apiVersion)

	return c.doRequestWithContentType("PATCH", url, body, "application/json-patch+json")
}

// decode decodes a JSON response into the given target
func decode(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// WorkItemWebURL returns the web URL for a work item
func (c *Client) WorkItemWebURL(id int) string {
	return fmt.Sprintf("%s/_workitems/edit/%d", c.webURL, id)
}

// Organization returns the organization name
func (c *Client) Organization() string {
	return c.organization
}

// Project returns the project name
func (c *Client) Project() string {
	return c.project
}

// Team returns the team name
func (c *Client) Team() string {
	return c.team
}
