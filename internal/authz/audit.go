package authz

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// AuditLogger defines the interface for logging authorization decisions.
//
// Implementations can write to files, databases, or external audit services
// (e.g., AWS CloudTrail, Splunk, Datadog).
type AuditLogger interface {
	// LogDecision logs an authorization decision.
	LogDecision(ctx context.Context, entry *AuditEntry) error

	// Close closes the audit logger and flushes any buffered entries.
	Close() error
}

// AuditEntry represents a single authorization decision event.
type AuditEntry struct {
	// Timestamp when the decision was made
	Timestamp time.Time `json:"timestamp"`

	// Decision outcome
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`

	// Subject (who)
	UserID         string `json:"user_id"`
	Email          string `json:"email,omitempty"`
	OrganizationID string `json:"organization_id"`
	Role           string `json:"role"`

	// Action (what)
	Action string `json:"action"`

	// Resource (where)
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id,omitempty"`

	// Context (how/when)
	Environment map[string]interface{} `json:"environment,omitempty"`

	// Policy details
	PolicyIDs []string `json:"policy_ids,omitempty"`

	// Request metadata
	RequestID  string        `json:"request_id,omitempty"`
	Duration   time.Duration `json:"duration_ms,omitempty"`
	ErrorMsg   string        `json:"error,omitempty"`
}

// AuditLoggerConfig holds configuration for audit logging.
type AuditLoggerConfig struct {
	// Writer is the output destination for audit logs.
	// Defaults to os.Stdout if not specified.
	Writer io.Writer

	// LogAllDecisions determines whether to log all decisions or only denials.
	// Defaults to false (log only denials).
	LogAllDecisions bool

	// IncludeEnvironment determines whether to include environment attributes.
	// Defaults to true.
	IncludeEnvironment bool

	// BufferSize is the size of the buffered channel for async logging.
	// Defaults to 1000. Set to 0 for synchronous logging.
	BufferSize int
}

// DefaultAuditLogger provides a JSON-formatted audit logger.
type DefaultAuditLogger struct {
	writer             io.Writer
	logAllDecisions    bool
	includeEnvironment bool

	// Async logging support
	entryChan chan *AuditEntry
	wg        sync.WaitGroup
	closed    chan struct{}
	mu        sync.Mutex
}

// NewDefaultAuditLogger creates a new default audit logger.
func NewDefaultAuditLogger(cfg AuditLoggerConfig) *DefaultAuditLogger {
	if cfg.Writer == nil {
		cfg.Writer = os.Stdout
	}

	if cfg.BufferSize == 0 {
		cfg.BufferSize = 1000
	}

	logger := &DefaultAuditLogger{
		writer:             cfg.Writer,
		logAllDecisions:    cfg.LogAllDecisions,
		includeEnvironment: cfg.IncludeEnvironment,
		entryChan:          make(chan *AuditEntry, cfg.BufferSize),
		closed:             make(chan struct{}),
	}

	// Start async logging goroutine
	logger.wg.Add(1)
	go logger.processEntries()

	return logger
}

// LogDecision logs an authorization decision asynchronously.
func (l *DefaultAuditLogger) LogDecision(ctx context.Context, entry *AuditEntry) error {
	// Filter based on configuration
	if !l.logAllDecisions && entry.Allowed {
		// Only log denials
		return nil
	}

	// Filter environment if configured
	if !l.includeEnvironment {
		entry.Environment = nil
	}

	select {
	case l.entryChan <- entry:
		return nil
	case <-l.closed:
		return io.ErrClosedPipe
	default:
		// Buffer full - log synchronously to avoid blocking
		return l.writeEntry(entry)
	}
}

// processEntries processes audit entries asynchronously.
func (l *DefaultAuditLogger) processEntries() {
	defer l.wg.Done()

	for {
		select {
		case entry := <-l.entryChan:
			if err := l.writeEntry(entry); err != nil {
				log.Printf("audit: failed to write entry: %v", err)
			}
		case <-l.closed:
			// Drain remaining entries
			for {
				select {
				case entry := <-l.entryChan:
					if err := l.writeEntry(entry); err != nil {
						log.Printf("audit: failed to write entry during shutdown: %v", err)
					}
				default:
					return
				}
			}
		}
	}
}

// writeEntry writes a single audit entry as JSON.
func (l *DefaultAuditLogger) writeEntry(entry *AuditEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	encoder := json.NewEncoder(l.writer)
	return encoder.Encode(entry)
}

// Close closes the audit logger and flushes buffered entries.
func (l *DefaultAuditLogger) Close() error {
	close(l.closed)
	l.wg.Wait()
	close(l.entryChan)
	return nil
}

// NewAuditEntry creates an audit entry from an authorization request and decision.
func NewAuditEntry(req *AuthorizationRequest, decision *Decision, duration time.Duration) *AuditEntry {
	entry := &AuditEntry{
		Timestamp:      decision.Timestamp,
		Allowed:        decision.Allowed,
		Reason:         decision.Reason,
		Action:         req.Action,
		ResourceType:   req.Resource.Type,
		ResourceID:     req.Resource.ID,
		PolicyIDs:      decision.PolicyIDs,
		Duration:       duration,
	}

	// Extract subject information
	if req.Subject != nil {
		entry.UserID = req.Subject.UserID
		entry.Email = req.Subject.Email
		entry.OrganizationID = req.Subject.OrganizationID
		entry.Role = req.Subject.OrganizationRole
	}

	// Copy environment attributes
	if len(req.Environment) > 0 {
		entry.Environment = make(map[string]interface{})
		for k, v := range req.Environment {
			entry.Environment[k] = v
		}
	}

	return entry
}

// NoOpAuditLogger is an audit logger that does nothing.
// Useful for testing or when audit logging is disabled.
type NoOpAuditLogger struct{}

// NewNoOpAuditLogger creates a no-op audit logger.
func NewNoOpAuditLogger() *NoOpAuditLogger {
	return &NoOpAuditLogger{}
}

// LogDecision does nothing.
func (l *NoOpAuditLogger) LogDecision(ctx context.Context, entry *AuditEntry) error {
	return nil
}

// Close does nothing.
func (l *NoOpAuditLogger) Close() error {
	return nil
}

// InMemoryAuditLogger stores audit entries in memory for testing.
type InMemoryAuditLogger struct {
	mu      sync.RWMutex
	entries []*AuditEntry
}

// NewInMemoryAuditLogger creates an in-memory audit logger.
func NewInMemoryAuditLogger() *InMemoryAuditLogger {
	return &InMemoryAuditLogger{
		entries: make([]*AuditEntry, 0),
	}
}

// LogDecision stores the audit entry in memory.
func (l *InMemoryAuditLogger) LogDecision(ctx context.Context, entry *AuditEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Create a copy to avoid external mutations
	entryCopy := *entry
	if entry.Environment != nil {
		entryCopy.Environment = make(map[string]interface{})
		for k, v := range entry.Environment {
			entryCopy.Environment[k] = v
		}
	}
	if entry.PolicyIDs != nil {
		entryCopy.PolicyIDs = make([]string, len(entry.PolicyIDs))
		copy(entryCopy.PolicyIDs, entry.PolicyIDs)
	}

	l.entries = append(l.entries, &entryCopy)
	return nil
}

// GetEntries returns all stored audit entries.
func (l *InMemoryAuditLogger) GetEntries() []*AuditEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make([]*AuditEntry, len(l.entries))
	copy(result, l.entries)
	return result
}

// Clear clears all stored entries.
func (l *InMemoryAuditLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries = make([]*AuditEntry, 0)
}

// Close does nothing.
func (l *InMemoryAuditLogger) Close() error {
	return nil
}

// WithAuditLogger adds audit logging to an Engine.
func WithAuditLogger(engine *Engine, logger AuditLogger) *Engine {
	engine.auditLogger = logger
	return engine
}

// Update Engine struct to include audit logger (this will be added to authz.go)
