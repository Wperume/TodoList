package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"todolist-api/internal/models"

	"github.com/google/uuid"
)

// Client is a type-safe HTTP client for the TodoList API
type Client struct {
	baseURL          string
	httpClient       *http.Client
	token            string // Bearer token for authentication
	refreshToken     string // Refresh token for automatic renewal
	onTokenRefreshed func(accessToken, refreshToken string) error // Callback when tokens are refreshed
}

// Config holds the client configuration
type Config struct {
	BaseURL            string
	Token              string
	RefreshToken       string
	InsecureSkipVerify bool // Skip TLS verification (for self-signed certs)
	Timeout            time.Duration
	OnTokenRefreshed   func(accessToken, refreshToken string) error // Callback when tokens are refreshed
}

// NewClient creates a new API client
func NewClient(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		baseURL:          cfg.BaseURL,
		token:            cfg.Token,
		refreshToken:     cfg.RefreshToken,
		onTokenRefreshed: cfg.OnTokenRefreshed,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: cfg.InsecureSkipVerify,
				},
			},
		},
	}
}

// SetToken updates the authentication token
func (c *Client) SetToken(token string) {
	c.token = token
}

// GetToken returns the current authentication token
func (c *Client) GetToken() string {
	return c.token
}

// doRequest performs an HTTP request with proper error handling and automatic token refresh
func (c *Client) doRequest(method, path string, body interface{}, auth bool) (*http.Response, error) {
	return c.doRequestWithRetry(method, path, body, auth, true)
}

// doRequestWithRetry performs an HTTP request with optional retry on token expiration
func (c *Client) doRequestWithRetry(method, path string, body interface{}, auth bool, retryOnTokenExpired bool) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if auth && c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	// Check for TOKEN_EXPIRED error and attempt refresh if enabled
	if retryOnTokenExpired && resp.StatusCode == http.StatusUnauthorized {
		// Peek at the response to check for TOKEN_EXPIRED
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var errResp models.ErrorResponse
		if err := json.Unmarshal(bodyBytes, &errResp); err == nil && errResp.Code == "TOKEN_EXPIRED" {
			// Attempt to refresh the token
			if c.refreshToken != "" {
				if err := c.attemptTokenRefresh(); err == nil {
					// Retry the request with the new token (without further retry)
					return c.doRequestWithRetry(method, path, body, auth, false)
				}
			}
		}

		// Restore the response body for normal error handling
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	return resp, nil
}

// attemptTokenRefresh attempts to refresh the access token using the refresh token
func (c *Client) attemptTokenRefresh() error {
	if c.refreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	// Call the refresh endpoint
	authResp, err := c.RefreshToken(c.refreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update internal tokens
	c.token = authResp.AccessToken
	c.refreshToken = authResp.RefreshToken

	// Call the callback to persist the new tokens
	if c.onTokenRefreshed != nil {
		if err := c.onTokenRefreshed(authResp.AccessToken, authResp.RefreshToken); err != nil {
			return fmt.Errorf("failed to persist refreshed tokens: %w", err)
		}
	}

	return nil
}

// parseResponse parses the HTTP response into the target type
func parseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for error responses
	if resp.StatusCode >= 400 {
		var errResp models.ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
		}
		return fmt.Errorf("%s: %s", errResp.Code, errResp.Message)
	}

	if target != nil {
		if err := json.Unmarshal(body, target); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// ============================
// Authentication Methods
// ============================

// Register creates a new user account
func (c *Client) Register(req models.RegisterRequest) (*models.AuthResponse, error) {
	resp, err := c.doRequest(http.MethodPost, "/auth/register", req, false)
	if err != nil {
		return nil, err
	}

	var authResp models.AuthResponse
	if err := parseResponse(resp, &authResp); err != nil {
		return nil, err
	}

	// Automatically set the token for subsequent requests
	c.token = authResp.AccessToken

	return &authResp, nil
}

// Login authenticates a user and returns tokens
func (c *Client) Login(req models.LoginRequest) (*models.AuthResponse, error) {
	resp, err := c.doRequest(http.MethodPost, "/auth/login", req, false)
	if err != nil {
		return nil, err
	}

	var authResp models.AuthResponse
	if err := parseResponse(resp, &authResp); err != nil {
		return nil, err
	}

	// Automatically set the token for subsequent requests
	c.token = authResp.AccessToken

	return &authResp, nil
}

// RefreshToken refreshes the access token using a refresh token
func (c *Client) RefreshToken(refreshToken string) (*models.AuthResponse, error) {
	req := models.RefreshTokenRequest{RefreshToken: refreshToken}
	resp, err := c.doRequest(http.MethodPost, "/auth/refresh", req, false)
	if err != nil {
		return nil, err
	}

	var authResp models.AuthResponse
	if err := parseResponse(resp, &authResp); err != nil {
		return nil, err
	}

	// Automatically update the token
	c.token = authResp.AccessToken

	return &authResp, nil
}

// Logout revokes the current refresh token
func (c *Client) Logout(refreshToken string) error {
	req := models.RefreshTokenRequest{RefreshToken: refreshToken}
	resp, err := c.doRequest(http.MethodPost, "/auth/logout", req, true)
	if err != nil {
		return err
	}

	return parseResponse(resp, nil)
}

// GetProfile retrieves the current user's profile
func (c *Client) GetProfile() (*models.UserInfo, error) {
	resp, err := c.doRequest(http.MethodGet, "/auth/profile", nil, true)
	if err != nil {
		return nil, err
	}

	var user models.UserInfo
	if err := parseResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateProfile updates the current user's profile
func (c *Client) UpdateProfile(req models.UpdateProfileRequest) (*models.UserInfo, error) {
	resp, err := c.doRequest(http.MethodPut, "/auth/profile", req, true)
	if err != nil {
		return nil, err
	}

	var user models.UserInfo
	if err := parseResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// ChangePassword changes the current user's password
func (c *Client) ChangePassword(req models.ChangePasswordRequest) error {
	resp, err := c.doRequest(http.MethodPut, "/auth/password", req, true)
	if err != nil {
		return err
	}

	return parseResponse(resp, nil)
}

// ============================
// TodoList Methods
// ============================

// GetLists retrieves all todo lists for the current user
func (c *Client) GetLists() ([]models.TodoList, error) {
	resp, err := c.doRequest(http.MethodGet, "/lists", nil, true)
	if err != nil {
		return nil, err
	}

	var paginatedResp models.PaginatedListsResponse
	if err := parseResponse(resp, &paginatedResp); err != nil {
		return nil, err
	}

	return paginatedResp.Data, nil
}

// CreateList creates a new todo list
func (c *Client) CreateList(req models.CreateTodoListRequest) (*models.TodoList, error) {
	resp, err := c.doRequest(http.MethodPost, "/lists", req, true)
	if err != nil {
		return nil, err
	}

	var list models.TodoList
	if err := parseResponse(resp, &list); err != nil {
		return nil, err
	}

	return &list, nil
}

// GetList retrieves a specific todo list by ID
func (c *Client) GetList(listID uuid.UUID) (*models.TodoList, error) {
	path := fmt.Sprintf("/lists/%s", listID)
	resp, err := c.doRequest(http.MethodGet, path, nil, true)
	if err != nil {
		return nil, err
	}

	var list models.TodoList
	if err := parseResponse(resp, &list); err != nil {
		return nil, err
	}

	return &list, nil
}

// UpdateList updates a todo list
func (c *Client) UpdateList(listID uuid.UUID, req models.UpdateTodoListRequest) (*models.TodoList, error) {
	path := fmt.Sprintf("/lists/%s", listID)
	resp, err := c.doRequest(http.MethodPut, path, req, true)
	if err != nil {
		return nil, err
	}

	var list models.TodoList
	if err := parseResponse(resp, &list); err != nil {
		return nil, err
	}

	return &list, nil
}

// DeleteList deletes a todo list
func (c *Client) DeleteList(listID uuid.UUID) error {
	path := fmt.Sprintf("/lists/%s", listID)
	resp, err := c.doRequest(http.MethodDelete, path, nil, true)
	if err != nil {
		return err
	}

	return parseResponse(resp, nil)
}

// ============================
// Todo Methods
// ============================

// GetTodos retrieves all todos in a list
func (c *Client) GetTodos(listID uuid.UUID) ([]models.Todo, error) {
	return c.GetTodosWithFilter(listID, nil, nil, nil, "", "")
}

// GetTodosWithFilter retrieves todos in a list with optional filters
func (c *Client) GetTodosWithFilter(listID uuid.UUID, priority *models.Priority, completed, flagged *bool, sortBy, sortOrder string) ([]models.Todo, error) {
	path := fmt.Sprintf("/lists/%s/todos", listID)

	// Build query parameters
	query := ""
	params := []string{}

	if priority != nil {
		params = append(params, fmt.Sprintf("priority=%s", *priority))
	}
	if completed != nil {
		params = append(params, fmt.Sprintf("completed=%t", *completed))
	}
	if flagged != nil {
		params = append(params, fmt.Sprintf("flagged=%t", *flagged))
	}
	if sortBy != "" {
		params = append(params, fmt.Sprintf("sortBy=%s", sortBy))
	}
	if sortOrder != "" {
		params = append(params, fmt.Sprintf("sortOrder=%s", sortOrder))
	}

	if len(params) > 0 {
		query = "?" + params[0]
		for i := 1; i < len(params); i++ {
			query += "&" + params[i]
		}
	}

	resp, err := c.doRequest(http.MethodGet, path+query, nil, true)
	if err != nil {
		return nil, err
	}

	var todos []models.Todo
	if err := parseResponse(resp, &todos); err != nil {
		return nil, err
	}

	return todos, nil
}

// CreateTodo creates a new todo in a list
func (c *Client) CreateTodo(listID uuid.UUID, req models.CreateTodoRequest) (*models.Todo, error) {
	path := fmt.Sprintf("/lists/%s/todos", listID)
	resp, err := c.doRequest(http.MethodPost, path, req, true)
	if err != nil {
		return nil, err
	}

	var todo models.Todo
	if err := parseResponse(resp, &todo); err != nil {
		return nil, err
	}

	return &todo, nil
}

// GetTodo retrieves a specific todo by ID
func (c *Client) GetTodo(listID, todoID uuid.UUID) (*models.Todo, error) {
	path := fmt.Sprintf("/lists/%s/todos/%s", listID, todoID)
	resp, err := c.doRequest(http.MethodGet, path, nil, true)
	if err != nil {
		return nil, err
	}

	var todo models.Todo
	if err := parseResponse(resp, &todo); err != nil {
		return nil, err
	}

	return &todo, nil
}

// UpdateTodo updates a todo
func (c *Client) UpdateTodo(listID, todoID uuid.UUID, req models.UpdateTodoRequest) (*models.Todo, error) {
	path := fmt.Sprintf("/lists/%s/todos/%s", listID, todoID)
	resp, err := c.doRequest(http.MethodPut, path, req, true)
	if err != nil {
		return nil, err
	}

	var todo models.Todo
	if err := parseResponse(resp, &todo); err != nil {
		return nil, err
	}

	return &todo, nil
}

// DeleteTodo deletes a todo
func (c *Client) DeleteTodo(listID, todoID uuid.UUID) error {
	path := fmt.Sprintf("/lists/%s/todos/%s", listID, todoID)
	resp, err := c.doRequest(http.MethodDelete, path, nil, true)
	if err != nil {
		return err
	}

	return parseResponse(resp, nil)
}

// ============================
// Health Check Methods
// ============================

// HealthResponse represents the basic health check response
type HealthResponse struct {
	Status string `json:"status"`
}

// HealthCheck performs a basic health check
func (c *Client) HealthCheck() (*HealthResponse, error) {
	resp, err := c.doRequest(http.MethodGet, "/../health", nil, false)
	if err != nil {
		return nil, err
	}

	var health HealthResponse
	if err := parseResponse(resp, &health); err != nil {
		return nil, err
	}

	return &health, nil
}
