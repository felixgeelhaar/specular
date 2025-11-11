package attestation

import (
	"crypto"
	"encoding/json"
	"time"
)

// Attestation represents a cryptographic attestation of a workflow execution
type Attestation struct {
	// Version of the attestation format
	Version string `json:"version"`

	// Workflow information
	WorkflowID string    `json:"workflowId"`
	Goal       string    `json:"goal"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
	Duration   string    `json:"duration"`
	Status     string    `json:"status"` // success, failed, cancelled

	// Provenance data
	Provenance Provenance `json:"provenance"`

	// Plan and output hashes
	PlanHash   string `json:"planHash"`   // SHA256 of the execution plan
	OutputHash string `json:"outputHash"` // SHA256 of the JSON output

	// Signature metadata
	SignedAt  time.Time `json:"signedAt"`
	SignedBy  string    `json:"signedBy"`  // Email or identity
	Signature string    `json:"signature"` // Base64-encoded signature
	PublicKey string    `json:"publicKey"` // Base64-encoded public key
}

// Provenance contains information about the execution environment
type Provenance struct {
	// Host information
	Hostname string `json:"hostname"`
	Platform string `json:"platform"` // darwin, linux, windows
	Arch     string `json:"arch"`     // amd64, arm64

	// Git context (if available)
	GitRepo   string `json:"gitRepo,omitempty"`
	GitCommit string `json:"gitCommit,omitempty"`
	GitBranch string `json:"gitBranch,omitempty"`
	GitDirty  bool   `json:"gitDirty,omitempty"`

	// Specular version
	SpecularVersion string `json:"specularVersion"`

	// Profile used
	Profile string `json:"profile"`

	// Model information
	Models []ModelUsage `json:"models"`

	// Cost tracking
	TotalCost     float64 `json:"totalCost"`
	TasksExecuted int     `json:"tasksExecuted"`
	TasksFailed   int     `json:"tasksFailed"`
}

// ModelUsage tracks which models were used during execution
type ModelUsage struct {
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	Requests int     `json:"requests"`
	Cost     float64 `json:"cost"`
}

// Signer is the interface for signing attestations
type Signer interface {
	// Sign generates a signature for the attestation
	Sign(data []byte) (signature []byte, publicKey crypto.PublicKey, err error)

	// Identity returns the identity of the signer (e.g., email)
	Identity() string
}

// Verifier is the interface for verifying attestations
type Verifier interface {
	// Verify checks the signature on an attestation
	Verify(attestation *Attestation) error

	// VerifyProvenance validates the provenance data
	VerifyProvenance(attestation *Attestation) error
}

// ToJSON serializes the attestation to JSON
func (a *Attestation) ToJSON() ([]byte, error) {
	return json.MarshalIndent(a, "", "  ")
}

// FromJSON deserializes an attestation from JSON
func FromJSON(data []byte) (*Attestation, error) {
	var attestation Attestation
	if err := json.Unmarshal(data, &attestation); err != nil {
		return nil, err
	}
	return &attestation, nil
}
