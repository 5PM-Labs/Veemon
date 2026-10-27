package main

// This file demonstrates the REST PASETO authentication flow.
// Run with: go run ./examples

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
}

type FlowInput struct {
	Email    string
	Password string
	Name     string
	Phone    string
	Register bool
}

type FlowResult struct {
	Registered     RegisterResult
	LoginToken     string
	RefreshedToken string
	CurrentUser    UserProfile
	Users          []UserProfile
	Pagination     Pagination
	LogoutMessage  string
}

type RegisterResult struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type LoginResult struct {
	Token string      `json:"token"`
	User  UserProfile `json:"user"`
}

type RefreshTokenResult struct {
	Token string `json:"token"`
}

type LogoutResult struct {
	Message string `json:"message"`
}

type UserProfile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type Pagination struct {
	Page       int   `json:"page"`
	Size       int   `json:"size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

type ListUsersParams struct {
	Page      int
	Size      int
	Search    string
	SortBy    string
	SortOrder string
}

type apiResponse[T any] struct {
	Success bool       `json:"success"`
	Data    T          `json:"data"`
	Meta    Pagination `json:"meta"`
	Error   apiError   `json:"error"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func RunAuthenticatedFlow(ctx context.Context, client *APIClient, input FlowInput) (*FlowResult, error) {
	result := &FlowResult{}

	if input.Register {
		registered, err := client.Register(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("register user: %w", err)
		}
		result.Registered = registered
	}

	login, err := client.Login(ctx, input.Email, input.Password)
	if err != nil {
		return nil, fmt.Errorf("login user: %w", err)
	}
	result.LoginToken = login.Token

	me, err := client.GetMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current user: %w", err)
	}
	result.CurrentUser = me

	refreshedToken, err := client.RefreshToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}
	result.RefreshedToken = refreshedToken

	users, pagination, err := client.ListUsers(ctx, ListUsersParams{
		Page:      1,
		Size:      10,
		SortBy:    "created_at",
		SortOrder: "desc",
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	result.Users = users
	result.Pagination = pagination

	logoutMessage, err := client.Logout(ctx)
	if err != nil {
		return nil, fmt.Errorf("logout user: %w", err)
	}
	result.LogoutMessage = logoutMessage

	return result, nil
}

func (c *APIClient) Register(ctx context.Context, input FlowInput) (RegisterResult, error) {
	body := map[string]string{
		"email":    input.Email,
		"password": input.Password,
		"name":     input.Name,
		"phone":    input.Phone,
	}

	var resp apiResponse[RegisterResult]
	err := c.do(ctx, http.MethodPost, "/api/v1/auth/register", nil, body, &resp)
	return resp.Data, err
}

func (c *APIClient) Login(ctx context.Context, email, password string) (LoginResult, error) {
	body := map[string]string{
		"email":    email,
		"password": password,
	}

	var resp apiResponse[LoginResult]
	if err := c.do(ctx, http.MethodPost, "/api/v1/auth/login", nil, body, &resp); err != nil {
		return LoginResult{}, err
	}

	c.Token = resp.Data.Token
	return resp.Data, nil
}

func (c *APIClient) GetMe(ctx context.Context) (UserProfile, error) {
	var resp apiResponse[UserProfile]
	err := c.do(ctx, http.MethodGet, "/api/v1/auth/me", nil, nil, &resp)
	return resp.Data, err
}

func (c *APIClient) RefreshToken(ctx context.Context) (string, error) {
	var resp apiResponse[RefreshTokenResult]
	if err := c.do(ctx, http.MethodPost, "/api/v1/auth/refresh", nil, nil, &resp); err != nil {
		return "", err
	}

	c.Token = resp.Data.Token
	return resp.Data.Token, nil
}

func (c *APIClient) ListUsers(ctx context.Context, params ListUsersParams) ([]UserProfile, Pagination, error) {
	query := url.Values{}
	query.Set("page", fmt.Sprintf("%d", valueOrDefault(params.Page, 1)))
	query.Set("size", fmt.Sprintf("%d", valueOrDefault(params.Size, 10)))
	query.Set("sortBy", defaultString(params.SortBy, "created_at"))
	query.Set("sortOrder", defaultString(params.SortOrder, "desc"))
	if params.Search != "" {
		query.Set("search", params.Search)
	}

	var resp apiResponse[[]UserProfile]
	if err := c.do(ctx, http.MethodGet, "/api/v1/users/", query, nil, &resp); err != nil {
		return nil, Pagination{}, err
	}

	return resp.Data, resp.Meta, nil
}

func (c *APIClient) Logout(ctx context.Context) (string, error) {
	var resp apiResponse[LogoutResult]
	if err := c.do(ctx, http.MethodPost, "/api/v1/auth/logout", nil, nil, &resp); err != nil {
		return "", err
	}

	return resp.Data.Message, nil
}

func (c *APIClient) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var requestBody io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		requestBody = bytes.NewReader(payload)
	}

	endpoint := c.BaseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, requestBody)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%s %s failed with HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return err
	}

	if failure, ok := out.(interface{ failed() (bool, string) }); ok {
		if failed, message := failure.failed(); failed {
			return fmt.Errorf("%s %s failed: %s", method, path, message)
		}
	}

	return nil
}

func (r apiResponse[T]) failed() (bool, string) {
	if r.Success {
		return false, ""
	}
	if r.Error.Message != "" {
		return true, r.Error.Message
	}
	return true, "response success=false"
}

func valueOrDefault(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func main() {
	baseURL := flag.String("base-url", "http://localhost:3000", "API base URL")
	email := flag.String("email", "admin@example.com", "user email")
	password := flag.String("password", "Admin123!", "user password")
	name := flag.String("name", "Demo Admin", "name used when -register is enabled")
	phone := flag.String("phone", "081234567890", "phone used when -register is enabled")
	register := flag.Bool("register", false, "call /api/v1/auth/register before login")
	flag.Parse()

	result, err := RunAuthenticatedFlow(context.Background(), NewAPIClient(*baseURL), FlowInput{
		Email:    *email,
		Password: *password,
		Name:     *name,
		Phone:    *phone,
		Register: *register,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("PASETO authenticated flow completed")
	if result.Registered.ID != "" {
		fmt.Printf("Registered: %s <%s>\n", result.Registered.Name, result.Registered.Email)
	}
	fmt.Printf("Login token: %s\n", maskToken(result.LoginToken))
	fmt.Printf("Current user: %s <%s>\n", result.CurrentUser.Name, result.CurrentUser.Email)
	fmt.Printf("Refreshed token: %s\n", maskToken(result.RefreshedToken))
	fmt.Printf("Users page %d/%d: %d of %d total\n",
		result.Pagination.Page,
		result.Pagination.TotalPages,
		len(result.Users),
		result.Pagination.Total,
	)
	for _, user := range result.Users {
		fmt.Printf("- %s <%s> [%s]\n", user.Name, user.Email, user.Status)
	}
	fmt.Printf("Logout: %s\n", result.LogoutMessage)
}

func maskToken(token string) string {
	if len(token) <= 20 {
		return token
	}
	return token[:12] + "..." + token[len(token)-6:]
}
