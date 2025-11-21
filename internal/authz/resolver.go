package authz

import (
	"context"
	"sync"
	"time"

	"github.com/felixgeelhaar/specular/internal/auth"
)

// DefaultAttributeResolver provides a basic attribute resolver implementation.
type DefaultAttributeResolver struct {
	resourceStore ResourceStore
	cache         *attributeCache
}

// ResourceStore defines an interface for fetching resource attributes.
type ResourceStore interface {
	GetResourceAttributes(ctx context.Context, resourceType, resourceID string) (Attributes, error)
}

// NewDefaultAttributeResolver creates a new attribute resolver.
func NewDefaultAttributeResolver(resourceStore ResourceStore) *DefaultAttributeResolver {
	return &DefaultAttributeResolver{
		resourceStore: resourceStore,
		cache:         newAttributeCache(1 * time.Minute), // 1min TTL
	}
}

// GetSubjectAttributes extracts attributes from the authenticated session.
func (r *DefaultAttributeResolver) GetSubjectAttributes(ctx context.Context, subject *auth.Session) (Attributes, error) {
	if subject == nil {
		return make(Attributes), nil
	}

	attrs := Attributes{
		"user_id":           subject.UserID,
		"email":             subject.Email,
		"provider":          subject.Provider,
		"organization_id":   subject.OrganizationID,
		"organization_role": subject.OrganizationRole,
		"role":              subject.OrganizationRole, // Alias for easier policy writing
	}

	// Add team attributes if present
	if subject.TeamID != nil {
		attrs["team_id"] = *subject.TeamID
	}
	if subject.TeamRole != nil {
		attrs["team_role"] = *subject.TeamRole
	}

	// Add custom attributes
	for key, val := range subject.Attributes {
		attrs[key] = val
	}

	return attrs, nil
}

// GetResourceAttributes fetches attributes for a specific resource.
func (r *DefaultAttributeResolver) GetResourceAttributes(ctx context.Context, resourceType, resourceID string) (Attributes, error) {
	if resourceID == "" {
		// No specific resource, return type-level attributes
		return Attributes{
			"type": resourceType,
		}, nil
	}

	// Check cache
	cacheKey := resourceType + ":" + resourceID
	if cached, ok := r.cache.Get(cacheKey); ok {
		return cached, nil
	}

	// Fetch from store
	attrs, err := r.resourceStore.GetResourceAttributes(ctx, resourceType, resourceID)
	if err != nil {
		return nil, err
	}

	// Add resource metadata
	attrs["type"] = resourceType
	attrs["id"] = resourceID

	// Cache the attributes
	r.cache.Set(cacheKey, attrs)

	return attrs, nil
}

// InvalidateResourceCache invalidates cached attributes for a resource.
func (r *DefaultAttributeResolver) InvalidateResourceCache(resourceType, resourceID string) {
	cacheKey := resourceType + ":" + resourceID
	r.cache.Delete(cacheKey)
}

// attributeCache provides a simple in-memory cache with TTL.
type attributeCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	attrs     Attributes
	expiresAt time.Time
}

func newAttributeCache(ttl time.Duration) *attributeCache {
	cache := &attributeCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

func (c *attributeCache) Get(key string) (Attributes, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	// Check expiration
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.attrs, true
}

func (c *attributeCache) Set(key string, attrs Attributes) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cacheEntry{
		attrs:     attrs,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *attributeCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

func (c *attributeCache) cleanup() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.After(entry.expiresAt) {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}

// InMemoryResourceStore provides a simple in-memory resource store for testing.
type InMemoryResourceStore struct {
	mu        sync.RWMutex
	resources map[string]Attributes // key: "type:id"
}

// NewInMemoryResourceStore creates a new in-memory resource store.
func NewInMemoryResourceStore() *InMemoryResourceStore {
	return &InMemoryResourceStore{
		resources: make(map[string]Attributes),
	}
}

// SetResourceAttributes sets attributes for a resource.
func (s *InMemoryResourceStore) SetResourceAttributes(resourceType, resourceID string, attrs Attributes) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceType + ":" + resourceID
	s.resources[key] = attrs
}

// GetResourceAttributes retrieves attributes for a resource.
func (s *InMemoryResourceStore) GetResourceAttributes(ctx context.Context, resourceType, resourceID string) (Attributes, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := resourceType + ":" + resourceID
	attrs, ok := s.resources[key]
	if !ok {
		// Return empty attributes if not found
		return Attributes{
			"type": resourceType,
			"id":   resourceID,
		}, nil
	}

	// Return a copy to avoid external mutations
	result := make(Attributes)
	for k, v := range attrs {
		result[k] = v
	}

	return result, nil
}

// DeleteResource removes a resource from the store.
func (s *InMemoryResourceStore) DeleteResource(resourceType, resourceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceType + ":" + resourceID
	delete(s.resources, key)
}
