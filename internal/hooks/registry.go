package hooks

import (
	"context"
	"fmt"
	"sync"
)

// Registry manages hooks and their lifecycle
type Registry struct {
	mu sync.RWMutex

	// hooks maps event types to registered hooks
	hooks map[EventType][]Hook

	// factories maps hook types to their factory functions
	factories map[string]HookFactory

	// executor executes hooks
	executor *Executor
}

// NewRegistry creates a new hook registry
func NewRegistry() *Registry {
	return &Registry{
		hooks:     make(map[EventType][]Hook),
		factories: make(map[string]HookFactory),
		executor:  NewExecutor(),
	}
}

// RegisterFactory registers a hook factory
func (r *Registry) RegisterFactory(hookType string, factory HookFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.factories[hookType] = factory
}

// Register adds a hook to the registry
func (r *Registry) Register(hook Hook) error {
	if hook == nil {
		return fmt.Errorf("hook cannot be nil")
	}

	if !hook.Enabled() {
		// Don't register disabled hooks
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Register hook for each event type it handles
	for _, eventType := range hook.EventTypes() {
		r.hooks[eventType] = append(r.hooks[eventType], hook)
	}

	return nil
}

// RegisterFromConfig creates and registers a hook from configuration
func (r *Registry) RegisterFromConfig(config *HookConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if !config.Enabled {
		// Skip disabled hooks
		return nil
	}

	// Get factory for this hook type
	r.mu.RLock()
	factory, exists := r.factories[config.Type]
	r.mu.RUnlock()

	if !exists {
		return fmt.Errorf("unknown hook type: %s", config.Type)
	}

	// Create hook using factory
	hook, err := factory(config)
	if err != nil {
		return fmt.Errorf("failed to create hook %s: %w", config.Name, err)
	}

	// Register the hook
	return r.Register(hook)
}

// Unregister removes a hook from the registry
func (r *Registry) Unregister(hookName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove hook from all event types
	for eventType, hooks := range r.hooks {
		filtered := make([]Hook, 0, len(hooks))
		for _, hook := range hooks {
			if hook.Name() != hookName {
				filtered = append(filtered, hook)
			}
		}
		r.hooks[eventType] = filtered
	}
}

// Trigger executes all hooks registered for an event type
func (r *Registry) Trigger(ctx context.Context, event *Event) []ExecutionResult {
	r.mu.RLock()
	hooks := r.hooks[event.Type]
	r.mu.RUnlock()

	if len(hooks) == 0 {
		return nil
	}

	// Execute all hooks for this event
	return r.executor.ExecuteAll(ctx, hooks, event)
}

// GetHooks returns all hooks for an event type
func (r *Registry) GetHooks(eventType EventType) []Hook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hooks := r.hooks[eventType]
	// Return a copy to prevent external modification
	result := make([]Hook, len(hooks))
	copy(result, hooks)
	return result
}

// GetAllHooks returns all registered hooks
func (r *Registry) GetAllHooks() map[EventType][]Hook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a deep copy
	result := make(map[EventType][]Hook)
	for eventType, hooks := range r.hooks {
		hooksCopy := make([]Hook, len(hooks))
		copy(hooksCopy, hooks)
		result[eventType] = hooksCopy
	}
	return result
}

// Count returns the total number of registered hooks
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Count unique hooks (a hook may be registered for multiple events)
	seen := make(map[string]bool)
	for _, hooks := range r.hooks {
		for _, hook := range hooks {
			seen[hook.Name()] = true
		}
	}
	return len(seen)
}

// Clear removes all hooks from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.hooks = make(map[EventType][]Hook)
}

// HasHooksFor checks if there are any hooks registered for an event type
func (r *Registry) HasHooksFor(eventType EventType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hooks, exists := r.hooks[eventType]
	return exists && len(hooks) > 0
}
