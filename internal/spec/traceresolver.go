package spec

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type TraceResolver struct {
	root      string
	docIndex  map[string][]string
	testIndex map[string][]string
	apiIndex  map[string][]string
}

var (
	docDirs  = []string{"docs", "documentation", "design", "specs"}
	testDirs = []string{"internal", "pkg", "cmd", "src", "tests", "apps", "services", "backend", "frontend", "lib", "web"}
	apiDirs  = []string{".specular/openapi", "openapi", "api", "apis", "docs/api"}

	docExtensions = map[string]struct{}{
		".md":   {},
		".mdx":  {},
		".rst":  {},
		".adoc": {},
	}
	testExtensions = map[string]struct{}{
		".go":   {},
		".ts":   {},
		".tsx":  {},
		".js":   {},
		".jsx":  {},
		".py":   {},
		".rb":   {},
		".java": {},
		".cs":   {},
		".rs":   {},
		".php":  {},
	}
	apiExtensions = map[string]struct{}{
		".yaml": {},
		".yml":  {},
		".json": {},
	}
	skipDirNames = map[string]struct{}{
		".git":         {},
		"node_modules": {},
		"dist":         {},
		"build":        {},
		"out":          {},
		"vendor":       {},
		".idea":        {},
		".vscode":      {},
		".venv":        {},
		"env":          {},
		"target":       {},
		"coverage":     {},
		"bin":          {},
	}
)

var (
	resolverCacheMu sync.RWMutex
	resolverCache   = make(map[string]*TraceResolver)
)

// NewTraceResolver builds indexes of existing artifacts to map features to real traces.
func NewTraceResolver(workspace string) *TraceResolver {
	ws := strings.TrimSpace(workspace)
	if ws == "" {
		ws = "."
	}
	root, err := filepath.Abs(ws)
	if err != nil {
		return nil
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil
	}

	resolver := &TraceResolver{
		root:      root,
		docIndex:  make(map[string][]string),
		testIndex: make(map[string][]string),
		apiIndex:  make(map[string][]string),
	}

	resolver.indexDocs()
	resolver.indexTests()
	resolver.indexAPIs()

	return resolver
}

// EnhanceTraceArtifacts enriches traces using a resolver built from the workspace.
func EnhanceTraceArtifacts(spec *ProductSpec, workspace string) {
	if spec == nil {
		return
	}
	resolver := getTraceResolver(workspace)
	if resolver == nil {
		return
	}
	resolver.EnhanceSpec(spec)
}

// EnhanceSpec updates all features within the spec.
func (r *TraceResolver) EnhanceSpec(spec *ProductSpec) {
	if r == nil || spec == nil {
		return
	}
	for i := range spec.Features {
		r.enhanceFeature(&spec.Features[i])
	}
}

func (r *TraceResolver) enhanceFeature(feature *Feature) {
	if feature == nil {
		return
	}
	slugs := featureSlugs(*feature)
	if len(slugs) == 0 {
		return
	}

	if doc := r.findDoc(slugs); doc != "" {
		placeholder := filepath.ToSlash(filepath.Clean(filepath.Join("docs/features", feature.ID.String()+".md")))
		feature.Trace = removeEntries(feature.Trace, placeholder)
		feature.Trace = appendUnique(feature.Trace, doc)
	}

	if tests := r.findTests(slugs); len(tests) > 0 {
		placeholder := filepath.ToSlash(filepath.Clean(filepath.Join(".specular/tests", feature.ID.String()+"_test.go")))
		feature.Trace = removeEntries(feature.Trace, placeholder)
		feature.Trace = appendUnique(feature.Trace, tests...)
	}

	if len(feature.API) > 0 {
		if api := r.findAPI(slugs); api != "" {
			placeholder := filepath.ToSlash(filepath.Clean(filepath.Join(".specular/openapi", feature.ID.String()+".yaml")))
			feature.Trace = removeEntries(feature.Trace, placeholder)
			feature.Trace = appendUnique(feature.Trace, api)
		}
	}
}

func (r *TraceResolver) indexDocs() {
	for _, dir := range docDirs {
		r.walkDir(dir, r.addDoc)
	}
}

func (r *TraceResolver) indexTests() {
	for _, dir := range testDirs {
		r.walkDir(dir, r.addTest)
	}
}

func (r *TraceResolver) indexAPIs() {
	for _, dir := range apiDirs {
		r.walkDir(dir, r.addAPI)
	}
}

func (r *TraceResolver) walkDir(rel string, handler func(string, fs.DirEntry)) {
	if rel == "" {
		return
	}
	root := filepath.Join(r.root, rel)
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return
	}
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if _, skip := skipDirNames[strings.ToLower(d.Name())]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		handler(path, d)
		return nil
	})
}

func (r *TraceResolver) addDoc(path string, _ fs.DirEntry) {
	if !hasExtension(path, docExtensions) {
		return
	}
	slug := slugify(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if slug == "" {
		return
	}
	rel := r.rel(path)
	r.docIndex[slug] = appendUnique(r.docIndex[slug], rel)
}

func (r *TraceResolver) addTest(path string, _ fs.DirEntry) {
	if !hasExtension(path, testExtensions) {
		return
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	slug := slugify(trimTestAffixes(base))
	if slug == "" {
		return
	}
	rel := r.rel(path)
	r.testIndex[slug] = appendUnique(r.testIndex[slug], rel)
}

func (r *TraceResolver) addAPI(path string, _ fs.DirEntry) {
	if !hasExtension(path, apiExtensions) {
		return
	}
	slug := slugify(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if slug == "" {
		return
	}
	rel := r.rel(path)
	r.apiIndex[slug] = appendUnique(r.apiIndex[slug], rel)
}

func (r *TraceResolver) findDoc(slugs []string) string {
	for _, slug := range slugs {
		if paths, ok := r.docIndex[slug]; ok && len(paths) > 0 {
			copyPaths := append([]string(nil), paths...)
			sort.Strings(copyPaths)
			return copyPaths[0]
		}
	}
	return ""
}

func (r *TraceResolver) findTests(slugs []string) []string {
	seen := make(map[string]struct{})
	var results []string
	for _, slug := range slugs {
		paths, ok := r.testIndex[slug]
		if !ok {
			continue
		}
		copyPaths := append([]string(nil), paths...)
		sort.Strings(copyPaths)
		for _, path := range copyPaths {
			if _, exists := seen[path]; exists {
				continue
			}
			seen[path] = struct{}{}
			results = append(results, path)
		}
	}
	return results
}

func (r *TraceResolver) findAPI(slugs []string) string {
	for _, slug := range slugs {
		if paths, ok := r.apiIndex[slug]; ok && len(paths) > 0 {
			copyPaths := append([]string(nil), paths...)
			sort.Strings(copyPaths)
			return copyPaths[0]
		}
	}
	return ""
}

func (r *TraceResolver) rel(path string) string {
	rel, err := filepath.Rel(r.root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func featureSlugs(feature Feature) []string {
	var slugs []string
	if id := slugify(feature.ID.String()); id != "" {
		slugs = appendUnique(slugs, id)
	}
	if title := slugify(feature.Title); title != "" {
		slugs = appendUnique(slugs, title)
	}
	return slugs
}

func slugify(value string) string {
	if value == "" {
		return ""
	}
	var builder strings.Builder
	previousHyphen := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			previousHyphen = false
			continue
		}
		if previousHyphen {
			continue
		}
		builder.WriteRune('-')
		previousHyphen = true
	}
	slug := strings.Trim(builder.String(), "-")
	return slug
}

func trimTestAffixes(value string) string {
	lower := strings.ToLower(value)
	for _, suffix := range []string{"_test", "_tests", "-test", "-tests", ".test", ".tests", ".spec", ".specs", "test", "tests"} {
		if strings.HasSuffix(lower, suffix) {
			lower = strings.TrimSuffix(lower, suffix)
			break
		}
	}
	for _, prefix := range []string{"test_", "tests_", "spec_", "specs_"} {
		if strings.HasPrefix(lower, prefix) {
			lower = strings.TrimPrefix(lower, prefix)
		}
	}
	return lower
}

func hasExtension(path string, allowed map[string]struct{}) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := allowed[ext]
	return ok
}

func getTraceResolver(workspace string) *TraceResolver {
	ws := strings.TrimSpace(workspace)
	if ws == "" {
		ws = "."
	}

	abs, err := filepath.Abs(ws)
	if err != nil {
		return nil
	}

	resolverCacheMu.RLock()
	if resolver, ok := resolverCache[abs]; ok && resolver != nil {
		resolverCacheMu.RUnlock()
		return resolver
	}
	resolverCacheMu.RUnlock()

	resolver := NewTraceResolver(abs)
	if resolver == nil {
		return nil
	}

	resolverCacheMu.Lock()
	resolverCache[abs] = resolver
	resolverCacheMu.Unlock()
	return resolver
}
