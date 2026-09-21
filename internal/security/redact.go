package security

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

// evidencePatterns match common secret shapes in free-form text (goals,
// prompts, notes) without requiring key=value labeling.
var evidencePatterns = []*regexp.Regexp{
	regexp.MustCompile(`ghp_[A-Za-z0-9_]{36,}`),
	regexp.MustCompile(`gho_[A-Za-z0-9_]{36,}`),
	regexp.MustCompile(`ghu_[A-Za-z0-9_]{36,}`),
	regexp.MustCompile(`ghs_[A-Za-z0-9_]{36,}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`xox[baprs]-[0-9]{10,12}-[0-9]{10,12}-[A-Za-z0-9]{24,}`),
	regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`),
	regexp.MustCompile(`(?i)(postgres|mysql|mongodb|redis)://[^\s'"]+:[^\s'"]+@`),
	regexp.MustCompile(`-----BEGIN[ A-Z]*PRIVATE KEY-----[\s\S]*?-----END[ A-Z]*PRIVATE KEY-----`),
	// Labeled secrets in free text: "api_key=..." / "token: ..."
	regexp.MustCompile(`(?i)(api[\s_-]?key|secret|password|token)\s*[:=]\s*["']?[A-Za-z0-9_\-+=/.]{16,}["']?`),
}

const redactedPlaceholder = "[REDACTED]"

// RedactSecrets replaces known secret shapes in text with [REDACTED].
// Safe for goals, prompts, and other free-form evidence fields.
func RedactSecrets(text string) string {
	if text == "" {
		return text
	}
	out := text
	for _, pat := range evidencePatterns {
		out = pat.ReplaceAllString(out, redactedPlaceholder)
	}
	return out
}

// ContainsSecret reports whether text appears to include a secret shape.
func ContainsSecret(text string) bool {
	return RedactSecrets(text) != text
}

// GoalDigest returns a sha256 hex digest of the original (pre-redaction) goal
// so auditors can correlate without storing the raw secret-bearing text.
func GoalDigest(goal string) string {
	sum := sha256.Sum256([]byte(goal))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// SafeGoal returns a redacted goal plus digest of the original.
func SafeGoal(goal string) (redacted, digest string) {
	return RedactSecrets(goal), GoalDigest(goal)
}
