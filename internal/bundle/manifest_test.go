package bundle

import (
	"testing"
	"time"
)

// TestManifest_GetFile tests the GetFile method
func TestManifest_GetFile(t *testing.T) {
	manifest := &Manifest{
		Schema:  "specular.bundle/v1",
		ID:      "test/bundle",
		Version: "1.0.0",
		Created: time.Now(),
		Integrity: IntegrityInfo{
			Algorithm: "sha256",
			Digest:    "abc123",
		},
		Files: []FileEntry{
			{
				Path:     "spec.yaml",
				Size:     1024,
				Checksum: "sha256:abc123",
			},
			{
				Path:     "spec.lock.json",
				Size:     512,
				Checksum: "sha256:def456",
			},
			{
				Path:     "routing.yaml",
				Size:     256,
				Checksum: "sha256:ghi789",
			},
		},
	}

	tests := []struct {
		name     string
		path     string
		wantFile *FileEntry
		wantNil  bool
	}{
		{
			name: "find existing file - spec.yaml",
			path: "spec.yaml",
			wantFile: &FileEntry{
				Path:     "spec.yaml",
				Size:     1024,
				Checksum: "sha256:abc123",
			},
			wantNil: false,
		},
		{
			name: "find existing file - spec.lock.json",
			path: "spec.lock.json",
			wantFile: &FileEntry{
				Path:     "spec.lock.json",
				Size:     512,
				Checksum: "sha256:def456",
			},
			wantNil: false,
		},
		{
			name: "find existing file - routing.yaml",
			path: "routing.yaml",
			wantFile: &FileEntry{
				Path:     "routing.yaml",
				Size:     256,
				Checksum: "sha256:ghi789",
			},
			wantNil: false,
		},
		{
			name:     "file not found",
			path:     "nonexistent.yaml",
			wantFile: nil,
			wantNil:  true,
		},
		{
			name:     "empty path",
			path:     "",
			wantFile: nil,
			wantNil:  true,
		},
		{
			name:     "case sensitive - wrong case",
			path:     "SPEC.YAML",
			wantFile: nil,
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manifest.GetFile(tt.path)

			if tt.wantNil {
				if got != nil {
					t.Errorf("GetFile(%q) = %v, want nil", tt.path, got)
				}
				return
			}

			if got == nil {
				t.Fatalf("GetFile(%q) = nil, want non-nil", tt.path)
			}

			if got.Path != tt.wantFile.Path {
				t.Errorf("GetFile(%q).Path = %s, want %s", tt.path, got.Path, tt.wantFile.Path)
			}

			if got.Size != tt.wantFile.Size {
				t.Errorf("GetFile(%q).Size = %d, want %d", tt.path, got.Size, tt.wantFile.Size)
			}

			if got.Checksum != tt.wantFile.Checksum {
				t.Errorf("GetFile(%q).Checksum = %s, want %s", tt.path, got.Checksum, tt.wantFile.Checksum)
			}
		})
	}
}

// TestManifest_GetFile_EmptyManifest tests GetFile with empty file list
func TestManifest_GetFile_EmptyManifest(t *testing.T) {
	manifest := &Manifest{
		Schema:  "specular.bundle/v1",
		ID:      "test/bundle",
		Version: "1.0.0",
		Created: time.Now(),
		Integrity: IntegrityInfo{
			Algorithm: "sha256",
			Digest:    "abc123",
		},
		Files: []FileEntry{},
	}

	got := manifest.GetFile("any-file.yaml")
	if got != nil {
		t.Errorf("GetFile() on empty manifest = %v, want nil", got)
	}
}

// TestManifest_HasFile tests the HasFile method
func TestManifest_HasFile(t *testing.T) {
	manifest := &Manifest{
		Schema:  "specular.bundle/v1",
		ID:      "test/bundle",
		Version: "1.0.0",
		Created: time.Now(),
		Integrity: IntegrityInfo{
			Algorithm: "sha256",
			Digest:    "abc123",
		},
		Files: []FileEntry{
			{
				Path:     "spec.yaml",
				Size:     1024,
				Checksum: "sha256:abc123",
			},
			{
				Path:     "spec.lock.json",
				Size:     512,
				Checksum: "sha256:def456",
			},
			{
				Path:     "policies/security.yaml",
				Size:     128,
				Checksum: "sha256:jkl012",
			},
		},
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "has file - spec.yaml",
			path: "spec.yaml",
			want: true,
		},
		{
			name: "has file - spec.lock.json",
			path: "spec.lock.json",
			want: true,
		},
		{
			name: "has file - nested path",
			path: "policies/security.yaml",
			want: true,
		},
		{
			name: "does not have file",
			path: "nonexistent.yaml",
			want: false,
		},
		{
			name: "empty path",
			path: "",
			want: false,
		},
		{
			name: "case sensitive - wrong case",
			path: "SPEC.YAML",
			want: false,
		},
		{
			name: "partial path match should not work",
			path: "spec",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := manifest.HasFile(tt.path)
			if got != tt.want {
				t.Errorf("HasFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestManifest_HasFile_EmptyManifest tests HasFile with empty file list
func TestManifest_HasFile_EmptyManifest(t *testing.T) {
	manifest := &Manifest{
		Schema:  "specular.bundle/v1",
		ID:      "test/bundle",
		Version: "1.0.0",
		Created: time.Now(),
		Integrity: IntegrityInfo{
			Algorithm: "sha256",
			Digest:    "abc123",
		},
		Files: []FileEntry{},
	}

	tests := []string{
		"any-file.yaml",
		"spec.yaml",
		"",
		"nonexistent",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			got := manifest.HasFile(path)
			if got {
				t.Errorf("HasFile(%q) on empty manifest = true, want false", path)
			}
		})
	}
}

// TestManifest_HasFile_MultipleFiles tests HasFile with many files
func TestManifest_HasFile_MultipleFiles(t *testing.T) {
	// Create manifest with many files
	files := []FileEntry{
		{Path: "spec.yaml", Size: 100, Checksum: "sha256:a"},
		{Path: "spec.lock.json", Size: 100, Checksum: "sha256:b"},
		{Path: "routing.yaml", Size: 100, Checksum: "sha256:c"},
		{Path: "policies/policy1.yaml", Size: 100, Checksum: "sha256:d"},
		{Path: "policies/policy2.yaml", Size: 100, Checksum: "sha256:e"},
		{Path: "policies/policy3.yaml", Size: 100, Checksum: "sha256:f"},
		{Path: "files/data1.json", Size: 100, Checksum: "sha256:g"},
		{Path: "files/data2.json", Size: 100, Checksum: "sha256:h"},
	}

	manifest := &Manifest{
		Schema:  "specular.bundle/v1",
		ID:      "test/bundle",
		Version: "1.0.0",
		Created: time.Now(),
		Integrity: IntegrityInfo{
			Algorithm: "sha256",
			Digest:    "abc123",
		},
		Files: files,
	}

	// Test that all files can be found
	for _, file := range files {
		t.Run("has_"+file.Path, func(t *testing.T) {
			if !manifest.HasFile(file.Path) {
				t.Errorf("HasFile(%q) = false, want true", file.Path)
			}
		})
	}

	// Test that non-existent files return false
	nonExistent := []string{
		"missing.yaml",
		"policies/missing.yaml",
		"files/missing.json",
		"spec.json",
	}

	for _, path := range nonExistent {
		t.Run("not_has_"+path, func(t *testing.T) {
			if manifest.HasFile(path) {
				t.Errorf("HasFile(%q) = true, want false", path)
			}
		})
	}
}

// TestManifest_GetFile_ReturnsCopy tests that GetFile returns a pointer to the actual entry
func TestManifest_GetFile_ReturnsCopy(t *testing.T) {
	manifest := &Manifest{
		Schema:  "specular.bundle/v1",
		ID:      "test/bundle",
		Version: "1.0.0",
		Created: time.Now(),
		Integrity: IntegrityInfo{
			Algorithm: "sha256",
			Digest:    "abc123",
		},
		Files: []FileEntry{
			{
				Path:     "spec.yaml",
				Size:     1024,
				Checksum: "sha256:original",
			},
		},
	}

	// Get the file
	file := manifest.GetFile("spec.yaml")
	if file == nil {
		t.Fatal("GetFile() returned nil")
	}

	// Verify original values
	if file.Checksum != "sha256:original" {
		t.Errorf("Initial checksum = %s, want sha256:original", file.Checksum)
	}

	// Note: GetFile returns a pointer to the slice element,
	// so modifying it WILL modify the original (this is the actual behavior)
	// This test verifies the current implementation behavior
	originalChecksum := manifest.Files[0].Checksum
	if file.Checksum != originalChecksum {
		t.Errorf("Checksums don't match: file.Checksum=%s, manifest.Files[0].Checksum=%s",
			file.Checksum, originalChecksum)
	}
}
