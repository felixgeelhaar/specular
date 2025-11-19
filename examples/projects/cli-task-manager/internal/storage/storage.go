// Package storage provides task persistence using JSON files.
package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/felixgeelhaar/specular/examples/cli-task-manager/internal/task"
)

// Storage defines the interface for task persistence.
type Storage interface {
	List() ([]task.Task, error)
	Get(id int) (*task.Task, error)
	Create(t *task.Task) error
	Update(t *task.Task) error
	Delete(id int) error
}

// FileStorage implements Storage using JSON files.
type FileStorage struct {
	path string
	mu   sync.RWMutex
}

// NewFileStorage creates a new file-based storage.
func NewFileStorage(path string) (*FileStorage, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	return &FileStorage{path: path}, nil
}

// List returns all tasks.
func (s *FileStorage) List() ([]task.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks, err := s.load()
	if err != nil {
		return nil, err
	}

	// Sort by created date, newest first
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})

	return tasks, nil
}

// Get returns a task by ID.
func (s *FileStorage) Get(id int) (*task.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks, err := s.load()
	if err != nil {
		return nil, err
	}

	for _, t := range tasks {
		if t.ID == id {
			return &t, nil
		}
	}

	return nil, errors.New("task not found")
}

// Create adds a new task and assigns an ID.
func (s *FileStorage) Create(t *task.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks, err := s.load()
	if err != nil {
		return err
	}

	// Assign next ID
	maxID := 0
	for _, existing := range tasks {
		if existing.ID > maxID {
			maxID = existing.ID
		}
	}
	t.ID = maxID + 1

	tasks = append(tasks, *t)
	return s.save(tasks)
}

// Update modifies an existing task.
func (s *FileStorage) Update(t *task.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks, err := s.load()
	if err != nil {
		return err
	}

	found := false
	for i, existing := range tasks {
		if existing.ID == t.ID {
			tasks[i] = *t
			found = true
			break
		}
	}

	if !found {
		return errors.New("task not found")
	}

	return s.save(tasks)
}

// Delete removes a task by ID.
func (s *FileStorage) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks, err := s.load()
	if err != nil {
		return err
	}

	newTasks := make([]task.Task, 0, len(tasks))
	found := false
	for _, t := range tasks {
		if t.ID == id {
			found = true
			continue
		}
		newTasks = append(newTasks, t)
	}

	if !found {
		return errors.New("task not found")
	}

	return s.save(newTasks)
}

// load reads tasks from the JSON file.
func (s *FileStorage) load() ([]task.Task, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []task.Task{}, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return []task.Task{}, nil
	}

	var tasks []task.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

// save writes tasks to the JSON file atomically.
func (s *FileStorage) save(tasks []task.Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file first for atomic operation
	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}

	// Rename for atomic update
	return os.Rename(tmpPath, s.path)
}
