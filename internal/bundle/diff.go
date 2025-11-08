package bundle

import (
	"fmt"
	"sort"
)

// DiffResult represents the differences between two bundles.
type DiffResult struct {
	FilesAdded              []FileEntry
	FilesRemoved            []FileEntry
	FilesModified           []FileChange
	ApprovalsAdded          []*Approval
	ApprovalsRemoved        []*Approval
	AttestationChanged      bool
	MetadataChanged         bool
	ManifestMetadataChanges map[string]string
}

// FileChange represents a modified file with its old and new checksums.
type FileChange struct {
	Path        string
	OldChecksum string
	NewChecksum string
}

// DiffBundles compares two bundles and returns their differences.
func DiffBundles(bundleA, bundleB *Bundle) (*DiffResult, error) {
	if bundleA == nil || bundleB == nil {
		return nil, fmt.Errorf("cannot diff nil bundles")
	}

	if bundleA.Manifest == nil || bundleB.Manifest == nil {
		return nil, fmt.Errorf("bundles must have valid manifests")
	}

	result := &DiffResult{
		FilesAdded:              []FileEntry{},
		FilesRemoved:            []FileEntry{},
		FilesModified:           []FileChange{},
		ApprovalsAdded:          []*Approval{},
		ApprovalsRemoved:        []*Approval{},
		ManifestMetadataChanges: make(map[string]string),
	}

	// Compare files
	compareFiles(bundleA.Manifest, bundleB.Manifest, result)

	// Compare approvals
	compareApprovals(bundleA.Approvals, bundleB.Approvals, result)

	// Compare attestations
	result.AttestationChanged = attestationsChanged(bundleA.Attestation, bundleB.Attestation)

	// Compare metadata
	compareManifestMetadata(bundleA.Manifest, bundleB.Manifest, result)

	return result, nil
}

// compareFiles compares file entries between two manifests.
func compareFiles(manifestA, manifestB *Manifest, result *DiffResult) {
	// Create maps for O(1) lookup
	filesA := make(map[string]FileEntry)
	filesB := make(map[string]FileEntry)

	for _, file := range manifestA.Files {
		filesA[file.Path] = file
	}

	for _, file := range manifestB.Files {
		filesB[file.Path] = file
	}

	// Find added files (in B but not in A)
	for path, file := range filesB {
		if _, exists := filesA[path]; !exists {
			result.FilesAdded = append(result.FilesAdded, file)
		}
	}

	// Find removed files (in A but not in B)
	for path, file := range filesA {
		if _, exists := filesB[path]; !exists {
			result.FilesRemoved = append(result.FilesRemoved, file)
		}
	}

	// Find modified files (in both but with different checksums)
	for path, fileA := range filesA {
		if fileB, exists := filesB[path]; exists {
			if fileA.Checksum != fileB.Checksum {
				result.FilesModified = append(result.FilesModified, FileChange{
					Path:        path,
					OldChecksum: fileA.Checksum,
					NewChecksum: fileB.Checksum,
				})
			}
		}
	}

	// Sort results for consistent output
	sort.Slice(result.FilesAdded, func(i, j int) bool {
		return result.FilesAdded[i].Path < result.FilesAdded[j].Path
	})
	sort.Slice(result.FilesRemoved, func(i, j int) bool {
		return result.FilesRemoved[i].Path < result.FilesRemoved[j].Path
	})
	sort.Slice(result.FilesModified, func(i, j int) bool {
		return result.FilesModified[i].Path < result.FilesModified[j].Path
	})
}

// compareApprovals compares approvals between two bundles.
func compareApprovals(approvalsA, approvalsB []*Approval, result *DiffResult) {
	// Create maps for lookup (key: role-user)
	approvalMapA := make(map[string]*Approval)
	approvalMapB := make(map[string]*Approval)

	for _, approval := range approvalsA {
		key := fmt.Sprintf("%s-%s", approval.Role, approval.User)
		approvalMapA[key] = approval
	}

	for _, approval := range approvalsB {
		key := fmt.Sprintf("%s-%s", approval.Role, approval.User)
		approvalMapB[key] = approval
	}

	// Find added approvals (in B but not in A)
	for key, approval := range approvalMapB {
		if _, exists := approvalMapA[key]; !exists {
			result.ApprovalsAdded = append(result.ApprovalsAdded, approval)
		}
	}

	// Find removed approvals (in A but not in B)
	for key, approval := range approvalMapA {
		if _, exists := approvalMapB[key]; !exists {
			result.ApprovalsRemoved = append(result.ApprovalsRemoved, approval)
		}
	}

	// Sort results
	sort.Slice(result.ApprovalsAdded, func(i, j int) bool {
		return result.ApprovalsAdded[i].Role < result.ApprovalsAdded[j].Role
	})
	sort.Slice(result.ApprovalsRemoved, func(i, j int) bool {
		return result.ApprovalsRemoved[i].Role < result.ApprovalsRemoved[j].Role
	})
}

// attestationsChanged checks if attestations differ between bundles.
func attestationsChanged(attestationA, attestationB *Attestation) bool {
	// Both nil - no change
	if attestationA == nil && attestationB == nil {
		return false
	}

	// One nil, one not - changed
	if (attestationA == nil) != (attestationB == nil) {
		return true
	}

	// Compare attestation subjects
	if attestationA.Subject.Name != attestationB.Subject.Name {
		return true
	}

	// Compare attestation signatures
	if attestationA.Signature.Signature != attestationB.Signature.Signature {
		return true
	}

	// Compare attestation timestamps
	if !attestationA.Timestamp.Equal(attestationB.Timestamp) {
		return true
	}

	return false
}

// compareManifestMetadata compares manifest metadata fields.
func compareManifestMetadata(manifestA, manifestB *Manifest, result *DiffResult) {
	// Compare version
	if manifestA.Version != manifestB.Version {
		result.MetadataChanged = true
		result.ManifestMetadataChanges["version"] = fmt.Sprintf("%s → %s", manifestA.Version, manifestB.Version)
	}

	// Compare ID
	if manifestA.ID != manifestB.ID {
		result.MetadataChanged = true
		result.ManifestMetadataChanges["id"] = fmt.Sprintf("%s → %s", manifestA.ID, manifestB.ID)
	}

	// Compare description
	if manifestA.Description != manifestB.Description {
		result.MetadataChanged = true
		result.ManifestMetadataChanges["description"] = "changed"
	}

	// Compare governance level
	if manifestA.GovernanceLevel != manifestB.GovernanceLevel {
		result.MetadataChanged = true
		result.ManifestMetadataChanges["governance_level"] = fmt.Sprintf("%s → %s", manifestA.GovernanceLevel, manifestB.GovernanceLevel)
	}
}

// HasChanges returns true if there are any differences between bundles.
func (r *DiffResult) HasChanges() bool {
	return len(r.FilesAdded) > 0 ||
		len(r.FilesRemoved) > 0 ||
		len(r.FilesModified) > 0 ||
		len(r.ApprovalsAdded) > 0 ||
		len(r.ApprovalsRemoved) > 0 ||
		r.AttestationChanged ||
		r.MetadataChanged
}

// Summary returns a brief summary of changes.
func (r *DiffResult) Summary() string {
	if !r.HasChanges() {
		return "No differences found"
	}

	changes := []string{}

	if len(r.FilesAdded) > 0 {
		changes = append(changes, fmt.Sprintf("%d file(s) added", len(r.FilesAdded)))
	}
	if len(r.FilesRemoved) > 0 {
		changes = append(changes, fmt.Sprintf("%d file(s) removed", len(r.FilesRemoved)))
	}
	if len(r.FilesModified) > 0 {
		changes = append(changes, fmt.Sprintf("%d file(s) modified", len(r.FilesModified)))
	}
	if len(r.ApprovalsAdded) > 0 {
		changes = append(changes, fmt.Sprintf("%d approval(s) added", len(r.ApprovalsAdded)))
	}
	if len(r.ApprovalsRemoved) > 0 {
		changes = append(changes, fmt.Sprintf("%d approval(s) removed", len(r.ApprovalsRemoved)))
	}
	if r.AttestationChanged {
		changes = append(changes, "attestation changed")
	}
	if r.MetadataChanged {
		changes = append(changes, "metadata changed")
	}

	summary := ""
	for i, change := range changes {
		if i > 0 {
			summary += ", "
		}
		summary += change
	}

	return summary
}
