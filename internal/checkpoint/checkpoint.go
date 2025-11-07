package checkpoint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State represents the checkpoint state for a long-running operation
type State struct {
	Version     string            `json:"version"`
	OperationID string            `json:"operation_id"`
	StartedAt   time.Time         `json:"started_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Status      string            `json:"status"` // running, completed, failed
	Tasks       map[string]Task   `json:"tasks"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Task represents the state of an individual task
type Task struct {
	ID          string    `json:"id"`
	Status      string    `json:"status"` // pending, running, completed, failed, skipped
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
	Attempts    int       `json:"attempts"`
	Artifacts   []string  `json:"artifacts,omitempty"`
}

// Manager handles checkpoint persistence and recovery
type Manager struct {
	checkpointDir string
	autoSave      bool
	saveInterval  time.Duration
}

// NewManager creates a new checkpoint manager
func NewManager(checkpointDir string, autoSave bool, saveInterval time.Duration) *Manager {
	return &Manager{
		checkpointDir: checkpointDir,
		autoSave:      autoSave,
		saveInterval:  saveInterval,
	}
}

// NewState creates a new checkpoint state
func NewState(operationID string) *State {
	now := time.Now()
	return &State{
		Version:     "1.0",
		OperationID: operationID,
		StartedAt:   now,
		UpdatedAt:   now,
		Status:      "running",
		Tasks:       make(map[string]Task),
		Metadata:    make(map[string]string),
	}
}

// Save persists the checkpoint state to disk
func (m *Manager) Save(state *State) error {
	if state == nil {
		return fmt.Errorf("checkpoint state is nil")
	}

	// Update timestamp
	state.UpdatedAt = time.Now()

	// Create checkpoint directory if it doesn't exist
	if err := os.MkdirAll(m.checkpointDir, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	// Write checkpoint file
	checkpointPath := filepath.Join(m.checkpointDir, fmt.Sprintf("%s.json", state.OperationID))
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint state: %w", err)
	}

	if err := os.WriteFile(checkpointPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint file: %w", err)
	}

	return nil
}

// Load reads the checkpoint state from disk
func (m *Manager) Load(operationID string) (*State, error) {
	checkpointPath := filepath.Join(m.checkpointDir, fmt.Sprintf("%s.json", operationID))

	data, err := os.ReadFile(checkpointPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("checkpoint not found: %s", operationID)
		}
		return nil, fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal checkpoint state: %w", err)
	}

	return &state, nil
}

// Exists checks if a checkpoint exists for the given operation ID
func (m *Manager) Exists(operationID string) bool {
	checkpointPath := filepath.Join(m.checkpointDir, fmt.Sprintf("%s.json", operationID))
	_, err := os.Stat(checkpointPath)
	return err == nil
}

// Delete removes a checkpoint file
func (m *Manager) Delete(operationID string) error {
	checkpointPath := filepath.Join(m.checkpointDir, fmt.Sprintf("%s.json", operationID))
	if err := os.Remove(checkpointPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}
	return nil
}

// List returns all checkpoint operation IDs
func (m *Manager) List() ([]string, error) {
	entries, err := os.ReadDir(m.checkpointDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read checkpoint directory: %w", err)
	}

	var operationIDs []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			// Remove .json extension to get operation ID
			operationID := entry.Name()[:len(entry.Name())-5]
			operationIDs = append(operationIDs, operationID)
		}
	}

	return operationIDs, nil
}

// UpdateTask updates or creates a task in the checkpoint state
func (s *State) UpdateTask(taskID, status string, err error) {
	task, exists := s.Tasks[taskID]
	if !exists {
		task = Task{
			ID:       taskID,
			Status:   "pending",
			Attempts: 0,
		}
	}

	// Update status
	oldStatus := task.Status
	task.Status = status

	// Track timing
	now := time.Now()
	if oldStatus == "pending" && status == "running" {
		task.StartedAt = now
		task.Attempts++
	}
	if status == "completed" || status == "failed" || status == "skipped" {
		task.CompletedAt = now
	}

	// Store error if provided
	if err != nil {
		task.Error = err.Error()
	}

	s.Tasks[taskID] = task
	s.UpdatedAt = now
}

// GetPendingTasks returns all tasks that haven't been completed
func (s *State) GetPendingTasks() []string {
	var pending []string
	for id, task := range s.Tasks {
		if task.Status == "pending" || task.Status == "running" {
			pending = append(pending, id)
		}
	}
	return pending
}

// GetCompletedTasks returns all successfully completed tasks
func (s *State) GetCompletedTasks() []string {
	var completed []string
	for id, task := range s.Tasks {
		if task.Status == "completed" {
			completed = append(completed, id)
		}
	}
	return completed
}

// GetFailedTasks returns all failed tasks
func (s *State) GetFailedTasks() []string {
	var failed []string
	for id, task := range s.Tasks {
		if task.Status == "failed" {
			failed = append(failed, id)
		}
	}
	return failed
}

// IsComplete returns true if all tasks are completed or skipped
func (s *State) IsComplete() bool {
	for _, task := range s.Tasks {
		if task.Status != "completed" && task.Status != "skipped" {
			return false
		}
	}
	return len(s.Tasks) > 0
}

// Progress returns the completion percentage (0.0 to 1.0)
func (s *State) Progress() float64 {
	if len(s.Tasks) == 0 {
		return 0.0
	}

	completed := 0
	for _, task := range s.Tasks {
		if task.Status == "completed" || task.Status == "skipped" {
			completed++
		}
	}

	return float64(completed) / float64(len(s.Tasks))
}

// AddArtifact adds an artifact path to a task
func (s *State) AddArtifact(taskID, artifactPath string) {
	task, exists := s.Tasks[taskID]
	if !exists {
		return
	}

	task.Artifacts = append(task.Artifacts, artifactPath)
	s.Tasks[taskID] = task
	s.UpdatedAt = time.Now()
}

// SetMetadata sets a metadata key-value pair
func (s *State) SetMetadata(key, value string) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	s.Metadata[key] = value
	s.UpdatedAt = time.Now()
}

// GetMetadata retrieves a metadata value
func (s *State) GetMetadata(key string) (string, bool) {
	if s.Metadata == nil {
		return "", false
	}
	value, ok := s.Metadata[key]
	return value, ok
}
