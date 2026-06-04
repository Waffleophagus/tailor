package cloudapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Waffleophagus/tailor/internal/devtailnet"
	"github.com/Waffleophagus/tailor/internal/policy"
)

const (
	DefaultBaseURL = "https://api.tailscale.com"
)

var ErrNotAuthenticated = errors.New("cloud api is not authenticated")

type Client struct {
	httpClient *http.Client
	baseURL    string
	logger     *slog.Logger

	mu      sync.Mutex
	session *Session
}

type Session struct {
	Tailnet string
	APIKey  string
	Policy  string
	DevMode bool
}

type AuthRequest struct {
	Tailnet string
	APIKey  string
}

type AuthStatus struct {
	Authenticated bool
	Tailnet       string
	HasPolicy     bool
	DevMode       bool
}

type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("tailscale api returned status %d", e.StatusCode)
	}
	return e.Message
}

func New(options ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    DefaultBaseURL,
		logger:     slog.New(slog.DiscardHandler),
	}
	for _, option := range options {
		option(c)
	}
	return c
}

type Option func(*Client)

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		if baseURL != "" {
			c.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(c *Client) {
		if logger != nil {
			c.logger = logger
		}
	}
}

func (c *Client) Authenticate(ctx context.Context, request AuthRequest) (AuthStatus, error) {
	request.Tailnet = strings.TrimSpace(request.Tailnet)
	request.APIKey = strings.TrimSpace(request.APIKey)
	if request.Tailnet == "" {
		return AuthStatus{}, errors.New("tailnet is required")
	}
	if request.APIKey == "" {
		return AuthStatus{}, errors.New("Tailscale API key is required")
	}
	if !strings.HasPrefix(request.APIKey, "tskey-api-") {
		return AuthStatus{}, errors.New("Tailscale API key must start with tskey-api-")
	}

	if devtailnet.Enabled && devtailnet.IsDevAPIKey(request.APIKey) {
		tailnet := request.Tailnet
		if tailnet == "" || tailnet == "-" {
			tailnet = devtailnet.Name
		}
		c.mu.Lock()
		c.session = &Session{
			Tailnet: tailnet,
			APIKey:  request.APIKey,
			Policy:  devtailnet.Policy(),
			DevMode: true,
		}
		status := c.statusLocked()
		c.mu.Unlock()
		return status, nil
	}

	policy, err := c.fetchPolicy(ctx, request.Tailnet, request.APIKey)
	if err != nil {
		return AuthStatus{}, err
	}

	c.mu.Lock()
	c.session = &Session{
		Tailnet: request.Tailnet,
		APIKey:  request.APIKey,
		Policy:  policy,
	}
	status := c.statusLocked()
	c.mu.Unlock()
	return status, nil
}

func (c *Client) Status() AuthStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.statusLocked()
}

func (c *Client) Policy(ctx context.Context) (string, error) {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return "", err
	}
	if session.Policy != "" {
		return session.Policy, nil
	}
	policy, err := c.fetchPolicy(ctx, session.Tailnet, session.APIKey)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	if c.session != nil {
		c.session.Policy = policy
	}
	c.mu.Unlock()
	return policy, nil
}

func (c *Client) RefreshPolicy(ctx context.Context) (string, error) {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return "", err
	}
	if session.DevMode {
		return session.Policy, nil
	}
	policy, err := c.fetchPolicy(ctx, session.Tailnet, session.APIKey)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	if c.session != nil {
		c.session.Policy = policy
	}
	c.mu.Unlock()
	return policy, nil
}

func (c *Client) ValidatePolicy(ctx context.Context, draft string) error {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(draft) == "" {
		return errors.New("draft policy is required")
	}
	if session.DevMode {
		if _, err := policy.Parse(draft); err != nil {
			return err
		}
		return nil
	}
	return c.sendPolicy(ctx, http.MethodPost, session.Tailnet, session.APIKey, "/validate", draft)
}

func (c *Client) SavePolicy(ctx context.Context, draft string) (string, error) {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(draft) == "" {
		return "", errors.New("draft policy is required")
	}
	if session.DevMode {
		if _, err := policy.Parse(draft); err != nil {
			return "", err
		}
		c.mu.Lock()
		if c.session != nil {
			c.session.Policy = draft
		}
		saved := c.session.Policy
		c.mu.Unlock()
		return saved, nil
	}
	if err := c.sendPolicy(ctx, http.MethodPost, session.Tailnet, session.APIKey, "", draft); err != nil {
		return "", err
	}
	policy, err := c.fetchPolicy(ctx, session.Tailnet, session.APIKey)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	if c.session != nil {
		c.session.Policy = policy
	}
	c.mu.Unlock()
	return policy, nil
}

func (c *Client) ensureSession(ctx context.Context) (*Session, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == nil {
		return nil, ErrNotAuthenticated
	}
	session := *c.session
	return &session, nil
}

func (c *Client) statusLocked() AuthStatus {
	if c.session == nil {
		return AuthStatus{}
	}
	return AuthStatus{
		Authenticated: true,
		Tailnet:       c.session.Tailnet,
		HasPolicy:     c.session.Policy != "",
		DevMode:       c.session.DevMode,
	}
}

func (c *Client) fetchPolicy(ctx context.Context, tailnet, apiKey string) (string, error) {
	endpoint := c.baseURL + "/api/v2/tailnet/" + url.PathEscape(tailnet) + "/acl"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(apiKey, "")
	req.Header.Set("Accept", "application/hujson, application/json, text/plain")

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logCloudAPI(http.MethodGet, tailnet, "", 0, time.Since(start), err)
		return "", fmt.Errorf("policy fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		c.logCloudAPI(http.MethodGet, tailnet, "", resp.StatusCode, time.Since(start), err)
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := apiError(resp.StatusCode, body, "policy fetch failed")
		c.logCloudAPI(http.MethodGet, tailnet, "", resp.StatusCode, time.Since(start), apiErr)
		return "", apiErr
	}
	c.logCloudAPI(http.MethodGet, tailnet, "", resp.StatusCode, time.Since(start), nil)
	return string(body), nil
}

func (c *Client) sendPolicy(ctx context.Context, method, tailnet, apiKey, suffix, policy string) error {
	endpoint := c.baseURL + "/api/v2/tailnet/" + url.PathEscape(tailnet) + "/acl" + suffix
	req, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(policy))
	if err != nil {
		return err
	}
	req.SetBasicAuth(apiKey, "")
	req.Header.Set("Content-Type", "application/hujson")
	req.Header.Set("Accept", "application/hujson, application/json, text/plain")

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logCloudAPI(method, tailnet, suffix, 0, time.Since(start), err)
		return fmt.Errorf("policy request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		c.logCloudAPI(method, tailnet, suffix, resp.StatusCode, time.Since(start), err)
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := apiError(resp.StatusCode, body, "policy request failed")
		c.logCloudAPI(method, tailnet, suffix, resp.StatusCode, time.Since(start), apiErr)
		return apiErr
	}
	c.logCloudAPI(method, tailnet, suffix, resp.StatusCode, time.Since(start), nil)
	return nil
}

func (c *Client) logCloudAPI(method, tailnet, suffix string, status int, elapsed time.Duration, err error) {
	operation := "policy_fetch"
	if suffix == "/validate" {
		operation = "policy_validate"
	} else if method == http.MethodPost && suffix == "" {
		operation = "policy_save"
	}

	attrs := []any{
		"method", method,
		"operation", operation,
		"tailnet", tailnet,
		"latency_ms", elapsed.Milliseconds(),
	}
	if suffix != "" {
		attrs = append(attrs, "suffix", suffix)
	}
	if status > 0 {
		attrs = append(attrs, "status", status)
	}

	switch {
	case err != nil:
		attrs = append(attrs, "error", err.Error())
		c.logger.Warn("cloud api request failed", attrs...)
	case method == http.MethodGet:
		c.logger.Debug("cloud api request", attrs...)
	default:
		c.logger.Info("cloud api request", attrs...)
	}
}

func apiError(statusCode int, body []byte, fallback string) error {
	message := strings.TrimSpace(string(body))
	var structured struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if json.Unmarshal(body, &structured) == nil {
		if structured.Message != "" {
			message = structured.Message
		} else if structured.Error != "" {
			message = structured.Error
		}
	}
	message = scrubSecrets(message)
	if message == "" {
		message = fallback
	}
	return &Error{StatusCode: statusCode, Message: fmt.Sprintf("%s (status %d)", message, statusCode)}
}

func scrubSecrets(message string) string {
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")
	return string(bytes.TrimSpace([]byte(message)))
}
