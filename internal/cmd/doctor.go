package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/felixgeelhaar/specular/internal/detect"
	"github.com/felixgeelhaar/specular/internal/ux"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run system diagnostics and health checks",
	Long: `Run comprehensive system diagnostics to check if Specular is properly configured.

Checks include:
  • Container runtime (Docker/Podman) availability
  • AI provider availability and configuration
  • Project structure (.specular/ directory)
  • Required files (spec.yaml, policy.yaml, router.yaml)
  • Git repository status
  • Environment variables and API keys

Examples:
  # Run diagnostics with colored output
  specular doctor

  # Output as JSON for CI/CD
  specular doctor --format json
`,
	RunE: runDoctor,
}

// DoctorReport represents the complete health check report
type DoctorReport struct {
	Docker    *DoctorCheck            `json:"docker"`
	Podman    *DoctorCheck            `json:"podman,omitempty"`
	Providers map[string]*DoctorCheck `json:"providers"`
	Spec      *DoctorCheck            `json:"spec"`
	Lock      *DoctorCheck            `json:"lock"`
	Policy    *DoctorCheck            `json:"policy"`
	Router    *DoctorCheck            `json:"router"`
	Git       *DoctorCheck            `json:"git"`
	Issues    []string                `json:"issues"`
	Warnings  []string                `json:"warnings"`
	NextSteps []string                `json:"next_steps"`
	Healthy   bool                    `json:"healthy"`
}

// DoctorCheck represents a single health check result
type DoctorCheck struct {
	Name    string                 `json:"name"`
	Status  string                 `json:"status"` // "ok", "warning", "error", "missing"
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	// Extract command context
	cmdCtx, err := NewCommandContext(cmd)
	if err != nil {
		return fmt.Errorf("failed to create command context: %w", err)
	}

	// Detect context
	ctx, err := detect.DetectAll()
	if err != nil {
		return ux.FormatError(err, "detecting system context")
	}

	// Run all health checks
	report := &DoctorReport{
		Providers: make(map[string]*DoctorCheck),
		Issues:    []string{},
		Warnings:  []string{},
		NextSteps: []string{},
	}

	// Check container runtime
	checkContainerRuntime(ctx, report)

	// Check AI providers
	checkProviders(ctx, report)

	// Check project structure
	checkProjectStructure(report)

	// Check Git
	checkGit(ctx, report)

	// Generate next steps
	generateNextSteps(report)

	// Determine overall health
	report.Healthy = len(report.Issues) == 0

	// Output report
	if cmdCtx.Format == "json" {
		return outputJSON(report)
	}

	return outputText(report)
}

func checkContainerRuntime(ctx *detect.Context, report *DoctorReport) {
	if ctx.Docker.Available {
		status := "ok"
		message := fmt.Sprintf("Docker is available (version %s)", ctx.Docker.Version)
		if !ctx.Docker.Running {
			status = "warning"
			message = fmt.Sprintf("Docker CLI found (version %s) but daemon may not be running", ctx.Docker.Version)
			report.Warnings = append(report.Warnings, "Docker daemon might not be running")
		}

		report.Docker = &DoctorCheck{
			Name:    "Docker",
			Status:  status,
			Message: message,
			Details: map[string]interface{}{
				"version": ctx.Docker.Version,
				"running": ctx.Docker.Running,
			},
		}
	} else if ctx.Podman.Available {
		report.Podman = &DoctorCheck{
			Name:    "Podman",
			Status:  "ok",
			Message: fmt.Sprintf("Podman is available (version %s)", ctx.Podman.Version),
			Details: map[string]interface{}{
				"version": ctx.Podman.Version,
				"running": ctx.Podman.Running,
			},
		}
	} else {
		report.Docker = &DoctorCheck{
			Name:    "Docker",
			Status:  "error",
			Message: "No container runtime detected (Docker or Podman)",
		}
		report.Issues = append(report.Issues, "Container runtime not found")
	}
}

func checkProviders(ctx *detect.Context, report *DoctorReport) {
	foundProvider := false

	for name, info := range ctx.Providers {
		check := &DoctorCheck{
			Name: name,
		}

		if info.Available {
			foundProvider = true
			status := "ok"
			message := fmt.Sprintf("%s is available (%s)", name, info.Type)

			if info.EnvVar != "" {
				check.Details = map[string]interface{}{
					"type":    info.Type,
					"env_var": info.EnvVar,
					"env_set": info.EnvSet,
				}

				if info.Version != "" {
					check.Details["version"] = info.Version
				}

				if !info.EnvSet {
					status = "warning"
					message = fmt.Sprintf("%s available but %s not set", name, info.EnvVar)
					report.Warnings = append(report.Warnings, fmt.Sprintf("%s requires %s environment variable", name, info.EnvVar))
				}
			}

			check.Status = status
			check.Message = message
		} else {
			check.Status = "missing"
			check.Message = fmt.Sprintf("%s is not available", name)
		}

		report.Providers[name] = check
	}

	if !foundProvider {
		report.Issues = append(report.Issues, "No AI providers detected")
	}
}

func checkProjectStructure(report *DoctorReport) {
	defaults := ux.NewPathDefaults()

	// Check spec file
	specPath := defaults.SpecFile()
	if _, err := os.Stat(specPath); err == nil {
		report.Spec = &DoctorCheck{
			Name:    "Spec",
			Status:  "ok",
			Message: fmt.Sprintf("Spec file exists at %s", specPath),
			Details: map[string]interface{}{
				"path": specPath,
			},
		}
	} else {
		report.Spec = &DoctorCheck{
			Name:    "Spec",
			Status:  "missing",
			Message: "Spec file not found",
		}
		report.NextSteps = append(report.NextSteps, "Create spec with 'specular interview' or 'specular spec generate'")
	}

	// Check lock file
	lockPath := defaults.SpecLockFile()
	if _, err := os.Stat(lockPath); err == nil {
		report.Lock = &DoctorCheck{
			Name:    "SpecLock",
			Status:  "ok",
			Message: fmt.Sprintf("SpecLock file exists at %s", lockPath),
			Details: map[string]interface{}{
				"path": lockPath,
			},
		}
	} else {
		report.Lock = &DoctorCheck{
			Name:    "SpecLock",
			Status:  "missing",
			Message: "SpecLock file not found",
		}
		if report.Spec != nil && report.Spec.Status == "ok" {
			report.NextSteps = append(report.NextSteps, "Generate lock file with 'specular spec lock'")
		}
	}

	// Check policy file
	policyPath := defaults.PolicyFile()
	if _, err := os.Stat(policyPath); err == nil {
		report.Policy = &DoctorCheck{
			Name:    "Policy",
			Status:  "ok",
			Message: fmt.Sprintf("Policy file exists at %s", policyPath),
			Details: map[string]interface{}{
				"path": policyPath,
			},
		}
	} else {
		report.Policy = &DoctorCheck{
			Name:    "Policy",
			Status:  "warning",
			Message: "Policy file not found (will use defaults)",
		}
		report.Warnings = append(report.Warnings, "No policy file found - using default policies")
	}

	// Check router file
	routerPath := defaults.RouterFile()
	if _, err := os.Stat(routerPath); err == nil {
		report.Router = &DoctorCheck{
			Name:    "Router",
			Status:  "ok",
			Message: fmt.Sprintf("Router file exists at %s", routerPath),
			Details: map[string]interface{}{
				"path": routerPath,
			},
		}
	} else {
		report.Router = &DoctorCheck{
			Name:    "Router",
			Status:  "warning",
			Message: "Router file not found",
		}
		report.Warnings = append(report.Warnings, "No router file found - run 'specular init' to create one")
	}
}

func checkGit(ctx *detect.Context, report *DoctorReport) {
	if ctx.Git.Initialized {
		status := "ok"
		message := fmt.Sprintf("Git repository initialized (branch: %s)", ctx.Git.Branch)

		if ctx.Git.Dirty {
			status = "warning"
			message = fmt.Sprintf("Git repository has %d uncommitted changes", ctx.Git.Uncommitted)
			report.Warnings = append(report.Warnings, fmt.Sprintf("%d uncommitted changes in Git", ctx.Git.Uncommitted))
		}

		report.Git = &DoctorCheck{
			Name:    "Git",
			Status:  status,
			Message: message,
			Details: map[string]interface{}{
				"root":        filepath.Base(ctx.Git.Root),
				"branch":      ctx.Git.Branch,
				"dirty":       ctx.Git.Dirty,
				"uncommitted": ctx.Git.Uncommitted,
			},
		}
	} else {
		report.Git = &DoctorCheck{
			Name:    "Git",
			Status:  "missing",
			Message: "Not a Git repository",
		}
		report.NextSteps = append(report.NextSteps, "Initialize Git repository with 'git init'")
	}
}

func generateNextSteps(report *DoctorReport) {
	// Add logical next steps based on current state
	if report.Spec == nil || report.Spec.Status == "missing" {
		// Already added in checkProjectStructure
		return
	}

	if report.Lock == nil || report.Lock.Status == "missing" {
		// Already added in checkProjectStructure
		return
	}

	// If spec and lock exist, suggest plan generation
	if report.Spec.Status == "ok" && report.Lock.Status == "ok" {
		report.NextSteps = append(report.NextSteps, "Generate plan with 'specular plan'")
	}

	// If issues exist, prioritize fixing them
	if len(report.Issues) > 0 {
		if report.Docker != nil && report.Docker.Status == "error" {
			report.NextSteps = append([]string{"Install Docker from https://docker.com"}, report.NextSteps...)
		}
	}
}

func outputJSON(report *DoctorReport) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func outputText(report *DoctorReport) error {
	printHeader()
	printContainerRuntime(report)
	printAIProviders(report)
	printProjectStructure(report)
	printGitRepository(report)
	printIssues(report)
	printWarnings(report)
	printNextSteps(report)
	return printOverallHealth(report)
}

// printHeader prints the diagnostics header
func printHeader() {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    System Diagnostics                        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// printContainerRuntime prints container runtime checks
func printContainerRuntime(report *DoctorReport) {
	fmt.Println("Container Runtime:")
	if report.Docker != nil {
		printCheck(report.Docker)
	}
	if report.Podman != nil {
		printCheck(report.Podman)
	}
	fmt.Println()
}

// printAIProviders prints AI provider checks
func printAIProviders(report *DoctorReport) {
	fmt.Println("AI Providers:")
	for _, name := range []string{"ollama", "anthropic", "openai", "gemini", "claude"} {
		if check, ok := report.Providers[name]; ok {
			printCheck(check)
		}
	}
	fmt.Println()
}

// printProjectStructure prints project structure checks
func printProjectStructure(report *DoctorReport) {
	fmt.Println("Project Structure:")
	if report.Spec != nil {
		printCheck(report.Spec)
	}
	if report.Lock != nil {
		printCheck(report.Lock)
	}
	if report.Policy != nil {
		printCheck(report.Policy)
	}
	if report.Router != nil {
		printCheck(report.Router)
	}
	fmt.Println()
}

// printGitRepository prints git repository check
func printGitRepository(report *DoctorReport) {
	if report.Git != nil {
		fmt.Println("Git Repository:")
		printCheck(report.Git)
		fmt.Println()
	}
}

// printIssues prints issues if any exist
func printIssues(report *DoctorReport) {
	if len(report.Issues) > 0 {
		fmt.Println("❌ Issues:")
		for _, issue := range report.Issues {
			fmt.Printf("   • %s\n", issue)
		}
		fmt.Println()
	}
}

// printWarnings prints warnings if any exist
func printWarnings(report *DoctorReport) {
	if len(report.Warnings) > 0 {
		fmt.Println("⚠️  Warnings:")
		for _, warning := range report.Warnings {
			fmt.Printf("   • %s\n", warning)
		}
		fmt.Println()
	}
}

// printNextSteps prints next steps if any exist
func printNextSteps(report *DoctorReport) {
	if len(report.NextSteps) > 0 {
		fmt.Println("📋 Next Steps:")
		for i, step := range report.NextSteps {
			fmt.Printf("   %d. %s\n", i+1, step)
		}
		fmt.Println()
	}
}

// printOverallHealth prints overall health status and returns error if unhealthy
func printOverallHealth(report *DoctorReport) error {
	if report.Healthy {
		fmt.Println("✅ System is healthy and ready to use!")
		return nil
	}

	fmt.Println("❌ System has issues that need attention")
	if len(report.Issues) == 0 {
		fmt.Println("   (Warnings present but system is functional)")
	}
	return fmt.Errorf("system health check failed")
}

func printCheck(check *DoctorCheck) {
	icon := " "
	switch check.Status {
	case "ok":
		icon = "✓"
	case "warning":
		icon = "⚠"
	case "error":
		icon = "✗"
	case "missing":
		icon = "○"
	}

	fmt.Printf("  %s %s: %s\n", icon, check.Name, check.Message)
}
