package platform

import (
	"fmt"
	"time"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         User      `json:"user"`
}

// User represents a platform user
type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

// Login authenticates with the platform and returns tokens
func (c *Client) Login(email, password string) (*LoginResponse, error) {
	req := LoginRequest{
		Email:    email,
		Password: password,
	}

	resp, err := c.doRequest("POST", "/api/v1/auth/login", req)
	if err != nil {
		return nil, err
	}

	var loginResp LoginResponse
	if err := parseResponse(resp, &loginResp); err != nil {
		return nil, err
	}

	// Automatically set the token for future requests
	c.SetToken(loginResp.AccessToken)

	return &loginResp, nil
}

// Register creates a new user account and automatically logs in
func (c *Client) Register(username, email, password, firstName, lastName string) (*LoginResponse, error) {
	req := map[string]string{
		"username":   username,
		"email":      email,
		"password":   password,
		"first_name": firstName,
		"last_name":  lastName,
	}

	// Step 1: Register the user
	resp, err := c.doRequest("POST", "/api/v1/auth/register", req)
	if err != nil {
		return nil, err
	}

	var user User
	if err := parseResponse(resp, &user); err != nil {
		return nil, err
	}

	// Step 2: Automatically login to get tokens
	loginResp, err := c.Login(email, password)
	if err != nil {
		return nil, fmt.Errorf("registration succeeded but login failed: %w", err)
	}

	return loginResp, nil
}

// GetCurrentUser retrieves the currently authenticated user
func (c *Client) GetCurrentUser() (*User, error) {
	resp, err := c.doRequest("GET", "/api/v1/users/me", nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := parseResponse(resp, &user); err != nil {
		return nil, err
	}

	return &user, nil
}
