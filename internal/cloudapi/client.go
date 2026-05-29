package cloudapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

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

	mu      sync.Mutex
	session *Session
}

type Session struct {
	Tailnet        string
	APIKey         string
	Policy         string
	ValidatedDraft string
	DevMode        bool
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

	if devtailnet.IsDevAPIKey(request.APIKey) {
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
		c.mu.Lock()
		if c.session != nil {
			c.session.ValidatedDraft = draft
		}
		c.mu.Unlock()
		return nil
	}
	if err := c.sendPolicy(ctx, http.MethodPost, session.Tailnet, session.APIKey, "/validate", draft); err != nil {
		return err
	}
	c.mu.Lock()
	if c.session != nil {
		c.session.ValidatedDraft = draft
	}
	c.mu.Unlock()
	return nil
}

func (c *Client) SaveValidatedPolicy(ctx context.Context) (string, error) {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(session.ValidatedDraft) == "" {
		return "", errors.New("validate a draft policy before saving")
	}
	if session.DevMode {
		c.mu.Lock()
		if c.session != nil {
			c.session.Policy = c.session.ValidatedDraft
			c.session.ValidatedDraft = ""
		}
		saved := c.session.Policy
		c.mu.Unlock()
		return saved, nil
	}
	if err := c.sendPolicy(ctx, http.MethodPost, session.Tailnet, session.APIKey, "", session.ValidatedDraft); err != nil {
		return "", err
	}
	policy, err := c.fetchPolicy(ctx, session.Tailnet, session.APIKey)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	if c.session != nil {
		c.session.Policy = policy
		c.session.ValidatedDraft = ""
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("policy fetch failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", apiError(resp.StatusCode, body, "policy fetch failed")
	}
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("policy request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiError(resp.StatusCode, body, "policy request failed")
	}
	return nil
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
