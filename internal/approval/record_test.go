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

func TestCloseOpenException(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)
	exp := now.Add(48 * time.Hour)
	if _, err := Write(root, &Record{
		Type:       TypeException,
		ResourceID: "exception-EX-192",
		ApprovedBy: "alice",
		ApprovedAt: now,
		Reason:     "hotfix",
		Policy:     "provenance",
		ExpiresAt:  &exp,
	}); err != nil {
		t.Fatal(err)
	}

	open, err := OpenExceptions(root, now)
	if err != nil || len(open) != 1 {
		t.Fatalf("open before=%v err=%v", open, err)
	}

	closed, err := Close(root, "exception-EX-192", CloseOptions{
		Now:    now.Add(time.Minute),
		By:     "alice",
		Reason: "incident mitigated",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !closed.IsClosed() || closed.ClosedBy != "alice" {
		t.Fatalf("closed=%+v", closed)
	}
	if closed.CloseReason != "incident mitigated" {
		t.Fatalf("reason=%q", closed.CloseReason)
	}
	if closed.ExpiresAt == nil || closed.ExpiresAt.After(now.Add(time.Minute)) {
		t.Fatalf("expires=%v", closed.ExpiresAt)
	}
	if closed.IsOpen(now.Add(time.Minute)) {
		t.Fatal("expected not open after close")
	}

	openAfter, err := OpenExceptions(root, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(openAfter) != 0 {
		t.Fatalf("open after close=%+v", openAfter)
	}

	// File still on disk with closed_at.
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(closed.Path)))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"closed_at:", "closed_by: alice", "close_reason: incident mitigated"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("missing %q in:\n%s", want, raw)
		}
	}

	// Idempotent second close.
	again, err := Close(root, "EX-192", CloseOptions{Now: now.Add(2 * time.Minute), By: "bob"})
	if err != nil {
		t.Fatal(err)
	}
	if again.ClosedBy != "alice" {
		t.Fatalf("idempotent should keep original closer, got %q", again.ClosedBy)
	}
}

func TestCloseMissingAndNonException(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Now().UTC()
	if _, err := Close(root, "exception-missing", CloseOptions{Now: now, By: "x"}); err == nil {
		t.Fatal("expected missing error")
	}
	if _, err := Write(root, &Record{
		Type: TypeBundle, ResourceID: "bundle-abc", ApprovedBy: "a", ApprovedAt: now, Message: "m",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := Close(root, "bundle-abc", CloseOptions{Now: now, By: "x"}); err == nil ||
		!strings.Contains(err.Error(), "not an exception") {
		t.Fatalf("err=%v", err)
	}
}

func TestCloseAlreadyExpiredStampsClosed(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	now := time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	if _, err := Write(root, &Record{
		Type:       TypeException,
		ResourceID: "exception-stale",
		ApprovedBy: "alice",
		ApprovedAt: now.Add(-2 * time.Hour),
		Reason:     "stale",
		ExpiresAt:  &past,
	}); err != nil {
		t.Fatal(err)
	}
	rec, err := Close(root, "exception-stale", CloseOptions{Now: now, By: "ops", Reason: "ack"})
	if err != nil {
		t.Fatal(err)
	}
	if !rec.IsClosed() || rec.ClosedBy != "ops" {
		t.Fatalf("%+v", rec)
	}
	// Do not move expires_at earlier than original past.
	if rec.ExpiresAt == nil || !rec.ExpiresAt.Equal(past) {
		t.Fatalf("expires=%v want %v", rec.ExpiresAt, past)
	}
}

func TestFilterByStatus(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	exp := now.Add(24 * time.Hour)
	past := now.Add(-time.Hour)
	closedAt := now.Add(-30 * time.Minute)

	openRec := Record{
		Type: TypeException, ResourceID: "exception-open",
		ApprovedAt: now, ExpiresAt: &exp,
	}
	expiredRec := Record{
		Type: TypeException, ResourceID: "exception-expired",
		ApprovedAt: now.Add(-2 * time.Hour), ExpiresAt: &past,
	}
	closedRec := Record{
		Type: TypeException, ResourceID: "exception-closed",
		ApprovedAt: now.Add(-time.Hour), ExpiresAt: &exp, ClosedAt: &closedAt, ClosedBy: "ops",
	}
	bundle := Record{
		Type: TypeBundle, ResourceID: "bundle-abc", ApprovedAt: now,
	}
	all := []Record{openRec, expiredRec, closedRec, bundle}

	open, err := FilterByStatus(all, "open", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 2 {
		t.Fatalf("open=%v", idsOfApprovals(open))
	}
	for _, rec := range open {
		if rec.Lifecycle(now) != StatusOpen {
			t.Fatalf("%s lifecycle=%s", rec.ResourceID, rec.Lifecycle(now))
		}
	}

	closed, err := FilterByStatus(all, "closed", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(closed) != 1 || closed[0].ResourceID != "exception-closed" {
		t.Fatalf("closed=%v", idsOfApprovals(closed))
	}

	expired, err := FilterByStatus(all, "expired", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(expired) != 1 || expired[0].ResourceID != "exception-expired" {
		t.Fatalf("expired=%v", idsOfApprovals(expired))
	}

	passthrough, err := FilterByStatus(all, "", now)
	if err != nil || len(passthrough) != 4 {
		t.Fatalf("empty: %v len=%d", err, len(passthrough))
	}
	if _, err := FilterByStatus(all, "pending", now); err == nil {
		t.Fatal("expected invalid status error")
	}
}

func TestFilterByTypePolicyScope(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	exp := now.Add(24 * time.Hour)
	recs := []Record{
		{
			Type: TypeException, ResourceID: "exception-app",
			ApprovedAt: now, ExpiresAt: &exp,
			Policy: "provenance", Scope: "auth-service",
		},
		{
			Type: TypeException, ResourceID: "exception-sec",
			ApprovedAt: now, ExpiresAt: &exp,
			Policy: "security", Scope: "payments",
		},
		{
			Type: TypeBundle, ResourceID: "bundle-abc", ApprovedAt: now,
			Policy: "soc2", Scope: "release",
		},
	}

	ex, err := FilterByType(recs, "exception")
	if err != nil {
		t.Fatal(err)
	}
	if len(ex) != 2 {
		t.Fatalf("exception=%v", idsOfApprovals(ex))
	}
	if _, err := FilterByType(recs, "unknown"); err == nil {
		t.Fatal("expected invalid type")
	}

	pol := FilterByPolicy(recs, "PROVENANCE")
	if len(pol) != 1 || pol[0].ResourceID != "exception-app" {
		t.Fatalf("policy=%v", idsOfApprovals(pol))
	}
	scope := FilterByScope(recs, "pay")
	if len(scope) != 1 || scope[0].ResourceID != "exception-sec" {
		t.Fatalf("scope=%v", idsOfApprovals(scope))
	}

	// Combinable: type + policy
	typed, err := FilterByType(recs, "exception")
	if err != nil {
		t.Fatal(err)
	}
	combo := FilterByPolicy(typed, "sec")
	if len(combo) != 1 || combo[0].ResourceID != "exception-sec" {
		t.Fatalf("combo=%v", idsOfApprovals(combo))
	}
}

func idsOfApprovals(recs []Record) []string {
	out := make([]string, len(recs))
	for i, r := range recs {
		out[i] = r.ResourceID
	}
	return out
}
