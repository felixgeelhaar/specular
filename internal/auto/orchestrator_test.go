package auto

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/felixgeelhaar/specular/internal/hooks"
	"github.com/felixgeelhaar/specular/internal/trace"
)

type recordingHook struct {
	executed atomic.Bool
}

func (h *recordingHook) Name() string {
	return "recording"
}

func (h *recordingHook) EventTypes() []hooks.EventType {
	return []hooks.EventType{hooks.EventWorkflowStart}
}

func (h *recordingHook) Execute(ctx context.Context, event *hooks.Event) error {
	h.executed.Store(true)
	return nil
}

func (h *recordingHook) Enabled() bool {
	return true
}

func (h *recordingHook) wasExecuted() bool {
	return h.executed.Load()
}

func TestOrchestratorSettersAndHookTrigger(t *testing.T) {
	orch := NewOrchestrator(nil, DefaultConfig())

	cfg := trace.DefaultConfig()
	cfg.Enabled = false
	tracer, err := trace.NewLogger(cfg)
	if err != nil {
		t.Fatalf("failed to create trace logger: %v", err)
	}

	orch.SetTracer(tracer)
	if orch.tracer != tracer {
		t.Fatal("SetTracer did not store the provided tracer")
	}

	workDir := t.TempDir()
	patchDir := filepath.Join(t.TempDir(), "patches")
	orch.SetPatchGenerator(workDir, patchDir)
	if orch.patchGenerator == nil || orch.patchWriter == nil {
		t.Fatalf("expected patch generator/writer to be initialized")
	}

	registry := hooks.NewRegistry()
	hook := &recordingHook{}
	if err := registry.Register(hook); err != nil {
		t.Fatalf("failed to register hook: %v", err)
	}

	orch.SetHookRegistry(registry)
	if orch.hookRegistry != registry {
		t.Fatalf("SetHookRegistry did not store registry")
	}

	ctx := context.Background()
	orch.triggerHook(ctx, hooks.EventWorkflowStart, "wf", map[string]interface{}{"state": "testing"})
	if !hook.wasExecuted() {
		t.Fatal("expected hook to be executed after triggerHook")
	}
}
