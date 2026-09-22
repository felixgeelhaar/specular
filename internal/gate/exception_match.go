package gate

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DenyKind is the hard-DENY category a scoped exception can soft-ALLOW.
type DenyKind string

const (
	// DenyKindDrift soft-ALLOWs a drift evaluation DENY.
	DenyKindDrift DenyKind = "drift"
	// DenyKindPolicy soft-ALLOWs a policy verification DENY.
	DenyKindPolicy DenyKind = "policy"
	// DenyKindRisk soft-ALLOWs a risk-adaptive approval DENY.
	DenyKindRisk DenyKind = "risk"
)

// ExceptionOverrule records that an open exception soft-ALLOWed a DENY.
type ExceptionOverrule struct {
	ResourceID string `json:"resource_id"`
	Kind       string `json:"kind"` // drift | policy | risk
	Binding    string `json:"binding"`
	Reason     string `json:"reason,omitempty"`
	ApprovedBy string `json:"approved_by,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Policy     string `json:"policy,omitempty"`
}

func findExceptionOverrule(res *Result, kind DenyKind) *ExceptionOverrule {
	if res == nil {
		return nil
	}
	for _, ex := range res.Approvals.Exceptions {
		if ex.Expired {
			continue
		}
		if binding, ok := exceptionMatches(ex, kind, res); ok {
			return &ExceptionOverrule{
				ResourceID: ex.ResourceID,
				Kind:       string(kind),
				Binding:    binding,
				Reason:     ex.Reason,
				ApprovedBy: ex.ApprovedBy,
				Scope:      ex.Scope,
				Policy:     ex.Policy,
			}
		}
	}
	return nil
}

func exceptionMatches(ex ApprovalSummary, kind DenyKind, res *Result) (binding string, ok bool) {
	policy := normToken(ex.Policy)
	scope := strings.TrimSpace(ex.Scope)
	if policy == "" && scope == "" {
		return "", false
	}
	switch kind {
	case DenyKindDrift:
		return matchDriftException(policy, scope, res.Drift)
	case DenyKindPolicy:
		return matchPolicyException(policy, scope, res.Policy)
	case DenyKindRisk:
		return matchRiskException(policy, scope, res.Risk)
	default:
		return "", false
	}
}

func matchDriftException(policy, scope string, drift DriftSection) (string, bool) {
	for _, f := range drift.Findings {
		sev := strings.ToLower(strings.TrimSpace(f.Severity))
		if sev != "" && sev != "error" {
			continue
		}
		code := normToken(f.Code)
		if policy != "" && policy == code {
			return "policy→finding:" + f.Code, true
		}
		path := firstNonEmpty(f.Path, f.Location)
		if scope != "" && path != "" && scopeMatchesPath(scope, path) {
			return "scope→" + filepath.ToSlash(path), true
		}
	}
	// Category bind: policy=drift + scope only when no path-bearing findings
	// exist (otherwise scope must hit a finding path above).
	if policy == "drift" && scope != "" && drift.Status == StatusFail && !driftHasPathFindings(drift) {
		return "policy=drift+scope=" + scope, true
	}
	return "", false
}

func driftHasPathFindings(drift DriftSection) bool {
	for _, f := range drift.Findings {
		if firstNonEmpty(f.Path, f.Location) != "" {
			return true
		}
	}
	return false
}

func matchPolicyException(policy, scope string, pol PolicySection) (string, bool) {
	for _, name := range pol.FailedChecks {
		n := normToken(name)
		if n == "" {
			continue
		}
		if policy == n || normToken(scope) == n {
			return "check→" + name, true
		}
	}
	if policy == "policy" && strings.TrimSpace(scope) != "" {
		return "policy=policy+scope=" + strings.TrimSpace(scope), true
	}
	return "", false
}

func matchRiskException(policy, scope string, risk RiskSection) (string, bool) {
	level := normToken(risk.Level)
	for _, cand := range []string{policy, normToken(scope)} {
		if cand == "" {
			continue
		}
		if cand == "risk" || (level != "" && cand == level) {
			return "risk→" + cand, true
		}
	}
	return "", false
}

func normToken(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// scopeMatchesPath reports whether scope covers path (exact, prefix, or trailing **).
func scopeMatchesPath(scope, path string) bool {
	s := filepath.ToSlash(strings.TrimSpace(scope))
	p := filepath.ToSlash(strings.TrimSpace(path))
	if s == "" || p == "" {
		return false
	}
	s = strings.TrimPrefix(s, "./")
	p = strings.TrimPrefix(p, "./")
	if strings.HasSuffix(s, "/**") {
		prefix := strings.TrimSuffix(s, "/**")
		return p == prefix || strings.HasPrefix(p, prefix+"/")
	}
	if strings.HasSuffix(s, "**") {
		prefix := strings.TrimSuffix(s, "**")
		return strings.HasPrefix(p, prefix)
	}
	if s == p {
		return true
	}
	return strings.HasPrefix(p, strings.TrimSuffix(s, "/")+"/")
}

func updateApprovalsNoteForOverrule(res *Result) {
	if res == nil || len(res.Approvals.Overrules) == 0 {
		return
	}
	kinds := make([]string, 0, len(res.Approvals.Overrules))
	for _, o := range res.Approvals.Overrules {
		kinds = append(kinds, o.Kind)
	}
	res.Approvals.Note = fmt.Sprintf(
		"%d open exception(s); %d overruled %s DENY → soft-ALLOW",
		len(res.Approvals.Exceptions),
		len(res.Approvals.Overrules),
		strings.Join(kinds, "+"),
	)
}
