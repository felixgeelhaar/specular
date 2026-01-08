package templates

import (
	"embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/pkg/specular/types"
)

//go:embed *.yaml
var templateFS embed.FS

// Template represents a pre-built spec template
type Template struct {
	Name        string       `yaml:"name"`
	ID          string       `yaml:"id"`
	Description string       `yaml:"description"`
	Language    string       `yaml:"language"`
	Tags        []string     `yaml:"tags"`
	Spec        SpecTemplate `yaml:"spec"`
}

// SpecTemplate is the template version of ProductSpec
type SpecTemplate struct {
	Product       string                `yaml:"product"`
	Goals         []string              `yaml:"goals"`
	Features      []FeatureTemplate     `yaml:"features"`
	NonFunctional NonFunctionalTemplate `yaml:"non_functional"`
	Acceptance    []string              `yaml:"acceptance"`
}

// FeatureTemplate is the template version of Feature
type FeatureTemplate struct {
	ID       string        `yaml:"id"`
	Title    string        `yaml:"title"`
	Desc     string        `yaml:"desc"`
	Priority string        `yaml:"priority"`
	API      []APITemplate `yaml:"api,omitempty"`
	Success  []string      `yaml:"success"`
	Trace    []string      `yaml:"trace"`
	Refs     []string      `yaml:"refs,omitempty"`
}

// APITemplate is the template version of API
type APITemplate struct {
	Method   string `yaml:"method"`
	Path     string `yaml:"path"`
	Request  string `yaml:"request,omitempty"`
	Response string `yaml:"response,omitempty"`
}

// NonFunctionalTemplate is the template version of NonFunctional
type NonFunctionalTemplate struct {
	Performance  []string `yaml:"performance,omitempty"`
	Security     []string `yaml:"security,omitempty"`
	Scalability  []string `yaml:"scalability,omitempty"`
	Availability []string `yaml:"availability,omitempty"`
}

// Registry holds available templates
type Registry struct {
	templates map[string]*Template
}

// NewRegistry creates a new template registry and loads embedded templates
func NewRegistry() (*Registry, error) {
	r := &Registry{
		templates: make(map[string]*Template),
	}

	// Load all embedded YAML files
	entries, err := templateFS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("failed to read template directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		data, err := templateFS.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read template %s: %w", entry.Name(), err)
		}

		var tmpl Template
		if err := yaml.Unmarshal(data, &tmpl); err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", entry.Name(), err)
		}

		r.templates[tmpl.ID] = &tmpl
	}

	return r, nil
}

// Get returns a template by ID
func (r *Registry) Get(id string) (*Template, error) {
	tmpl, ok := r.templates[id]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return tmpl, nil
}

// List returns all available templates
func (r *Registry) List() []*Template {
	result := make([]*Template, 0, len(r.templates))
	for _, tmpl := range r.templates {
		result = append(result, tmpl)
	}
	return result
}

// ListByTag returns templates matching a tag
func (r *Registry) ListByTag(tag string) []*Template {
	var result []*Template
	for _, tmpl := range r.templates {
		for _, t := range tmpl.Tags {
			if strings.EqualFold(t, tag) {
				result = append(result, tmpl)
				break
			}
		}
	}
	return result
}

// ListByLanguage returns templates for a specific language
func (r *Registry) ListByLanguage(lang string) []*Template {
	var result []*Template
	for _, tmpl := range r.templates {
		if strings.EqualFold(tmpl.Language, lang) {
			result = append(result, tmpl)
		}
	}
	return result
}

// ToProductSpec converts a template to a ProductSpec
func (t *Template) ToProductSpec(productName string) *spec.ProductSpec {
	// Use provided product name or template default
	name := productName
	if name == "" {
		name = t.Spec.Product
	}

	ps := &spec.ProductSpec{
		Product:    name,
		Goals:      t.Spec.Goals,
		Features:   make([]spec.Feature, len(t.Spec.Features)),
		Acceptance: t.Spec.Acceptance,
		NonFunctional: spec.NonFunctional{
			Performance:  t.Spec.NonFunctional.Performance,
			Security:     t.Spec.NonFunctional.Security,
			Scalability:  t.Spec.NonFunctional.Scalability,
			Availability: t.Spec.NonFunctional.Availability,
		},
	}

	// Convert features
	for i, ft := range t.Spec.Features {
		f := spec.Feature{
			ID:       types.FeatureID(ft.ID),
			Title:    ft.Title,
			Desc:     ft.Desc,
			Priority: types.Priority(ft.Priority),
			Success:  ft.Success,
			Trace:    ft.Trace,
			Refs:     ft.Refs,
		}

		// Convert API endpoints
		if len(ft.API) > 0 {
			f.API = make([]spec.API, len(ft.API))
			for j, api := range ft.API {
				f.API[j] = spec.API{
					Method:   api.Method,
					Path:     api.Path,
					Request:  api.Request,
					Response: api.Response,
				}
			}
		}

		ps.Features[i] = f
	}

	return ps
}

// AvailableTemplates returns a formatted list of available templates
func AvailableTemplates() string {
	registry, err := NewRegistry()
	if err != nil {
		return "Error loading templates"
	}

	var sb strings.Builder
	sb.WriteString("Available templates:\n")

	for _, tmpl := range registry.List() {
		sb.WriteString(fmt.Sprintf("  • %s - %s\n", tmpl.ID, tmpl.Description))
		if tmpl.Language != "" {
			sb.WriteString(fmt.Sprintf("    Language: %s\n", tmpl.Language))
		}
	}

	return sb.String()
}

// GetTemplateIDs returns all template IDs for validation
func GetTemplateIDs() []string {
	registry, err := NewRegistry()
	if err != nil {
		return nil
	}

	ids := make([]string, 0, len(registry.templates))
	for id := range registry.templates {
		ids = append(ids, id)
	}
	return ids
}
