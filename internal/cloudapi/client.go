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

// ErrPolicyChanged reports that the tailnet policy changed after it was read, so a
// conditional save (If-Match) was rejected by Tailscale.
var ErrPolicyChanged = errors.New("tailnet policy changed since it was read")

// PolicyVersion is a tailnet policy together with the ETag Tailscale returned for it.
// ETag is empty in dev mode.
type PolicyVersion struct {
	HuJSON string
	ETag   string
}

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

type Device struct {
	Addresses       []string       `json:"addresses"`
	ID              string         `json:"id"`
	NodeID          string         `json:"nodeId"`
	User            string         `json:"user"`
	Name            string         `json:"name"`
	Hostname        string         `json:"hostname"`
	ClientVersion   string         `json:"clientVersion"`
	OS              string         `json:"os"`
	Tags            []string       `json:"tags"`
	LastSeen        string         `json:"lastSeen"`
	IsExternal      bool           `json:"isExternal"`
	EnabledRoutes   []string       `json:"enabledRoutes"`
	PostureIdentity map[string]any `json:"postureIdentity"`
}

type devicesResponse struct {
	Devices []Device `json:"devices"`
}

type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	LoginName   string `json:"loginName"`
	Type        string `json:"type"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

type usersResponse struct {
	Users []User `json:"users"`
}

type DevicePostureAttributes struct {
	Attributes map[string]any `json:"attributes"`
}

type VIPService struct {
	Name        string            `json:"name"`
	Addrs       []string          `json:"addrs"`
	Comment     string            `json:"comment,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Ports       []string          `json:"ports,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type vipServicesResponse struct {
	VIPServices []VIPService `json:"vipServices"`
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("tailscale api returned status %d", e.StatusCode)
	}
	return e.Message
}

func New(options ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
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

func normalizeAuthRequest(request AuthRequest) (AuthRequest, error) {
	request.Tailnet = strings.TrimSpace(request.Tailnet)
	request.APIKey = strings.TrimSpace(request.APIKey)
	if request.Tailnet == "" {
		return request, errors.New("tailnet is required")
	}
	if request.APIKey == "" {
		return request, errors.New("Tailscale API key is required")
	}
	if !strings.HasPrefix(request.APIKey, "tskey-api-") {
		return request, errors.New("Tailscale API key must start with tskey-api-")
	}
	return request, nil
}

// IsDevCredential reports whether request uses the built-in demo tailnet key, which
// only dev builds accept.
func IsDevCredential(request AuthRequest) bool {
	return devtailnet.Enabled && devtailnet.IsDevAPIKey(strings.TrimSpace(request.APIKey))
}

// CredentialInfo is what a candidate API key can see. It is used to vet a key
// before Tailor stores it.
type CredentialInfo struct {
	Devices []Device
	Users   []User
}

// InspectCredentials reads the devices and users visible to request's key without
// storing the key or changing the current session.
func (c *Client) InspectCredentials(ctx context.Context, request AuthRequest) (CredentialInfo, error) {
	request, err := normalizeAuthRequest(request)
	if err != nil {
		return CredentialInfo{}, err
	}
	candidate := &Session{Tailnet: request.Tailnet, APIKey: request.APIKey}
	var devices devicesResponse
	if err := c.getJSON(ctx, candidate, "/devices", url.Values{"fields": []string{"all"}}, &devices); err != nil {
		return CredentialInfo{}, err
	}
	users, err := c.fetchUsers(ctx, candidate)
	if err != nil {
		return CredentialInfo{}, err
	}
	return CredentialInfo{Devices: devices.Devices, Users: users}, nil
}

func (c *Client) Authenticate(ctx context.Context, request AuthRequest) (AuthStatus, error) {
	request, err := normalizeAuthRequest(request)
	if err != nil {
		return AuthStatus{}, err
	}

	if IsDevCredential(request) {
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

	policy, _, err := c.fetchPolicy(ctx, request.Tailnet, request.APIKey)
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
	policy, _, err := c.fetchPolicy(ctx, session.Tailnet, session.APIKey)
	if err != nil {
		return "", err
	}
	c.cachePolicy(session, policy)
	return policy, nil
}

// RefreshPolicy bypasses the session cache and returns the live tailnet policy.
func (c *Client) RefreshPolicy(ctx context.Context) (string, error) {
	version, err := c.RefreshPolicyVersion(ctx)
	return version.HuJSON, err
}

// RefreshPolicyVersion returns the live tailnet policy and its ETag, for use with
// SavePolicyIfMatch.
func (c *Client) RefreshPolicyVersion(ctx context.Context) (PolicyVersion, error) {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return PolicyVersion{}, err
	}
	if session.DevMode {
		return PolicyVersion{HuJSON: session.Policy}, nil
	}
	policy, etag, err := c.fetchPolicy(ctx, session.Tailnet, session.APIKey)
	if err != nil {
		return PolicyVersion{}, err
	}
	c.cachePolicy(session, policy)
	return PolicyVersion{HuJSON: policy, ETag: etag}, nil
}

// cachePolicy stores a fetched policy, unless the session was replaced (a different
// tailnet or key) while the request was in flight.
func (c *Client) cachePolicy(from *Session, policy string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session != nil && c.session.Tailnet == from.Tailnet && c.session.APIKey == from.APIKey {
		c.session.Policy = policy
	}
}

func (c *Client) Devices(ctx context.Context) ([]Device, error) {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return nil, err
	}
	if session.DevMode {
		return nil, nil
	}
	var out devicesResponse
	if err := c.getJSON(ctx, session, "/devices", url.Values{"fields": []string{"all"}}, &out); err != nil {
		return nil, err
	}
	return out.Devices, nil
}

func (c *Client) Users(ctx context.Context) ([]User, error) {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return nil, err
	}
	if session.DevMode {
		return nil, nil
	}
	return c.fetchUsers(ctx, session)
}

func (c *Client) fetchUsers(ctx context.Context, session *Session) ([]User, error) {
	var out usersResponse
	query := url.Values{
		"type": []string{"all"},
		"role": []string{"all"},
	}
	if err := c.getJSON(ctx, session, "/users", query, &out); err != nil {
		return nil, err
	}
	return out.Users, nil
}

func (c *Client) DevicePostureAttributes(ctx context.Context, deviceID string) (DevicePostureAttributes, error) {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return DevicePostureAttributes{}, err
	}
	if session.DevMode {
		return DevicePostureAttributes{}, nil
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return DevicePostureAttributes{}, errors.New("device ID is required")
	}
	var out DevicePostureAttributes
	if err := c.getJSONAPI(ctx, session, "/device/"+url.PathEscape(deviceID)+"/attributes", nil, &out); err != nil {
		return DevicePostureAttributes{}, err
	}
	return out, nil
}

func (c *Client) VIPServices(ctx context.Context) ([]VIPService, error) {
	session, err := c.ensureSession(ctx)
	if err != nil {
		return nil, err
	}
	if session.DevMode {
		return nil, nil
	}
	var out vipServicesResponse
	if err := c.getJSON(ctx, session, "/services", nil, &out); err != nil {
		var apiErr *Error
		if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
			return nil, err
		}
		if err := c.getJSON(ctx, session, "/vip-services", nil, &out); err != nil {
			return nil, err
		}
	}
	return out.VIPServices, nil
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
	return c.sendPolicy(ctx, http.MethodPost, session.Tailnet, session.APIKey, "/validate", draft, "")
}

func (c *Client) SavePolicy(ctx context.Context, draft string) (string, error) {
	return c.SavePolicyIfMatch(ctx, draft, "")
}

// SavePolicyIfMatch replaces the tailnet policy. When etag is non-empty, Tailscale
// only accepts the write if the live policy still has that ETag; otherwise it
// returns ErrPolicyChanged.
func (c *Client) SavePolicyIfMatch(ctx context.Context, draft, etag string) (string, error) {
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
		c.cachePolicy(session, draft)
		return draft, nil
	}
	if err := c.sendPolicy(ctx, http.MethodPost, session.Tailnet, session.APIKey, "", draft, etag); err != nil {
		var apiErr *Error
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusPreconditionFailed {
			return "", fmt.Errorf("%w: %w", ErrPolicyChanged, err)
		}
		return "", err
	}
	policy, _, err := c.fetchPolicy(ctx, session.Tailnet, session.APIKey)
	if err != nil {
		return "", err
	}
	c.cachePolicy(session, policy)
	return policy, nil
}

func (c *Client) ensureSession(ctx context.Context) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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

func (c *Client) fetchPolicy(ctx context.Context, tailnet, apiKey string) (policy, etag string, err error) {
	endpoint := c.baseURL + "/api/v2/tailnet/" + url.PathEscape(tailnet) + "/acl"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", "", err
	}
	req.SetBasicAuth(apiKey, "")
	req.Header.Set("Accept", "application/hujson, application/json, text/plain")

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logCloudAPI(http.MethodGet, tailnet, "", 0, time.Since(start), err)
		return "", "", fmt.Errorf("policy fetch failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Warn("close cloud api response body", "error", err)
		}
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		c.logCloudAPI(http.MethodGet, tailnet, "", resp.StatusCode, time.Since(start), err)
		return "", "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := apiError(resp.StatusCode, body, "policy fetch failed")
		c.logCloudAPI(http.MethodGet, tailnet, "", resp.StatusCode, time.Since(start), apiErr)
		return "", "", apiErr
	}
	c.logCloudAPI(http.MethodGet, tailnet, "", resp.StatusCode, time.Since(start), nil)
	return string(body), resp.Header.Get("ETag"), nil
}

func (c *Client) getJSON(ctx context.Context, session *Session, path string, query url.Values, target any) error {
	endpoint := c.baseURL + "/api/v2/tailnet/" + url.PathEscape(session.Tailnet) + path
	return c.getJSONEndpoint(ctx, session, endpoint, query, target)
}

func (c *Client) getJSONAPI(ctx context.Context, session *Session, path string, query url.Values, target any) error {
	endpoint := c.baseURL + "/api/v2" + path
	return c.getJSONEndpoint(ctx, session, endpoint, query, target)
}

func (c *Client) getJSONEndpoint(ctx context.Context, session *Session, endpoint string, query url.Values, target any) error {
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(session.APIKey, "")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Warn("close cloud api response body", "error", err)
		}
	}()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apiError(resp.StatusCode, body, "metadata request failed")
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode tailscale api response: %w", err)
	}
	return nil
}

func (c *Client) sendPolicy(ctx context.Context, method, tailnet, apiKey, suffix, policy, ifMatch string) error {
	endpoint := c.baseURL + "/api/v2/tailnet/" + url.PathEscape(tailnet) + "/acl" + suffix
	req, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(policy))
	if err != nil {
		return err
	}
	req.SetBasicAuth(apiKey, "")
	req.Header.Set("Content-Type", "application/hujson")
	req.Header.Set("Accept", "application/hujson, application/json, text/plain")
	if ifMatch != "" {
		req.Header.Set("If-Match", ifMatch)
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logCloudAPI(method, tailnet, suffix, 0, time.Since(start), err)
		return fmt.Errorf("policy request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Warn("close cloud api response body", "error", err)
		}
	}()

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
