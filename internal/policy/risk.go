package policy

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// StringList unmarshals a YAML string or sequence of strings.
// Used for PRODUCT_INTENT `approval: none` vs `approval: [code-owner]`.
type StringList []string

// UnmarshalYAML accepts a scalar string or a sequence.
func (s *StringList) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		*s = nil
		return nil
	}
	switch value.Kind {
	case yaml.ScalarNode:
		var single string
		if err := value.Decode(&single); err != nil {
			return err
		}
		if strings.TrimSpace(single) == "" {
			*s = nil
			return nil
		}
		*s = []string{single}
		return nil
	case yaml.SequenceNode:
		var multi []string
		if err := value.Decode(&multi); err != nil {
			return err
		}
		*s = multi
		return nil
	case yaml.AliasNode:
		if value.Alias != nil {
			return s.UnmarshalYAML(value.Alias)
		}
	}
	*s = nil
	return nil
}

// HasRiskGovernance reports whether any risk tier is configured.
func (p *Policy) HasRiskGovernance() bool {
	return p != nil && p.Risk != nil && p.Risk.hasAnyTier()
}

func (r *RiskGovernance) hasAnyTier() bool {
	if r == nil {
		return false
	}
	return r.Low != nil || r.Medium != nil || r.High != nil || r.Critical != nil
}

// TierFor returns the configured tier for a risk level (case-insensitive).
// CRITICAL falls back to HIGH when critical is unset.
func (r *RiskGovernance) TierFor(level string) *RiskTier {
	if r == nil {
		return nil
	}
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "LOW":
		return r.Low
	case "MEDIUM":
		return r.Medium
	case "HIGH":
		return r.High
	case "CRITICAL":
		if r.Critical != nil {
			return r.Critical
		}
		return r.High
	default:
		return nil
	}
}

// RequiredApprovals returns normalized role names for this tier.
// "none" / empty / whitespace-only entries are dropped.
func (t *RiskTier) RequiredApprovals() []string {
	if t == nil {
		return nil
	}
	raw := append([]string{}, t.Approval...)
	raw = append(raw, t.Approvals...)
	var out []string
	seen := map[string]struct{}{}
	for _, a := range raw {
		role := strings.ToLower(strings.TrimSpace(a))
		if role == "" || role == "none" {
			continue
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	return out
}

// BlocksAutonomous reports whether this tier sets autonomous_execution: false.
func (t *RiskTier) BlocksAutonomous() bool {
	return t != nil && t.AutonomousExecution != nil && !*t.AutonomousExecution
}
