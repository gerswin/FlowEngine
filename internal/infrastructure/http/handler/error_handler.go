package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/LaFabric-LinkTIC/FlowEngine/internal/domain/shared"
)

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Code    string                 `json:"code,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// handleError converts domain errors to HTTP responses.
func handleError(c *gin.Context, err error) {
	var domainErr *shared.DomainError
	if errors.As(err, &domainErr) {
		handleDomainError(c, domainErr)
		return
	}

	// Generic error
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:   "Internal Server Error",
		Message: err.Error(),
	})
}

// handleDomainError converts domain errors to appropriate HTTP status codes.
func handleDomainError(c *gin.Context, err *shared.DomainError) {
	status := getHTTPStatus(err)

	response := ErrorResponse{
		Error:   string(err.Code()),
		Message: err.Message(),
		Code:    string(err.Code()),
		Context: err.Context(),
	}

	c.JSON(status, response)
}

// getHTTPStatus maps domain error codes to HTTP status codes.
func getHTTPStatus(err *shared.DomainError) int {
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
