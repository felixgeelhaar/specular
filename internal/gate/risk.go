package gate

import (
	"path/filepath"
	"sort"
	"strings"
)

// RiskSection is an advisory change-risk profile for the gate board.
// Factors are explainable strings derived from path heuristics and provenance.
// Risk never flips ALLOW→DENY by itself; drift and policy remain decisive.
type RiskSection struct {
	Level   string   `json:"level"` // NONE | LOW | MEDIUM | HIGH — advisory only
	Factors []string `json:"factors,omitempty"`
	Note    string   `json:"note,omitempty"`
}

// Known explainable factor strings (stable for tests and JSON consumers).
const (
	FactorAuthPaths       = "authentication / authorization paths modified"
	FactorDependencyPaths = "dependency or lockfile changes"
	FactorInfraCIPaths    = "infrastructure or CI configuration changes"
	FactorSecretsPaths    = "secrets-adjacent paths modified"
	FactorUnattested      = "unattested provenance (no session attestation)"
)

func assessRisk(provenance ProvenanceSection, root string) RiskSection {
	paths := collectChangePaths(root)
	factors := detectRiskFactors(paths, provenance.Attested)
	sec := RiskSection{
		Level:   advisoryRiskLevel(factors),
		Factors: factors,
	}
	switch {
	case len(factors) == 0:
		sec.Note = "no elevated risk factors from path heuristics"
	default:
		sec.Note = "advisory only — does not change gate verdict"
	}
	return sec
}

func collectChangePaths(root string) []string {
	var paths []string
	if out, err := runGit(root, "status", "--porcelain", "-uall"); err == nil {
		paths = append(paths, parsePorcelainPaths(out)...)
	}
	if len(paths) == 0 {
		for _, base := range []string{"origin/main", "origin/master", "main", "master"} {
			out, err := runGit(root, "diff", "--name-only", base+"...HEAD")
			if err != nil {
				continue
			}
			lines := nonEmptyLines(out)
			if len(lines) > 0 {
				paths = append(paths, lines...)
				break
			}
		}
	}
	return unique(paths)
}

func parsePorcelainPaths(out string) []string {
	var paths []string
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		rest := strings.TrimSpace(line[3:])
		if rest == "" {
			continue
		}
		if i := strings.Index(rest, " -> "); i >= 0 {
			rest = rest[i+4:]
		}
		rest = strings.Trim(rest, `"`)
		if rest != "" {
			paths = append(paths, filepath.ToSlash(rest))
		}
	}
	return paths
}

func detectRiskFactors(paths []string, attested bool) []string {
	var factors []string
	var auth, deps, infra, secrets bool
	for _, p := range paths {
		n := strings.ToLower(filepath.ToSlash(p))
		base := filepath.Base(n)
		if !auth && isAuthPath(n, base) {
			auth = true
		}
		if !deps && isDependencyPath(n, base) {
			deps = true
		}
		if !infra && isInfraCIPath(n, base) {
			infra = true
		}
		if !secrets && isSecretsAdjacentPath(n, base) {
			secrets = true
		}
	}
	if auth {
		factors = append(factors, FactorAuthPaths)
	}
	if deps {
		factors = append(factors, FactorDependencyPaths)
	}
	if infra {
		factors = append(factors, FactorInfraCIPaths)
	}
	if secrets {
		factors = append(factors, FactorSecretsPaths)
	}
	if !attested {
		factors = append(factors, FactorUnattested)
	}
	sort.Strings(factors)
	return factors
}

func advisoryRiskLevel(factors []string) string {
	n := len(factors)
	switch {
	case n == 0:
		return "NONE"
	case n == 1:
		return "LOW"
	case n == 2:
		return "MEDIUM"
	default:
		return "HIGH"
	}
}

func isAuthPath(n, base string) bool {
	authTokens := []string{
		"/auth/", "/auth.", "/oauth", "/oidc", "/saml", "/jwt",
		"/rbac", "/iam/", "/login", "/logout", "/sso",
		"/permission", "/authorize", "/authentication", "/authorization",
		"/passwd", "/password", "/session/token",
	}
	for _, t := range authTokens {
		if strings.Contains(n, t) {
			return true
		}
	}
	switch base {
	case "auth.go", "auth.ts", "auth.js", "auth.py", "authorizer.go",
		"authentication.go", "authorization.go", "oauth.go", "jwt.go":
		return true
	}
	return strings.HasPrefix(base, "auth_") || strings.HasPrefix(base, "auth-")
}

func isDependencyPath(n, base string) bool {
	switch base {
	case "go.mod", "go.sum",
		"package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
		"npm-shrinkwrap.json", "bun.lockb",
		"cargo.toml", "cargo.lock",
		"requirements.txt", "pipfile", "pipfile.lock", "poetry.lock",
		"gemfile", "gemfile.lock",
		"composer.json", "composer.lock",
		"mix.lock", "pdm.lock", "renv.lock":
		return true
	}
	if strings.HasPrefix(base, "requirements") && strings.HasSuffix(base, ".txt") {
		return true
	}
	return strings.Contains(n, "/vendor/") && strings.HasSuffix(n, "/modules.txt")
}

func isInfraCIPath(n, base string) bool {
	if strings.Contains(n, "/.github/workflows/") || strings.HasPrefix(n, ".github/workflows/") {
		return true
	}
	if strings.Contains(n, "/.circleci/") || strings.HasPrefix(n, ".circleci/") {
		return true
	}
	if strings.Contains(n, "/.gitlab-ci") || base == ".gitlab-ci.yml" {
		return true
	}
	switch base {
	case "dockerfile", "docker-compose.yml", "docker-compose.yaml",
		"jenkinsfile", "cloudbuild.yaml", "cloudbuild.yml",
		"buildkite.yml", "buildkite.yaml", "chart.yaml":
		return true
	}
	if strings.HasSuffix(n, ".tf") || strings.HasSuffix(n, ".tfvars") {
		return true
	}
	infraDirs := []string{"/terraform/", "/helm/", "/k8s/", "/kubernetes/", "/deploy/", "/infra/"}
	for _, d := range infraDirs {
		if strings.Contains(n, d) {
			return true
		}
	}
	return strings.HasPrefix(base, "dockerfile.")
}

func isSecretsAdjacentPath(n, base string) bool {
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	secretTokens := []string{
		"/secrets/", "/secret/", "/credentials/", "/credential/",
		"/certs/", "/certificates/", "/.vault", "/vault/",
	}
	for _, t := range secretTokens {
		if strings.Contains(n, t) {
			return true
		}
	}
	switch {
	case strings.HasSuffix(base, ".pem"),
		strings.HasSuffix(base, ".p12"),
		strings.HasSuffix(base, ".pfx"),
		strings.HasSuffix(base, ".key"),
		base == "id_rsa", base == "id_ed25519", base == "id_ecdsa",
		base == "kubeconfig", base == "service-account.json",
		strings.Contains(base, "credentials"):
		return true
	}
	return false
}
