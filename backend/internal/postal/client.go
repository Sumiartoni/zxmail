package postal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	ErrManualSetupRequired  = errors.New("postal operation requires manual setup or API contract validation")
	ErrUnsupportedOperation = errors.New("postal operation is not wired to a confirmed API endpoint")
)

type Client struct {
	baseURL        string
	apiKey         string
	smtpPublicHost string
	httpClient     *http.Client
}

type HealthCheckResult struct {
	BaseURL          string `json:"base_url"`
	Reachable        bool   `json:"reachable"`
	StatusCode       int    `json:"status_code"`
	SMTPPublicHost   string `json:"smtp_public_host"`
	APIKeyConfigured bool   `json:"api_key_configured"`
	Note             string `json:"note"`
}

type ServerPlaceholderRequest struct {
	Name string `json:"name"`
}

type CredentialPlaceholderRequest struct {
	Username string `json:"username"`
	Domain   string `json:"domain"`
}

type MessagePlaceholderResult struct {
	MessageID string `json:"message_id"`
	Note      string `json:"note"`
}

func NewClient(baseURL, apiKey, smtpPublicHost string) *Client {
	return &Client{
		baseURL:        strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:         strings.TrimSpace(apiKey),
		smtpPublicHost: strings.TrimSpace(smtpPublicHost),
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (c *Client) HealthCheck(ctx context.Context) (*HealthCheckResult, error) {
	if c.baseURL == "" {
		return nil, fmt.Errorf("postal base URL is empty")
	}

	parsed, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid postal base URL: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}

	// TODO: Once the exact Postal API auth contract is validated in this deployment,
	// add the required API key header to this request and switch the probe to a
	// confirmed API endpoint instead of root reachability only.
	request.Header.Set("User-Agent", "zxmail-postal-healthcheck/1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	return &HealthCheckResult{
		BaseURL:          c.baseURL,
		Reachable:        response.StatusCode >= 200 && response.StatusCode < 500,
		StatusCode:       response.StatusCode,
		SMTPPublicHost:   c.smtpPublicHost,
		APIKeyConfigured: c.apiKey != "",
		Note:             "Checks Postal base URL reachability only. It does not confirm specific Postal API capabilities yet.",
	}, nil
}

func (c *Client) CreateServerPlaceholder(ctx context.Context, request ServerPlaceholderRequest) error {
	_ = ctx
	_ = request

	// TODO: Validate the exact Postal server creation endpoint and auth scheme
	// before implementing this. Production v1 currently expects this step to be
	// completed manually in Postal or via a later confirmed API contract.
	return fmt.Errorf("%w: createServerPlaceholder", ErrManualSetupRequired)
}

func (c *Client) CreateCredentialPlaceholder(ctx context.Context, request CredentialPlaceholderRequest) error {
	_ = ctx
	_ = request

	// TODO: Confirm whether Postal exposes a direct credential provisioning
	// endpoint for the zxMail flow. Until that contract is verified, do not
	// pretend the credential exists in Postal.
	return fmt.Errorf("%w: createCredentialPlaceholder", ErrUnsupportedOperation)
}

func (c *Client) GetMessagePlaceholder(ctx context.Context, messageID string) (*MessagePlaceholderResult, error) {
	_ = ctx
	_ = messageID

	// TODO: Wire this to a confirmed Postal message lookup endpoint once the
	// exact API surface is validated. Manual inspection in Postal remains
	// required for now.
	return nil, fmt.Errorf("%w: getMessagePlaceholder", ErrUnsupportedOperation)
}
