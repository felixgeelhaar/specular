package attestation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/felixgeelhaar/specular/internal/auto"
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

	// Serialize attestation data for signing (without signature fields)
	dataToSign, err := g.serializeForSigning(attestation)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize for signing: %w", err)
	}

	// Sign the attestation
	signature, publicKey, err := g.signer.Sign(dataToSign)
	if err != nil {
		return nil, fmt.Errorf("failed to sign attestation: %w", err)
	}

	// Encode signature and public key
	attestation.Signature = EncodeSignature(signature)

	// Get public key bytes
	if ephemeralSigner, ok := g.signer.(*EphemeralSigner); ok {
		var pubKeyBytes []byte
		pubKeyBytes, err = ephemeralSigner.PublicKey()
		if err != nil {
			return nil, fmt.Errorf("failed to encode public key: %w", err)
		}
		attestation.PublicKey = EncodePublicKey(pubKeyBytes)
	} else {
		// For other signer types, we'll need to handle this differently
		_ = publicKey              // Use the returned publicKey parameter
		attestation.PublicKey = "" // Placeholder
	}

	return attestation, nil
}

// gatherProvenance collects provenance information
func (g *Generator) gatherProvenance(result *auto.Result, config *auto.Config) (*Provenance, error) {
	hostname, _ := os.Hostname() // Hostname is best-effort, empty string is acceptable

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
	}

	// Try to gather git information
	gitInfo, err := gatherGitInfo()
	if err == nil {
		provenance.GitRepo = gitInfo.Repo
		provenance.GitCommit = gitInfo.Commit
		provenance.GitBranch = gitInfo.Branch
		provenance.GitDirty = gitInfo.Dirty
	}

	return provenance, nil
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

// gatherGitInfo collects git repository information
func gatherGitInfo() (*gitInfo, error) {
	info := &gitInfo{}

	// Get remote URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err == nil {
		info.Repo = strings.TrimSpace(string(output))
	}

	// Get current commit
	cmd = exec.Command("git", "rev-parse", "HEAD")
	output, err = cmd.Output()
	if err == nil {
		info.Commit = strings.TrimSpace(string(output))
	}

	// Get current branch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err = cmd.Output()
	if err == nil {
		info.Branch = strings.TrimSpace(string(output))
	}

	// Check for uncommitted changes
	cmd = exec.Command("git", "status", "--porcelain")
	output, err = cmd.Output()
	if err == nil {
		info.Dirty = len(strings.TrimSpace(string(output))) > 0
	}

	// Return error if we couldn't get any git info
	if info.Repo == "" && info.Commit == "" {
		return nil, fmt.Errorf("not a git repository")
	}

	return info, nil
}
