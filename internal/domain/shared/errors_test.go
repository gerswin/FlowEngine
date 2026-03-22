package shared

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDomainError(t *testing.T) {
	err := NewDomainError(ErrCodeNotFound, "resource not found", nil)

	require.NotNil(t, err)
	assert.Equal(t, ErrCodeNotFound, err.Code())
	assert.Equal(t, "resource not found", err.Message())
	assert.Nil(t, err.Cause())
	assert.NotNil(t, err.Context())
	assert.Empty(t, err.Context())
}

func TestDomainError_ErrorWithoutCause(t *testing.T) {
	err := NewDomainError(ErrCodeNotFound, "user not found", nil)

	assert.Equal(t, "[NOT_FOUND] user not found", err.Error())
}

func TestDomainError_ErrorWithCause(t *testing.T) {
	cause := fmt.Errorf("database connection failed")
	err := NewDomainError(ErrCodeInternal, "internal error", cause)

	assert.Equal(t, "[INTERNAL] internal error: database connection failed", err.Error())
}

func TestDomainError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := NewDomainError(ErrCodeInternal, "wrapper", cause)

	assert.Equal(t, cause, err.Unwrap())
	assert.True(t, errors.Is(err, cause))
}

func TestDomainError_UnwrapNil(t *testing.T) {
	err := NewDomainError(ErrCodeNotFound, "no cause", nil)

	assert.Nil(t, err.Unwrap())
}

func TestDomainError_WithContext(t *testing.T) {
	err := NewDomainError(ErrCodeNotFound, "not found", nil).
		WithContext("entity", "workflow").
		WithContext("id", "123")

	ctx := err.Context()
	assert.Equal(t, "workflow", ctx["entity"])
	assert.Equal(t, "123", ctx["id"])
}

func TestDomainError_Is(t *testing.T) {
	t.Run("same code matches", func(t *testing.T) {
		err1 := NewDomainError(ErrCodeNotFound, "a", nil)
		err2 := NewDomainError(ErrCodeNotFound, "b", nil)

		assert.True(t, err1.Is(err2))
	})

	t.Run("different code does not match", func(t *testing.T) {
		err1 := NewDomainError(ErrCodeNotFound, "a", nil)
		err2 := NewDomainError(ErrCodeConflict, "b", nil)

		assert.False(t, err1.Is(err2))
	})

	t.Run("nil target returns false", func(t *testing.T) {
		err := NewDomainError(ErrCodeNotFound, "a", nil)

		assert.False(t, err.Is(nil))
	})

	t.Run("non-DomainError target returns false", func(t *testing.T) {
		err := NewDomainError(ErrCodeNotFound, "a", nil)

		assert.False(t, err.Is(fmt.Errorf("plain error")))
	})
}

// --- Helper function tests ---

func TestNotFoundError(t *testing.T) {
	err := NotFoundError("workflow not found")

	assert.Equal(t, ErrCodeNotFound, err.Code())
	assert.Equal(t, "workflow not found", err.Message())
	assert.Nil(t, err.Cause())
}

func TestInvalidInputError(t *testing.T) {
	err := InvalidInputError("name is required")

	assert.Equal(t, ErrCodeInvalidInput, err.Code())
	assert.Equal(t, "name is required", err.Message())
}

func TestConflictError(t *testing.T) {
	err := ConflictError("version mismatch")

	assert.Equal(t, ErrCodeConflict, err.Code())
	assert.Equal(t, "version mismatch", err.Message())
}

func TestInvalidStateError(t *testing.T) {
	err := InvalidStateError("cannot transition")

	assert.Equal(t, ErrCodeInvalidState, err.Code())
	assert.Equal(t, "cannot transition", err.Message())
}

func TestAlreadyExistsError(t *testing.T) {
	err := AlreadyExistsError("workflow already exists")

	assert.Equal(t, ErrCodeAlreadyExists, err.Code())
	assert.Equal(t, "workflow already exists", err.Message())
}

func TestUnauthorizedError(t *testing.T) {
	err := UnauthorizedError("invalid credentials")

	assert.Equal(t, ErrCodeUnauthorized, err.Code())
	assert.Equal(t, "invalid credentials", err.Message())
}

func TestForbiddenError(t *testing.T) {
	err := ForbiddenError("access denied")

	assert.Equal(t, ErrCodeForbidden, err.Code())
	assert.Equal(t, "access denied", err.Message())
}

func TestInternalError(t *testing.T) {
	cause := fmt.Errorf("db error")
	err := InternalError("something went wrong", cause)

	assert.Equal(t, ErrCodeInternal, err.Code())
	assert.Equal(t, "something went wrong", err.Message())
	assert.Equal(t, cause, err.Cause())
}

// --- Checker function tests ---

func TestIsNotFoundError(t *testing.T) {
	t.Run("true for not found", func(t *testing.T) {
		err := NotFoundError("missing")
		assert.True(t, IsNotFoundError(err))
	})

	t.Run("false for other code", func(t *testing.T) {
		err := ConflictError("conflict")
		assert.False(t, IsNotFoundError(err))
	})

	t.Run("false for plain error", func(t *testing.T) {
		err := fmt.Errorf("plain error")
		assert.False(t, IsNotFoundError(err))
	})

	t.Run("true for wrapped not found", func(t *testing.T) {
		inner := NotFoundError("missing")
		wrapped := fmt.Errorf("wrapped: %w", inner)
		assert.True(t, IsNotFoundError(wrapped))
	})
}

func TestIsInvalidInputError(t *testing.T) {
	assert.True(t, IsInvalidInputError(InvalidInputError("bad input")))
	assert.False(t, IsInvalidInputError(NotFoundError("missing")))
	assert.False(t, IsInvalidInputError(fmt.Errorf("plain")))
}

func TestIsConflictError(t *testing.T) {
	assert.True(t, IsConflictError(ConflictError("conflict")))
	assert.False(t, IsConflictError(NotFoundError("missing")))
	assert.False(t, IsConflictError(fmt.Errorf("plain")))
}

func TestIsInvalidStateError(t *testing.T) {
	assert.True(t, IsInvalidStateError(InvalidStateError("bad state")))
	assert.False(t, IsInvalidStateError(NotFoundError("missing")))
	assert.False(t, IsInvalidStateError(fmt.Errorf("plain")))
}

// --- Sentinel error tests ---

func TestSentinelErrors(t *testing.T) {
	assert.True(t, IsNotFoundError(ErrNotFound))
	assert.True(t, IsInvalidInputError(ErrInvalidInput))
	assert.True(t, IsConflictError(ErrConflict))
	assert.True(t, IsInvalidStateError(ErrInvalidState))

	assert.Equal(t, ErrCodeAlreadyExists, ErrAlreadyExists.Code())
	assert.Equal(t, ErrCodeUnauthorized, ErrUnauthorized.Code())
	assert.Equal(t, ErrCodeForbidden, ErrForbidden.Code())
	assert.Equal(t, ErrCodeInternal, ErrInternal.Code())
}
