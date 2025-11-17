package exec

import (
	"fmt"
	"testing"
)

// BenchmarkBuildDockerArgs_Minimal benchmarks with minimal configuration
func BenchmarkBuildDockerArgs_Minimal(b *testing.B) {
	step := Step{
		ID:     "step-001",
		Runner: "docker",
		Image:  "alpine:latest",
		Cmd:    []string{"echo", "hello"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := buildDockerArgs(step)
		if len(args) < 5 {
			b.Fatalf("Expected at least 5 args, got %d", len(args))
		}
	}
}

// BenchmarkBuildDockerArgs_WithWorkdir benchmarks with working directory
func BenchmarkBuildDockerArgs_WithWorkdir(b *testing.B) {
	step := Step{
		ID:      "step-002",
		Runner:  "docker",
		Image:   "golang:1.21",
		Cmd:     []string{"go", "build", "./..."},
		Workdir: "/Users/test/project",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := buildDockerArgs(step)
		if len(args) < 10 {
			b.Fatalf("Expected at least 10 args, got %d", len(args))
		}
	}
}

// BenchmarkBuildDockerArgs_WithEnv benchmarks with environment variables
func BenchmarkBuildDockerArgs_WithEnv(b *testing.B) {
	step := Step{
		ID:     "step-003",
		Runner: "docker",
		Image:  "node:18",
		Cmd:    []string{"npm", "test"},
		Env: map[string]string{
			"NODE_ENV":     "test",
			"CI":           "true",
			"API_KEY":      "test-key",
			"DATABASE_URL": "postgres://localhost:5432/test",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := buildDockerArgs(step)
		if len(args) < 15 {
			b.Fatalf("Expected at least 15 args, got %d", len(args))
		}
	}
}

// BenchmarkBuildDockerArgs_WithResourceLimits benchmarks with CPU and memory limits
func BenchmarkBuildDockerArgs_WithResourceLimits(b *testing.B) {
	step := Step{
		ID:     "step-004",
		Runner: "docker",
		Image:  "python:3.11",
		Cmd:    []string{"python", "script.py"},
		CPU:    "2.0",
		Mem:    "512m",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := buildDockerArgs(step)
		if len(args) < 12 {
			b.Fatalf("Expected at least 12 args, got %d", len(args))
		}
	}
}

// BenchmarkBuildDockerArgs_WithNetwork benchmarks with network configuration
func BenchmarkBuildDockerArgs_WithNetwork(b *testing.B) {
	step := Step{
		ID:      "step-005",
		Runner:  "docker",
		Image:   "alpine:latest",
		Cmd:     []string{"wget", "https://example.com"},
		Network: "bridge",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := buildDockerArgs(step)
		if len(args) < 10 {
			b.Fatalf("Expected at least 10 args, got %d", len(args))
		}
	}
}

// BenchmarkBuildDockerArgs_FullConfig benchmarks with all configuration options
func BenchmarkBuildDockerArgs_FullConfig(b *testing.B) {
	env := make(map[string]string)
	for i := 0; i < 10; i++ {
		env[fmt.Sprintf("VAR_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	step := Step{
		ID:      "step-006",
		Runner:  "docker",
		Image:   "ubuntu:22.04",
		Cmd:     []string{"bash", "-c", "echo hello && ls -la"},
		Workdir: "/Users/test/complex-project",
		Env:     env,
		Network: "host",
		CPU:     "4.0",
		Mem:     "2g",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := buildDockerArgs(step)
		if len(args) < 30 {
			b.Fatalf("Expected at least 30 args, got %d", len(args))
		}
	}
}

// BenchmarkBuildDockerArgs_ManyEnvVars benchmarks with many environment variables
func BenchmarkBuildDockerArgs_ManyEnvVars(b *testing.B) {
	env := make(map[string]string)
	for i := 0; i < 50; i++ {
		env[fmt.Sprintf("ENV_VAR_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	step := Step{
		ID:     "step-007",
		Runner: "docker",
		Image:  "node:18",
		Cmd:    []string{"node", "app.js"},
		Env:    env,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := buildDockerArgs(step)
		if len(args) < 100 {
			b.Fatalf("Expected at least 100 args, got %d", len(args))
		}
	}
}

// BenchmarkBuildDockerArgs_LongCommand benchmarks with long command
func BenchmarkBuildDockerArgs_LongCommand(b *testing.B) {
	cmd := make([]string, 50)
	cmd[0] = "bash"
	cmd[1] = "-c"
	for i := 2; i < 50; i++ {
		cmd[i] = fmt.Sprintf("arg%d", i)
	}

	step := Step{
		ID:     "step-008",
		Runner: "docker",
		Image:  "alpine:latest",
		Cmd:    cmd,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := buildDockerArgs(step)
		if len(args) < 55 {
			b.Fatalf("Expected at least 55 args, got %d", len(args))
		}
	}
}

// BenchmarkBuildDockerArgs_ComplexPath benchmarks with complex working directory path
func BenchmarkBuildDockerArgs_ComplexPath(b *testing.B) {
	step := Step{
		ID:      "step-009",
		Runner:  "docker",
		Image:   "golang:1.21",
		Cmd:     []string{"go", "test", "-v", "./..."},
		Workdir: "/Users/developer/projects/complex-microservice-architecture/backend/services/auth-service/v2",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := buildDockerArgs(step)
		if len(args) < 15 {
			b.Fatalf("Expected at least 15 args, got %d", len(args))
		}
	}
}

// BenchmarkBuildDockerArgs_Parallel benchmarks parallel argument building
func BenchmarkBuildDockerArgs_Parallel(b *testing.B) {
	step := Step{
		ID:      "step-010",
		Runner:  "docker",
		Image:   "python:3.11",
		Cmd:     []string{"python", "-m", "pytest", "tests/"},
		Workdir: "/workspace",
		Env: map[string]string{
			"PYTHONPATH": "/workspace/src",
			"ENV":        "test",
		},
		CPU: "2.0",
		Mem: "1g",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			args := buildDockerArgs(step)
			if len(args) < 20 {
				b.Fatalf("Expected at least 20 args, got %d", len(args))
			}
		}
	})
}

// BenchmarkBuildDockerArgs_Realistic benchmarks a realistic production scenario
func BenchmarkBuildDockerArgs_Realistic(b *testing.B) {
	step := Step{
		ID:      "task-build-api",
		Runner:  "docker",
		Image:   "golang:1.21-alpine",
		Cmd:     []string{"go", "build", "-o", "/app/bin/api", "./cmd/api"},
		Workdir: "/Users/developer/projects/microservices/api-gateway",
		Env: map[string]string{
			"CGO_ENABLED":   "0",
			"GOOS":          "linux",
			"GOARCH":        "amd64",
			"GO111MODULE":   "on",
			"GOFLAGS":       "-mod=readonly",
			"BUILD_VERSION": "1.2.3",
		},
		Network: "none",
		CPU:     "2.0",
		Mem:     "1g",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		args := buildDockerArgs(step)
		if len(args) < 25 {
			b.Fatalf("Expected at least 25 args, got %d", len(args))
		}
	}
}
