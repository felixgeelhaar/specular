package quickstart

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateMinimalConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "specular-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	provider := &ProviderSelection{
		Name:   "anthropic",
		Type:   "api",
		Ready:  true,
		EnvVar: "ANTHROPIC_API_KEY",
		Reason: "test",
	}

	docker := DockerStatus{
		Available: true,
		Runtime:   "docker",
		Version:   "24.0.0",
	}

	files, err := GenerateMinimalConfig(provider, docker)
	if err != nil {
		t.Fatalf("GenerateMinimalConfig failed: %v", err)
	}

	// Verify files were created
	if _, err := os.Stat(files.RouterPath); os.IsNotExist(err) {
		t.Error("routing.yaml was not created")
	}
	if _, err := os.Stat(files.PolicyPath); os.IsNotExist(err) {
		t.Error("policy.yaml was not created")
	}
	if _, err := os.Stat(files.SettingsPath); os.IsNotExist(err) {
		t.Error("settings.json was not created")
	}

	// Verify router content
	routerContent, err := os.ReadFile(files.RouterPath)
	if err != nil {
		t.Fatalf("failed to read routing.yaml: %v", err)
	}
	if !strings.Contains(string(routerContent), "anthropic") {
		t.Error("routing.yaml should contain anthropic provider")
	}

	// Verify policy content
	policyContent, err := os.ReadFile(files.PolicyPath)
	if err != nil {
		t.Fatalf("failed to read policy.yaml: %v", err)
	}
	if !strings.Contains(string(policyContent), "docker") {
		t.Error("policy.yaml should contain docker settings")
	}
	if !strings.Contains(string(policyContent), "required: true") {
		t.Error("policy.yaml should require docker when available")
	}

	// Verify settings content is valid JSON
	settingsContent, err := os.ReadFile(files.SettingsPath)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	var settings Settings
	if err := json.Unmarshal(settingsContent, &settings); err != nil {
		t.Errorf("settings.json is not valid JSON: %v", err)
	}
	if settings.Provider.Name != "anthropic" {
		t.Errorf("settings.json provider name should be anthropic, got %s", settings.Provider.Name)
	}
}

func TestGenerateMinimalConfig_LocalProvider(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "specular-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	provider := &ProviderSelection{
		Name:   "ollama",
		Type:   "local",
		Ready:  true,
		Reason: "Ollama detected",
	}

	docker := DockerStatus{
		Available: false,
		Warning:   "Docker not installed",
	}

	files, err := GenerateMinimalConfig(provider, docker)
	if err != nil {
		t.Fatalf("GenerateMinimalConfig failed: %v", err)
	}

	// Verify router contains local provider config
	routerContent, err := os.ReadFile(files.RouterPath)
	if err != nil {
		t.Fatalf("failed to read routing.yaml: %v", err)
	}
	if !strings.Contains(string(routerContent), "ollama") {
		t.Error("routing.yaml should contain ollama provider")
	}
	if !strings.Contains(string(routerContent), "localhost:11434") {
		t.Error("routing.yaml should contain ollama endpoint")
	}

	// Verify policy has docker disabled
	policyContent, err := os.ReadFile(files.PolicyPath)
	if err != nil {
		t.Fatalf("failed to read policy.yaml: %v", err)
	}
	if !strings.Contains(string(policyContent), "required: false") {
		t.Error("policy.yaml should not require docker when unavailable")
	}
}

func TestConfigExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "specular-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Initially should not exist
	if ConfigExists() {
		t.Error("ConfigExists should return false for empty directory")
	}

	// Create .specular directory with a config file
	os.MkdirAll(".specular", 0755)
	os.WriteFile(".specular/routing.yaml", []byte("test"), 0644)

	// Now should exist
	if !ConfigExists() {
		t.Error("ConfigExists should return true when config files exist")
	}
}

func TestBackupExistingConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "specular-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// Create existing config
	os.MkdirAll(".specular", 0755)
	os.WriteFile(".specular/routing.yaml", []byte("original content"), 0644)

	// Backup
	backupDir, err := BackupExistingConfig()
	if err != nil {
		t.Fatalf("BackupExistingConfig failed: %v", err)
	}

	// Verify backup was created
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Error("backup directory was not created")
	}

	// Verify original was moved
	if _, err := os.Stat(".specular"); !os.IsNotExist(err) {
		t.Error(".specular should have been moved")
	}

	// Verify content was preserved
	backupContent, err := os.ReadFile(filepath.Join(backupDir, "routing.yaml"))
	if err != nil {
		t.Fatalf("failed to read backup: %v", err)
	}
	if string(backupContent) != "original content" {
		t.Error("backup content was not preserved")
	}
}

func TestGenerateRouterYAML_APIProvider(t *testing.T) {
	provider := &ProviderSelection{
		Name:   "openai",
		Type:   "api",
		EnvVar: "OPENAI_API_KEY",
	}

	content := generateRouterYAML(provider)
	yaml := string(content)

	if !strings.Contains(yaml, "openai") {
		t.Error("should contain openai")
	}
	if !strings.Contains(yaml, "type: api") {
		t.Error("should contain type: api")
	}
	if !strings.Contains(yaml, "OPENAI_API_KEY") {
		t.Error("should reference env var")
	}
}

func TestGenerateRouterYAML_LocalProvider(t *testing.T) {
	provider := &ProviderSelection{
		Name: "ollama",
		Type: "local",
	}

	content := generateRouterYAML(provider)
	yaml := string(content)

	if !strings.Contains(yaml, "ollama") {
		t.Error("should contain ollama")
	}
	if !strings.Contains(yaml, "type: local") {
		t.Error("should contain type: local")
	}
	if !strings.Contains(yaml, "max_cost: 0.0") {
		t.Error("should have zero cost for local")
	}
}

func TestGeneratePolicyYAML_WithDocker(t *testing.T) {
	docker := DockerStatus{
		Available: true,
		Runtime:   "docker",
	}

	content := generatePolicyYAML(docker)
	yaml := string(content)

	if !strings.Contains(yaml, "required: true") {
		t.Error("should require docker when available")
	}
	if !strings.Contains(yaml, "sandboxed execution enabled") {
		t.Error("should indicate sandboxed execution")
	}
}

func TestGeneratePolicyYAML_WithoutDocker(t *testing.T) {
	docker := DockerStatus{
		Available: false,
	}

	content := generatePolicyYAML(docker)
	yaml := string(content)

	if !strings.Contains(yaml, "required: false") {
		t.Error("should not require docker when unavailable")
	}
	if !strings.Contains(yaml, "Docker not detected") {
		t.Error("should indicate docker not detected")
	}
}
