package plugin

import (
	"fmt"
	"sort"
	"strings"
)

// DependencyResolver resolves plugin dependencies
type DependencyResolver struct {
	manager  *Manager
	registry *Registry
	resolved map[string]*ResolvedDependency
	visited  map[string]bool
	stack    []string
}

// ResolvedDependency represents a resolved dependency
type ResolvedDependency struct {
	// Name is the plugin name
	Name string `json:"name"`
	// Version is the resolved version
	Version string `json:"version"`
	// Source is where to install from
	Source string `json:"source"`
	// Dependencies are transitive dependencies
	Dependencies []string `json:"dependencies,omitempty"`
	// Optional indicates if this is an optional dependency
	Optional bool `json:"optional,omitempty"`
	// Depth is how deep in the dependency tree
	Depth int `json:"depth"`
}

// ConflictError represents a version conflict between dependencies
type ConflictError struct {
	Plugin      string
	Requested   []VersionRequest
	Description string
}

// VersionRequest represents a version request from a dependent
type VersionRequest struct {
	Version     string
	RequestedBy string
}

func (e *ConflictError) Error() string {
	var requests []string
	for _, r := range e.Requested {
		requests = append(requests, fmt.Sprintf("%s (requested by %s)", r.Version, r.RequestedBy))
	}
	return fmt.Sprintf("version conflict for %s: %s", e.Plugin, strings.Join(requests, ", "))
}

// CircularDependencyError represents a circular dependency
type CircularDependencyError struct {
	Chain []string
}

func (e *CircularDependencyError) Error() string {
	return fmt.Sprintf("circular dependency detected: %s", strings.Join(e.Chain, " -> "))
}

// ResolutionResult contains the result of dependency resolution
type ResolutionResult struct {
	// Resolved is the list of resolved dependencies in installation order
	Resolved []*ResolvedDependency
	// Conflicts contains any version conflicts detected
	Conflicts []*ConflictError
	// Circular contains any circular dependencies detected
	Circular []*CircularDependencyError
	// Missing contains dependencies that couldn't be found
	Missing []string
}

// IsSuccess returns true if resolution was successful
func (r *ResolutionResult) IsSuccess() bool {
	return len(r.Conflicts) == 0 && len(r.Circular) == 0 && len(r.Missing) == 0
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(manager *Manager, registry *Registry) *DependencyResolver {
	return &DependencyResolver{
		manager:  manager,
		registry: registry,
		resolved: make(map[string]*ResolvedDependency),
		visited:  make(map[string]bool),
		stack:    make([]string, 0),
	}
}

// Resolve resolves all dependencies for a plugin manifest
func (r *DependencyResolver) Resolve(manifest *Manifest) (*ResolutionResult, error) {
	result := &ResolutionResult{
		Resolved:  make([]*ResolvedDependency, 0),
		Conflicts: make([]*ConflictError, 0),
		Circular:  make([]*CircularDependencyError, 0),
		Missing:   make([]string, 0),
	}

	// Reset state
	r.resolved = make(map[string]*ResolvedDependency)
	r.visited = make(map[string]bool)
	r.stack = make([]string, 0)

	// Track version requests for conflict detection
	versionRequests := make(map[string][]VersionRequest)

	// Resolve each dependency
	for _, dep := range manifest.Dependencies {
		if err := r.resolveDependency(dep, manifest.Name, 0, versionRequests, result); err != nil {
			return nil, err
		}
	}

	// Check for conflicts
	for plugin, requests := range versionRequests {
		if conflict := r.detectConflict(plugin, requests); conflict != nil {
			result.Conflicts = append(result.Conflicts, conflict)
		}
	}

	// Build resolved list in topological order
	result.Resolved = r.topologicalSort()

	return result, nil
}

// resolveDependency resolves a single dependency recursively
func (r *DependencyResolver) resolveDependency(
	dep PluginDependency,
	requestedBy string,
	depth int,
	versionRequests map[string][]VersionRequest,
	result *ResolutionResult,
) error {
	name := dep.Name

	// Check for circular dependency
	if r.isInStack(name) {
		chain := append(r.stack, name)
		result.Circular = append(result.Circular, &CircularDependencyError{Chain: chain})
		return nil
	}

	// Record version request
	versionRequests[name] = append(versionRequests[name], VersionRequest{
		Version:     dep.Version,
		RequestedBy: requestedBy,
	})

	// If already resolved, check version compatibility
	if existing, ok := r.resolved[name]; ok {
		// Version already resolved, verify compatibility
		if dep.Version != "" && existing.Version != dep.Version {
			// Check if existing version satisfies the constraint
			if !r.versionSatisfies(existing.Version, dep.Version) {
				// Will be caught as conflict later
				return nil
			}
		}
		return nil
	}

	// Mark as being processed
	r.stack = append(r.stack, name)
	r.visited[name] = true

	// Resolve version
	version, source, err := r.resolveVersion(name, dep.Version)
	if err != nil {
		if dep.Optional {
			// Optional dependency not found, skip
			r.stack = r.stack[:len(r.stack)-1]
			return nil
		}
		result.Missing = append(result.Missing, name)
		r.stack = r.stack[:len(r.stack)-1]
		return nil
	}

	// Create resolved dependency
	resolved := &ResolvedDependency{
		Name:     name,
		Version:  version,
		Source:   source,
		Optional: dep.Optional,
		Depth:    depth,
	}

	// Get transitive dependencies
	transitiveDeps, err := r.getTransitiveDependencies(name, version)
	if err != nil {
		// Log warning but continue
		transitiveDeps = nil
	}

	// Resolve transitive dependencies
	for _, tdep := range transitiveDeps {
		if err := r.resolveDependency(tdep, name, depth+1, versionRequests, result); err != nil {
			return err
		}
		resolved.Dependencies = append(resolved.Dependencies, tdep.Name)
	}

	r.resolved[name] = resolved
	r.stack = r.stack[:len(r.stack)-1]

	return nil
}

// resolveVersion resolves a version constraint to a specific version
func (r *DependencyResolver) resolveVersion(name, constraint string) (version, source string, err error) {
	// First check if plugin is already installed
	if r.manager != nil {
		if p, ok := r.manager.Get(name); ok {
			if constraint == "" || r.versionSatisfies(p.Manifest.Version, constraint) {
				return p.Manifest.Version, "local", nil
			}
		}
	}

	// Check registry
	if r.registry != nil {
		version, err := r.registry.ResolveVersion(name, constraint)
		if err == nil {
			return version, fmt.Sprintf("registry:%s@%s", name, version), nil
		}
	}

	return "", "", fmt.Errorf("could not resolve %s@%s", name, constraint)
}

// getTransitiveDependencies gets dependencies for a plugin version
func (r *DependencyResolver) getTransitiveDependencies(name, version string) ([]PluginDependency, error) {
	// Check local installation first
	if r.manager != nil {
		if p, ok := r.manager.Get(name); ok {
			return p.Manifest.Dependencies, nil
		}
	}

	// Check registry
	if r.registry != nil {
		plugin, v, err := r.registry.GetVersion(name, version)
		if err == nil {
			// Get dependencies from registry version
			_ = plugin
			return v.Dependencies, nil
		}
	}

	return nil, fmt.Errorf("could not get dependencies for %s@%s", name, version)
}

// versionSatisfies checks if a version satisfies a constraint
func (r *DependencyResolver) versionSatisfies(version, constraint string) bool {
	if constraint == "" {
		return true
	}

	v, err := ParseVersion(version)
	if err != nil {
		return false
	}

	c, err := ParseConstraint(constraint)
	if err != nil {
		// Try as exact version match
		return version == constraint
	}

	return v.Satisfies(c)
}

// isInStack checks if a name is in the current resolution stack
func (r *DependencyResolver) isInStack(name string) bool {
	for _, n := range r.stack {
		if n == name {
			return true
		}
	}
	return false
}

// detectConflict checks for version conflicts
func (r *DependencyResolver) detectConflict(plugin string, requests []VersionRequest) *ConflictError {
	if len(requests) <= 1 {
		return nil
	}

	// Check if all requests are compatible
	var resolvedVersion string
	for _, req := range requests {
		if req.Version == "" {
			continue
		}

		if resolvedVersion == "" {
			resolvedVersion = req.Version
			continue
		}

		// Check compatibility
		if !r.areVersionsCompatible(resolvedVersion, req.Version) {
			return &ConflictError{
				Plugin:    plugin,
				Requested: requests,
			}
		}
	}

	return nil
}

// areVersionsCompatible checks if two version constraints are compatible
func (r *DependencyResolver) areVersionsCompatible(v1, v2 string) bool {
	// If either is empty (any version), they're compatible
	if v1 == "" || v2 == "" {
		return true
	}

	// If exact versions, must match
	ver1, err1 := ParseVersion(v1)
	ver2, err2 := ParseVersion(v2)

	if err1 == nil && err2 == nil {
		// Both are exact versions
		return ver1.Compare(ver2) == 0
	}

	// One or both are constraints
	c1, err1 := ParseConstraint(v1)
	c2, err2 := ParseConstraint(v2)

	if err1 != nil || err2 != nil {
		// Can't parse, assume compatible for now
		return true
	}

	// Check if there's a version that satisfies both constraints
	// This is a simplified check - in practice would need more sophisticated resolution
	testVersion, _ := ParseVersion("999.0.0")
	if testVersion.Satisfies(c1) && testVersion.Satisfies(c2) {
		return true
	}

	return false
}

// topologicalSort returns resolved dependencies in installation order
func (r *DependencyResolver) topologicalSort() []*ResolvedDependency {
	var result []*ResolvedDependency
	visited := make(map[string]bool)

	var visit func(name string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true

		if dep, ok := r.resolved[name]; ok {
			// Visit dependencies first
			for _, depName := range dep.Dependencies {
				visit(depName)
			}
			result = append(result, dep)
		}
	}

	// Visit all resolved dependencies
	for name := range r.resolved {
		visit(name)
	}

	return result
}

// DetectCircular checks for circular dependencies in a list of dependencies
func DetectCircular(deps []PluginDependency) *CircularDependencyError {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, dep := range deps {
		graph[dep.Name] = nil // Initialize node
	}

	// Track visited and recursion stack
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var chain []string

	var detectCycle func(node string) bool
	detectCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true
		chain = append(chain, node)

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if detectCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				// Found cycle
				chain = append(chain, neighbor)
				return true
			}
		}

		recStack[node] = false
		chain = chain[:len(chain)-1]
		return false
	}

	for node := range graph {
		if !visited[node] {
			if detectCycle(node) {
				return &CircularDependencyError{Chain: chain}
			}
		}
	}

	return nil
}

// CheckConflicts checks for version conflicts in a list of resolved dependencies
func CheckConflicts(deps []*ResolvedDependency) []*ConflictError {
	versions := make(map[string][]VersionRequest)

	for _, dep := range deps {
		versions[dep.Name] = append(versions[dep.Name], VersionRequest{
			Version:     dep.Version,
			RequestedBy: "direct",
		})
	}

	var conflicts []*ConflictError
	for name, reqs := range versions {
		if len(reqs) > 1 {
			// Check if versions conflict
			first := reqs[0].Version
			for _, req := range reqs[1:] {
				if req.Version != first {
					conflicts = append(conflicts, &ConflictError{
						Plugin:    name,
						Requested: reqs,
					})
					break
				}
			}
		}
	}

	return conflicts
}

// InstallOrder returns the optimal installation order for dependencies
func InstallOrder(deps []*ResolvedDependency) []*ResolvedDependency {
	// Sort by depth (deepest dependencies first)
	sorted := make([]*ResolvedDependency, len(deps))
	copy(sorted, deps)

	sort.Slice(sorted, func(i, j int) bool {
		// Higher depth = install first
		if sorted[i].Depth != sorted[j].Depth {
			return sorted[i].Depth > sorted[j].Depth
		}
		// Alphabetical for consistent ordering
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}

// FilterOptional removes optional dependencies from the list
func FilterOptional(deps []*ResolvedDependency) []*ResolvedDependency {
	var result []*ResolvedDependency
	for _, dep := range deps {
		if !dep.Optional {
			result = append(result, dep)
		}
	}
	return result
}

// DependencyTree represents a dependency tree for display
type DependencyTree struct {
	Name        string
	Version     string
	Children    []*DependencyTree
	Optional    bool
	Conflicting bool
	Missing     bool
}

// BuildTree builds a dependency tree for visualization
func BuildTree(resolved []*ResolvedDependency, root string) *DependencyTree {
	// Index resolved deps by name
	index := make(map[string]*ResolvedDependency)
	for _, dep := range resolved {
		index[dep.Name] = dep
	}

	var build func(name string, visited map[string]bool) *DependencyTree
	build = func(name string, visited map[string]bool) *DependencyTree {
		if visited[name] {
			return &DependencyTree{Name: name, Conflicting: true}
		}
		visited[name] = true

		dep, ok := index[name]
		if !ok {
			return &DependencyTree{Name: name, Missing: true}
		}

		tree := &DependencyTree{
			Name:     dep.Name,
			Version:  dep.Version,
			Optional: dep.Optional,
		}

		for _, childName := range dep.Dependencies {
			child := build(childName, visited)
			tree.Children = append(tree.Children, child)
		}

		delete(visited, name)
		return tree
	}

	return build(root, make(map[string]bool))
}

// PrintTree prints a dependency tree to a string
func PrintTree(tree *DependencyTree, prefix string, isLast bool) string {
	var sb strings.Builder

	// Print current node
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	if prefix == "" {
		connector = ""
	}

	status := ""
	if tree.Conflicting {
		status = " [CONFLICT]"
	} else if tree.Missing {
		status = " [MISSING]"
	} else if tree.Optional {
		status = " (optional)"
	}

	version := ""
	if tree.Version != "" {
		version = "@" + tree.Version
	}

	sb.WriteString(fmt.Sprintf("%s%s%s%s%s\n", prefix, connector, tree.Name, version, status))

	// Print children
	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	for i, child := range tree.Children {
		isLastChild := i == len(tree.Children)-1
		sb.WriteString(PrintTree(child, childPrefix, isLastChild))
	}

	return sb.String()
}
