package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/infrastructure/security"
	"github.com/LaFabric-LinkTIC/FlowEngine/pkg/jsonapi"
)

const (
	ContextUserIDKey = "userID"
	ContextRolesKey  = "roles"
)

// AuthMiddleware creates a Gin middleware for JWT authentication.
func AuthMiddleware(tokenService *security.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			jsonapi.ErrorResponse(c, http.StatusUnauthorized, []*jsonapi.Error{
				jsonapi.NewError(http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", "Missing Authorization header"),
			})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			jsonapi.ErrorResponse(c, http.StatusUnauthorized, []*jsonapi.Error{
				jsonapi.NewError(http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token format", "Format must be 'Bearer <token>'"),
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := tokenService.ValidateToken(tokenString)
		if err != nil {
			jsonapi.ErrorResponse(c, http.StatusUnauthorized, []*jsonapi.Error{
				jsonapi.NewError(http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token", err.Error()),
			})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextRolesKey, claims.Roles)

		c.Next()
	}
}
