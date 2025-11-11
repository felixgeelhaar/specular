package patch

import (
	"encoding/json"
	"time"
)

// Patch represents a single patch with metadata
type Patch struct {
	// Metadata
	StepID      string    `json:"stepId"`
	StepType    string    `json:"stepType"`
	Timestamp   time.Time `json:"timestamp"`
	WorkflowID  string    `json:"workflowId"`
	Description string    `json:"description"`

	// Patch content
	Files []FilePatch `json:"files"`

	// Statistics
	FilesChanged int `json:"filesChanged"`
	Insertions   int `json:"insertions"`
	Deletions    int `json:"deletions"`
}

// FilePatch represents changes to a single file
type FilePatch struct {
	// File identification
	Path    string     `json:"path"`
	OldPath string     `json:"oldPath,omitempty"` // For renames
	Status  FileStatus `json:"status"`            // added, modified, deleted, renamed

	// Content
	OldContent string `json:"oldContent,omitempty"` // For rollback
	NewContent string `json:"newContent,omitempty"` // For forward application
	Diff       string `json:"diff"`                 // Unified diff format

	// Statistics
	Insertions int `json:"insertions"`
	Deletions  int `json:"deletions"`
}

// FileStatus represents the type of change to a file
type FileStatus string

const (
	FileStatusAdded    FileStatus = "added"
	FileStatusModified FileStatus = "modified"
	FileStatusDeleted  FileStatus = "deleted"
	FileStatusRenamed  FileStatus = "renamed"
)

// PatchMetadata contains high-level patch information
type PatchMetadata struct {
	StepID      string    `json:"stepId"`
	StepType    string    `json:"stepType"`
	Timestamp   time.Time `json:"timestamp"`
	WorkflowID  string    `json:"workflowId"`
	Description string    `json:"description"`

	// Summary statistics
	FilesChanged int `json:"filesChanged"`
	Insertions   int `json:"insertions"`
	Deletions    int `json:"deletions"`
}

// ToJSON converts patch to JSON
func (p *Patch) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// FromJSON parses a patch from JSON
func FromJSON(data []byte) (*Patch, error) {
	var patch Patch
	if err := json.Unmarshal(data, &patch); err != nil {
		return nil, err
	}
	return &patch, nil
}

// GetMetadata returns patch metadata without full content
func (p *Patch) GetMetadata() *PatchMetadata {
	return &PatchMetadata{
		StepID:       p.StepID,
		StepType:     p.StepType,
		Timestamp:    p.Timestamp,
		WorkflowID:   p.WorkflowID,
		Description:  p.Description,
		FilesChanged: p.FilesChanged,
		Insertions:   p.Insertions,
		Deletions:    p.Deletions,
	}
}

// IsEmpty returns true if the patch contains no changes
func (p *Patch) IsEmpty() bool {
	return len(p.Files) == 0
}

// CalculateStats updates the patch statistics from file patches
func (p *Patch) CalculateStats() {
	p.FilesChanged = len(p.Files)
	p.Insertions = 0
	p.Deletions = 0

	for _, filePatch := range p.Files {
		p.Insertions += filePatch.Insertions
		p.Deletions += filePatch.Deletions
	}
}
