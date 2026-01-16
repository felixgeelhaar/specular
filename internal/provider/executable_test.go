package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestExecutableProviderGenerateStreamHealth(t *testing.T) {
	t.Parallel()

	scriptPath := writeFakeProviderScript(t)
	config := &ProviderConfig{
		Name:    "fake-cli",
		Type:    ProviderTypeCLI,
		Version: "v1.0.0",
		Config: map[string]interface{}{
			"args": []interface{}{"--debug"},
			"capabilities": map[string]interface{}{
				"streaming": true,
			},
			"trust_level": string(TrustLevelVerified),
		},
	}

	provider, err := NewExecutableProvider(scriptPath, config)
	if err != nil {
		t.Fatalf("NewExecutableProvider() error = %v", err)
	}

	if !provider.GetCapabilities().SupportsStreaming {
		t.Fatalf("expected streaming capability to be enabled")
	}

	if provider.GetInfo().TrustLevel != TrustLevelVerified {
		t.Fatalf("expected trust level to be %s, got %s", TrustLevelVerified, provider.GetInfo().TrustLevel)
	}

	resp, err := provider.Generate(context.Background(), &GenerateRequest{Prompt: "hello"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if resp.Content != "hello from provider" {
		t.Fatalf("Generate() content = %s, want hello from provider", resp.Content)
	}

	if resp.Provider != "fake-cli" {
		t.Fatalf("Generate() provider = %s, want fake-cli", resp.Provider)
	}

	chunks, err := provider.Stream(context.Background(), &GenerateRequest{Prompt: "stream"})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	collected := make([]StreamChunk, 0, 2)
	for chunk := range chunks {
		collected = append(collected, chunk)
		if chunk.Done {
			break
		}
	}

	if len(collected) != 2 {
		t.Fatalf("expected 2 stream chunks, got %d", len(collected))
	}
	if !collected[1].Done {
		t.Fatal("expected final stream chunk to be marked as done")
	}

	if err := provider.Health(context.Background()); err != nil {
		t.Fatalf("Health() error = %v", err)
	}
}

func writeFakeProviderScript(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "fake-provider.sh")
	script := `#!/usr/bin/env bash
cmd=""
for arg in "$@"; do
  case "$arg" in
    generate|stream|health)
      cmd="$arg"
      break
      ;;
  esac
done

if [ -z "$cmd" ]; then
  echo "missing command" >&2
  exit 1
fi

case "$cmd" in
generate)
cat <<'EOF'
{"content":"hello from provider","tokens_used":1,"provider":"fake-cli"}
EOF
;;
stream)
cat <<'EOF'
{"content":"chunk one","delta":"chunk one","done":false,"tokens_used":0}
{"content":"chunk two","delta":"chunk two","done":true,"tokens_used":1}
EOF
;;
health)
echo "ok"
;;
*)
echo "unsupported command: $cmd" >&2
exit 1
;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
	return path
}
