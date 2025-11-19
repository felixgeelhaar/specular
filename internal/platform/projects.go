package platform

import (
	"fmt"
	"time"
)

// Project represents a platform project
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     string    `json:"owner_id"`
	Status      string    `json:"status"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateProjectRequest represents a request to create a project
type CreateProjectRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Visibility  string                 `json:"visibility"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ListProjectsResponse represents a list of projects
type ListProjectsResponse struct {
	Projects   []Project `json:"projects"`
	TotalCount int       `json:"total_count"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
}

// CreateProject creates a new project
func (c *Client) CreateProject(name, description, visibility string, metadata map[string]interface{}) (*Project, error) {
	req := CreateProjectRequest{
		Name:        name,
		Description: description,
		Visibility:  visibility,
		Metadata:    metadata,
	}

	resp, err := c.doRequest("POST", "/api/v1/projects", req)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := parseResponse(resp, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// GetProject retrieves a project by ID
func (c *Client) GetProject(projectID string) (*Project, error) {
	path := fmt.Sprintf("/api/v1/projects/%s", projectID)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := parseResponse(resp, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// ListProjects retrieves all projects for the authenticated user
func (c *Client) ListProjects(page, pageSize int) (*ListProjectsResponse, error) {
	path := fmt.Sprintf("/api/v1/projects?page=%d&page_size=%d", page, pageSize)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var projects ListProjectsResponse
	if err := parseResponse(resp, &projects); err != nil {
		return nil, err
	}

	return &projects, nil
}

// UpdateProject updates an existing project
func (c *Client) UpdateProject(projectID, name, description string, metadata map[string]interface{}) (*Project, error) {
	req := CreateProjectRequest{
		Name:        name,
		Description: description,
		Metadata:    metadata,
	}

	path := fmt.Sprintf("/api/v1/projects/%s", projectID)
	resp, err := c.doRequest("PUT", path, req)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := parseResponse(resp, &project); err != nil {
		return nil, err
	}

	return &project, nil
}

// DeleteProject deletes a project
func (c *Client) DeleteProject(projectID string) error {
	path := fmt.Sprintf("/api/v1/projects/%s", projectID)
	resp, err := c.doRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to delete project: status %d", resp.StatusCode)
	}

	return nil
}
