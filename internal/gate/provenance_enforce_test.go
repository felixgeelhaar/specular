package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/specular/internal/approval"
)

func TestEvaluateProvenanceProtocolEnforceDenyMissing(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	writeRiskPolicy(t, root, "provenance:\n  protocol: enforce\n")
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	att := `{"provenance":{"harness":"claude-code","governed":true}}`
	if err := os.WriteFile(filepath.Join(dir, "auth.attestation.json"), []byte(att), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != Deny {
		t.Fatalf("verdict=%s reason=%s", res.Verdict, res.Reason)
	}
	if !res.Provenance.Enforced || res.Provenance.Status != StatusFail {
		t.Fatalf("enforced=%v status=%s", res.Provenance.Enforced, res.Provenance.Status)
	}
	if !strings.Contains(res.Reason, "missing .provenance.json") {
		t.Fatalf("reason=%s", res.Reason)
	}
	text := FormatText(res)
	if !strings.Contains(text, "Status         FAIL") || !strings.Contains(text, "APP protocol enforce") {
		t.Fatalf("board:\n%s", text)
	}
}

func TestEvaluateProvenanceProtocolEnforceDenyInvalid(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	writeRiskPolicy(t, root, "provenance:\n  protocol: enforce\n")
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	att := `{"provenance":{"harness":"claude-code"}}`
	if err := os.WriteFile(filepath.Join(dir, "auth.attestation.json"), []byte(att), 0o600); err != nil {
		t.Fatal(err)
	}
	// Missing required session field → Validate fails.
	bad := `{"schema":"specular.provenance/v1","version":"1","harness":"claude-code"}`
	if err := os.WriteFile(filepath.Join(dir, "auth.provenance.json"), []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != Deny {
		t.Fatalf("verdict=%s reason=%s docs=%d ok=%d", res.Verdict, res.Reason,
			res.Provenance.ProtocolDocs, res.Provenance.ProtocolOK)
	}
	if res.Provenance.ProtocolDocs != 1 || res.Provenance.ProtocolOK != 0 {
		t.Fatalf("docs=%d ok=%d", res.Provenance.ProtocolDocs, res.Provenance.ProtocolOK)
	}
	if !strings.Contains(res.Reason, "docs valid") {
		t.Fatalf("reason=%s", res.Reason)
	}
}

func TestEvaluateProvenanceProtocolEnforceAllowValid(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	writeRiskPolicy(t, root, "provenance:\n  protocol: enforce\n")
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	att := `{"provenance":{"harness":"claude-code","governed":true}}`
	if err := os.WriteFile(filepath.Join(dir, "auth.attestation.json"), []byte(att), 0o600); err != nil {
		t.Fatal(err)
	}
	prov := `{"schema":"specular.provenance/v1","version":"1","session":"auth","harness":"claude-code","governed":true}`
	if err := os.WriteFile(filepath.Join(dir, "auth.provenance.json"), []byte(prov), 0o600); err != nil {
		t.Fatal(err)
	}

	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != Allow {
		t.Fatalf("verdict=%s reason=%s", res.Verdict, res.Reason)
	}
	if !res.Provenance.Enforced || res.Provenance.Status != StatusPass {
		t.Fatalf("enforced=%v status=%s", res.Provenance.Enforced, res.Provenance.Status)
	}
	if !strings.Contains(res.Reason, "APP protocol ok") {
		t.Fatalf("reason=%s", res.Reason)
	}
}

func TestEvaluateProvenanceProtocolEnforceSoftAllow(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	writeRiskPolicy(t, root, "provenance:\n  protocol: enforce\n")
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	att := `{"provenance":{"harness":"claude-code"}}`
	if err := os.WriteFile(filepath.Join(dir, "auth.attestation.json"), []byte(att), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := approval.Write(root, &approval.Record{
		Type:       approval.TypeException,
		ResourceID: "exception-app-protocol",
		ApprovedBy: "platform",
		Reason:     "migrate attest hooks",
		Policy:     "provenance",
	}); err != nil {
		t.Fatal(err)
	}

	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != Allow {
		t.Fatalf("verdict=%s reason=%s", res.Verdict, res.Reason)
	}
	if res.Provenance.Status != StatusFail {
		t.Fatalf("underlying status should stay FAIL, got %s", res.Provenance.Status)
	}
	if len(res.Approvals.Overrules) != 1 || res.Approvals.Overrules[0].Kind != "provenance" {
		t.Fatalf("overrules=%+v", res.Approvals.Overrules)
	}
	if !strings.Contains(res.Reason, "overruled provenance DENY") {
		t.Fatalf("reason=%s", res.Reason)
	}
}

func TestEvaluateProvenanceProtocolAdvisoryWithoutPolicy(t *testing.T) {
	t.Parallel()
	root := initTempRepo(t)
	dir := filepath.Join(root, ".specular", "sessions")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	att := `{"provenance":{"harness":"claude-code"}}`
	if err := os.WriteFile(filepath.Join(dir, "auth.attestation.json"), []byte(att), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err := Evaluate(Options{ProjectRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if res.Verdict != Allow {
		t.Fatalf("verdict=%s", res.Verdict)
	}
	if res.Provenance.Enforced || res.Provenance.Status != StatusPass {
		t.Fatalf("enforced=%v status=%s", res.Provenance.Enforced, res.Provenance.Status)
	}
}
