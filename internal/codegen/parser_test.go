package codegen

import (
	"strings"
	"testing"
)

func TestResponseParser_ParseResponse_WithPathAnnotations(t *testing.T) {
	parser := NewResponseParser()

	response := `Here's the implementation:

` + "```go:cmd/main.go" + `
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
` + "```" + `

And here's the handler:

` + "```go:internal/handler/task.go" + `
package handler

type TaskHandler struct{}

func (h *TaskHandler) Handle() error {
	return nil
}
` + "```"

	files, err := parser.ParseResponse(response, "go")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}

	if len(files) != 2 {
		t.Errorf("ParseResponse() got %d files, want 2", len(files))
	}

	// Check first file
	if files[0].Path != "cmd/main.go" {
		t.Errorf("First file path = %q, want %q", files[0].Path, "cmd/main.go")
	}
	if files[0].Language != "go" {
		t.Errorf("First file language = %q, want %q", files[0].Language, "go")
	}
	if !strings.Contains(files[0].Content, "package main") {
		t.Error("First file content doesn't contain 'package main'")
	}

	// Check second file
	if files[1].Path != "internal/handler/task.go" {
		t.Errorf("Second file path = %q, want %q", files[1].Path, "internal/handler/task.go")
	}
}

func TestResponseParser_ParseResponse_SimpleBlocks(t *testing.T) {
	parser := NewResponseParser()

	response := `Here's some Go code:

` + "```go" + `
package main

func main() {}
` + "```"

	files, err := parser.ParseResponse(response, "go")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}

	if len(files) != 1 {
		t.Errorf("ParseResponse() got %d files, want 1", len(files))
	}

	// Should infer path from content
	if !strings.Contains(files[0].Path, "main") || !strings.HasSuffix(files[0].Path, ".go") {
		t.Errorf("Inferred path = %q, expected to contain 'main' and end with '.go'", files[0].Path)
	}
}

func TestResponseParser_ParseResponse_NoCodeBlocks(t *testing.T) {
	parser := NewResponseParser()

	response := "This is just plain text without any code blocks."

	_, err := parser.ParseResponse(response, "go")
	if err == nil {
		t.Error("ParseResponse() expected error for response without code blocks")
	}
}

func TestResponseParser_ParseResponse_EmptyCodeBlock(t *testing.T) {
	parser := NewResponseParser()

	response := "```go\n```"

	_, err := parser.ParseResponse(response, "go")
	if err == nil {
		t.Error("ParseResponse() expected error for empty code block")
	}
}

func TestResponseParser_ParseResponse_MultipleLanguages(t *testing.T) {
	parser := NewResponseParser()

	response := `
` + "```go:main.go" + `
package main
func main() {}
` + "```" + `

` + "```sql:migrations/001_create_users.sql" + `
CREATE TABLE users (id SERIAL PRIMARY KEY);
` + "```" + `

` + "```typescript:src/App.tsx" + `
export default function App() { return <div>Hello</div>; }
` + "```"

	files, err := parser.ParseResponse(response, "")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}

	if len(files) != 3 {
		t.Errorf("ParseResponse() got %d files, want 3", len(files))
	}

	// Verify languages
	languages := map[string]string{
		"main.go":                         "go",
		"migrations/001_create_users.sql": "sql",
		"src/App.tsx":                     "typescript", // tsx is normalized to typescript
	}

	for _, f := range files {
		expectedLang, ok := languages[f.Path]
		if !ok {
			t.Errorf("Unexpected file path: %s", f.Path)
			continue
		}
		if f.Language != expectedLang {
			t.Errorf("File %s language = %q, want %q", f.Path, f.Language, expectedLang)
		}
	}
}

func TestResponseParser_ParseResponse_DuplicatePaths(t *testing.T) {
	parser := NewResponseParser()

	// Response with duplicate paths - should only keep first
	response := `
` + "```go:main.go" + `
package main
// First version
` + "```" + `

` + "```go:main.go" + `
package main
// Second version - should be ignored
` + "```"

	files, err := parser.ParseResponse(response, "go")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}

	if len(files) != 1 {
		t.Errorf("ParseResponse() got %d files, want 1 (duplicates should be ignored)", len(files))
	}

	if !strings.Contains(files[0].Content, "First version") {
		t.Error("Expected first version to be kept, not second")
	}
}

func TestResponseParser_ParseSingleFile(t *testing.T) {
	parser := NewResponseParser()

	response := `
` + "```go:cmd/main.go" + `
package main
func main() {}
` + "```"

	file, err := parser.ParseSingleFile(response, "expected/path.go", "go")
	if err != nil {
		t.Fatalf("ParseSingleFile() error = %v", err)
	}

	// When expectedPath is provided, it should override the parsed path
	if file.Path != "expected/path.go" {
		t.Errorf("ParseSingleFile() path = %q, want %q", file.Path, "expected/path.go")
	}
}

func TestResponseParser_ValidateResponse(t *testing.T) {
	parser := NewResponseParser()

	tests := []struct {
		name     string
		response string
		wantErr  bool
	}{
		{
			name:     "valid response",
			response: "```go:main.go\npackage main\n```",
			wantErr:  false,
		},
		{
			name:     "empty response",
			response: "",
			wantErr:  true,
		},
		{
			name:     "no code blocks",
			response: "Just text",
			wantErr:  true,
		},
		{
			name:     "empty code block",
			response: "```go\n```",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidateResponse(tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeLanguage(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"go", "go"},
		{"golang", "go"},
		{"GOLANG", "go"},
		{"typescript", "typescript"},
		{"ts", "typescript"},
		{"python", "python"},
		{"py", "python"},
		{"python3", "python"},
		{"javascript", "javascript"},
		{"js", "javascript"},
		{"bash", "bash"},
		{"shell", "bash"},
		{"sh", "bash"},
		{"yml", "yaml"},
		{"terraform", "hcl"},
		{"tf", "hcl"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeLanguage(tt.input)
			if got != tt.want {
				t.Errorf("normalizeLanguage(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestInferFilePath(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		language string
		index    int
		wantSub  string // substring that should be in path
	}{
		{
			name:     "go main package",
			content:  "package main\nfunc main() {}",
			language: "go",
			index:    0,
			wantSub:  "cmd/main.go",
		},
		{
			name:     "go handler package",
			content:  "package handler\ntype Handler struct{}",
			language: "go",
			index:    0,
			wantSub:  "internal/handler/handler.go",
		},
		{
			name:     "go test file",
			content:  "package handler\nfunc TestHandler(t *testing.T) {}",
			language: "go",
			index:    0,
			wantSub:  "_test.go",
		},
		{
			name:     "typescript with export",
			content:  "export default function MyComponent() { return <div />; }",
			language: "tsx",
			index:    0,
			wantSub:  "MyComponent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferFilePath(tt.content, tt.language, tt.index)
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("inferFilePath() = %q, want to contain %q", got, tt.wantSub)
			}
		})
	}
}

func TestHashContent(t *testing.T) {
	content := "test content"
	hash1 := hashContent(content)
	hash2 := hashContent(content)

	if hash1 != hash2 {
		t.Error("hashContent() should return same hash for same content")
	}

	differentHash := hashContent("different content")
	if hash1 == differentHash {
		t.Error("hashContent() should return different hash for different content")
	}

	if len(hash1) != 64 { // SHA-256 produces 64 hex characters
		t.Errorf("hashContent() length = %d, want 64", len(hash1))
	}
}

func TestExtractCodeBlocks(t *testing.T) {
	parser := NewResponseParser()

	response := `
` + "```go:main.go" + `
package main
` + "```" + `

` + "```python" + `
print("hello")
` + "```"

	blocks := parser.ExtractCodeBlocks(response)
	if len(blocks) != 2 {
		t.Errorf("ExtractCodeBlocks() got %d blocks, want 2", len(blocks))
	}
}
