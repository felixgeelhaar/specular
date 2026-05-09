package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/felixgeelhaar/specular/internal/telemetry"
)

// activationMarkerName is the file persisted under .specular/ to track the
// activation funnel across sessions. It records when init started so that the
// next successful command can emit time-to-first-success.
const activationMarkerName = ".activation.json"

// activationMarker captures activation timing state across CLI invocations.
type activationMarker struct {
	StartedAt      time.Time `json:"started_at"`
	InitCompleteAt time.Time `json:"init_complete_at,omitempty"`
	FirstSuccessAt time.Time `json:"first_success_at,omitempty"`
}

// activationMarkerPath resolves the marker path inside the given .specular dir.
func activationMarkerPath(specDir string) string {
	return filepath.Join(specDir, activationMarkerName)
}

// findSpecularDir walks up from cwd looking for a .specular directory. Returns
// the directory path or empty string if none is found.
func findSpecularDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := cwd
	for {
		candidate := filepath.Join(dir, ".specular")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// writeActivationStart persists the activation start timestamp. It is invoked
// from the init command before user-visible work happens so that drop-off can
// be measured even if the user abandons mid-flow.
func writeActivationStart(specDir string, startedAt time.Time) error {
	marker := activationMarker{StartedAt: startedAt}
	return writeActivationMarker(specDir, marker)
}

// markActivationInitComplete updates the marker after init finishes
// successfully. It preserves the original StartedAt so the cross-session
// time-to-first-success calculation remains anchored to the first attempt.
func markActivationInitComplete(specDir string, completedAt time.Time) error {
	marker, err := readActivationMarker(specDir)
	if err != nil {
		// If the marker is missing or corrupt, recreate it with completedAt as
		// both start and complete to keep the funnel observable.
		marker = activationMarker{StartedAt: completedAt}
	}
	marker.InitCompleteAt = completedAt
	return writeActivationMarker(specDir, marker)
}

// recordFirstSuccessIfPending emits time-to-first-success the first time a
// non-init command finishes successfully after init. It is a no-op if the
// marker is missing, already recorded, or the command is one we want to
// exclude from activation tracking (init, help, version, completion).
func recordFirstSuccessIfPending(ctx context.Context, cmdName string) {
	if isActivationExcludedCommand(cmdName) {
		return
	}

	specDir := findSpecularDir()
	if specDir == "" {
		return
	}

	marker, err := readActivationMarker(specDir)
	if err != nil {
		return
	}
	if !marker.FirstSuccessAt.IsZero() || marker.StartedAt.IsZero() {
		return
	}

	now := time.Now()
	marker.FirstSuccessAt = now

	telemetry.RecordActivationStep(ctx, telemetry.ActivationStepFirstSuccess, telemetry.ActivationStatusOK)
	telemetry.RecordActivationDuration(ctx, telemetry.ActivationMilestoneFirstSuccess, now.Sub(marker.StartedAt))

	// Best-effort persistence: failing to write must not break the user's
	// command, so swallow the error after recording metrics.
	_ = writeActivationMarker(specDir, marker)
}

func isActivationExcludedCommand(name string) bool {
	switch name {
	case "init", "i", "new",
		"help", "version",
		"completion", "__complete",
		"specular": // root command name when no subcommand specified
		return true
	}
	return false
}

func readActivationMarker(specDir string) (activationMarker, error) {
	var marker activationMarker
	data, err := os.ReadFile(activationMarkerPath(specDir))
	if err != nil {
		return marker, err
	}
	if err := json.Unmarshal(data, &marker); err != nil {
		return marker, err
	}
	return marker, nil
}

func writeActivationMarker(specDir string, marker activationMarker) error {
	if specDir == "" {
		return errors.New("activation marker: empty .specular directory")
	}
	if err := os.MkdirAll(specDir, 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(activationMarkerPath(specDir), data, 0o600)
}
