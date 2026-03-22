package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/actor"
	"github.com/LaFabric-LinkTIC/FlowEngine/pkg/jsonapi"
)

// RequirePermission creates middleware that checks if the user has the required permission.
func RequirePermission(perm actor.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get(ContextRolesKey)
		if !exists {
			jsonapi.ErrorResponse(c, http.StatusForbidden, []*jsonapi.Error{
				jsonapi.NewError(http.StatusForbidden, "FORBIDDEN", "Access denied", "No roles found in token"),
			})
			c.Abort()
			return
		}

		userRoles, ok := roles.([]string)
		if !ok {
			jsonapi.ErrorResponse(c, http.StatusForbidden, []*jsonapi.Error{
				jsonapi.NewError(http.StatusForbidden, "FORBIDDEN", "Access denied", "Invalid roles format"),
			})
			c.Abort()
			return
		}

		if !actor.HasAnyPermission(userRoles, perm) {
			jsonapi.ErrorResponse(c, http.StatusForbidden, []*jsonapi.Error{
				jsonapi.NewError(http.StatusForbidden, "FORBIDDEN", "Access denied",
					"Required permission: "+string(perm)),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
