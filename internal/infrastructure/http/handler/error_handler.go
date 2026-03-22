package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
	"github.com/LaFabric-LinkTIC/FlowEngine/pkg/jsonapi"
)

// handleDomainError converts domain errors to JSON:API error responses.
// It inspects the error for a *shared.DomainError and maps its code to an
// HTTP status. Non-domain errors fall back to 500 Internal Server Error.
func handleDomainError(c *gin.Context, err error) {
	var domainErr *shared.DomainError
	if errors.As(err, &domainErr) {
		status := domainErrorToHTTPStatus(domainErr)
		jsonapi.ErrorResponse(c, status, []*jsonapi.Error{
			jsonapi.NewError(status, string(domainErr.Code()), domainErr.Message(), domainErr.Error()),
		})
		return
	}

	// Generic / infrastructure error
	jsonapi.ErrorResponse(c, http.StatusInternalServerError, []*jsonapi.Error{
		jsonapi.NewError(http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", err.Error()),
	})
}

// domainErrorToHTTPStatus maps domain error codes to HTTP status codes.
func domainErrorToHTTPStatus(err *shared.DomainError) int {
	switch err.Code() {
	case shared.ErrCodeNotFound:
		return http.StatusNotFound
	case shared.ErrCodeInvalidInput:
		return http.StatusBadRequest
	case shared.ErrCodeInvalidState:
		return http.StatusConflict
	case shared.ErrCodeConflict:
		return http.StatusConflict
	case shared.ErrCodeAlreadyExists:
		return http.StatusConflict
	case shared.ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case shared.ErrCodeForbidden:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
