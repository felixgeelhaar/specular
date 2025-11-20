package auth

import (
	"fmt"
)

// Error codes for authentication failures
const (
	// Authentication errors
	ErrInvalidCredentials = "AUTH_INVALID_CREDENTIALS"
	ErrProviderUnavailable = "AUTH_PROVIDER_UNAVAILABLE"
	ErrNoProviders        = "AUTH_NO_PROVIDERS"
	ErrDuplicateProvider  = "AUTH_DUPLICATE_PROVIDER"
	ErrProviderNotFound   = "AUTH_PROVIDER_NOT_FOUND"

	// Session errors
	ErrSessionExpired     = "AUTH_SESSION_EXPIRED"
	ErrSessionInvalid     = "AUTH_SESSION_INVALID"
	ErrSessionNotFound    = "AUTH_SESSION_NOT_FOUND"
	ErrSessionStoreFailed = "AUTH_SESSION_STORE_FAILED"
	ErrRefreshFailed      = "AUTH_REFRESH_FAILED"

	// Token errors
	ErrTokenInvalid       = "AUTH_TOKEN_INVALID"
	ErrTokenExpired       = "AUTH_TOKEN_EXPIRED"
	ErrTokenMalformed     = "AUTH_TOKEN_MALFORMED"
	ErrTokenSigningFailed = "AUTH_TOKEN_SIGNING_FAILED"

	// SAML errors
	ErrSAMLAssertionInvalid = "AUTH_SAML_ASSERTION_INVALID"
	ErrSAMLSignatureInvalid = "AUTH_SAML_SIGNATURE_INVALID"
	ErrSAMLMetadataFailed   = "AUTH_SAML_METADATA_FAILED"

	// OIDC errors
	ErrOIDCAuthorizationFailed = "AUTH_OIDC_AUTHORIZATION_FAILED"
	ErrOIDCTokenExchangeFailed = "AUTH_OIDC_TOKEN_EXCHANGE_FAILED"
	ErrOIDCIDTokenInvalid      = "AUTH_OIDC_ID_TOKEN_INVALID"
)

// AuthError represents an authentication error with code and context.
type AuthError struct {
	// Code is the error code (e.g., AUTH_SESSION_EXPIRED)
	Code string

	// Message is a human-readable error message
	Message string

	// Context provides additional details about the error
	Context map[string]interface{}

	// Cause is the underlying error that caused this error
	Cause error
}

// Error implements the error interface.
func (e *AuthError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *AuthError) Unwrap() error {
	return e.Cause
}

// NewError creates a new AuthError.
func NewError(code, message string, context map[string]interface{}) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
		Context: context,
	}
}

// WrapError wraps an existing error with an AuthError.
func WrapError(code, message string, cause error, context map[string]interface{}) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
		Context: context,
		Cause:   cause,
	}
}

// IsAuthError checks if an error is an AuthError with the given code.
func IsAuthError(err error, code string) bool {
	if authErr, ok := err.(*AuthError); ok {
		return authErr.Code == code
	}
	return false
}
