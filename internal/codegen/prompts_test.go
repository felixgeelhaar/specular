package codegen

import (
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

func TestNewPromptBuilder(t *testing.T) {
	pb := NewPromptBuilder()
	if pb == nil {
		t.Fatal("NewPromptBuilder() returned nil")
	}

	// Should have system prompts for known skills
	skills := []string{"go-backend", "ui-react", "database", "testing", "infra", "default"}
	for _, skill := range skills {
		prompt := pb.getSystemPrompt(skill)
		if prompt == "" {
			t.Errorf("No system prompt for skill %q", skill)
		}
	}
}

func TestPromptBuilder_BuildPrompt_GoBackend(t *testing.T) {
	pb := NewPromptBuilder()

	req := CodeGenerationRequest{
		FeatureID:    types.FeatureID("feat-001"),
		FeatureTitle: "User Authentication",
		FeatureDesc:  "Implement JWT-based user authentication",
		SuccessCriteria: []string{
			"Users can register with email and password",
			"Users can log in and receive JWT token",
			"Protected routes require valid JWT",
		},
		Skill:          "go-backend",
		ProductContext: "Product: TaskManager\nGoals: [Build a task management app]",
		APIs: []APIEndpoint{
			{Method: "POST", Path: "/api/auth/register", Request: "RegisterRequest", Response: "UserResponse"},
			{Method: "POST", Path: "/api/auth/login", Request: "LoginRequest", Response: "TokenResponse"},
		},
		Priority: types.Priority("P0"),
	}

	systemPrompt, userPrompt := pb.BuildPrompt(req)

	// Check system prompt
	if !strings.Contains(systemPrompt, "Go developer") {
		t.Error("System prompt should mention Go developer")
	}
	if !strings.Contains(systemPrompt, "Clean Architecture") {
		t.Error("System prompt should mention Clean Architecture")
	}
	if !strings.Contains(systemPrompt, "```go:") {
		t.Error("System prompt should include output format example")
	}

	// Check user prompt
	if !strings.Contains(userPrompt, "feat-001") {
		t.Error("User prompt should include feature ID")
	}
	if !strings.Contains(userPrompt, "User Authentication") {
		t.Error("User prompt should include feature title")
	}
	if !strings.Contains(userPrompt, "JWT-based") {
		t.Error("User prompt should include feature description")
	}
	if !strings.Contains(userPrompt, "Users can register") {
		t.Error("User prompt should include success criteria")
	}
	if !strings.Contains(userPrompt, "POST") {
		t.Error("User prompt should include API endpoints")
	}
	if !strings.Contains(userPrompt, "/api/auth/register") {
		t.Error("User prompt should include API paths")
	}
	if !strings.Contains(userPrompt, "TaskManager") {
		t.Error("User prompt should include product context")
	}
}

func TestPromptBuilder_BuildPrompt_UIReact(t *testing.T) {
	pb := NewPromptBuilder()

	req := CodeGenerationRequest{
		FeatureID:       types.FeatureID("feat-002"),
		FeatureTitle:    "Task List Component",
		FeatureDesc:     "Display a list of tasks with filtering",
		SuccessCriteria: []string{"Shows task name, status, and due date"},
		Skill:           "ui-react",
	}

	systemPrompt, userPrompt := pb.BuildPrompt(req)

	// Check system prompt
	if !strings.Contains(systemPrompt, "React") {
		t.Error("System prompt should mention React")
	}
	if !strings.Contains(systemPrompt, "TypeScript") {
		t.Error("System prompt should mention TypeScript")
	}
	if !strings.Contains(systemPrompt, ".tsx") {
		t.Error("System prompt should include tsx file extension")
	}

	// Check user prompt
	if !strings.Contains(userPrompt, "Task List Component") {
		t.Error("User prompt should include feature title")
	}
}

func TestPromptBuilder_BuildPrompt_Database(t *testing.T) {
	pb := NewPromptBuilder()

	req := CodeGenerationRequest{
		FeatureID:       types.FeatureID("feat-003"),
		FeatureTitle:    "User Table Migration",
		FeatureDesc:     "Create users table with proper indexes",
		SuccessCriteria: []string{"Table has id, email, password_hash columns"},
		Skill:           "database",
	}

	systemPrompt, _ := pb.BuildPrompt(req)

	// Check system prompt
	if !strings.Contains(systemPrompt, "database") {
		t.Error("System prompt should mention database")
	}
	if !strings.Contains(systemPrompt, "migration") {
		t.Error("System prompt should mention migrations")
	}
	if !strings.Contains(systemPrompt, ".sql") {
		t.Error("System prompt should include sql file extension")
	}
}

func TestPromptBuilder_BuildPrompt_Testing(t *testing.T) {
	pb := NewPromptBuilder()

	req := CodeGenerationRequest{
		FeatureID:       types.FeatureID("feat-004"),
		FeatureTitle:    "Auth Handler Tests",
		FeatureDesc:     "Unit tests for authentication handlers",
		SuccessCriteria: []string{"Test login success and failure cases"},
		Skill:           "testing",
	}

	systemPrompt, _ := pb.BuildPrompt(req)

	// Check system prompt
	if !strings.Contains(systemPrompt, "test") {
		t.Error("System prompt should mention testing")
	}
	if !strings.Contains(systemPrompt, "table-driven") {
		t.Error("System prompt should mention table-driven tests")
	}
}

func TestPromptBuilder_BuildPrompt_Infra(t *testing.T) {
	pb := NewPromptBuilder()

	req := CodeGenerationRequest{
		FeatureID:       types.FeatureID("feat-005"),
		FeatureTitle:    "Kubernetes Deployment",
		FeatureDesc:     "Deploy app to Kubernetes",
		SuccessCriteria: []string{"Deployment with 3 replicas"},
		Skill:           "infra",
	}

	systemPrompt, _ := pb.BuildPrompt(req)

	// Check system prompt
	if !strings.Contains(systemPrompt, "infrastructure") || !strings.Contains(systemPrompt, "DevOps") {
		t.Error("System prompt should mention infrastructure/DevOps")
	}
	if !strings.Contains(systemPrompt, ".yaml") {
		t.Error("System prompt should include yaml file extension")
	}
}

func TestPromptBuilder_BuildPrompt_UnknownSkill(t *testing.T) {
	pb := NewPromptBuilder()

	req := CodeGenerationRequest{
		FeatureID:    types.FeatureID("feat-006"),
		FeatureTitle: "Unknown Feature",
		FeatureDesc:  "Some feature",
		Skill:        "unknown-skill",
	}

	systemPrompt, _ := pb.BuildPrompt(req)

	// Should fall back to default
	if systemPrompt == "" {
		t.Error("Should return default system prompt for unknown skill")
	}
	if !strings.Contains(systemPrompt, "software developer") {
		t.Error("Default prompt should mention software developer")
	}
}

func TestPromptBuilder_BuildPrompt_WithExistingFiles(t *testing.T) {
	pb := NewPromptBuilder()

	req := CodeGenerationRequest{
		FeatureID:    types.FeatureID("feat-007"),
		FeatureTitle: "Add Validation",
		FeatureDesc:  "Add input validation to handler",
		Skill:        "go-backend",
		ExistingFiles: map[string]string{
			"internal/handler/task.go": "package handler\n\ntype TaskHandler struct{}",
		},
	}

	_, userPrompt := pb.BuildPrompt(req)

	// Check that existing files are included
	if !strings.Contains(userPrompt, "Existing Code Context") {
		t.Error("User prompt should include existing code context section")
	}
	if !strings.Contains(userPrompt, "internal/handler/task.go") {
		t.Error("User prompt should include existing file path")
	}
	if !strings.Contains(userPrompt, "TaskHandler") {
		t.Error("User prompt should include existing file content")
	}
}

func TestPromptBuilder_AddSystemPrompt(t *testing.T) {
	pb := NewPromptBuilder()

	customPrompt := "You are a Rust expert."
	pb.AddSystemPrompt("rust-backend", customPrompt)

	got := pb.getSystemPrompt("rust-backend")
	if got != customPrompt {
		t.Errorf("getSystemPrompt() = %q, want %q", got, customPrompt)
	}
}

func TestSkillToLanguage(t *testing.T) {
	tests := []struct {
		skill    string
		wantLang string
	}{
		{"go-backend", "go"},
		{"ui-react", "typescript"},
		{"database", "sql"},
		{"testing", "go"},
		{"infra", "yaml"},
		{"unknown", "go"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.skill, func(t *testing.T) {
			got := SkillToLanguage(tt.skill)
			if got != tt.wantLang {
				t.Errorf("SkillToLanguage(%q) = %q, want %q", tt.skill, got, tt.wantLang)
			}
		})
	}
}

func TestGetFileExtension(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"component.tsx", "tsx"},
		{"styles.css", "css"},
		{"config.yaml", "yaml"},
		{"Dockerfile", ""},
		{"path/to/file.py", "python"},
		{"path/to/file.sql", "sql"},
		{"file.unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := getFileExtension(tt.path)
			if got != tt.want {
				t.Errorf("getFileExtension(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
