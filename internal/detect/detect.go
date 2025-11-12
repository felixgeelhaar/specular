package detect

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Context represents the detected project context
type Context struct {
	// Container runtime
	Docker  ContainerRuntime
	Podman  ContainerRuntime
	Runtime string // "docker", "podman", or ""

	// AI Providers
	Providers map[string]ProviderInfo

	// Language and framework
	Languages  []string
	Frameworks []string

	// Git context
	Git GitContext

	// CI/CD environment
	CI CIInfo
}

// ContainerRuntime holds container runtime detection results
type ContainerRuntime struct {
	Available bool
	Version   string
	Running   bool
}

// ProviderInfo holds AI provider detection results
type ProviderInfo struct {
	Name      string
	Available bool
	Type      string // "cli", "api", "local"
	Version   string
	EnvVar    string // Required environment variable
	EnvSet    bool   // Is environment variable set
}

// GitContext holds Git repository information
type GitContext struct {
	Initialized bool
	Root        string
	Dirty       bool
	Uncommitted int
	Branch      string
}

// CIInfo holds CI/CD environment information
type CIInfo struct {
	Detected bool
	Name     string // "github", "gitlab", "jenkins", "circleci", etc.
}

// DetectAll runs all detection checks and returns the context
func DetectAll() (*Context, error) {
	ctx := &Context{
		Providers: make(map[string]ProviderInfo),
	}

	// Detect container runtime
	ctx.Docker = detectDocker()
	ctx.Podman = detectPodman()
	if ctx.Docker.Available {
		ctx.Runtime = "docker"
	} else if ctx.Podman.Available {
		ctx.Runtime = "podman"
	}

	// Detect AI providers
	ctx.Providers["ollama"] = detectOllama()
	ctx.Providers["claude"] = detectClaude()
	ctx.Providers["openai"] = detectOpenAI()
	ctx.Providers["gemini"] = detectGemini()
	ctx.Providers["anthropic"] = detectAnthropic()

	// Detect languages and frameworks
	ctx.Languages, ctx.Frameworks = detectLanguagesAndFrameworks()

	// Detect Git context
	ctx.Git = detectGit()

	// Detect CI environment
	ctx.CI = detectCI()

	return ctx, nil
}

// detectDocker checks if Docker is available and running
func detectDocker() ContainerRuntime {
	runtime := ContainerRuntime{}

	// Check if docker command exists
	path, err := exec.LookPath("docker")
	if err != nil {
		return runtime
	}

	runtime.Available = true

	// Get version
	cmd := exec.Command(path, "version", "--format", "{{.Server.Version}}")
	output, err := cmd.Output()
	if err == nil {
		runtime.Version = strings.TrimSpace(string(output))
		runtime.Running = true
	} else {
		// Docker CLI exists but daemon might not be running
		cmd = exec.Command(path, "--version")
		output, err = cmd.Output()
		if err == nil {
			// Parse version from "Docker version 24.0.7, build afdd53b"
			parts := strings.Split(string(output), " ")
			if len(parts) >= 3 {
				runtime.Version = strings.TrimSuffix(parts[2], ",")
			}
		}
	}

	return runtime
}

// detectPodman checks if Podman is available
func detectPodman() ContainerRuntime {
	runtime := ContainerRuntime{}

	path, err := exec.LookPath("podman")
	if err != nil {
		return runtime
	}

	runtime.Available = true

	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err == nil {
		// Parse version from "podman version 4.7.2"
		parts := strings.Split(string(output), " ")
		if len(parts) >= 3 {
			runtime.Version = strings.TrimSpace(parts[2])
		}
		runtime.Running = true
	}

	return runtime
}

// detectOllama checks if Ollama is available
func detectOllama() ProviderInfo {
	info := ProviderInfo{
		Name: "ollama",
		Type: "local",
	}

	path, err := exec.LookPath("ollama")
	if err != nil {
		return info
	}

	info.Available = true

	// Get version
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err == nil {
		info.Version = strings.TrimSpace(string(output))
	}

	return info
}

// detectClaude checks if Claude CLI is available
func detectClaude() ProviderInfo {
	info := ProviderInfo{
		Name:   "claude",
		Type:   "cli",
		EnvVar: "ANTHROPIC_API_KEY",
	}

	path, err := exec.LookPath("claude")
	if err != nil {
		return info
	}

	info.Available = true
	info.EnvSet = os.Getenv(info.EnvVar) != ""

	// Get version if available
	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err == nil {
		info.Version = strings.TrimSpace(string(output))
	}

	return info
}

// detectProviderWithCLI checks if a provider is available via CLI or API key
func detectProviderWithCLI(name, cliName, envVar string) ProviderInfo {
	info := ProviderInfo{
		Name:   name,
		Type:   "api",
		EnvVar: envVar,
	}

	// Check for CLI
	path, err := exec.LookPath(cliName)
	if err == nil {
		info.Available = true
		info.Type = "cli"

		cmd := exec.Command(path, "--version")
		var output []byte
		output, err = cmd.Output()
		if err == nil {
			info.Version = strings.TrimSpace(string(output))
		}
	}

	// Check for API key (always check, even if CLI not found)
	info.EnvSet = os.Getenv(info.EnvVar) != ""
	if info.EnvSet {
		info.Available = true
	}

	return info
}

// detectOpenAI checks if OpenAI is available (API or CLI)
func detectOpenAI() ProviderInfo {
	return detectProviderWithCLI("openai", "openai", "OPENAI_API_KEY")
}

// detectGemini checks if Gemini is available
func detectGemini() ProviderInfo {
	return detectProviderWithCLI("gemini", "gemini", "GEMINI_API_KEY")
}

// detectAnthropic checks if Anthropic API is available
func detectAnthropic() ProviderInfo {
	info := ProviderInfo{
		Name:   "anthropic",
		Type:   "api",
		EnvVar: "ANTHROPIC_API_KEY",
	}

	// Check for API key
	info.EnvSet = os.Getenv(info.EnvVar) != ""
	info.Available = info.EnvSet

	return info
}

// detectLanguagesAndFrameworks detects programming languages and frameworks in current directory
func detectLanguagesAndFrameworks() ([]string, []string) {
	languages := []string{}
	frameworks := []string{}

	// Language detection based on common files
	fileChecks := map[string]string{
		"package.json":     "javascript",
		"go.mod":           "go",
		"requirements.txt": "python",
		"Pipfile":          "python",
		"pyproject.toml":   "python",
		"Cargo.toml":       "rust",
		"pom.xml":          "java",
		"build.gradle":     "java",
		"Gemfile":          "ruby",
		"composer.json":    "php",
	}

	for file, lang := range fileChecks {
		if _, err := os.Stat(file); err == nil {
			if !contains(languages, lang) {
				languages = append(languages, lang)
			}
		}
	}

	// Framework detection
	if _, err := os.Stat("package.json"); err == nil {
		// Check for common Node.js frameworks
		if hasInFile("package.json", "react") {
			frameworks = append(frameworks, "react")
		}
		if hasInFile("package.json", "next") {
			frameworks = append(frameworks, "nextjs")
		}
		if hasInFile("package.json", "express") {
			frameworks = append(frameworks, "express")
		}
		if hasInFile("package.json", "vue") {
			frameworks = append(frameworks, "vue")
		}
	}

	if _, err := os.Stat("go.mod"); err == nil {
		// Check for common Go frameworks
		if hasInFile("go.mod", "gin-gonic/gin") {
			frameworks = append(frameworks, "gin")
		}
		if hasInFile("go.mod", "gofiber/fiber") {
			frameworks = append(frameworks, "fiber")
		}
	}

	return languages, frameworks
}

// detectGit detects Git repository information
func detectGit() GitContext {
	git := GitContext{}

	// Check if in a git repository
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return git
	}

	git.Initialized = true
	git.Root = strings.TrimSpace(string(output))

	// Get current branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err = cmd.Output()
	if err == nil {
		git.Branch = strings.TrimSpace(string(output))
	}

	// Check if working directory is dirty
	cmd = exec.Command("git", "status", "--porcelain")
	output, err = cmd.Output()
	if err == nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			statusLines := strings.Split(trimmed, "\n")
			git.Uncommitted = len(statusLines)
			git.Dirty = true
		}
	}

	return git
}

// detectCI detects CI/CD environment
func detectCI() CIInfo {
	ci := CIInfo{}

	// Check for common CI environment variables
	ciChecks := map[string]string{
		"GITHUB_ACTIONS": "github",
		"GITLAB_CI":      "gitlab",
		"JENKINS_HOME":   "jenkins",
		"CIRCLECI":       "circleci",
		"TRAVIS":         "travis",
		"BUILDKITE":      "buildkite",
	}

	for envVar, name := range ciChecks {
		if os.Getenv(envVar) != "" {
			ci.Detected = true
			ci.Name = name
			return ci
		}
	}

	return ci
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func hasInFile(filename, searchString string) bool {
	content, err := os.ReadFile(filename)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), searchString)
}

// GetRecommendedProviders returns a list of recommended providers based on detected context
func (c *Context) GetRecommendedProviders() []string {
	recommended := []string{}

	// Prioritize local-first if ollama is available
	if p, ok := c.Providers["ollama"]; ok && p.Available {
		recommended = append(recommended, "ollama")
	}

	// Add API providers that have keys set
	for name, info := range c.Providers {
		if name != "ollama" && info.Available && info.EnvSet {
			recommended = append(recommended, name)
		}
	}

	// If no providers detected, recommend ollama as default
	if len(recommended) == 0 {
		recommended = append(recommended, "ollama")
	}

	return recommended
}

// Summary returns a human-readable summary of the detected context
func (c *Context) Summary() string {
	var sb strings.Builder

	sb.WriteString("Detected Context:\n\n")

	// Container runtime
	if c.Runtime != "" {
		runtime := c.Docker
		if c.Runtime == "podman" {
			runtime = c.Podman
		}
		fmt.Fprintf(&sb, "  Container Runtime: %s (version %s, running: %v)\n",
			c.Runtime, runtime.Version, runtime.Running)
	} else {
		sb.WriteString("  Container Runtime: Not detected\n")
	}

	// AI Providers
	sb.WriteString("\n  AI Providers:\n")
	for name, info := range c.Providers {
		if info.Available {
			fmt.Fprintf(&sb, "    ✓ %s (%s", name, info.Type)
			if info.Version != "" {
				fmt.Fprintf(&sb, ", version %s", info.Version)
			}
			if info.EnvVar != "" {
				fmt.Fprintf(&sb, ", API key: %v", info.EnvSet)
			}
			sb.WriteString(")\n")
		}
	}

	// Languages and Frameworks
	if len(c.Languages) > 0 {
		fmt.Fprintf(&sb, "\n  Languages: %s\n", strings.Join(c.Languages, ", "))
	}
	if len(c.Frameworks) > 0 {
		fmt.Fprintf(&sb, "  Frameworks: %s\n", strings.Join(c.Frameworks, ", "))
	}

	// Git
	if c.Git.Initialized {
		fmt.Fprintf(&sb, "\n  Git Repository: %s (branch: %s, dirty: %v, uncommitted: %d)\n",
			filepath.Base(c.Git.Root), c.Git.Branch, c.Git.Dirty, c.Git.Uncommitted)
	}

	// CI
	if c.CI.Detected {
		fmt.Fprintf(&sb, "  CI Environment: %s\n", c.CI.Name)
	}

	return sb.String()
}
