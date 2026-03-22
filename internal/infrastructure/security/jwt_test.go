package security

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTokenService() *TokenService {
	return &TokenService{
		secretKey: []byte("test-secret-key-for-unit-tests"),
	}
}

func TestGenerateToken_CreatesValidToken(t *testing.T) {
	svc := newTestTokenService()

	token, err := svc.GenerateToken("user-123", []string{"admin"}, time.Hour)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken_SucceedsForValidToken(t *testing.T) {
	svc := newTestTokenService()

	token, err := svc.GenerateToken("user-123", []string{"admin"}, time.Hour)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)

	require.NoError(t, err)
	assert.NotNil(t, claims)
}

func TestValidateToken_FailsForExpiredToken(t *testing.T) {
	svc := newTestTokenService()

	token, err := svc.GenerateToken("user-123", []string{"admin"}, -1*time.Hour)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidateToken_FailsForInvalidTokenString(t *testing.T) {
	svc := newTestTokenService()

	claims, err := svc.ValidateToken("this-is-not-a-valid-jwt-token")

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_FailsForTamperedToken(t *testing.T) {
	svc := newTestTokenService()

	token, err := svc.GenerateToken("user-123", []string{"admin"}, time.Hour)
	require.NoError(t, err)

	// Tamper with the token by modifying a character
	tampered := token[:len(token)-5] + "XXXXX"

	claims, err := svc.ValidateToken(tampered)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_ExtractsCorrectUserIDAndRoles(t *testing.T) {
	svc := newTestTokenService()

	token, err := svc.GenerateToken("user-456", []string{"editor"}, time.Hour)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)

	require.NoError(t, err)
	assert.Equal(t, "user-456", claims.UserID)
	assert.Equal(t, []string{"editor"}, claims.Roles)
	assert.Equal(t, "flow-engine", claims.Issuer)
}

func TestGenerateToken_WithMultipleRolesPreservesAllRoles(t *testing.T) {
	svc := newTestTokenService()
	roles := []string{"admin", "editor", "viewer", "auditor"}

	token, err := svc.GenerateToken("user-789", roles, time.Hour)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)

	require.NoError(t, err)
	assert.Equal(t, roles, claims.Roles)
	assert.Len(t, claims.Roles, 4)
}

func TestValidateToken_FailsWithDifferentSecret(t *testing.T) {
	svc1 := &TokenService{secretKey: []byte("secret-one")}
	svc2 := &TokenService{secretKey: []byte("secret-two")}

	token, err := svc1.GenerateToken("user-123", []string{"admin"}, time.Hour)
	require.NoError(t, err)

	claims, err := svc2.ValidateToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_EmptyToken(t *testing.T) {
	svc := newTestTokenService()

	claims, err := svc.ValidateToken("")
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_MalformedToken(t *testing.T) {
	svc := newTestTokenService()

	// Just a random base64-looking string with wrong structure
	claims, err := svc.ValidateToken("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.payload")
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateToken_InvalidSigningMethod(t *testing.T) {
	svc := newTestTokenService()

	// Create a token with RSA algorithm header but HMAC signature (invalid)
	// This should be caught by the signing method check
	claims, err := svc.ValidateToken("eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.fakesignature")
	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestNewTokenService_UsesEnvVariable(t *testing.T) {
	t.Setenv("JWT_SECRET", "my-test-secret")
	svc := NewTokenService()

	// Generate and validate to confirm it works with env secret
	token, err := svc.GenerateToken("user-1", []string{"admin"}, time.Hour)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
}

func TestNewTokenService_FallbackSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	svc := NewTokenService()

	// Should still work with fallback secret
	token, err := svc.GenerateToken("user-1", []string{"viewer"}, time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", claims.UserID)
}

func TestGenerateToken_WithEmptyRoles(t *testing.T) {
	svc := newTestTokenService()

	token, err := svc.GenerateToken("user-1", []string{}, time.Hour)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Empty(t, claims.Roles)
}

func TestGenerateToken_WithNilRoles(t *testing.T) {
	svc := newTestTokenService()

	token, err := svc.GenerateToken("user-1", nil, time.Hour)
	require.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	require.NoError(t, err)
	assert.Nil(t, claims.Roles)
}
