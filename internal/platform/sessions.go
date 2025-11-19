package platform

import (
	"fmt"
	"time"
)

// Session represents a platform AI session
type Session struct {
	ID            string                 `json:"id"`
	ProjectID     string                 `json:"project_id"`
	UserID        string                 `json:"user_id"`
	Title         string                 `json:"title"`
	Status        string                 `json:"status"`
	Provider      string                 `json:"provider"`
	Model         string                 `json:"model"`
	MessageCount  int                    `json:"message_count"`
	TokensUsed    int                    `json:"tokens_used"`
	EstimatedCost float64                `json:"estimated_cost"`
	Context       map[string]interface{} `json:"context"`
	Tags          []string               `json:"tags"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// Message represents a message in a session
type Message struct {
	ID               string     `json:"id"`
	SessionID        string     `json:"session_id"`
	Role             string     `json:"role"`
	Content          string     `json:"content"`
	PromptTokens     int        `json:"prompt_tokens"`
	CompletionTokens int        `json:"completion_tokens"`
	TotalTokens      int        `json:"total_tokens"`
	ToolCalls        []ToolCall `json:"tool_calls"`
	CreatedAt        time.Time  `json:"created_at"`
}

// ToolCall represents a tool call in a message
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// CreateSessionRequest represents a request to create a session
type CreateSessionRequest struct {
	ProjectID string                 `json:"project_id"`
	Title     string                 `json:"title"`
	Provider  string                 `json:"provider"`
	Model     string                 `json:"model"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
}

// SendMessageRequest represents a request to send a message
type SendMessageRequest struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

// SendMessageResponse represents the response to sending a message
type SendMessageResponse struct {
	UserMessage      Message `json:"user_message"`
	AssistantMessage Message `json:"assistant_message"`
	Session          Session `json:"session"`
}

// ListSessionsResponse represents a list of sessions
type ListSessionsResponse struct {
	Sessions   []Session `json:"sessions"`
	TotalCount int       `json:"total_count"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
}

// ListMessagesResponse represents a list of messages
type ListMessagesResponse struct {
	Messages   []Message `json:"messages"`
	TotalCount int       `json:"total_count"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
}

// CreateSession creates a new AI session
func (c *Client) CreateSession(projectID, title, provider, model string, context map[string]interface{}, tags []string) (*Session, error) {
	req := CreateSessionRequest{
		ProjectID: projectID,
		Title:     title,
		Provider:  provider,
		Model:     model,
		Context:   context,
		Tags:      tags,
	}

	resp, err := c.doRequest("POST", "/api/v1/sessions", req)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := parseResponse(resp, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// GetSession retrieves a session by ID
func (c *Client) GetSession(sessionID string) (*Session, error) {
	path := fmt.Sprintf("/api/v1/sessions/%s", sessionID)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := parseResponse(resp, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// ListSessions retrieves all sessions for a project
func (c *Client) ListSessions(projectID string, page, pageSize int) (*ListSessionsResponse, error) {
	path := fmt.Sprintf("/api/v1/sessions?project_id=%s&page=%d&page_size=%d", projectID, page, pageSize)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var sessions ListSessionsResponse
	if err := parseResponse(resp, &sessions); err != nil {
		return nil, err
	}

	return &sessions, nil
}

// SendMessage sends a message to a session and gets AI response
func (c *Client) SendMessage(sessionID, content string) (*SendMessageResponse, error) {
	req := SendMessageRequest{
		Content: content,
		Role:    "user",
	}

	path := fmt.Sprintf("/api/v1/sessions/%s/messages", sessionID)
	resp, err := c.doRequest("POST", path, req)
	if err != nil {
		return nil, err
	}

	var messageResp SendMessageResponse
	if err := parseResponse(resp, &messageResp); err != nil {
		return nil, err
	}

	return &messageResp, nil
}

// ListMessages retrieves all messages for a session
func (c *Client) ListMessages(sessionID string, page, pageSize int) (*ListMessagesResponse, error) {
	path := fmt.Sprintf("/api/v1/sessions/%s/messages?page=%d&page_size=%d", sessionID, page, pageSize)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var messages ListMessagesResponse
	if err := parseResponse(resp, &messages); err != nil {
		return nil, err
	}

	return &messages, nil
}

// PauseSession pauses an active session
func (c *Client) PauseSession(sessionID string) (*Session, error) {
	path := fmt.Sprintf("/api/v1/sessions/%s/pause", sessionID)
	resp, err := c.doRequest("POST", path, nil)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := parseResponse(resp, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// ResumeSession resumes a paused session
func (c *Client) ResumeSession(sessionID string) (*Session, error) {
	path := fmt.Sprintf("/api/v1/sessions/%s/resume", sessionID)
	resp, err := c.doRequest("POST", path, nil)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := parseResponse(resp, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// CompleteSession marks a session as completed
func (c *Client) CompleteSession(sessionID string) (*Session, error) {
	path := fmt.Sprintf("/api/v1/sessions/%s/complete", sessionID)
	resp, err := c.doRequest("POST", path, nil)
	if err != nil {
		return nil, err
	}

	var session Session
	if err := parseResponse(resp, &session); err != nil {
		return nil, err
	}

	return &session, nil
}
