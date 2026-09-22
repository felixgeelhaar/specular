package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felixgeelhaar/specular/internal/bundle"
	"github.com/felixgeelhaar/specular/internal/ux"
)

// sessionEvidenceBundleOptions configures the thin evidence packager used by
// `session wait --bundle`.
type sessionEvidenceBundleOptions struct {
	Output   string
	Policies []string
	Includes []string
	Quiet    bool
}

// runSessionEvidenceBundle creates a governance bundle from session evidence
// (attestations, APP .provenance.json, drift SARIF) plus optional policy
// library fragments.
func runSessionEvidenceBundle(opts sessionEvidenceBundleOptions) error {
	output := opts.Output
	if output == "" {
		output = "session-evidence.sbundle.tgz"
	}

	defaults := ux.NewPathDefaults()
	bo := bundle.BundleOptions{
		IncludePaths:    append([]string{}, opts.Includes...),
		GovernanceLevel: "L2",
		Metadata: map[string]string{
			"source": "session-wait-bundle",
		},
	}
	// Library fragments are control→evidence YAML, not execution policy.Policy —
	// include them as opaque evidence files for auditors.
	for _, p := range opts.Policies {
		if fileExists(p) && !containsPath(bo.IncludePaths, p) {
			bo.IncludePaths = append(bo.IncludePaths, p)
		}
	}
	if p := defaults.SpecFile(); fileExists(p) {
		bo.SpecPath = p
	}
	if p := defaults.SpecLockFile(); fileExists(p) {
		bo.LockPath = p
	}

	for _, candidate := range []string{"drift.sarif", filepath.Join(".specular", "drift.sarif")} {
		if fileExists(candidate) && !containsPath(bo.IncludePaths, candidate) {
			bo.IncludePaths = append(bo.IncludePaths, candidate)
		}
	}

	attestDir := filepath.Join(".specular", "sessions")
	if entries, err := os.ReadDir(attestDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".attestation.json") &&
				!strings.HasSuffix(name, ".provenance.json") {
				continue
			}
			path := filepath.Join(attestDir, name)
			if !containsPath(bo.IncludePaths, path) {
				bo.IncludePaths = append(bo.IncludePaths, path)
			}
		}
	}
	for _, p := range opts.Includes {
		if fileExists(p) && !containsPath(bo.IncludePaths, p) {
			bo.IncludePaths = append(bo.IncludePaths, p)
		}
	}

	if err := bo.Validate(); err != nil {
		return fmt.Errorf("session bundle: %w (install a policy library seed or run --attest/--gate first)", err)
	}

	builder, err := bundle.NewBuilder(bo)
	if err != nil {
		return fmt.Errorf("session bundle: builder: %w", err)
	}
	if buildErr := builder.Build(output); buildErr != nil {
		return fmt.Errorf("session bundle: build: %w", buildErr)
	}
	if !opts.Quiet {
		info, statErr := os.Stat(output)
		size := int64(0)
		if statErr == nil {
			size = info.Size()
		}
		fmt.Printf("\nSession evidence bundle: %s (%.1f KB)\n", output, float64(size)/1024)
		fmt.Printf("  Includes: %d\n", len(bo.IncludePaths))
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func containsPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}
