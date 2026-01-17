package plugin

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

// ScaffoldConfig contains configuration for plugin scaffolding
type ScaffoldConfig struct {
	Name        string
	Type        string
	Lang        string
	Author      string
	Description string
	Version     string
}

// GetSupportedLanguages returns available template languages
func GetSupportedLanguages() []string {
	return []string{"go", "python", "node", "shell"}
}

// Scaffold creates a new plugin from templates
func Scaffold(dir string, cfg ScaffoldConfig) error {
	// Set defaults
	if cfg.Version == "" {
		cfg.Version = "0.1.0"
	}
	if cfg.Description == "" {
		cfg.Description = fmt.Sprintf("A Specular %s plugin", cfg.Type)
	}

	// Validate language
	validLangs := GetSupportedLanguages()
	if !containsString(validLangs, cfg.Lang) {
		return fmt.Errorf("unsupported language: %s (valid: %v)", cfg.Lang, validLangs)
	}

	// Validate type
	validTypes := []string{"provider", "validator", "formatter", "hook", "notifier"}
	if !containsString(validTypes, cfg.Type) {
		return fmt.Errorf("unsupported plugin type: %s (valid: %v)", cfg.Type, validTypes)
	}

	// Get template path
	tmplPath := fmt.Sprintf("templates/%s/%s", cfg.Lang, cfg.Type)

	// Check if template exists
	entries, err := templateFS.ReadDir(tmplPath)
	if err != nil {
		return fmt.Errorf("template not found for %s/%s: %w", cfg.Lang, cfg.Type, err)
	}

	// Create plugin directory
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Process each template file
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := processTemplate(tmplPath, entry.Name(), dir, cfg); err != nil {
			return err
		}
	}

	return nil
}

func processTemplate(tmplPath, filename, outDir string, cfg ScaffoldConfig) error {
	// Read template content
	content, err := templateFS.ReadFile(filepath.Join(tmplPath, filename))
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", filename, err)
	}

	// Parse template
	tmpl, err := template.New(filename).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", filename, err)
	}

	// Determine output filename (strip .tmpl suffix)
	outName := filename
	if filepath.Ext(outName) == ".tmpl" {
		outName = outName[:len(outName)-5]
	}

	// Create output file
	outPath := filepath.Join(outDir, outName)
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", outPath, err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, cfg); err != nil {
		return fmt.Errorf("failed to execute template %s: %w", filename, err)
	}

	// Make entrypoints executable
	if isEntrypoint(outName) {
		// #nosec G302 -- Entrypoint scripts need execute permission
		if err := os.Chmod(outPath, 0700); err != nil {
			return fmt.Errorf("failed to make %s executable: %w", outPath, err)
		}
	}

	return nil
}

func isEntrypoint(name string) bool {
	switch name {
	case "main.go", "main.py", "index.js", "entrypoint.sh":
		return true
	}
	return false
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// PrintNextSteps prints language-specific next steps after scaffolding
func PrintNextSteps(pluginName, lang string) {
	fmt.Println("Next steps:")
	fmt.Println()

	switch lang {
	case "go":
		fmt.Printf("  1. cd %s\n", pluginName)
		fmt.Println("  2. Edit main.go to implement your plugin logic")
		fmt.Println("  3. go build -o " + pluginName)
		fmt.Printf("  4. echo '{\"action\":\"health\"}' | ./%s\n", pluginName)
		fmt.Printf("  5. specular plugin install ./%s\n", pluginName)
	case "python":
		fmt.Printf("  1. cd %s\n", pluginName)
		fmt.Println("  2. Edit main.py to implement your plugin logic")
		fmt.Println("  3. echo '{\"action\":\"health\"}' | python3 main.py")
		fmt.Printf("  4. specular plugin install ./%s\n", pluginName)
	case "node":
		fmt.Printf("  1. cd %s\n", pluginName)
		fmt.Println("  2. Edit index.js to implement your plugin logic")
		fmt.Println("  3. npm install (if you add dependencies)")
		fmt.Println("  4. echo '{\"action\":\"health\"}' | node index.js")
		fmt.Printf("  5. specular plugin install ./%s\n", pluginName)
	default: // shell
		fmt.Printf("  1. cd %s\n", pluginName)
		fmt.Println("  2. Edit entrypoint.sh to implement your plugin logic")
		fmt.Println("  3. chmod +x entrypoint.sh")
		fmt.Println("  4. echo '{\"action\":\"health\"}' | ./entrypoint.sh")
		fmt.Printf("  5. specular plugin install ./%s\n", pluginName)
	}
}
