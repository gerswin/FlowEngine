package jsonapi

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const ContentType = "application/vnd.api+json"

// SuccessResponse sends a successful JSON:API response.
func SuccessResponse(c *gin.Context, code int, data interface{}, meta interface{}, links *Links) {
	c.Header("Content-Type", ContentType)
	c.JSON(code, Response{
		Data:  data,
		Meta:  meta,
		Links: links,
		Jsonapi: &Jsonapi{
			Version: "1.0",
		},
	})
}

// ErrorResponse sends an error JSON:API response.
func ErrorResponse(c *gin.Context, status int, errs []*Error) {
	c.Header("Content-Type", ContentType)
	// Ensure the status code in the header matches, even if individual errors vary
	c.JSON(status, Response{
		Errors: errs,
		Jsonapi: &Jsonapi{
			Version: "1.0",
		},
	})
}

// NewError creates a single error object.
func NewError(status int, code, title, detail string) *Error {
	return &Error{
		Status: strconv.Itoa(status),
		Code:   code,
		Title:  title,
		Detail: detail,
	}
}

// NewValidationError creates an error for a specific field.
func NewValidationError(field, detail string) *Error {
	return &Error{
		Status: "400",
		Code:   "VALIDATION_ERROR",
		Title:  "Invalid Attribute",
		Detail: detail,
		Source: &ErrorSource{
			Pointer: "/data/attributes/" + field,
		},
	}
}

// NewResourceHelper is a utility to create a Resource struct easily.
func NewResource(resourceType, id string, attributes map[string]interface{}) *Resource {
	return &Resource{
		Type:       resourceType,
		ID:         id,
		Attributes: attributes,
	}
}
