// Package task defines the task domain model and operations.
package task

import (
	"errors"
	"time"
)

// Priority represents task priority levels.
type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

// Status represents task completion status.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
)

// Task represents a single task item.
type Task struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Priority    Priority   `json:"priority"`
	Status      Status     `json:"status"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// Validate checks if the task has valid data.
func (t *Task) Validate() error {
	if t.Title == "" {
		return errors.New("title is required")
	}
	if len(t.Title) > 200 {
		return errors.New("title must be 200 characters or less")
	}
	return nil
}

// Complete marks the task as completed.
func (t *Task) Complete() {
	now := time.Now()
	t.Status = StatusCompleted
	t.CompletedAt = &now
}

// IsOverdue checks if the task is past its due date.
func (t *Task) IsOverdue() bool {
	if t.DueDate == nil || t.Status == StatusCompleted {
		return false
	}
	return time.Now().After(*t.DueDate)
}

// NewTask creates a new task with default values.
func NewTask(title string, priority Priority) *Task {
	return &Task{
		Title:     title,
		Priority:  priority,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}
}

// ParsePriority converts a string to a Priority.
func ParsePriority(s string) (Priority, error) {
	switch s {
	case "low":
		return PriorityLow, nil
	case "medium":
		return PriorityMedium, nil
	case "high":
		return PriorityHigh, nil
	default:
		return "", errors.New("invalid priority: must be low, medium, or high")
	}
}
