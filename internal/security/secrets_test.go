package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSecretScanner(t *testing.T) {
	scanner := NewSecretScanner()

	if scanner == nil {
		t.Fatal("Scanner should not be nil")
	}

	if len(scanner.patterns) == 0 {
		t.Error("Scanner should have default patterns")
	}

	if len(scanner.excludePaths) == 0 {
		t.Error("Scanner should have default exclude paths")
	}
}

func TestScanFileWithSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "config.go")

	// Create file with various secrets
	content := `package config

const (
	// AWS credentials
	AWSAccessKey = "AKIAIOSFODNN7EXAMPLE"
	AWSSecretKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"

	// GitHub token
	GitHubToken = "ghp_1234567890abcdefghijklmnopqrstuvwxyz"

	// API key
	APIKey = "sk-test1234567890abcdefghijklmnopqrstuvwxyz"
)
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewSecretScanner()
	matches, err := scanner.ScanFile(testFile)
	if err != nil {
		t.Fatalf("Failed to scan file: %v", err)
	}

	if len(matches) == 0 {
		t.Error("Expected to find secrets in test file")
	}

	// Verify we found some key secret types
	foundTypes := make(map[SecretType]bool)
	for _, match := range matches {
		foundTypes[match.Type] = true
	}

	if !foundTypes[SecretAWSKey] && !foundTypes[SecretAWSSecret] {
		t.Error("Expected to find AWS credentials")
	}

	if !foundTypes[SecretGitHubToken] {
		t.Error("Expected to find GitHub token")
	}
}

func TestScanFileNoSecrets(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "clean.go")

	// Create file without secrets
	content := `package main

func main() {
	println("Hello, World!")
}
`

	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	scanner := NewSecretScanner()
	matches, err := scanner.ScanFile(testFile)
	if err != nil {
		t.Fatalf("Failed to scan file: %v", err)
	}

	if len(matches) != 0 {
		t.Errorf("Expected no secrets, found %d", len(matches))
	}
}

func TestScanDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure with secrets
	file1 := filepath.Join(tmpDir, "config.go")
	file2 := filepath.Join(tmpDir, "secrets.yaml")
	subDir := filepath.Join(tmpDir, "internal")
	file3 := filepath.Join(subDir, "creds.json")

	os.MkdirAll(subDir, 0755)

	// File with secret
	os.WriteFile(file1, []byte(`
		package config
		const GitHubToken = "ghp_1234567890abcdefghijklmnopqrstuvwxyz"
	`), 0644)

	// Another file with secret
	os.WriteFile(file2, []byte(`
		api_key: sk-test1234567890abcdefghijklmnopqrstuvwxyz
	`), 0644)

	// Nested file with secret
	os.WriteFile(file3, []byte(`
		{"aws_access_key": "AKIAIOSFODNN7EXAMPLE"}
	`), 0644)

	scanner := NewSecretScanner()
	matches, err := scanner.ScanDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to scan directory: %v", err)
	}

	// Should find at least 2 secrets (some patterns may overlap)
	if len(matches) < 2 {
		t.Errorf("Expected at least 2 secrets across files, found %d", len(matches))
	}

	// Verify secrets found in different files
	filesSeen := make(map[string]bool)
	for _, match := range matches {
		filesSeen[match.File] = true
	}

	if len(filesSeen) < 2 {
		t.Error("Expected to find secrets in multiple files")
	}
}

func TestExcludedPaths(t *testing.T) {
	tmpDir := t.TempDir()

	// Create node_modules directory with "secret"
	nodeModules := filepath.Join(tmpDir, "node_modules")
	os.MkdirAll(nodeModules, 0755)

	secretFile := filepath.Join(nodeModules, "secret.js")
	os.WriteFile(secretFile, []byte(`
		const apiKey = "sk-test1234567890abcdefghijklmnopqrstuvwxyz"
	`), 0644)

	scanner := NewSecretScanner()
	matches, err := scanner.ScanDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to scan directory: %v", err)
	}

	// node_modules should be excluded
	for _, match := range matches {
		if strings.Contains(match.File, "node_modules") {
			t.Error("node_modules should be excluded from scanning")
		}
	}
}

func TestScanGitDiff(t *testing.T) {
	diff := `diff --git a/config.go b/config.go
index 1234567..abcdefg 100644
--- a/config.go
+++ b/config.go
@@ -1,3 +1,5 @@
 package config

+const GitHubToken = "ghp_1234567890abcdefghijklmnopqrstuvwxyz"
+const AWSKey = "AKIAIOSFODNN7EXAMPLE"
`

	scanner := NewSecretScanner()
	matches, err := scanner.ScanGitDiff(diff)
	if err != nil {
		t.Fatalf("Failed to scan diff: %v", err)
	}

	if len(matches) < 2 {
		t.Errorf("Expected at least 2 secrets in diff, found %d", len(matches))
	}

	// Verify line numbers are set
	for _, match := range matches {
		if match.Line == 0 {
			t.Error("Line number should be set for diff matches")
		}
	}
}

func TestScanGitDiffIgnoresRemovedLines(t *testing.T) {
	diff := `diff --git a/config.go b/config.go
index 1234567..abcdefg 100644
--- a/config.go
+++ b/config.go
@@ -1,5 +1,3 @@
 package config

-const GitHubToken = "ghp_1234567890abcdefghijklmnopqrstuvwxyz"
`

	scanner := NewSecretScanner()
	matches, err := scanner.ScanGitDiff(diff)
	if err != nil {
		t.Fatalf("Failed to scan diff: %v", err)
	}

	// Should not find secrets in removed lines
	if len(matches) != 0 {
		t.Errorf("Expected no secrets in removed lines, found %d", len(matches))
	}
}

func TestSecretTypes(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		secretType SecretType
	}{
		{
			name:       "AWS Access Key",
			content:    `aws_access_key_id = "AKIAIOSFODNN7EXAMPLE"`,
			secretType: SecretAWSKey,
		},
		{
			name:       "GitHub Personal Token",
			content:    `github_token = "ghp_1234567890abcdefghijklmnopqrstuvwxyz"`,
			secretType: SecretGitHubToken,
		},
		{
			name:       "Slack Token",
			content:    `slack_token = "xoxb-1234567890-1234567890-abcdefghijklmnopqrstuvwx"`,
			secretType: SecretSlackToken,
		},
		{
			name:       "Private Key",
			content:    `-----BEGIN RSA PRIVATE KEY-----`,
			secretType: SecretPrivateKey,
		},
		{
			name:       "JWT Token",
			content:    `token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"`,
			secretType: SecretJWT,
		},
		{
			name:       "Database URL",
			content:    `db_url = "postgres://user:password@localhost:5432/db"`,
			secretType: SecretDatabaseURL,
		},
	}

	tmpDir := t.TempDir()
	scanner := NewSecretScanner()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.name+".txt")
			err := os.WriteFile(testFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			matches, err := scanner.ScanFile(testFile)
			if err != nil {
				t.Fatalf("Failed to scan file: %v", err)
			}

			if len(matches) == 0 {
				t.Errorf("Expected to find %s", tt.secretType)
				return
			}

			found := false
			for _, match := range matches {
				if match.Type == tt.secretType {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected to find secret type %s, found types: %v", tt.secretType, matches)
			}
		})
	}
}

func TestSecretMatchRedaction(t *testing.T) {
	scanner := NewSecretScanner()

	// Test short string
	shortMatch := scanner.redactMatch("short")
	if shortMatch != "***REDACTED***" {
		t.Errorf("Short match redaction incorrect: got %s", shortMatch)
	}

	// Test long string
	longMatch := "this is a very long secret key that should be partially redacted"
	redacted := scanner.redactMatch(longMatch)

	if !strings.Contains(redacted, "***REDACTED***") {
		t.Error("Long match should contain redacted text")
	}

	if !strings.HasPrefix(redacted, longMatch[:10]) {
		t.Error("Long match should show first 10 characters")
	}

	if !strings.HasSuffix(redacted, longMatch[len(longMatch)-10:]) {
		t.Error("Long match should show last 10 characters")
	}
}

func TestAddExcludePath(t *testing.T) {
	scanner := NewSecretScanner()
	initialCount := len(scanner.excludePaths)

	scanner.AddExcludePath("custom-exclude")

	if len(scanner.excludePaths) != initialCount+1 {
		t.Error("Failed to add exclude path")
	}

	// Verify the path is excluded
	testPath := "/some/path/custom-exclude/file.go"
	if !scanner.shouldExclude(testPath) {
		t.Error("Custom exclude path should be excluded")
	}
}

func TestFormatReport(t *testing.T) {
	// Test empty matches
	report := FormatReport([]*SecretMatch{})
	if !strings.Contains(report, "No secrets detected") {
		t.Error("Empty report should indicate no secrets")
	}

	// Test with matches
	matches := []*SecretMatch{
		{
			Type:        SecretAWSKey,
			File:        "config.go",
			Line:        10,
			Severity:    "critical",
			Description: "AWS Access Key",
			Match:       "AKIAIOSFO***REDACTED***EXAMPLE",
		},
		{
			Type:        SecretGitHubToken,
			File:        "main.go",
			Line:        25,
			Severity:    "high",
			Description: "GitHub Token",
			Match:       "ghp_12345***REDACTED***67890",
		},
	}

	report = FormatReport(matches)

	if !strings.Contains(report, "Found 2 potential secret(s)") {
		t.Error("Report should show count of secrets")
	}

	if !strings.Contains(report, "CRITICAL") {
		t.Error("Report should show critical severity")
	}

	if !strings.Contains(report, "config.go:10") {
		t.Error("Report should show file and line number")
	}

	if !strings.Contains(report, "AWS Access Key") {
		t.Error("Report should show secret description")
	}
}

func TestSeverityGrouping(t *testing.T) {
	matches := []*SecretMatch{
		{Type: SecretAWSKey, File: "a.go", Line: 1, Severity: "critical", Description: "Critical"},
		{Type: SecretPassword, File: "b.go", Line: 2, Severity: "high", Description: "High"},
		{Type: SecretAPIKey, File: "c.go", Line: 3, Severity: "medium", Description: "Medium"},
	}

	report := FormatReport(matches)

	// Check that severities appear in order
	criticalPos := strings.Index(report, "CRITICAL")
	highPos := strings.Index(report, "HIGH")
	mediumPos := strings.Index(report, "MEDIUM")

	if criticalPos == -1 || highPos == -1 || mediumPos == -1 {
		t.Error("Report should contain all severity levels")
	}

	if !(criticalPos < highPos && highPos < mediumPos) {
		t.Error("Report should show severities in order: critical, high, medium")
	}
}

func TestExcludeTestFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with "secret"
	testFile := filepath.Join(tmpDir, "config_test.go")
	os.WriteFile(testFile, []byte(`
		package config
		const TestAPIKey = "sk-test1234567890abcdefghijklmnopqrstuvwxyz"
	`), 0644)

	scanner := NewSecretScanner()

	// Test files should be excluded by default pattern
	shouldExclude := scanner.shouldExclude(testFile)
	if !shouldExclude {
		t.Error("Test files should be excluded by default")
	}
}

func TestExcludeMinifiedFiles(t *testing.T) {
	scanner := NewSecretScanner()

	minFiles := []string{
		"app.min.js",
		"styles.min.css",
		"/path/to/bundle.min.js",
	}

	for _, file := range minFiles {
		if !scanner.shouldExclude(file) {
			t.Errorf("Minified file %s should be excluded", file)
		}
	}
}

func TestSecretMatchContainsFileInfo(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "secret.go")

	content := `package main

github_token = "ghp_1234567890abcdefghijklmnopqrstuvwxyz"
`

	os.WriteFile(testFile, []byte(content), 0644)

	scanner := NewSecretScanner()
	matches, err := scanner.ScanFile(testFile)
	if err != nil {
		t.Fatalf("Failed to scan file: %v", err)
	}

	if len(matches) == 0 {
		t.Fatal("Expected to find secret")
	}

	match := matches[0]

	if match.File != testFile {
		t.Errorf("File path mismatch: got %s, want %s", match.File, testFile)
	}

	if match.Line != 3 {
		t.Errorf("Line number mismatch: got %d, want 3", match.Line)
	}

	if match.Severity == "" {
		t.Error("Severity should be set")
	}

	if match.Description == "" {
		t.Error("Description should be set")
	}
}
