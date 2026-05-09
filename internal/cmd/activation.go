package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/felixgeelhaar/specular/internal/telemetry"
)

const (
	// activationMarkerName is the file persisted under .specular/ to track the
	// activation funnel across sessions. It records when init started so that
	// the next successful command can emit time-to-first-success.
	activationMarkerName = ".activation.json"

	// activationFirstSuccessSentinel is an exclusive-create lockfile that the
	// PostRunE hook uses to claim emission of first_success exactly once,
	// even under concurrent CLI invocations sharing a project directory.
	activationFirstSuccessSentinel = ".first_success.done"

	// activationOwnershipSibling must exist next to the marker before we will
	// write into a discovered .specular directory; this guards against a
	// stray .specular dir on a shared host hijacking the marker. spec.yaml
	// is the canonical artifact that runInit always creates, so it doubles
	// as the project-ownership marker.
	activationOwnershipSibling = "spec.yaml"

	// activationMarkerVersion is the current schema version of the marker
	// file. Readers must reject markers with a Version greater than this.
	activationMarkerVersion = 1

	// findSpecularDirMaxDepth bounds the parent walk so a symlink loop or
	// a stray .specular at / cannot be reached.
	findSpecularDirMaxDepth = 64
)

// activationMarker captures activation timing state across CLI invocations.
// The Version field is the schema version; bumping it requires a migration
// path documented in the activation package README.
type activationMarker struct {
	Version        int       `json:"v"`
	StartedAt      time.Time `json:"started_at"`
	InitCompleteAt time.Time `json:"init_complete_at,omitempty"`
	FirstSuccessAt time.Time `json:"first_success_at,omitempty"`
}

// activationMarkerPath resolves the marker path inside the given .specular dir.
func activationMarkerPath(specDir string) string {
	return filepath.Join(specDir, activationMarkerName)
}

// findSpecularDir walks up from cwd looking for a .specular directory that we
// can confidently claim as the current project's. The walk is bounded by:
//   - findSpecularDirMaxDepth iterations (defends against symlink loops),
//   - the user's home directory (we never write a marker above $HOME),
//   - presence of the activationOwnershipSibling inside the candidate
//     (defends against a stray .specular created by another user / job on a
//     shared host).
func findSpecularDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	homeBoundary := ""
	if home, herr := os.UserHomeDir(); herr == nil {
		homeBoundary = filepath.Clean(home)
	}

	dir := cwd
	for i := 0; i < findSpecularDirMaxDepth; i++ {
		candidate := filepath.Join(dir, ".specular")
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			if specularDirIsOwned(candidate) {
				return candidate
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		// Stop walking once we've crossed the user's home directory.
		if homeBoundary != "" && filepath.Clean(dir) == homeBoundary {
			return ""
		}
		dir = parent
	}
	return ""
}

// specularDirIsOwned returns true when the candidate .specular directory has
// the ownership sibling file we expect. Without this, a stray empty
// .specular/ on a shared filesystem could capture marker writes belonging to
// another project.
func specularDirIsOwned(candidate string) bool {
	if _, err := os.Stat(filepath.Join(candidate, activationOwnershipSibling)); err == nil {
		return true
	}
	return false
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
// non-init command finishes successfully after init. Concurrent invocations
// (e.g. parallel `specular plan` and `specular build` shells) must not
// double-emit, so the function claims a sentinel file with O_CREATE|O_EXCL
// before recording: only the goroutine that creates the sentinel is allowed
// to emit and persist the timestamp. The marker write itself uses
// temp+rename to keep the JSON readable under concurrent observers.
func recordFirstSuccessIfPending(ctx context.Context, cmdName string) {
	if isActivationExcludedCommand(cmdName) {
		return
	}

	specDir := findSpecularDir()
	if specDir == "" {
		return
	}

	marker, err := readActivationMarker(specDir)
	if err != nil || marker.StartedAt.IsZero() || !marker.FirstSuccessAt.IsZero() {
		return
	}

	sentinelPath := filepath.Join(specDir, activationFirstSuccessSentinel)
	f, err := os.OpenFile(sentinelPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		// Sentinel already claimed by another invocation — that goroutine
		// owns the metric emission for this project lifetime.
		return
	}
	_ = f.Close()

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
	if marker.Version > activationMarkerVersion {
		return marker, fmt.Errorf(
			"activation marker version %d is newer than supported %d; refusing to overwrite",
			marker.Version, activationMarkerVersion,
		)
	}
	return marker, nil
}

// writeActivationMarker persists the marker atomically: the JSON is written
// to a sibling .tmp file and then renamed into place so concurrent readers
// never observe a partially-written file. MkdirAll keeps the call usable
// from runInit (which creates .specular/) without callers having to
// pre-create the directory.
func writeActivationMarker(specDir string, marker activationMarker) error {
	if specDir == "" {
		return errors.New("activation marker: empty .specular directory")
	}
	if err := os.MkdirAll(specDir, 0o750); err != nil {
		return err
	}

	if marker.Version == 0 {
		marker.Version = activationMarkerVersion
	}

	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}

	target := activationMarkerPath(specDir)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
