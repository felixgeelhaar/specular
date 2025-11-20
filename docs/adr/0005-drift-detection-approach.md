# ADR 0005: Drift Detection and SARIF Output Format

**Status:** Accepted

**Date:** 2025-01-07

**Decision Makers:** Specular Core Team

## Context

Specular generates code based on product specifications. Over time, specifications change, code evolves, and infrastructure policies update. Without continuous validation, implementations can drift from their original intent, creating:

### Problems from Drift
1. **Plan Drift**: Spec changes without updating spec.lock → implementation doesn't match spec
2. **Code Drift**: Implementation doesn't conform to API contracts defined in OpenAPI
3. **Infrastructure Drift**: Code violates security/quality policies
4. **Compliance Drift**: Generated code no longer meets regulatory requirements

### Requirements
- Detect all three types of drift automatically
- Integrate with CI/CD pipelines (GitHub Actions, GitLab CI)
- Provide actionable feedback to developers
- Support security scanning tools (CodeQL, Snyk, Semgrep)
- Enable automated PR reviews
- Track drift trends over time

### Industry Standards
- **SARIF**: Static Analysis Results Interchange Format (OASIS standard)
- **CodeQL**: GitHub's code scanning uses SARIF
- **Semgrep**: Security scanning outputs SARIF
- **SonarQube**: Supports SARIF import

## Decision

**We will implement comprehensive drift detection with output in SARIF v2.1.0 format for maximum CI/CD integration.**

### Drift Detection Architecture

```
┌─────────────────────────────────────────┐
│     Specular Eval Command               │
│  --spec --plan --lock --api-spec        │
└────────────┬────────────────────────────┘
             │
             v
┌─────────────────────────────────────────┐
│        Drift Detection Engine           │
│  ┌──────────────────────────────────┐  │
│  │ 1. Plan Drift Detector           │  │
│  │    - Compare spec.lock hashes    │  │
│  │    - Identify changed features   │  │
│  └──────────────────────────────────┘  │
│  ┌──────────────────────────────────┐  │
│  │ 2. Code Drift Detector           │  │
│  │    - Validate against OpenAPI    │  │
│  │    - Check API conformance       │  │
│  └──────────────────────────────────┘  │
│  ┌──────────────────────────────────┐  │
│  │ 3. Infrastructure Drift Detector │  │
│  │    - Policy compliance check     │  │
│  │    - Security scanning           │  │
│  └──────────────────────────────────┘  │
└────────────┬────────────────────────────┘
             │
             v
┌─────────────────────────────────────────┐
│       SARIF Report Generator            │
│  - Convert findings to SARIF format     │
│  - Add severity, location, remediation  │
│  - Include metadata and metrics         │
└────────────┬────────────────────────────┘
             │
             v
       drift.sarif (JSON)
             │
        ┌────┴────┐
        │         │
        v         v
   ┌────────┐ ┌────────────┐
   │ GitHub │ │   CLI      │
   │CodeQL  │ │ Reporting  │
   └────────┘ └────────────┘
```

## Implementation Details

### Plan Drift Detection
```go
func DetectPlanDrift(lock *SpecLock, plan *Plan) []Finding {
    var findings []Finding

    for _, task := range plan.Tasks {
        lockedFeature := lock.Features[task.FeatureID]
        if lockedFeature == nil {
            findings = append(findings, Finding{
                Code: "MISSING_FEATURE_LOCK",
                Message: fmt.Sprintf("Feature %s not in spec lock", task.FeatureID),
                Severity: "error",
            })
            continue
        }

        // Compare hashes
        currentHash := computeFeatureHash(task.Feature)
        if currentHash != lockedFeature.Hash {
            findings = append(findings, Finding{
                Code: "FEATURE_HASH_MISMATCH",
                Message: fmt.Sprintf("Feature %s changed since lock", task.FeatureID),
                Severity: "warning",
                FeatureID: task.FeatureID,
                Expected: lockedFeature.Hash,
                Actual: currentHash,
            })
        }
    }

    return findings
}
```

### Code Drift Detection
```go
func DetectCodeDrift(spec *Spec, lock *SpecLock, opts CodeDriftOptions) []Finding {
    var findings []Finding

    // Validate OpenAPI spec if provided
    if opts.APISpecPath != "" {
        validator, err := NewOpenAPIValidator(opts.APISpecPath)
        if err != nil {
            findings = append(findings, Finding{
                Code: "INVALID_API_SPEC",
                Message: fmt.Sprintf("Invalid OpenAPI spec: %v", err),
                Severity: "error",
                Location: opts.APISpecPath,
            })
            return findings
        }

        // Check if API endpoints exist
        endpointFindings := validator.ValidateEndpoints(spec.Features)
        findings = append(findings, endpointFindings...)
    }

    return findings
}
```

### Infrastructure Drift Detection
```go
func DetectInfraDrift(opts InfraDriftOptions) []Finding {
    var findings []Finding

    // Check Docker image compliance
    for taskID, image := range opts.TaskImages {
        if !isImageAllowed(image, opts.Policy.Docker.ImageAllowlist) {
            findings = append(findings, Finding{
                Code: "DISALLOWED_IMAGE",
                Message: fmt.Sprintf("Image not in allowlist: %s", image),
                Severity: "error",
                TaskID: taskID,
                Location: fmt.Sprintf("task:%s", taskID),
            })
        }
    }

    return findings
}
```

### SARIF Report Generation
```go
type SARIFReport struct {
    Version string      `json:"version"`
    Schema  string      `json:"$schema"`
    Runs    []SARIFRun  `json:"runs"`
}

type SARIFRun struct {
    Tool     SARIFTool     `json:"tool"`
    Results  []SARIFResult `json:"results"`
}

type SARIFResult struct {
    RuleID    string          `json:"ruleId"`
    Level     string          `json:"level"` // "error", "warning", "note"
    Message   SARIFMessage    `json:"message"`
    Locations []SARIFLocation `json:"locations,omitempty"`
}

func (r *Report) ToSARIF() *SARIFReport {
    sarif := &SARIFReport{
        Version: "2.1.0",
        Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
        Runs: []SARIFRun{
            {
                Tool: SARIFTool{
                    Driver: SARIFDriver{
                        Name:    "Specular",
                        Version: "1.0.0",
                        InformationURI: "https://github.com/felixgeelhaar/specular",
                    },
                },
                Results: r.convertFindings(),
            },
        },
    }
    return sarif
}
```

### SARIF Severity Mapping
```go
func mapSeverity(severity string) string {
    switch severity {
    case "error":
        return "error"
    case "warning":
        return "warning"
    case "info", "note":
        return "note"
    default:
        return "warning"
    }
}
```

## Alternatives Considered

### Option 1: Custom JSON Format
**Pros:**
- Full control over structure
- Optimize for Specular use case
- Simpler schema

**Cons:**
- ❌ No CI/CD integration
- ❌ Can't use GitHub Code Scanning
- ❌ No ecosystem tooling
- ❌ Custom parsers needed

**Verdict:** REJECTED (reinventing wheel)

### Option 2: Plain Text Output
**Pros:**
- Human-readable
- Simple implementation
- No parsing needed

**Cons:**
- ❌ No structured data
- ❌ Can't integrate with tools
- ❌ No programmatic analysis
- ❌ Poor CI/CD experience

**Verdict:** REJECTED (too limited)

### Option 3: JUnit XML Format
**Pros:**
- CI/CD support
- Test result integration
- Widely supported

**Cons:**
- ❌ Designed for test results, not security findings
- ❌ Limited metadata support
- ❌ No location mapping
- ❌ Poor for code analysis

**Verdict:** REJECTED (wrong use case)

### Option 4: CycloneDX (SBOM Format)
**Pros:**
- Excellent for dependency tracking
- Security vulnerability mapping
- Supply chain focus

**Cons:**
- ❌ Overkill for drift detection
- ❌ Complex schema
- ❌ Less CI/CD integration than SARIF

**Verdict:** Future consideration for SBOM generation

## SARIF Benefits

### GitHub Integration
```yaml
# .github/workflows/specular.yml
- name: Detect Drift
  run: specular eval --fail-on-drift --report drift.sarif

- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v2
  with:
    sarif_file: drift.sarif
```

**Result**: Findings appear in GitHub Security tab with:
- Line-level annotations
- Severity classification
- Remediation guidance
- Trend tracking

### CI/CD Dashboards
- **GitLab**: Native SARIF support in Security Dashboard
- **Azure DevOps**: SARIF viewer extension
- **Jenkins**: SARIF plugin available

### Security Tools Integration
- **Snyk**: Combines with SARIF from Specular
- **SonarQube**: Import SARIF findings
- **Semgrep**: Merge scan results

## Consequences

### Positive
- ✅ **CI/CD Native**: Works with GitHub, GitLab, Azure DevOps
- ✅ **Industry Standard**: OASIS SARIF spec (stable)
- ✅ **Tool Ecosystem**: Parsers, validators, viewers available
- ✅ **Rich Metadata**: Locations, severity, remediation
- ✅ **Trending**: Track drift over time
- ✅ **Actionable**: Clear findings with context

### Negative
- ❌ **Verbose**: SARIF files can be large (~10KB-1MB)
- ❌ **Complexity**: More complex than plain text
- ❌ **Learning Curve**: SARIF spec is detailed

### Mitigations
- Compress SARIF for storage (gzip ~10x reduction)
- Provide human-readable summary in CLI
- Auto-upload to code scanning (no manual handling)
- Include SARIF examples in documentation

## Drift Detection Strategies

### 1. Continuous Validation (CI/CD)
```yaml
on: [push, pull_request]
jobs:
  drift:
    runs-on: ubuntu-latest
    steps:
      - uses: felixgeelhaar/specular-action@v1
        with:
          command: eval
          fail-on-drift: true
```

**Frequency**: Every commit
**Goal**: Catch drift immediately

### 2. Scheduled Audits
```yaml
on:
  schedule:
    - cron: '0 0 * * 0'  # Weekly
```

**Frequency**: Weekly/monthly
**Goal**: Detect gradual drift

### 3. Pre-Deployment Validation
```yaml
on:
  push:
    branches: [production]
```

**Frequency**: Before production deploy
**Goal**: Prevent drifted code in production

## Finding Codes

### Plan Drift Codes
- `MISSING_FEATURE_LOCK`: Feature not in spec.lock
- `FEATURE_HASH_MISMATCH`: Feature changed since lock
- `EXTRA_FEATURE`: Feature in lock but not in spec

### Code Drift Codes
- `MISSING_API_PATH`: API endpoint not in OpenAPI spec
- `MISSING_API_METHOD`: HTTP method not defined
- `INVALID_API_SPEC`: OpenAPI spec validation failed

### Infrastructure Drift Codes
- `DISALLOWED_IMAGE`: Docker image not in allowlist
- `RESOURCE_LIMIT_EXCEEDED`: Task exceeds resource limits
- `POLICY_VIOLATION`: General policy violation

## Future Enhancements

### v1.1-v1.2
- [ ] Auto-remediation suggestions
- [ ] Drift impact analysis (breaking vs. non-breaking)
- [ ] Historical drift trending
- [ ] Custom drift rules (via policy)

### v2.0+
- [ ] ML-based drift prediction
- [ ] Semantic drift detection (intent vs. implementation)
- [ ] Cross-repository drift analysis
- [ ] Drift visualization dashboard

## Related Decisions
- ADR 0001: Spec Lock Format (enables plan drift detection)
- ADR 0003: Docker-Only Execution (enables infrastructure drift detection)
- Future ADR: OpenAPI validation strategy

## References
- [SARIF v2.1.0 Specification](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html)
- [GitHub Code Scanning](https://docs.github.com/en/code-security/code-scanning)
- [SARIF Tutorials](https://github.com/microsoft/sarif-tutorials)
- [OpenAPI Specification](https://swagger.io/specification/)
