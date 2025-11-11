package security

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SecretType represents the type of secret detected
type SecretType string

// Secret type constants define the various types of secrets that can be detected
const (
	// SecretAWSKey represents AWS access key IDs
	SecretAWSKey        SecretType = "aws_access_key"
	SecretAWSSecret     SecretType = "aws_secret_key"
	SecretGitHubToken   SecretType = "github_token"
	SecretSlackToken    SecretType = "slack_token"
	SecretPrivateKey    SecretType = "private_key"
	SecretAPIKey        SecretType = "api_key"
	SecretPassword      SecretType = "password"
	SecretJWT           SecretType = "jwt_token"
	SecretDatabaseURL   SecretType = "database_url"
	SecretGenericSecret SecretType = "generic_secret"
)

// SecretMatch represents a detected secret
type SecretMatch struct {
	// Type is the secret type
	Type SecretType `json:"type"`

	// File is the file path where secret was found
	File string `json:"file"`

	// Line is the line number
	Line int `json:"line"`

	// Column is the column number (if available)
	Column int `json:"column"`

	// Match is the matched text (partially redacted)
	Match string `json:"match"`

	// Severity is the severity level
	Severity string `json:"severity"`

	// Description describes the secret type
	Description string `json:"description"`
}

// SecretPattern represents a regex pattern for detecting secrets
type SecretPattern struct {
	Type        SecretType
	Pattern     *regexp.Regexp
	Description string
	Severity    string
}

// SecretScanner scans files for secrets
type SecretScanner struct {
	patterns []SecretPattern

	// excludePaths are file paths to exclude from scanning
	excludePaths []string

	// excludePatterns are regex patterns for excluding files
	excludePatterns []*regexp.Regexp
}

// NewSecretScanner creates a new secret scanner
func NewSecretScanner() *SecretScanner {
	scanner := &SecretScanner{
		patterns: []SecretPattern{
			// AWS Keys
			{
				Type:        SecretAWSKey,
				Pattern:     regexp.MustCompile(`(?i)(aws|amazon)[\s\w]*key[\s\w]*[:=]\s*["']?(AKIA[0-9A-Z]{16})["']?`),
				Description: "AWS Access Key ID",
				Severity:    "critical",
			},
			{
				Type:        SecretAWSSecret,
				Pattern:     regexp.MustCompile(`(?i)(aws|amazon)[\s\w]*secret[\s\w]*[:=]\s*["']?([A-Za-z0-9/+=]{40})["']?`),
				Description: "AWS Secret Access Key",
				Severity:    "critical",
			},

			// GitHub Tokens
			{
				Type:        SecretGitHubToken,
				Pattern:     regexp.MustCompile(`(?i)github[\s\w]*token[\s\w]*[:=]\s*["']?(ghp_[A-Za-z0-9_]{36,})["']?`),
				Description: "GitHub Personal Access Token",
				Severity:    "high",
			},
			{
				Type:        SecretGitHubToken,
				Pattern:     regexp.MustCompile(`(?i)github[\s\w]*token[\s\w]*[:=]\s*["']?(gho_[A-Za-z0-9_]{36,})["']?`),
				Description: "GitHub OAuth Token",
				Severity:    "high",
			},

			// Slack Tokens
			{
				Type:        SecretSlackToken,
				Pattern:     regexp.MustCompile(`xox[baprs]-[0-9]{10,12}-[0-9]{10,12}-[A-Za-z0-9]{24,}`),
				Description: "Slack Token",
				Severity:    "high",
			},

			// Private Keys
			{
				Type:        SecretPrivateKey,
				Pattern:     regexp.MustCompile(`-----BEGIN\s+(RSA|DSA|EC|OPENSSH|PGP)\s+PRIVATE KEY-----`),
				Description: "Private Key",
				Severity:    "critical",
			},

			// Generic API Keys
			{
				Type:        SecretAPIKey,
				Pattern:     regexp.MustCompile(`(?i)api[\s_-]?key[\s\w]*[:=]\s*["']?([A-Za-z0-9_\-]{32,})["']?`),
				Description: "Generic API Key",
				Severity:    "medium",
			},

			// Passwords in code
			{
				Type:        SecretPassword,
				Pattern:     regexp.MustCompile(`(?i)password[\s\w]*[:=]\s*["']([^"']{8,})["']`),
				Description: "Hardcoded Password",
				Severity:    "high",
			},

			// JWT Tokens
			{
				Type:        SecretJWT,
				Pattern:     regexp.MustCompile(`eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*`),
				Description: "JWT Token",
				Severity:    "medium",
			},

			// Database URLs
			{
				Type:        SecretDatabaseURL,
				Pattern:     regexp.MustCompile(`(?i)(postgres|mysql|mongodb|redis)://[^\s'"]+:[^\s'"]+@`),
				Description: "Database Connection String with Credentials",
				Severity:    "critical",
			},

			// Generic secrets
			{
				Type:        SecretGenericSecret,
				Pattern:     regexp.MustCompile(`(?i)secret[\s\w]*[:=]\s*["']([A-Za-z0-9_\-+=]{20,})["']`),
				Description: "Generic Secret",
				Severity:    "medium",
			},
		},

		// Default exclusions
		excludePaths: []string{
			".git",
			"node_modules",
			"vendor",
			"dist",
			"build",
			".specular",
		},

		excludePatterns: []*regexp.Regexp{
			regexp.MustCompile(`\.(log|lock|sum|mod)$`),
			regexp.MustCompile(`_test\.go$`),
			regexp.MustCompile(`\.min\.(js|css)$`),
		},
	}

	return scanner
}

// AddExcludePath adds a path to exclude from scanning
func (s *SecretScanner) AddExcludePath(path string) {
	s.excludePaths = append(s.excludePaths, path)
}

// ScanFile scans a single file for secrets
func (s *SecretScanner) ScanFile(filePath string) ([]*SecretMatch, error) {
	// Check if file should be excluded
	if s.shouldExclude(filePath) {
		return nil, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	matches := []*SecretMatch{}
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Check each pattern
		for _, pattern := range s.patterns {
			if pattern.Pattern.MatchString(line) {
				match := &SecretMatch{
					Type:        pattern.Type,
					File:        filePath,
					Line:        lineNum,
					Match:       s.redactMatch(line),
					Severity:    pattern.Severity,
					Description: pattern.Description,
				}
				matches = append(matches, match)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file: %w", err)
	}

	return matches, nil
}

// ScanDirectory scans a directory recursively for secrets
func (s *SecretScanner) ScanDirectory(dirPath string) ([]*SecretMatch, error) {
	allMatches := []*SecretMatch{}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and excluded paths
		if info.IsDir() || s.shouldExclude(path) {
			if info.IsDir() && s.shouldExcludeDir(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Scan file
		matches, err := s.ScanFile(path)
		if err != nil {
			// Log error but continue scanning
			fmt.Fprintf(os.Stderr, "Warning: failed to scan %s: %v\n", path, err)
			return nil
		}

		allMatches = append(allMatches, matches...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return allMatches, nil
}

// ScanGitDiff scans a git diff for secrets
func (s *SecretScanner) ScanGitDiff(diff string) ([]*SecretMatch, error) {
	matches := []*SecretMatch{}
	lines := strings.Split(diff, "\n")

	currentFile := ""
	lineNum := 0

	for _, line := range lines {
		// Track current file
		if strings.HasPrefix(line, "+++") {
			currentFile = strings.TrimPrefix(line, "+++ b/")
			lineNum = 0
			continue
		}

		// Only check added lines
		if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue
		}

		lineNum++
		content := strings.TrimPrefix(line, "+")

		// Check each pattern
		for _, pattern := range s.patterns {
			if pattern.Pattern.MatchString(content) {
				match := &SecretMatch{
					Type:        pattern.Type,
					File:        currentFile,
					Line:        lineNum,
					Match:       s.redactMatch(content),
					Severity:    pattern.Severity,
					Description: pattern.Description,
				}
				matches = append(matches, match)
			}
		}
	}

	return matches, nil
}

// shouldExclude checks if a file should be excluded
func (s *SecretScanner) shouldExclude(path string) bool {
	// Check excluded paths
	for _, excluded := range s.excludePaths {
		if strings.Contains(path, excluded) {
			return true
		}
	}

	// Check excluded patterns
	for _, pattern := range s.excludePatterns {
		if pattern.MatchString(path) {
			return true
		}
	}

	return false
}

// shouldExcludeDir checks if a directory should be excluded
func (s *SecretScanner) shouldExcludeDir(path string) bool {
	basename := filepath.Base(path)
	for _, excluded := range s.excludePaths {
		if basename == excluded {
			return true
		}
	}
	return false
}

// redactMatch redacts sensitive parts of the match
func (s *SecretScanner) redactMatch(match string) string {
	if len(match) <= 20 {
		return "***REDACTED***"
	}

	// Show first 10 and last 10 characters
	return match[:10] + "***REDACTED***" + match[len(match)-10:]
}

// FormatReport formats scan results as a report
func FormatReport(matches []*SecretMatch) string {
	if len(matches) == 0 {
		return "✅ No secrets detected"
	}

	var report strings.Builder
	report.WriteString(fmt.Sprintf("🚨 Found %d potential secret(s):\n\n", len(matches)))

	// Group by severity
	bySeverity := make(map[string][]*SecretMatch)
	for _, match := range matches {
		bySeverity[match.Severity] = append(bySeverity[match.Severity], match)
	}

	// Report in severity order
	severities := []string{"critical", "high", "medium", "low"}
	for _, severity := range severities {
		matches := bySeverity[severity]
		if len(matches) == 0 {
			continue
		}

		report.WriteString(fmt.Sprintf("## %s Severity (%d)\n\n", strings.ToUpper(severity), len(matches)))

		for _, match := range matches {
			report.WriteString(fmt.Sprintf("- **%s** in `%s:%d`\n", match.Description, match.File, match.Line))
			report.WriteString(fmt.Sprintf("  Type: %s\n", match.Type))
			report.WriteString(fmt.Sprintf("  Match: %s\n\n", match.Match))
		}
	}

	return report.String()
}
