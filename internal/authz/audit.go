package authz

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	RequestID string        `json:"request_id,omitempty"`
	Duration  time.Duration `json:"duration_ms,omitempty"`
	ErrorMsg  string        `json:"error,omitempty"`

	// Cryptographic signature (ECDSA P-256)
	// These fields provide tamper-proof audit trails by signing the entry
	Signature string `json:"signature,omitempty"` // Base64-encoded ECDSA signature
	PublicKey string `json:"public_key,omitempty"` // Base64-encoded public key for verification
	SignedBy  string `json:"signed_by,omitempty"`  // Identity/email of the signer
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
		Timestamp:    decision.Timestamp,
		Allowed:      decision.Allowed,
		Reason:       decision.Reason,
		Action:       req.Action,
		ResourceType: req.Resource.Type,
		ResourceID:   req.Resource.ID,
		PolicyIDs:    decision.PolicyIDs,
		Duration:     duration,
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

// Signer defines the interface for signing audit entries.
type Signer interface {
	// Sign generates a signature for the data.
	Sign(data []byte) (signature []byte, publicKey []byte, err error)

	// Identity returns the identity of the signer.
	Identity() string
}

// SignedAuditLogger wraps an AuditLogger and adds cryptographic signatures
// to each audit entry using ECDSA P-256.
type SignedAuditLogger struct {
	wrapped AuditLogger
	signer  Signer
}

// NewSignedAuditLogger creates a new signed audit logger that wraps
// an existing logger and signs each entry.
func NewSignedAuditLogger(wrapped AuditLogger, signer Signer) *SignedAuditLogger {
	return &SignedAuditLogger{
		wrapped: wrapped,
		signer:  signer,
	}
}

// LogDecision signs the audit entry and then logs it using the wrapped logger.
func (l *SignedAuditLogger) LogDecision(ctx context.Context, entry *AuditEntry) error {
	// Sign the entry
	if err := l.signEntry(entry); err != nil {
		// If signing fails, log error but continue with unsigned entry
		// to ensure audit trail is not lost
		log.Printf("audit: failed to sign entry: %v", err)
	}

	// Pass to wrapped logger
	return l.wrapped.LogDecision(ctx, entry)
}

// Close closes both the signed logger and the wrapped logger.
func (l *SignedAuditLogger) Close() error {
	return l.wrapped.Close()
}

// signEntry signs an audit entry by computing a signature over its canonical JSON.
func (l *SignedAuditLogger) signEntry(entry *AuditEntry) error {
	// Clear signature fields to create canonical data
	entry.Signature = ""
	entry.PublicKey = ""
	entry.SignedBy = ""

	// Serialize to canonical JSON
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	// Sign the data
	signature, publicKey, err := l.signer.Sign(data)
	if err != nil {
		return fmt.Errorf("failed to sign: %w", err)
	}

	// Encode signature and public key as base64
	entry.Signature = encodeBase64(signature)
	entry.PublicKey = encodeBase64(publicKey)
	entry.SignedBy = l.signer.Identity()

	return nil
}

// encodeBase64 encodes bytes to base64 string.
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// decodeBase64 decodes a base64 string to bytes.
func decodeBase64(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// ECDSASigner defines the interface for ECDSA signing compatible with attestation.EphemeralSigner.
type ECDSASigner interface {
	// Sign generates a signature for the data.
	Sign(data []byte) (signature []byte, publicKey []byte, err error)

	// Identity returns the identity of the signer.
	Identity() string
}

// SignerAdapter adapts an attestation.EphemeralSigner to the Signer interface.
type SignerAdapter struct {
	identity   string
	signerFunc func([]byte) ([]byte, []byte, error)
}

// NewSignerAdapter creates a new signer adapter.
func NewSignerAdapter(identity string, signerFunc func([]byte) ([]byte, []byte, error)) *SignerAdapter {
	return &SignerAdapter{
		identity:   identity,
		signerFunc: signerFunc,
	}
}

// Sign generates a signature for the data.
func (a *SignerAdapter) Sign(data []byte) (signature []byte, publicKey []byte, err error) {
	return a.signerFunc(data)
}

// Identity returns the identity of the signer.
func (a *SignerAdapter) Identity() string {
	return a.identity
}

// Update Engine struct to include audit logger (this will be added to authz.go)
