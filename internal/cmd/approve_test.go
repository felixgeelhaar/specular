package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/felixgeelhaar/specular/internal/approval"
)

// TestApprovalRecord tests the ApprovalRecord struct definition
func TestApprovalRecord(t *testing.T) {
	// Create a sample approval to verify struct works
	now := time.Now()
	rec := ApprovalRecord{
		Version:      "1.0",
		Type:         "bundle",
		ResourceID:   "bundle-abc123",
		ResourceHash: "sha256:def456",
		ApprovedBy:   "alice@example.com",
		ApprovedAt:   now,
		Message:      "Approved for production",
		Metadata: map[string]string{
			"environment": "prod",
			"reviewer":    "bob",
		},
	}

	// Verify fields are accessible
	if rec.Version != "1.0" {
		t.Errorf("Version = %q, want %q", rec.Version, "1.0")
	}
	if rec.Type != "bundle" {
		t.Errorf("Type = %q, want %q", rec.Type, "bundle")
	}
	if rec.ResourceID != "bundle-abc123" {
		t.Errorf("ResourceID = %q, want %q", rec.ResourceID, "bundle-abc123")
	}
	if rec.ResourceHash != "sha256:def456" {
		t.Errorf("ResourceHash = %q, want %q", rec.ResourceHash, "sha256:def456")
	}
	if rec.ApprovedBy != "alice@example.com" {
		t.Errorf("ApprovedBy = %q, want %q", rec.ApprovedBy, "alice@example.com")
	}
	if !rec.ApprovedAt.Equal(now) {
		t.Errorf("ApprovedAt = %v, want %v", rec.ApprovedAt, now)
	}
	if rec.Message != "Approved for production" {
		t.Errorf("Message = %q, want %q", rec.Message, "Approved for production")
	}
	if len(rec.Metadata) != 2 {
		t.Errorf("Metadata length = %d, want %d", len(rec.Metadata), 2)
	}
	if rec.Metadata["environment"] != "prod" {
		t.Errorf("Metadata[environment] = %q, want %q", rec.Metadata["environment"], "prod")
	}
}

func TestFormatApprovalExplainException(t *testing.T) {
	t.Parallel()
	exp := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	text := formatApprovalExplain(approval.Record{
		Type:       approval.TypeException,
		ResourceID: "exception-EX-192",
		ApprovedBy: "alice",
		ApprovedAt: time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC),
		Message:    "temporary",
		Reason:     "emergency hotfix",
		Scope:      "internal/auth/**",
		Policy:     "SEC-17",
		Requester:  "bob",
		ExpiresAt:  &exp,
		EvidenceID: "ev_soft_allow",
		Path:       ".specular/approvals/exception-20260921-100000.yaml",
	})
	for _, want := range []string{
		"APPROVAL / EXCEPTION RECORD",
		"Type         exception",
		"Resource     exception-EX-192",
		"Reason       emergency hotfix",
		"Scope        internal/auth/**",
		"Policy       SEC-17",
		"Status       OPEN",
		"Evidence     ev_soft_allow",
		"Evidence     specular evidence show ev_soft_allow",
		"             specular explain ev_soft_allow",
		"specular approvals list",
		"Open         specular approvals list --status open",
		"Pending      specular approvals pending",
		"Doctor       specular doctor",
		"Gate         specular gate",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
}

func TestFormatApprovalExplainNoEvidenceRefs(t *testing.T) {
	t.Parallel()
	text := formatApprovalExplain(approval.Record{
		Type:       approval.TypeBundle,
		ResourceID: "bundle-abc",
		ApprovedBy: "alice",
		ApprovedAt: time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC),
		Message:    "ok",
	})
	if strings.Contains(text, "evidence show") || strings.Contains(text, "list --status open") {
		t.Fatalf("unexpected evidence/open refs:\n%s", text)
	}
	for _, want := range []string{
		"Pending      specular approvals pending",
		"Doctor       specular doctor",
		"Gate         specular gate",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing Soft trail %q:\n%s", want, text)
		}
	}
}

func TestPrintPendingOpenExceptions(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if !printPendingOpenExceptions() {
		// empty store → false
	} else {
		t.Fatal("expected false with no approvals")
	}

	now := time.Now().UTC()
	exp := now.Add(24 * time.Hour)
	if _, err := approval.Write(".", &approval.Record{
		Type: approval.TypeException, ResourceID: "exception-pending-open",
		ApprovedBy: "alice", ApprovedAt: now, ExpiresAt: &exp,
		Reason: "hotfix", Scope: "a.go", Policy: "drift", EvidenceID: "ev_pend",
	}); err != nil {
		t.Fatal(err)
	}

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	ok := printPendingOpenExceptions()
	_ = w.Close()
	os.Stdout = old
	if !ok {
		t.Fatal("expected open exceptions printed")
	}
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	text := string(buf[:n])
	for _, want := range []string{
		"Open exceptions:",
		"exception-pending-open  specular approvals show exception-pending-open",
		"specular evidence show ev_pend",
		"List   specular approvals list --status open",
		"Trail  specular approvals pending",
		"specular doctor",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
}

func TestPrintApprovalsCloseSoftTrail(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	printApprovalsCloseSoftTrail("exception-gone")
	_ = w.Close()
	os.Stdout = old
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	text := string(buf[:n])
	for _, want := range []string{
		"Refs",
		"Pending      specular approvals pending",
		"Open         specular approvals list --status open",
		"Doctor       specular doctor",
		"Gate         specular gate",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Open exceptions:") {
		t.Fatalf("unexpected open exceptions with empty store:\n%s", text)
	}

	now := time.Now().UTC()
	exp := now.Add(24 * time.Hour)
	if _, err := approval.Write(".", &approval.Record{
		Type: approval.TypeException, ResourceID: "exception-still-open",
		ApprovedBy: "alice", ApprovedAt: now, ExpiresAt: &exp,
		Reason: "hotfix", Scope: "a.go", Policy: "drift", EvidenceID: "ev_still",
	}); err != nil {
		t.Fatal(err)
	}
	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w2
	printApprovalsCloseSoftTrail("exception-gone")
	_ = w2.Close()
	os.Stdout = old
	buf2 := make([]byte, 4096)
	n2, _ := r2.Read(buf2)
	text2 := string(buf2[:n2])
	for _, want := range []string{
		"Open exceptions:",
		"exception-still-open  specular approvals show exception-still-open",
		"specular evidence show ev_still",
	} {
		if !strings.Contains(text2, want) {
			t.Fatalf("missing %q:\n%s", want, text2)
		}
	}
}

func TestPrintApprovalsCreateSoftTrail(t *testing.T) {
	t.Parallel()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	printApprovalsCreateSoftTrail(approval.Record{
		ResourceID: "exception-EX-1",
		EvidenceID: "ev_new",
	})
	_ = w.Close()
	os.Stdout = old
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	text := string(buf[:n])
	for _, want := range []string{
		"Refs",
		"Show         specular approvals show exception-EX-1",
		"Open         specular approvals list --status open",
		"Pending      specular approvals pending",
		"Doctor       specular doctor",
		"Gate         specular gate",
		"Evidence     specular evidence show ev_new",
		"             specular explain ev_new",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q:\n%s", want, text)
		}
	}

	r2, w2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w2
	printApprovalsCreateSoftTrail(approval.Record{ResourceID: "exception-bare"})
	_ = w2.Close()
	os.Stdout = old
	buf2 := make([]byte, 4096)
	n2, _ := r2.Read(buf2)
	text2 := string(buf2[:n2])
	if strings.Contains(text2, "evidence show") {
		t.Fatalf("bare exception should not invent evidence:\n%s", text2)
	}
	if !strings.Contains(text2, "Show         specular approvals show exception-bare") {
		t.Fatalf("missing show:\n%s", text2)
	}
}
