package codegen

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// ResponseParser extracts code blocks from AI responses
type ResponseParser struct {
	// Pattern for code blocks with path annotations
	// Matches: ```go:path/to/file.go or ```typescript:src/Component.tsx
	codeBlockPattern *regexp.Regexp

	// Pattern for simple code blocks without path
	// Matches: ```go or ```python
	simpleBlockPattern *regexp.Regexp
}

// NewResponseParser creates a new parser for AI responses
func NewResponseParser() *ResponseParser {
	return &ResponseParser{
		// Matches ```language:path/to/file.ext
		// Group 1: language
		// Group 2: file path
		// Group 3: content
		codeBlockPattern: regexp.MustCompile("```([a-zA-Z0-9_+-]+):([^\n`]+)\n([\\s\\S]*?)```"),

		// Matches ```language (without path)
		// Group 1: language
		// Group 2: content
		simpleBlockPattern: regexp.MustCompile("```([a-zA-Z0-9_+-]+)\n([\\s\\S]*?)```"),
	}
}

// ParseResponse extracts all code files from an AI response
func (p *ResponseParser) ParseResponse(response string, defaultLanguage string) ([]GeneratedFile, error) {
	var files []GeneratedFile
	seenPaths := make(map[string]bool)

	// First, try to extract code blocks with path annotations
	matches := p.codeBlockPattern.FindAllStringSubmatch(response, -1)
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		language := normalizeLanguage(match[1])
		path := strings.TrimSpace(match[2])
		content := strings.TrimRight(match[3], "\n")

		// Skip duplicates
		if seenPaths[path] {
			continue
		}
		seenPaths[path] = true

		file := GeneratedFile{
			Path:     path,
			Content:  content,
			Language: language,
			Hash:     hashContent(content),
			Size:     len(content),
		}

		files = append(files, file)
	}

	// If we found annotated code blocks, return them
	if len(files) > 0 {
		return files, nil
	}

	// Fallback: try to extract simple code blocks and infer paths
	simpleMatches := p.simpleBlockPattern.FindAllStringSubmatch(response, -1)
	for i, match := range simpleMatches {
		if len(match) < 3 {
			continue
		}

		language := normalizeLanguage(match[1])
		content := strings.TrimRight(match[2], "\n")

		// Skip empty blocks
		if strings.TrimSpace(content) == "" {
			continue
		}

		// Infer path from content or generate default
		path := inferFilePath(content, language, i)

		// Skip duplicates
		if seenPaths[path] {
			continue
		}
		seenPaths[path] = true

		file := GeneratedFile{
			Path:     path,
			Content:  content,
			Language: language,
			Hash:     hashContent(content),
			Size:     len(content),
		}

		files = append(files, file)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no code blocks found in response")
	}

	return files, nil
}

// ParseSingleFile parses a response expected to contain a single file
func (p *ResponseParser) ParseSingleFile(response string, expectedPath string, language string) (*GeneratedFile, error) {
	files, err := p.ParseResponse(response, language)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no code blocks found")
	}

	// If only one file, return it (with expected path if provided)
	if len(files) == 1 {
		file := files[0]
		if expectedPath != "" {
			file.Path = expectedPath
		}
		return &file, nil
	}

	// Multiple files - try to find the expected path
	if expectedPath != "" {
		for _, file := range files {
			if file.Path == expectedPath || strings.HasSuffix(file.Path, expectedPath) {
				return &file, nil
			}
		}
	}

	// Return the first file
	return &files[0], nil
}

// normalizeLanguage normalizes language identifiers
func normalizeLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))

	// Normalize common aliases
	aliases := map[string]string{
		"golang":     "go",
		"typescript": "typescript",
		"ts":         "typescript",
		"tsx":        "tsx",
		"javascript": "javascript",
		"js":         "javascript",
		"jsx":        "jsx",
		"python":     "python",
		"py":         "python",
		"python3":    "python",
		"postgresql": "sql",
		"postgres":   "sql",
		"mysql":      "sql",
		"sqlite":     "sql",
		"bash":       "bash",
		"shell":      "bash",
		"sh":         "bash",
		"zsh":        "bash",
		"yml":        "yaml",
		"dockerfile": "dockerfile",
		"docker":     "dockerfile",
		"terraform":  "hcl",
		"tf":         "hcl",
		"csharp":     "csharp",
		"c#":         "csharp",
	}

	if normalized, ok := aliases[lang]; ok {
		return normalized
	}

	return lang
}

// inferFilePath attempts to infer a file path from code content
func inferFilePath(content, language string, index int) string {
	// Try to extract package name for Go
	if language == "go" {
		if match := regexp.MustCompile(`package\s+(\w+)`).FindStringSubmatch(content); len(match) > 1 {
			pkg := match[1]
			if pkg == "main" {
				return "cmd/main.go"
			}
			// Check for _test suffix
			if strings.Contains(content, "func Test") {
				return fmt.Sprintf("internal/%s/%s_test.go", pkg, pkg)
			}
			return fmt.Sprintf("internal/%s/%s.go", pkg, pkg)
		}
	}

	// Try to extract component/module name for TypeScript/React
	if language == "typescript" || language == "tsx" {
		// Look for export default function/const ComponentName
		if match := regexp.MustCompile(`export\s+(?:default\s+)?(?:function|const|class)\s+(\w+)`).FindStringSubmatch(content); len(match) > 1 {
			name := match[1]
			ext := ".ts"
			if language == "tsx" || strings.Contains(content, "React") || strings.Contains(content, "jsx") {
				ext = ".tsx"
			}
			return fmt.Sprintf("src/components/%s%s", name, ext)
		}
	}

	// Default paths based on language
	defaults := map[string]string{
		"go":         "internal/generated/file%d.go",
		"typescript": "src/generated/file%d.ts",
		"tsx":        "src/components/Component%d.tsx",
		"javascript": "src/generated/file%d.js",
		"jsx":        "src/components/Component%d.jsx",
		"python":     "src/generated/file%d.py",
		"sql":        "migrations/generated_%d.sql",
		"yaml":       "config/generated%d.yaml",
		"json":       "config/generated%d.json",
		"bash":       "scripts/generated%d.sh",
		"dockerfile": "Dockerfile",
		"hcl":        "terraform/generated%d.tf",
	}

	if pattern, ok := defaults[language]; ok {
		return fmt.Sprintf(pattern, index+1)
	}

	return fmt.Sprintf("generated/file%d.txt", index+1)
}

// hashContent computes SHA-256 hash of content
func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}

// ExtractCodeBlocks returns all code block contents (for testing/debugging)
func (p *ResponseParser) ExtractCodeBlocks(response string) []string {
	var blocks []string

	// Extract annotated blocks
	matches := p.codeBlockPattern.FindAllStringSubmatch(response, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			blocks = append(blocks, match[3])
		}
	}

	// Extract simple blocks
	simpleMatches := p.simpleBlockPattern.FindAllStringSubmatch(response, -1)
	for _, match := range simpleMatches {
		if len(match) >= 3 {
			blocks = append(blocks, match[2])
		}
	}

	return blocks
}

// ValidateResponse checks if a response contains valid code blocks
func (p *ResponseParser) ValidateResponse(response string) error {
	if strings.TrimSpace(response) == "" {
		return fmt.Errorf("empty response")
	}

	// Check for code blocks
	if !strings.Contains(response, "```") {
		return fmt.Errorf("response does not contain code blocks")
	}

	// Try to parse
	files, err := p.ParseResponse(response, "")
	if err != nil {
		return fmt.Errorf("failed to parse code blocks: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no valid code files extracted")
	}

	// Validate each file
	for _, file := range files {
		if strings.TrimSpace(file.Content) == "" {
			return fmt.Errorf("empty content in file: %s", file.Path)
		}
	}

	return nil
}
