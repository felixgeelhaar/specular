package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/felixgeelhaar/specular/internal/approval"
	"github.com/felixgeelhaar/specular/internal/exec"
	"github.com/felixgeelhaar/specular/internal/license"
	"github.com/felixgeelhaar/specular/internal/telemetry"
)

var approveCmd = &cobra.Command{
	Use:   "approve <resource>",
	Short: "Approve bundle, drift, policy, plan, or record an exception",
	Long: `Create an approval or controlled exception record under .specular/approvals/.

Resources can be:
  • Bundle ID (from bundle create)     bundle-<id>
  • Drift hash (from eval drift)       drift-<id>
  • Policy change (from policy diff)   policy-<id>
  • Plan ID                            plan-<id>
  • Exception                          exception / exception-<id>

Exceptions (PRODUCT_INTENT §18) require --reason and should include --scope.
They are an explicit auditable trail — they do not silently bypass the gate.

Examples:
  specular approve bundle-abc123 --message "Reviewed for prod"
  specular approve drift-def456 --message "Accepted known drift"
  specular approve exception-EX-192 \
    --reason "Emergency auth hotfix" \
    --scope "internal/auth/**" \
    --policy SEC-17 \
    --expires 7d \
    --message "Approved by security on-call"
  specular approve exception --reason "Hotfix" --scope "payments" --expires 24h`,
	Args: cobra.ExactArgs(1),
	RunE: runApprove,
}

var approvalsCmd = &cobra.Command{
	Use:   "approvals",
	Short: "Manage approval records",
	Long:  `List and manage approval records for bundles, drift, policies, and exceptions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var approvalsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all approval records",
	Long: `Display approval records with optional filters.

Shows:
  • Approval type (bundle, drift, policy, plan, exception)
  • Resource ID, approver, timestamp
  • Exception reason/scope/policy/expiration when present

Filters (combinable):
  --status open|closed|expired   Lifecycle status (closed wins over expired)
  --type bundle|drift|policy|plan|exception
  --policy <substr>              Case-insensitive match on policy field
  --scope <substr>               Case-insensitive match on scope field

--json emits a machine-readable array of matching records.`,
	RunE: runApprovalsList,
}

var approvalsShowCmd = &cobra.Command{
	Use:   "show [resource-id]",
	Short: "Show one approval/exception in AI CHANGE RECORD style",
	Long: `Show the newest matching approval or exception record.

Without an argument, shows the most recent record.
With a resource id (e.g. exception-EX-192), shows the newest match.

--json emits the raw record.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runApprovalsShow,
}

var approvalsPendingCmd = &cobra.Command{
	Use:   "pending",
	Short: "Show pending approvals",
	Long: `Display resources that require approval but don't have one yet.

Checks:
  • Unapproved policy changes (from policy diff)
  • Unapproved bundles (from bundle create)
  • Unapproved drift (from eval drift)

Exit codes:
  0: No pending approvals
  1: Pending approvals found`,
	RunE: runApprovalsPending,
}

var approvalsCloseCmd = &cobra.Command{
	Use:     "close <exception-id>",
	Aliases: []string{"revoke"},
	Short:   "Close (revoke) an open exception early",
	Long: `Early-end an open exception so soft-ALLOW no longer applies.

Rewrites the existing YAML in place: stamps closed_at/closed_by and clamps
expires_at to now. Idempotent when already closed or expired.

  specular approvals close exception-EX-192
  specular approvals close EX-192 --reason "incident mitigated"
  specular approvals revoke exception-app-protocol --json

Uses the same Pro license gate as approve (approvals.create).`,
	Args: cobra.ExactArgs(1),
	RunE: runApprovalsClose,
}

// ApprovalRecord is retained for compatibility with existing tests.
// Prefer approval.Record for new code.
type ApprovalRecord struct {
	Version      string            `yaml:"version"`
	Type         string            `yaml:"type"`
	ResourceID   string            `yaml:"resource_id"`
	ResourceHash string            `yaml:"resource_hash,omitempty"`
	ApprovedBy   string            `yaml:"approved_by"`
	ApprovedAt   time.Time         `yaml:"approved_at"`
	Message      string            `yaml:"message,omitempty"`
	Metadata     map[string]string `yaml:"metadata,omitempty"`
}

func runApprove(cmd *cobra.Command, args []string) error {
	if err := license.RequireFeature("approvals.create", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "approve")
		return err
	}

	resourceID := args[0]
	message, _ := cmd.Flags().GetString("message")
	reason, _ := cmd.Flags().GetString("reason")
	scope, _ := cmd.Flags().GetString("scope")
	policyRef, _ := cmd.Flags().GetString("policy")
	expiresRaw, _ := cmd.Flags().GetString("expires")
	requester, _ := cmd.Flags().GetString("requester")
	artifact, _ := cmd.Flags().GetString("artifact")
	evidenceID, _ := cmd.Flags().GetString("evidence")

	resourceType, err := approval.TypeFromResourceID(resourceID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if resourceType == approval.TypeException {
		resourceID = approval.NormalizeExceptionID(resourceID, now)
		if strings.TrimSpace(reason) == "" {
			return fmt.Errorf("exception --reason is required (PRODUCT_INTENT §18)")
		}
		if strings.TrimSpace(message) == "" {
			message = reason
		}
	} else if strings.TrimSpace(message) == "" {
		return fmt.Errorf("approval message is required (use --message \"...\")")
	}

	expiresAt, err := approval.ParseExpires(expiresRaw, now)
	if err != nil {
		return err
	}

	approver := os.Getenv("USER")
	if approver == "" {
		approver = "unknown"
	}

	rec := &approval.Record{
		Version:    approval.SchemaVersion,
		Type:       resourceType,
		ResourceID: resourceID,
		ApprovedBy: approver,
		ApprovedAt: now,
		Message:    message,
		Reason:     strings.TrimSpace(reason),
		Scope:      strings.TrimSpace(scope),
		Policy:     strings.TrimSpace(policyRef),
		Requester:  strings.TrimSpace(requester),
		ExpiresAt:  expiresAt,
		Artifact:   strings.TrimSpace(artifact),
		EvidenceID: strings.TrimSpace(evidenceID),
	}

	approvalPath, err := approval.Write(".", rec)
	if err != nil {
		return err
	}

	telemetry.RecordIntervention(cmd.Context(), interventionGateForResource(resourceType), telemetry.InterventionDecisionApproved)

	if resourceType == approval.TypeException {
		fmt.Printf("⚠ Exception recorded: %s\n\n", resourceID)
		fmt.Printf("Reason:      %s\n", rec.Reason)
		if rec.Scope != "" {
			fmt.Printf("Scope:       %s\n", rec.Scope)
		}
		if rec.Policy != "" {
			fmt.Printf("Policy:      %s\n", rec.Policy)
		}
		if rec.Requester != "" {
			fmt.Printf("Requester:   %s\n", rec.Requester)
		}
		fmt.Printf("Approved by: %s\n", approver)
		if rec.ExpiresAt != nil {
			fmt.Printf("Expires:     %s\n", rec.ExpiresAt.Format(time.RFC3339))
		}
		fmt.Printf("Saved:       %s\n", approvalPath)
		fmt.Println("\nNote: an open, non-expired exception can soft-ALLOW a matching gate DENY")
		fmt.Println("(drift/policy/risk) when --policy/--scope binds to that deny; otherwise advisory.")
		return nil
	}

	fmt.Printf("✅ Approved %s: %s\n\n", resourceType, resourceID)
	fmt.Printf("Approved by: %s\n", approver)
	fmt.Printf("Approval saved: %s\n", approvalPath)
	fmt.Printf("Message: %s\n", message)
	return nil
}

func runApprovalsList(cmd *cobra.Command, args []string) error {
	if err := license.RequireFeature("approvals.list", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "approvals list")
		return err
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	status, _ := cmd.Flags().GetString("status")
	typ, _ := cmd.Flags().GetString("type")
	policySub, _ := cmd.Flags().GetString("policy")
	scopeSub, _ := cmd.Flags().GetString("scope")
	recs, err := approval.List(".")
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	recs, err = approval.FilterByStatus(recs, status, now)
	if err != nil {
		return err
	}
	recs, err = approval.FilterByType(recs, typ)
	if err != nil {
		return err
	}
	recs = approval.FilterByPolicy(recs, policySub)
	recs = approval.FilterByScope(recs, scopeSub)

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if recs == nil {
			recs = []approval.Record{}
		}
		return enc.Encode(recs)
	}

	if len(recs) == 0 {
		if filtered := strings.TrimSpace(status) != "" || strings.TrimSpace(typ) != "" ||
			strings.TrimSpace(policySub) != "" || strings.TrimSpace(scopeSub) != ""; filtered {
			fmt.Println("No approval records match the given filters.")
			return nil
		}
		fmt.Println("No approval records found.")
		fmt.Println("\nRun 'specular governance init' to create the governance workspace.")
		fmt.Println("Record an exception: specular approve exception-<id> --reason \"...\" --scope \"...\"")
		return nil
	}

	fmt.Println("APPROVAL / EXCEPTION TRAIL")
	fmt.Println("──────────────────────────────────────")

	approvalsByType := make(map[string][]approval.Record)
	for _, rec := range recs {
		approvalsByType[rec.Type] = append(approvalsByType[rec.Type], rec)
	}

	order := []string{
		approval.TypeException,
		approval.TypePolicy,
		approval.TypeBundle,
		approval.TypeDrift,
		approval.TypePlan,
	}
	for _, approvalType := range order {
		group := approvalsByType[approvalType]
		if len(group) == 0 {
			continue
		}
		title := strings.ToUpper(approvalType[:1]) + approvalType[1:]
		if approvalType == approval.TypeException {
			title = "Exception"
		}
		fmt.Printf("%s records: %d\n", title, len(group))
		for _, rec := range group {
			printApprovalHuman(rec, now)
			fmt.Println()
		}
	}

	fmt.Printf("Total: %d\n", len(recs))
	return nil
}

func runApprovalsShow(cmd *cobra.Command, args []string) error {
	if err := license.RequireFeature("approvals.list", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "approvals show")
		return err
	}

	jsonOut, _ := cmd.Flags().GetBool("json")
	recs, err := approval.List(".")
	if err != nil {
		return err
	}
	if len(recs) == 0 {
		return fmt.Errorf("no approval records found under .specular/approvals/")
	}

	var rec approval.Record
	if len(args) == 1 {
		id := strings.TrimSpace(args[0])
		matches, findErr := approval.FindByResourceID(".", id)
		if findErr != nil {
			return findErr
		}
		if len(matches) == 0 {
			return fmt.Errorf("no approval record for %q", id)
		}
		rec = matches[0]
	} else {
		rec = recs[0]
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rec)
	}

	fmt.Print(formatApprovalExplain(rec))
	return nil
}

func formatApprovalExplain(rec approval.Record) string {
	var b strings.Builder
	b.WriteString("APPROVAL / EXCEPTION RECORD\n")
	b.WriteString("──────────────────────────────────────\n")
	fmt.Fprintf(&b, "Type         %s\n", rec.Type)
	fmt.Fprintf(&b, "Resource     %s\n", rec.ResourceID)
	fmt.Fprintf(&b, "ApprovedBy   %s\n", rec.ApprovedBy)
	if !rec.ApprovedAt.IsZero() {
		fmt.Fprintf(&b, "ApprovedAt   %s\n", rec.ApprovedAt.UTC().Format(time.RFC3339))
	}
	if rec.Message != "" {
		fmt.Fprintf(&b, "Message      %s\n", rec.Message)
	}
	if rec.Type == approval.TypeException || rec.Reason != "" {
		b.WriteString("Exception\n")
		if rec.Reason != "" {
			fmt.Fprintf(&b, "Reason       %s\n", rec.Reason)
		}
		if rec.Scope != "" {
			fmt.Fprintf(&b, "Scope        %s\n", rec.Scope)
		}
		if rec.Policy != "" {
			fmt.Fprintf(&b, "Policy       %s\n", rec.Policy)
		}
		if rec.Requester != "" {
			fmt.Fprintf(&b, "Requester    %s\n", rec.Requester)
		}
		if rec.ExpiresAt != nil {
			fmt.Fprintf(&b, "Expires      %s\n", rec.ExpiresAt.UTC().Format(time.RFC3339))
		}
		switch {
		case rec.IsClosed():
			b.WriteString("Status       CLOSED\n")
			if rec.ClosedBy != "" {
				fmt.Fprintf(&b, "ClosedBy     %s\n", rec.ClosedBy)
			}
			if rec.ClosedAt != nil {
				fmt.Fprintf(&b, "ClosedAt     %s\n", rec.ClosedAt.UTC().Format(time.RFC3339))
			}
			if rec.CloseReason != "" {
				fmt.Fprintf(&b, "CloseReason  %s\n", rec.CloseReason)
			}
		case rec.ExpiresAt != nil && rec.IsExpired(time.Now().UTC()):
			b.WriteString("Status       EXPIRED\n")
		case rec.ExpiresAt != nil:
			b.WriteString("Status       OPEN\n")
		case rec.Type == approval.TypeException:
			b.WriteString("Status       OPEN (no expiration)\n")
		}
	}
	if rec.Artifact != "" {
		fmt.Fprintf(&b, "Artifact     %s\n", rec.Artifact)
	}
	if rec.EvidenceID != "" {
		fmt.Fprintf(&b, "Evidence     %s\n", rec.EvidenceID)
	}
	b.WriteString("──────────────────────────────────────\n")
	b.WriteString("Refs\n")
	if rec.Path != "" {
		fmt.Fprintf(&b, "File         %s\n", rec.Path)
	} else {
		b.WriteString("Store        .specular/approvals/\n")
	}
	b.WriteString("List         specular approvals list\n")
	b.WriteString("Gate trail   specular gate / specular explain\n")
	return b.String()
}

func printApprovalHuman(rec approval.Record, now time.Time) {
	fmt.Printf("  • %s\n", rec.ResourceID)
	fmt.Printf("    Approved by: %s\n", rec.ApprovedBy)
	fmt.Printf("    Approved at: %s\n", rec.ApprovedAt.Format("2006-01-02 15:04:05"))
	if rec.Message != "" {
		fmt.Printf("    Message: %s\n", rec.Message)
	}
	if rec.Reason != "" {
		fmt.Printf("    Reason: %s\n", rec.Reason)
	}
	if rec.Scope != "" {
		fmt.Printf("    Scope: %s\n", rec.Scope)
	}
	if rec.Policy != "" {
		fmt.Printf("    Policy: %s\n", rec.Policy)
	}
	if rec.ExpiresAt != nil {
		status := "open"
		switch {
		case rec.IsClosed():
			status = "closed"
		case rec.IsExpired(now):
			status = "expired"
		}
		fmt.Printf("    Expires: %s (%s)\n", rec.ExpiresAt.Format(time.RFC3339), status)
	} else if rec.IsClosed() {
		fmt.Printf("    Status: closed")
		if rec.ClosedAt != nil {
			fmt.Printf(" at %s", rec.ClosedAt.Format(time.RFC3339))
		}
		fmt.Println()
	}
	if rec.Path != "" {
		fmt.Printf("    File: %s\n", rec.Path)
	}
}

func runApprovalsClose(cmd *cobra.Command, args []string) error {
	if err := license.RequireFeature("approvals.create", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "approvals close")
		return err
	}

	reason, _ := cmd.Flags().GetString("reason")
	jsonOut, _ := cmd.Flags().GetBool("json")
	now := time.Now().UTC()
	id := approval.NormalizeExceptionID(args[0], now)

	by := os.Getenv("USER")
	if by == "" {
		by = os.Getenv("USERNAME")
	}
	if by == "" {
		by = "unknown"
	}

	rec, err := approval.Close(".", id, approval.CloseOptions{
		Now:    now,
		By:     by,
		Reason: reason,
	})
	if err != nil {
		return err
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rec)
	}

	fmt.Printf("⚠ Exception closed: %s\n", rec.ResourceID)
	if rec.ClosedBy != "" {
		fmt.Printf("Closed by:   %s\n", rec.ClosedBy)
	}
	if rec.ClosedAt != nil {
		fmt.Printf("Closed at:   %s\n", rec.ClosedAt.UTC().Format(time.RFC3339))
	}
	if rec.CloseReason != "" {
		fmt.Printf("Close note:  %s\n", rec.CloseReason)
	}
	if rec.ExpiresAt != nil {
		fmt.Printf("Expires:     %s\n", rec.ExpiresAt.UTC().Format(time.RFC3339))
	}
	if rec.Path != "" {
		fmt.Printf("Saved:       %s\n", rec.Path)
	}
	fmt.Println("Note: soft-ALLOW no longer applies for this id")
	return nil
}

func runApprovalsPending(cmd *cobra.Command, args []string) error {
	if err := license.RequireFeature("approvals.list", license.TierPro); err != nil {
		license.DisplayUpgradeMessage(err, "approvals pending")
		return err
	}

	fmt.Println("=== Pending Approvals ===")

	hasPending := false

	if hasPolicyChanges, err := checkPolicyChanges(); err == nil && hasPolicyChanges {
		fmt.Println("📋 Policy Changes:")
		fmt.Println("  • Policies have changed since last approval")
		fmt.Println("  • Run 'specular policy diff' to see changes")
		fmt.Println("  • Run 'specular policy approve' to approve")
		fmt.Println()
		hasPending = true
	}

	if pendingBundles, err := checkPendingBundles(); err == nil && len(pendingBundles) > 0 {
		fmt.Printf("📦 Bundles: %d pending\n", len(pendingBundles))
		for _, bundleID := range pendingBundles {
			fmt.Printf("  • %s\n", bundleID)
		}
		fmt.Println("  Run 'specular approve <bundle-id>' to approve")
		fmt.Println()
		hasPending = true
	}

	if hasDrift, err := checkDrift(); err == nil && hasDrift {
		fmt.Println("🔀 Drift Detected:")
		fmt.Println("  • Drift detected but not approved")
		fmt.Println("  • Run 'specular eval drift' to see details")
		fmt.Println("  • Run 'specular approve <drift-id>' to approve")
		fmt.Println()
		hasPending = true
	}

	if !hasPending {
		fmt.Println("✅ No pending approvals")
		fmt.Println("\nAll governance items are approved and up to date.")
		return nil
	}

	os.Exit(1)
	return nil
}

func checkPolicyChanges() (bool, error) {
	policiesPath := filepath.Join(".specular", "policies.yaml")
	if _, err := os.Stat(policiesPath); os.IsNotExist(err) {
		return false, nil
	}

	approvalsDir := filepath.Join(".specular", "approvals")
	entries, err := os.ReadDir(approvalsDir)
	if err != nil {
		return false, err
	}

	hasPolicyApproval := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "policy-") {
			hasPolicyApproval = true
			break
		}
	}

	if !hasPolicyApproval {
		return true, nil
	}

	currentHash, err := exec.HashFile(policiesPath)
	if err != nil {
		return false, fmt.Errorf("hash policies file: %w", err)
	}

	var latestApproval *ApprovalRecord
	var latestTime time.Time
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "policy-") {
			continue
		}

		approvalPath := filepath.Join(approvalsDir, entry.Name())
		data, err := os.ReadFile(approvalPath)
		if err != nil {
			continue
		}

		var rec ApprovalRecord
		if err := yaml.Unmarshal(data, &rec); err != nil {
			continue
		}

		if rec.ApprovedAt.After(latestTime) {
			latestTime = rec.ApprovedAt
			latestApproval = &rec
		}
	}

	if latestApproval != nil {
		return latestApproval.ResourceHash != currentHash, nil
	}

	return true, nil
}

func checkPendingBundles() ([]string, error) {
	bundlesDir := filepath.Join(".specular", "bundles")
	if _, err := os.Stat(bundlesDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(bundlesDir)
	if err != nil {
		return nil, err
	}

	approvalsDir := filepath.Join(".specular", "approvals")
	approvedBundles := make(map[string]bool)

	if approvalEntries, err := os.ReadDir(approvalsDir); err == nil {
		for _, entry := range approvalEntries {
			if !strings.HasPrefix(entry.Name(), "bundle-") {
				continue
			}

			approvalPath := filepath.Join(approvalsDir, entry.Name())
			data, err := os.ReadFile(approvalPath)
			if err != nil {
				continue
			}

			var rec ApprovalRecord
			if err := yaml.Unmarshal(data, &rec); err != nil {
				continue
			}

			approvedBundles[rec.ResourceID] = true
		}
	}

	var pending []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tar") {
			continue
		}

		bundleID := strings.TrimSuffix(entry.Name(), ".tar")
		if !approvedBundles[bundleID] {
			pending = append(pending, bundleID)
		}
	}

	return pending, nil
}

func checkDrift() (bool, error) {
	driftPath := filepath.Join(".specular", "drift-baseline.json")
	if _, err := os.Stat(driftPath); os.IsNotExist(err) {
		return false, nil
	}

	approvalsDir := filepath.Join(".specular", "approvals")
	entries, err := os.ReadDir(approvalsDir)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "drift-") {
			return false, nil
		}
	}

	return true, nil
}

func interventionGateForResource(resourceType string) string {
	switch resourceType {
	case approval.TypeBundle:
		return telemetry.InterventionGateBundleApproval
	case approval.TypeDrift:
		return telemetry.InterventionGateDriftApproval
	case approval.TypePolicy:
		return telemetry.InterventionGatePolicyApproval
	case approval.TypePlan:
		return telemetry.InterventionGatePlanApproval
	case approval.TypeException:
		return telemetry.InterventionGateOther
	default:
		return telemetry.InterventionGateOther
	}
}

func init() {
	rootCmd.AddCommand(approveCmd)
	rootCmd.AddCommand(approvalsCmd)
	approvalsCmd.AddCommand(approvalsListCmd)
	approvalsCmd.AddCommand(approvalsShowCmd)
	approvalsCmd.AddCommand(approvalsPendingCmd)
	approvalsCmd.AddCommand(approvalsCloseCmd)

	approveCmd.Flags().String("message", "", "Approval message or comment")
	approveCmd.Flags().String("reason", "", "Exception reason (required for exception-*)")
	approveCmd.Flags().String("scope", "", "Exception scope (paths, services, or change set)")
	approveCmd.Flags().String("policy", "", "Policy or control requiring the exception (e.g. SEC-17)")
	approveCmd.Flags().String("expires", "", "Exception expiration (RFC3339 or duration like 7d, 24h)")
	approveCmd.Flags().String("requester", "", "Who requested the exception")
	approveCmd.Flags().String("artifact", "", "Affected artifact digest or reference")
	approveCmd.Flags().String("evidence", "", "Related evidence id (ev_…)")

	approvalsListCmd.Flags().Bool("json", false, "Emit machine-readable JSON")
	approvalsListCmd.Flags().String("status", "", "Filter by lifecycle status (open|closed|expired)")
	approvalsListCmd.Flags().String("type", "", "Filter by record type (bundle|drift|policy|plan|exception)")
	approvalsListCmd.Flags().String("policy", "", "Filter by policy field substring (case-insensitive)")
	approvalsListCmd.Flags().String("scope", "", "Filter by scope field substring (case-insensitive)")
	approvalsShowCmd.Flags().Bool("json", false, "Emit machine-readable JSON")
	approvalsCloseCmd.Flags().String("reason", "", "Optional close note (audit)")
	approvalsCloseCmd.Flags().Bool("json", false, "Emit machine-readable JSON")
}
