package attestation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/auto"
	"github.com/felixgeelhaar/specular/internal/safeutil"
)

// Generator creates attestations for workflow executions
type Generator struct {
	signer  Signer
	version string
}

// NewGenerator creates a new attestation generator
func NewGenerator(signer Signer, version string) *Generator {
	return &Generator{
		signer:  signer,
		version: version,
	}
}

// Generate creates an attestation from a workflow result
func (g *Generator) Generate(result *auto.Result, config *auto.Config, planJSON []byte, outputJSON []byte) (*Attestation, error) {
	// Calculate hashes
	planHash := hashData(planJSON)
	outputHash := hashData(outputJSON)

	// Gather provenance data
	provenance, err := g.gatherProvenance(result, config)
	if err != nil {
		return nil, fmt.Errorf("failed to gather provenance: %w", err)
	}

	// Get workflow metadata from AutoOutput if available
	workflowID := "unknown"
	goal := config.Goal
	var startTime, endTime time.Time
	status := determineStatus(result)

	if result.AutoOutput != nil {
		workflowID = result.AutoOutput.Audit.CheckpointID
		goal = result.AutoOutput.Goal
		startTime = result.AutoOutput.Audit.StartedAt
		endTime = result.AutoOutput.Audit.CompletedAt
		status = result.AutoOutput.Status
	}

	// Create base attestation
	attestation := &Attestation{
		Version:    "1.0",
		WorkflowID: workflowID,
		Goal:       goal,
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   result.Duration.String(),
		Status:     status,
		Provenance: *provenance,
		PlanHash:   planHash,
		OutputHash: outputHash,
		SignedAt:   time.Now(),
		SignedBy:   g.signer.Identity(),
	}

	return g.signAttestation(attestation)
}

// gatherProvenance collects provenance information
func (g *Generator) gatherProvenance(result *auto.Result, config *auto.Config) (*Provenance, error) {
	hostname, _ := os.Hostname() // Hostname is best-effort, empty string is acceptable

	harness := config.Harness
	if harness == "" {
		harness = "specular-auto"
	}

	provenance := &Provenance{
		Hostname:        hostname,
		Platform:        runtime.GOOS,
		Arch:            runtime.GOARCH,
		SpecularVersion: g.version,
		Profile:         getProfileName(config),
		Models:          extractModelUsage(result),
		TotalCost:       result.TotalCost,
		TasksExecuted:   result.TasksExecuted,
		TasksFailed:     result.TasksFailed,
		Harness:         harness,
		WorktreePath:    config.WorktreePath,
		WorktreeBranch:  config.WorktreeBranch,
		WorktreeName:    config.WorktreeName,
	}

	// Prefer audit trail harness/worktree when AutoOutput is present
	if result.AutoOutput != nil {
		if result.AutoOutput.Audit.Harness != "" {
			provenance.Harness = result.AutoOutput.Audit.Harness
		}
		if result.AutoOutput.Audit.WorktreePath != "" {
			provenance.WorktreePath = result.AutoOutput.Audit.WorktreePath
			provenance.WorktreeBranch = result.AutoOutput.Audit.WorktreeBranch
			provenance.WorktreeName = result.AutoOutput.Audit.WorktreeName
		}
	}

	// Try to gather git information (from current cwd — typically the worktree)
	gitInfo, err := gatherGitInfo("")
	if err == nil {
		provenance.GitRepo = gitInfo.Repo
		provenance.GitCommit = gitInfo.Commit
		provenance.GitBranch = gitInfo.Branch
		provenance.GitDirty = gitInfo.Dirty
	}

	return provenance, nil
}

// SessionInput carries enough session metadata to emit an attestation without
// an auto.Result — used for native Claude/Codex/Gemini (and auto) sessions.
type SessionInput struct {
	ID             string
	Goal           string
	Harness        string
	Profile        string
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	WorktreePath   string
	WorktreeBranch string
	WorktreeName   string
}

// GenerateFromSession creates a signed attestation from a managed session record.
// Plan/output hashes are empty for native harnesses that do not emit auto JSON.
func (g *Generator) GenerateFromSession(in SessionInput) (*Attestation, error) {
	harness := strings.TrimSpace(in.Harness)
	if harness == "" {
		harness = "specular-auto"
	}
	profile := strings.TrimSpace(in.Profile)
	if profile == "" {
		profile = "ci"
	}
	hostname, _ := os.Hostname()
	provenance := &Provenance{
		Hostname:        hostname,
		Platform:        runtime.GOOS,
		Arch:            runtime.GOARCH,
		SpecularVersion: g.version,
		Profile:         profile,
		Models:          []ModelUsage{},
		Harness:         harness,
		WorktreePath:    in.WorktreePath,
		WorktreeBranch:  in.WorktreeBranch,
		WorktreeName:    in.WorktreeName,
	}
	gitDir := in.WorktreePath
	if gitInfo, err := gatherGitInfo(gitDir); err == nil {
		provenance.GitRepo = gitInfo.Repo
		provenance.GitCommit = gitInfo.Commit
		provenance.GitBranch = gitInfo.Branch
		provenance.GitDirty = gitInfo.Dirty
	}

	end := in.UpdatedAt
	if end.IsZero() {
		end = time.Now().UTC()
	}
	start := in.CreatedAt
	if start.IsZero() {
		start = end
	}
	attestation := &Attestation{
		Version:    "1.0",
		WorkflowID: "session-" + in.ID,
		Goal:       in.Goal,
		StartTime:  start,
		EndTime:    end,
		Duration:   end.Sub(start).String(),
		Status:     mapSessionAttestStatus(in.Status),
		Provenance: *provenance,
		PlanHash:   hashData(nil),
		OutputHash: hashData(nil),
		SignedAt:   time.Now().UTC(),
		SignedBy:   g.signer.Identity(),
	}
	return g.signAttestation(attestation)
}

func mapSessionAttestStatus(status string) string {
	switch status {
	case "completed":
		return "success"
	case "failed":
		return "failed"
	case "stopped":
		return "cancelled"
	default:
		return status
	}
}

func (g *Generator) signAttestation(attestation *Attestation) (*Attestation, error) {
	dataToSign, err := g.serializeForSigning(attestation)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize for signing: %w", err)
	}
	signature, publicKey, err := g.signer.Sign(dataToSign)
	if err != nil {
		return nil, fmt.Errorf("failed to sign attestation: %w", err)
	}
	attestation.Signature = EncodeSignature(signature)
	if ephemeralSigner, ok := g.signer.(*EphemeralSigner); ok {
		pubKeyBytes, pubErr := ephemeralSigner.PublicKey()
		if pubErr != nil {
			return nil, fmt.Errorf("failed to encode public key: %w", pubErr)
		}
		attestation.PublicKey = EncodePublicKey(pubKeyBytes)
	} else {
		_ = publicKey
		attestation.PublicKey = ""
	}
	return attestation, nil
}

// serializeForSigning creates a canonical JSON representation for signing
func (g *Generator) serializeForSigning(attestation *Attestation) ([]byte, error) {
	// Create a copy without signature fields
	copy := *attestation
	copy.Signature = ""
	copy.PublicKey = ""

	return copy.ToJSON()
}

// hashData computes SHA256 hash of data
func hashData(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// determineStatus determines the workflow status
func determineStatus(result *auto.Result) string {
	if result.TasksFailed > 0 {
		return "failed"
	}
	return "success"
}

// getProfileName extracts the profile name from config
func getProfileName(config *auto.Config) string {
	// This is a simplified version - would need to pass profile name through config
	return "default"
}

// extractModelUsage extracts model usage information from result
func extractModelUsage(result *auto.Result) []ModelUsage {
	// This is a placeholder - would need to track model usage in the result
	// For now, return empty slice
	return []ModelUsage{}
}

// gitInfo holds git repository information
type gitInfo struct {
	Repo   string
	Commit string
	Branch string
	Dirty  bool
}

// gatherGitInfo collects git repository information.
// When dir is non-empty, git runs in that directory (session worktree).
func gatherGitInfo(dir string) (*gitInfo, error) {
	info := &gitInfo{}
	run := func(args ...string) string {
		gitCmd, cmdErr := safeutil.SafeCommand(context.Background(), "git", args...)
		if cmdErr != nil {
			return ""
		}
		if dir != "" {
			gitCmd.Dir = dir
		}
		output, err := gitCmd.Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(output))
	}

	info.Repo = run("config", "--get", "remote.origin.url")
	info.Commit = run("rev-parse", "HEAD")
	info.Branch = run("rev-parse", "--abbrev-ref", "HEAD")
	if st := run("status", "--porcelain"); st != "" {
		info.Dirty = true
	}

	if info.Repo == "" && info.Commit == "" {
		return nil, fmt.Errorf("not a git repository")
	}
	return info, nil
}
