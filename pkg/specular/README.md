# Specular Public SDK

This package contains the public SDK for Specular, providing stable types and interfaces for:

1. **External integrations** - Third-party tools and plugins
2. **Future enterprise platform** - The private `specular-platform` repository will import these types
3. **API compatibility** - Ensures consistent data structures between free CLI and enterprise platform

## Package Structure

```
pkg/specular/
├── types/          # Core domain types (Spec, Plan, Policy, value objects)
├── provider/       # AI provider interface and types
├── client/         # Platform API client (stub for v2.0)
└── features/       # Feature flags for free vs. enterprise editions
```

## Design Principles

### 1. One-Way Dependency

The private enterprise platform (`specular-platform/`) will import from this public SDK:

```go
// ✅ ALLOWED: Private platform imports public SDK
// specular-platform/internal/service/spec.go
import "github.com/felixgeelhaar/specular/pkg/specular/types"

// ❌ FORBIDDEN: Public SDK never imports private code
// pkg/specular/types/spec.go
// import "github.com/felixgeelhaar/specular-platform/..." // NEVER!
```

### 2. Stable Public API

Types in this package form the public API contract. Breaking changes require:
- Semantic versioning (MAJOR bump)
- Migration guides
- Deprecation notices

### 3. Minimal Dependencies

The public SDK has minimal external dependencies to ensure:
- Easy adoption by third-party tools
- Fast compilation
- Reduced security surface area

## Usage Examples

### Using Core Types

```go
import "github.com/felixgeelhaar/specular/pkg/specular/types"

// Create a feature
feature := types.Feature{
    ID:       types.FeatureID("user-authentication"),
    Title:    "User Authentication",
    Priority: types.PriorityP0,
    Success:  []string{"Users can register and login securely"},
}

// Validate feature ID
featureID, err := types.NewFeatureID("user-auth-system")
if err != nil {
    log.Fatal(err)
}
```

### Implementing a Custom Provider

```go
import (
    "context"
    "github.com/felixgeelhaar/specular/pkg/specular/provider"
)

type MyCustomProvider struct {
    // ... implementation
}

func (p *MyCustomProvider) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResponse, error) {
    // ... custom AI provider logic
    return &provider.GenerateResponse{
        Content:    "Generated response",
        TokensUsed: 150,
        Model:      "my-custom-model",
    }, nil
}

func (p *MyCustomProvider) GetInfo() *provider.ProviderInfo {
    return &provider.ProviderInfo{
        Name:        "my-custom-provider",
        Version:     "1.0.0",
        Type:        provider.ProviderTypeCLI,
        TrustLevel:  provider.TrustLevelCommunity,
        Description: "My custom AI provider",
    }
}
```

### Using Feature Flags

```go
import "github.com/felixgeelhaar/specular/pkg/specular/features"

// Check if enterprise features are available
if features.IsEnabled(features.FlagMultiTenancy) {
    // Use multi-tenant features
    setupTenantIsolation()
} else {
    // Gracefully degrade to single-tenant mode
    setupSingleTenant()
}

// Get current edition
edition := features.GetEdition()
if edition == features.EditionEnterprise {
    // Enterprise-specific setup
}
```

### Calling Platform API (v2.0+)

```go
import "github.com/felixgeelhaar/specular/pkg/specular/client"

// Create platform client
client := client.New("https://platform.specular.io", apiKey)

// Generate spec via platform API
spec, err := client.GenerateSpec(ctx, &client.GenerateSpecRequest{
    Prompt: "Build a REST API for user management",
})
if err != nil {
    // Fall back to local generation
    spec = generateLocally(prompt)
}
```

## Migration Timeline

### v1.6.0 (Current)
- ✅ Public SDK created
- ✅ Types, provider interface, client stub, feature flags
- ⏳ Internal packages continue using `internal/domain`, `internal/spec`, etc.

### v2.0 M9 (Repository Split)
- 🔜 Private `specular-platform/` repository created
- 🔜 Platform code imports `pkg/specular/types`
- 🔜 Platform API implements endpoints for `pkg/specular/client`

### v2.0+ (Gradual Migration)
- 🔜 Internal packages gradually refactored to use `pkg/specular/types`
- 🔜 Reduced duplication between internal and public types

## Versioning

This SDK follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version: Incompatible API changes
- **MINOR** version: Backwards-compatible functionality additions
- **PATCH** version: Backwards-compatible bug fixes

The SDK version is independent of Specular CLI version until v2.0.

## Contributing

When adding new types to the public SDK:

1. **Consider stability** - Once public, types are hard to change
2. **Document thoroughly** - Public API needs excellent docs
3. **Add tests** - All public types should have tests
4. **Check backwards compatibility** - Use tools like `go-compat-check`

## License

This public SDK is licensed under the **Business Source License 1.1** (BSL 1.1), the same as the main Specular project.

- **Permitted**: Internal use, consulting, education, personal projects
- **Prohibited**: Competing commercial SaaS offerings
- **Automatic conversion**: Apache 2.0 after 2 years

See [LICENSE](../../LICENSE) for full details.
