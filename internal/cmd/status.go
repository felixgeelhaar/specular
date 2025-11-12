package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/detect"
	"github.com/felixgeelhaar/specular/internal/spec"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show environment and project status",
	Long: `Display an overview of the Specular environment and current project state.

Status information includes:
  • Environment health (Docker, AI providers)
  • Specification status (spec.yaml, spec.lock.json)
  • Plan status (plan.json)
  • Build and evaluation results
  • Configuration overview

Examples:
  # Display status in default text format
  specular debug status

  # Output as JSON for scripting
  specular debug status --format json

  # Output as YAML
  specular debug status --format yaml
`,
	RunE: runStatus,
}

// StatusReport represents the complete project status
type StatusReport struct {
	Timestamp   string               `json:"timestamp"`
	Environment EnvironmentStatus    `json:"environment"`
	Project     ProjectStatus        `json:"project"`
	Spec        SpecStatus           `json:"spec"`
	Plan        PlanStatus           `json:"plan"`
	LastBuild   *BuildStatus         `json:"last_build,omitempty"`
	Issues      []string             `json:"issues,omitempty"`
	Warnings    []string             `json:"warnings,omitempty"`
	NextSteps   []string             `json:"next_steps,omitempty"`
	Healthy     bool                 `json:"healthy"`
}

type EnvironmentStatus struct {
	Runtime   string   `json:"runtime"`    // "docker", "podman", or "none"
	Providers []string `json:"providers"`  // Available AI providers
	APIKeys   int      `json:"api_keys"`   // Number of configured API keys
	Healthy   bool     `json:"healthy"`
}

type ProjectStatus struct {
	Directory   string `json:"directory"`
	Initialized bool   `json:"initialized"` // .specular directory exists
	GitRepo     bool   `json:"git_repo"`
	GitBranch   string `json:"git_branch,omitempty"`
	GitDirty    bool   `json:"git_dirty"`
}

type SpecStatus struct {
	Exists      bool      `json:"exists"`
	Locked      bool      `json:"locked"`
	Version     string    `json:"version,omitempty"`
	Features    int       `json:"features"`
	LastUpdated time.Time `json:"last_updated,omitempty"`
}

type PlanStatus struct {
	Exists      bool      `json:"exists"`
	Tasks       int       `json:"tasks"`
	LastUpdated time.Time `json:"last_updated,omitempty"`
}

type BuildStatus struct {
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Duration  string    `json:"duration,omitempty"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	// Build status report
	report, err := buildStatusReport()
	if err != nil {
		return ux.FormatError(err, "building status report")
	}

	// Output report
	return outputStatus(cmdCtx, report)
}

func buildStatusReport() (*StatusReport, error) {
	report := &StatusReport{
		Timestamp: time.Now().Format(time.RFC3339),
		Issues:    []string{},
		Warnings:  []string{},
		NextSteps: []string{},
	}

	// Detect environment
	ctx, err := detect.DetectAll()
	if err != nil {
		return nil, fmt.Errorf("detecting environment: %w", err)
	}

	// Environment status
	report.Environment = buildEnvironmentStatus(ctx)

	// Project status
	report.Project = buildProjectStatus(ctx)

	// Spec status
	report.Spec = buildSpecStatus()

	// Plan status
	report.Plan = buildPlanStatus()

	// Build status (last build)
	report.LastBuild = getLastBuildStatus()

	// Analyze and determine issues/warnings
	analyzeStatus(report)

	// Determine overall health
	report.Healthy = len(report.Issues) == 0

	return report, nil
}

func buildEnvironmentStatus(ctx *detect.Context) EnvironmentStatus {
	env := EnvironmentStatus{
		Runtime:   ctx.Runtime,
		Providers: []string{},
		APIKeys:   0,
	}

	// Count available providers and API keys
	for name, info := range ctx.Providers {
		if info.Available {
			env.Providers = append(env.Providers, name)
			if info.EnvSet {
				env.APIKeys++
			}
		}
	}

	// Environment is healthy if we have runtime and at least one provider
	env.Healthy = ctx.Runtime != "" && len(env.Providers) > 0

	return env
}

func buildProjectStatus(ctx *detect.Context) ProjectStatus {
	cwd, _ := os.Getwd()

	project := ProjectStatus{
		Directory:   filepath.Base(cwd),
		GitRepo:     ctx.Git.Initialized,
		GitBranch:   ctx.Git.Branch,
		GitDirty:    ctx.Git.Dirty,
	}

	// Check if .specular directory exists
	defaults := ux.NewPathDefaults()
	if _, err := os.Stat(defaults.SpecularDir); err == nil {
		project.Initialized = true
	}

	return project
}

func buildSpecStatus() SpecStatus {
	defaults := ux.NewPathDefaults()
	status := SpecStatus{}

	// Check if spec.yaml exists
	specPath := defaults.SpecFile()
	if info, err := os.Stat(specPath); err == nil {
		status.Exists = true
		status.LastUpdated = info.ModTime()
	}

	// Check if spec.lock.json exists and load it
	lockPath := defaults.SpecLockFile()
	if lockInfo, err := os.Stat(lockPath); err == nil {
		status.Locked = true
		status.LastUpdated = lockInfo.ModTime()

		// Try to load lock file to get version and feature count
		if lock, err := spec.LoadSpecLock(lockPath); err == nil {
			status.Version = lock.Version
			status.Features = len(lock.Features)
		}
	}

	return status
}

func buildPlanStatus() PlanStatus {
	defaults := ux.NewPathDefaults()
	status := PlanStatus{}

	planPath := defaults.PlanFile()
	if info, err := os.Stat(planPath); err == nil {
		status.Exists = true
		status.LastUpdated = info.ModTime()

		// TODO: Load plan.json to count tasks
		// For now, we just know it exists
	}

	return status
}

func getLastBuildStatus() *BuildStatus {
	// TODO: Implement by checking .specular/runs/ directory for latest run manifest
	// For now, return nil
	return nil
}

func analyzeStatus(report *StatusReport) {
	// Environment issues
	if report.Environment.Runtime == "" {
		report.Issues = append(report.Issues, "No container runtime detected (Docker/Podman required)")
		report.NextSteps = append(report.NextSteps, "Install Docker from https://docker.com")
	}

	if len(report.Environment.Providers) == 0 {
		report.Issues = append(report.Issues, "No AI providers detected")
		report.NextSteps = append(report.NextSteps, "Install Ollama or set API keys (OPENAI_API_KEY, ANTHROPIC_API_KEY)")
	}

	// Project issues
	if !report.Project.Initialized {
		report.Issues = append(report.Issues, "Project not initialized")
		report.NextSteps = append(report.NextSteps, "Run 'specular init' to initialize project")
		return // Can't proceed without initialization
	}

	// Spec issues
	if !report.Spec.Exists {
		report.NextSteps = append(report.NextSteps, "Create specification with 'specular interview' or 'specular spec generate'")
		return
	}

	if !report.Spec.Locked {
		report.Warnings = append(report.Warnings, "Specification not locked")
		report.NextSteps = append(report.NextSteps, "Lock specification with 'specular spec lock'")
		return
	}

	// Plan issues
	if !report.Plan.Exists {
		report.NextSteps = append(report.NextSteps, "Generate plan with 'specular plan'")
		return
	}

	// If we got here, ready to build
	if len(report.NextSteps) == 0 {
		report.NextSteps = append(report.NextSteps, "Execute plan with 'specular build'")
	}

	// Git warnings
	if report.Project.GitRepo && report.Project.GitDirty {
		report.Warnings = append(report.Warnings, "Git working directory has uncommitted changes")
	}
}

func outputStatus(cmdCtx *CommandContext, report *StatusReport) error {
	// For JSON and YAML, use the formatter
	if cmdCtx.Format == "json" || cmdCtx.Format == "yaml" {
		formatter, err := ux.NewFormatter(cmdCtx.Format, &ux.FormatterOptions{
			NoColor: cmdCtx.NoColor,
		})
		if err != nil {
			return err
		}
		return formatter.Format(report)
	}

	// Text output
	printStatusHeader()
	printEnvironmentSection(report)
	printProjectSection(report)
	printSpecSection(report)
	printPlanSection(report)
	printIssuesSection(report)
	printWarningsSection(report)
	printNextStepsSection(report)
	printOverallStatus(report)

	return nil
}

func printStatusHeader() {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      Project Status                          ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func printEnvironmentSection(report *StatusReport) {
	fmt.Println("Environment:")
	if report.Environment.Runtime != "" {
		fmt.Printf("  ✓ Runtime: %s\n", report.Environment.Runtime)
	} else {
		fmt.Println("  ✗ Runtime: None detected")
	}

	if len(report.Environment.Providers) > 0 {
		fmt.Printf("  ✓ AI Providers: %d available\n", len(report.Environment.Providers))
		for _, provider := range report.Environment.Providers {
			fmt.Printf("    • %s\n", provider)
		}
	} else {
		fmt.Println("  ✗ AI Providers: None detected")
	}

	if report.Environment.APIKeys > 0 {
		fmt.Printf("  ✓ API Keys: %d configured\n", report.Environment.APIKeys)
	}
	fmt.Println()
}

func printProjectSection(report *StatusReport) {
	fmt.Println("Project:")
	fmt.Printf("  Directory: %s\n", report.Project.Directory)

	if report.Project.Initialized {
		fmt.Println("  ✓ Initialized (.specular directory exists)")
	} else {
		fmt.Println("  ✗ Not initialized")
	}

	if report.Project.GitRepo {
		fmt.Printf("  ✓ Git repository (branch: %s", report.Project.GitBranch)
		if report.Project.GitDirty {
			fmt.Println(", uncommitted changes)")
		} else {
			fmt.Println(", clean)")
		}
	}
	fmt.Println()
}

func printSpecSection(report *StatusReport) {
	fmt.Println("Specification:")
	if report.Spec.Exists {
		fmt.Print("  ✓ Spec file exists")
		if !report.Spec.LastUpdated.IsZero() {
			fmt.Printf(" (updated %s)", formatTime(report.Spec.LastUpdated))
		}
		fmt.Println()
	} else {
		fmt.Println("  ✗ Spec file not found")
	}

	if report.Spec.Locked {
		fmt.Printf("  ✓ Locked (version: %s, %d features)\n", report.Spec.Version, report.Spec.Features)
	} else if report.Spec.Exists {
		fmt.Println("  ⚠ Not locked")
	}
	fmt.Println()
}

func printPlanSection(report *StatusReport) {
	fmt.Println("Plan:")
	if report.Plan.Exists {
		fmt.Print("  ✓ Plan file exists")
		if !report.Plan.LastUpdated.IsZero() {
			fmt.Printf(" (updated %s)", formatTime(report.Plan.LastUpdated))
		}
		fmt.Println()
	} else {
		fmt.Println("  ✗ Plan file not found")
	}
	fmt.Println()
}

func printIssuesSection(report *StatusReport) {
	if len(report.Issues) > 0 {
		fmt.Println("❌ Issues:")
		for _, issue := range report.Issues {
			fmt.Printf("   • %s\n", issue)
		}
		fmt.Println()
	}
}

func printWarningsSection(report *StatusReport) {
	if len(report.Warnings) > 0 {
		fmt.Println("⚠️  Warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("   • %s\n", warning)
		}
		fmt.Println()
	}
}

func printNextStepsSection(report *StatusReport) {
	if len(report.NextSteps) > 0 {
		fmt.Println("📋 Next Steps:")
		for i, step := range report.NextSteps {
			fmt.Printf("   %d. %s\n", i+1, step)
		}
		fmt.Println()
	}
}

func printOverallStatus(report *StatusReport) {
	if report.Healthy {
		fmt.Println("✅ Project is healthy and ready")
	} else {
		fmt.Println("❌ Project has issues that need attention")
	}
}

func formatTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
