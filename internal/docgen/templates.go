package docgen

const prdTemplate = `# PRD

## Overview
- **Project:** {{.ProjectName}}
- **Generated:** {{.Timestamp.Format "2006-01-02"}}
- **Governance Level:** {{.Governance}} ({{.GovernanceDescription}})
- **Provider Strategy:** {{.ProviderStrategy}} ({{.ProviderStrategyDescription}})

{{- if .RecommendedProviders}}
### Recommended Providers
{{- range .RecommendedProviders}}
- {{.}}
{{- end}}
{{- end}}

## Objectives
{{- range .Features}}
- {{.}}
{{- end}}

## Providers
{{- range .Providers}}
- **{{.Name}}** ({{.Source}}, trust: {{.TrustLevel}})
  {{.Description}}
  {{- if .Capabilities}}
  Capabilities: {{.Capabilities}}
  {{- end}}
  {{- if .Hints}}
  {{.Hints}}
  {{- end}}
{{- end}}
`

const visionTemplate = `# Vision

This project keeps Specular's AI-native workflows governed, traceable, and provider-aware.

- **Project:** {{.ProjectName}}
- **Governance:** {{.Governance}} ({{.GovernanceDescription}})
- **Provider Strategy:** {{.ProviderStrategy}} ({{.ProviderStrategyDescription}})

Our vision is to make governance docs the single source of truth for every plan, keeping PRD/vision/roadmap tightly aligned with detected providers, descriptors, and policy guardrails.
`

const roadmapTemplate = `# Roadmap

1. Lock the governance docs (prd.md, vision.md, roadmap.md, tdd.md) inside .specular/docs and tie them to spec/plan artifacts.
2. Detect provider metadata (trust, source, capabilities) and surface it in docs, doctor output, and plans.
3. Keep the CLI wrappers packaged (scripts/package-cli-wrappers.sh → dist/providers/* → release archives) so workspaces start with the descriptors already satisfied.
4. Automate doc reviews/drift detection before any build/plan execution.
`

const tddTemplate = `# TDD

## Testing Strategy

- Validate provider availability using the descriptor catalog.
- Lock governance docs and detect drift before plan execution.
- Reference .specular/providers.yaml + .specular/providers.yaml.example to ensure safe provider usage.

## Acceptance Criteria

- All features on the plan are traceable back to the PRD/roadmap/vision documents.
- Provider health checks pass for the descriptor-led stack (Ollama + wrappers).
- Spec generation produces .specular/spec.yaml with governance metadata preserved.
`
