package approval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTypeFromResourceID(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id   string
		want string
		ok   bool
	}{
		{"bundle-abc", TypeBundle, true},
		{"drift-x", TypeDrift, true},
		{"policy-1", TypePolicy, true},
		{"plan-2", TypePlan, true},
		{"exception", TypeException, true},
		{"exception-EX-192", TypeException, true},
		{"unknown", "", false},
	}
	for _, tc := range cases {
		got, err := TypeFromResourceID(tc.id)
		if tc.ok && err != nil {
			t.Fatalf("%s: %v", tc.id, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%s: expected error", tc.id)
		}
		if got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.id, got, tc.want)
		}
	}
}

func TestNormalizeExceptionID(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	if got := NormalizeExceptionID("exception", at); got != "exception-20260921-100000" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeExceptionID("exception-EX-192", at); got != "exception-EX-192" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeExceptionID("EX-192", at); got != "exception-EX-192" {
		t.Fatalf("got %q", got)
	}
}

func TestParseExpires(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)

	got, err := ParseExpires("7d", now)
	if err != nil {
		t.Fatal(err)
	}
	want := now.Add(7 * 24 * time.Hour)
	if got == nil || !got.Equal(want) {
		t.Fatalf("7d: got %v want %v", got, want)
	}

	got, err = ParseExpires("2026-09-28T12:00:00Z", now)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Format(time.RFC3339) != "2026-09-28T12:00:00Z" {
		t.Fatalf("rfc3339: %v", got)
	}

	if _, err := ParseExpires("nope", now); err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteListOpenExceptions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	exp := now.Add(48 * time.Hour)
	past := now.Add(-time.Hour)

	openRec := &Record{
		Type:       TypeException,
		ResourceID: "exception-EX-192",
		ApprovedBy: "alice",
		ApprovedAt: now,
		Reason:     "emergency auth hotfix",
		Scope:      "internal/auth/**",
		Policy:     "SEC-17",
		Requester:  "bob",
		ExpiresAt:  &exp,
		Message:    "temporary exception",
	}
	path, err := Write(root, openRec)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, filepath.Join(".specular", "approvals")) {
		t.Fatalf("path=%s", path)
	}

	expiredRec := &Record{
		Type:       TypeException,
		ResourceID: "exception-old",
		ApprovedBy: "alice",
		ApprovedAt: now.Add(-2 * time.Hour),
		Reason:     "stale",
		ExpiresAt:  &past,
	}
	if _, err := Write(root, expiredRec); err != nil {
		t.Fatal(err)
	}

	bundle := &Record{
		Type:       TypeBundle,
		ResourceID: "bundle-abc",
		ApprovedBy: "carol",
		ApprovedAt: now.Add(-time.Minute),
		Message:    "ship it",
	}
	if _, err := Write(root, bundle); err != nil {
		t.Fatal(err)
	}

	all, err := List(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("list len=%d", len(all))
	}
	if all[0].ResourceID != "exception-EX-192" {
		t.Fatalf("newest=%s", all[0].ResourceID)
	}

	open, err := OpenExceptions(root, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].ResourceID != "exception-EX-192" {
		t.Fatalf("open=%+v", open)
	}

	found, err := FindByResourceID(root, "bundle-abc")
	if err != nil || len(found) != 1 {
		t.Fatalf("find: %v %+v", err, found)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"reason: emergency auth hotfix", "scope: internal/auth/**", "policy: SEC-17"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("missing %q in:\n%s", want, raw)
		}
	}
}

func TestWriteExceptionRequiresReason(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, err := Write(root, &Record{
		Type:       TypeException,
		ResourceID: "exception-x",
		ApprovedBy: "alice",
		ApprovedAt: time.Now().UTC(),
	})
	if err == nil || !strings.Contains(err.Error(), "reason") {
		t.Fatalf("got %v", err)
	}
}
