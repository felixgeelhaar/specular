package docgen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/felixgeelhaar/specular/internal/provider"
)

// DocProvider describes a provider in governance documents.
type DocProvider struct {
	Name         string
	Source       string
	TrustLevel   string
	Description  string
	Capabilities string
	Hints        string
}

// DocContext contains data shared across governance documents.
type DocContext struct {
	ProjectName                 string
	Timestamp                   time.Time
	Governance                  string
	GovernanceDescription       string
	ProviderStrategy            string
	ProviderStrategyDescription string
	RecommendedProviders        []string
	Providers                   []DocProvider
	Features                    []string
}

// FormatDetectionHints returns a human-readable hint summary.
func FormatDetectionHints(h provider.ProviderDetectionHints) string {
	var parts []string
	if len(h.Binaries) > 0 {
		parts = append(parts, fmt.Sprintf("binaries: %s", strings.Join(h.Binaries, ", ")))
	}
	if len(h.EnvVars) > 0 {
		parts = append(parts, fmt.Sprintf("env: %s", strings.Join(h.EnvVars, ", ")))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("Hints: %s", strings.Join(parts, "; "))
}

// WriteDocs renders governance documents into the target directory.
func WriteDocs(dir string, ctx DocContext) error {
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("create docs directory: %w", err)
	}
	templates := map[string]*template.Template{
		"prd.md":     template.Must(template.New("prd").Parse(prdTemplate)),
		"vision.md":  template.Must(template.New("vision").Parse(visionTemplate)),
		"roadmap.md": template.Must(template.New("roadmap").Parse(roadmapTemplate)),
		"tdd.md":     template.Must(template.New("tdd").Parse(tddTemplate)),
	}

	for name, tmpl := range templates {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, ctx); err != nil {
			return fmt.Errorf("render %s: %w", name, err)
		}
		targetPath := filepath.Join(dir, name)
		if err := os.WriteFile(targetPath, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	return nil
}
