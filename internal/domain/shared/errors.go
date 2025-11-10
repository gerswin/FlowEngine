package shared

import (
	"errors"
	"fmt"
)

// ErrorCode represents a domain error code.
type ErrorCode string

const (
	// ErrCodeNotFound indicates a resource was not found.
	ErrCodeNotFound ErrorCode = "NOT_FOUND"

	// ErrCodeInvalidInput indicates invalid input was provided.
	ErrCodeInvalidInput ErrorCode = "INVALID_INPUT"

	// ErrCodeConflict indicates a conflict occurred (e.g., optimistic lock failure).
	ErrCodeConflict ErrorCode = "CONFLICT"

	// ErrCodeInvalidState indicates an invalid state transition or operation.
	ErrCodeInvalidState ErrorCode = "INVALID_STATE"

	// ErrCodeAlreadyExists indicates a resource already exists.
	ErrCodeAlreadyExists ErrorCode = "ALREADY_EXISTS"

	// ErrCodeUnauthorized indicates unauthorized access.
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"

	// ErrCodeForbidden indicates forbidden access.
	ErrCodeForbidden ErrorCode = "FORBIDDEN"

	// ErrCodeInternal indicates an internal error.
	ErrCodeInternal ErrorCode = "INTERNAL"
)

// DomainError represents a domain-specific error with additional context.
type DomainError struct {
	code    ErrorCode
	message string
	cause   error
	context map[string]interface{}
}

// NewDomainError creates a new domain error.
func NewDomainError(code ErrorCode, message string, cause error) *DomainError {
	return &DomainError{
		code:    code,
		message: message,
		cause:   cause,
		context: make(map[string]interface{}),
	}
}

// Code returns the error code.
func (e *DomainError) Code() ErrorCode {
	return e.code
}

// Message returns the error message.
func (e *DomainError) Message() string {
	return e.message
}

// Cause returns the underlying cause of the error.
func (e *DomainError) Cause() error {
	return e.cause
}

// Context returns the error context.
func (e *DomainError) Context() map[string]interface{} {
	return e.context
}

// WithContext adds a context key-value pair to the error.
func (e *DomainError) WithContext(key string, value interface{}) *DomainError {
	e.context[key] = value
	return e
}

// Error implements the error interface.
func (e *DomainError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.code, e.message, e.cause)
	}
	return fmt.Sprintf("[%s] %s", e.code, e.message)
}

// Unwrap implements the errors.Unwrap interface.
func (e *DomainError) Unwrap() error {
	return e.cause
}

// Is implements the errors.Is interface for error comparison.
func (e *DomainError) Is(target error) bool {
	if target == nil {
		return false
	}

	var domainErr *DomainError
	if errors.As(target, &domainErr) {
		return e.code == domainErr.code
	}

	return false
}

// Common domain errors
var (
	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = NewDomainError(ErrCodeNotFound, "resource not found", nil)

	// ErrInvalidInput is returned when invalid input is provided.
	ErrInvalidInput = NewDomainError(ErrCodeInvalidInput, "invalid input", nil)

	// ErrConflict is returned when a conflict occurs.
	ErrConflict = NewDomainError(ErrCodeConflict, "conflict occurred", nil)

	// ErrInvalidState is returned when an invalid state operation is attempted.
	ErrInvalidState = NewDomainError(ErrCodeInvalidState, "invalid state", nil)

	// ErrAlreadyExists is returned when a resource already exists.
	ErrAlreadyExists = NewDomainError(ErrCodeAlreadyExists, "resource already exists", nil)

	// ErrUnauthorized is returned when access is unauthorized.
	ErrUnauthorized = NewDomainError(ErrCodeUnauthorized, "unauthorized access", nil)

	// ErrForbidden is returned when access is forbidden.
	ErrForbidden = NewDomainError(ErrCodeForbidden, "forbidden access", nil)

	// ErrInternal is returned when an internal error occurs.
	ErrInternal = NewDomainError(ErrCodeInternal, "internal error", nil)
)

// Helper functions to create specific errors

// NotFoundError creates a new not found error with a custom message.
func NotFoundError(message string) *DomainError {
	return NewDomainError(ErrCodeNotFound, message, nil)
}

// InvalidInputError creates a new invalid input error with a custom message.
func InvalidInputError(message string) *DomainError {
	return NewDomainError(ErrCodeInvalidInput, message, nil)
}

// ConflictError creates a new conflict error with a custom message.
func ConflictError(message string) *DomainError {
	return NewDomainError(ErrCodeConflict, message, nil)
}

// InvalidStateError creates a new invalid state error with a custom message.
func InvalidStateError(message string) *DomainError {
	return NewDomainError(ErrCodeInvalidState, message, nil)
}

// AlreadyExistsError creates a new already exists error with a custom message.
func AlreadyExistsError(message string) *DomainError {
	return NewDomainError(ErrCodeAlreadyExists, message, nil)
}

// UnauthorizedError creates a new unauthorized error with a custom message.
func UnauthorizedError(message string) *DomainError {
	return NewDomainError(ErrCodeUnauthorized, message, nil)
}

// ForbiddenError creates a new forbidden error with a custom message.
func ForbiddenError(message string) *DomainError {
	return NewDomainError(ErrCodeForbidden, message, nil)
}

// InternalError creates a new internal error with a custom message and cause.
func InternalError(message string, cause error) *DomainError {
	return NewDomainError(ErrCodeInternal, message, cause)
}

// IsNotFoundError checks if an error is a not found error.
func IsNotFoundError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.code == ErrCodeNotFound
	}
	return false
}

// IsInvalidInputError checks if an error is an invalid input error.
func IsInvalidInputError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.code == ErrCodeInvalidInput
	}
	return false
}

// IsConflictError checks if an error is a conflict error.
func IsConflictError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.code == ErrCodeConflict
	}
	return false
}

// IsInvalidStateError checks if an error is an invalid state error.
func IsInvalidStateError(err error) bool {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.code == ErrCodeInvalidState
	}
	return false
}
