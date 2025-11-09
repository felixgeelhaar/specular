package domain

import (
	"strings"
	"testing"
)

func TestNewTaskID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "valid simple ID",
			value:   "task-001",
			wantErr: false,
		},
		{
			name:    "valid ID with hyphen",
			value:   "implement-auth",
			wantErr: false,
		},
		{
			name:    "valid ID with multiple hyphens",
			value:   "implement-user-profile-api",
			wantErr: false,
		},
		{
			name:    "valid ID with numbers",
			value:   "task-123",
			wantErr: false,
		},
		{
			name:    "valid ID starts with letter",
			value:   "t123",
			wantErr: false,
		},
		{
			name:    "empty ID",
			value:   "",
			wantErr: true,
		},
		{
			name:    "ID starts with number",
			value:   "123-task",
			wantErr: true,
		},
		{
			name:    "ID starts with hyphen",
			value:   "-task",
			wantErr: true,
		},
		{
			name:    "ID ends with hyphen",
			value:   "task-",
			wantErr: true,
		},
		{
			name:    "ID with consecutive hyphens",
			value:   "task--001",
			wantErr: true,
		},
		{
			name:    "ID with uppercase letters",
			value:   "Task-001",
			wantErr: true,
		},
		{
			name:    "ID with special characters",
			value:   "task_001",
			wantErr: true,
		},
		{
			name:    "ID with spaces",
			value:   "task 001",
			wantErr: true,
		},
		{
			name:    "ID exceeds max length",
			value:   strings.Repeat("a", 101),
			wantErr: true,
		},
		{
			name:    "ID at max length",
			value:   strings.Repeat("a", 100),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTaskID(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTaskID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.value {
				t.Errorf("NewTaskID() = %v, want %v", got, tt.value)
			}
		})
	}
}

func TestTaskID_Validate(t *testing.T) {
	tests := []struct {
		name    string
		taskID  TaskID
		wantErr bool
	}{
		{"valid simple ID", TaskID("task-001"), false},
		{"valid with hyphens", TaskID("implement-user-profile"), false},
		{"valid with numbers", TaskID("task-123"), false},
		{"empty is invalid", TaskID(""), true},
		{"starts with number is invalid", TaskID("123-task"), true},
		{"ends with hyphen is invalid", TaskID("task-"), true},
		{"consecutive hyphens are invalid", TaskID("task--001"), true},
		{"uppercase is invalid", TaskID("Task"), true},
		{"too long is invalid", TaskID(strings.Repeat("a", 101)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.taskID.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("TaskID.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskID_String(t *testing.T) {
	tests := []struct {
		name   string
		taskID TaskID
		want   string
	}{
		{"simple ID", TaskID("task-001"), "task-001"},
		{"ID with hyphens", TaskID("implement-auth"), "implement-auth"},
		{"ID with numbers", TaskID("task-123"), "task-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.taskID.String(); got != tt.want {
				t.Errorf("TaskID.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTaskID_Equals(t *testing.T) {
	tests := []struct {
		name string
		id1  TaskID
		id2  TaskID
		want bool
	}{
		{"same IDs", TaskID("task-001"), TaskID("task-001"), true},
		{"different IDs", TaskID("task-001"), TaskID("task-002"), false},
		{"empty IDs", TaskID(""), TaskID(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id1.Equals(tt.id2); got != tt.want {
				t.Errorf("TaskID.Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}
