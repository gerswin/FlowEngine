package jsonapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func createTestContext(queryString string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/?"+queryString, nil)
	return c
}

func TestParseQueryOptions_DefaultValues(t *testing.T) {
	c := createTestContext("")
	opts := ParseQueryOptions(c)

	assert.Equal(t, 1, opts.PageNumber)
	assert.Equal(t, DefaultPageSize, opts.PageSize)
	assert.Empty(t, opts.Filters)
	assert.Nil(t, opts.Sort)
	assert.Nil(t, opts.Include)
	assert.Empty(t, opts.Fields)
}

func TestParseQueryOptions_CustomPageNumberAndSize(t *testing.T) {
	c := createTestContext("page[number]=3&page[size]=50")
	opts := ParseQueryOptions(c)

	assert.Equal(t, 3, opts.PageNumber)
	assert.Equal(t, 50, opts.PageSize)
}

func TestParseQueryOptions_PageSizeCappedAtMax(t *testing.T) {
	c := createTestContext("page[size]=999")
	opts := ParseQueryOptions(c)

	assert.Equal(t, MaxPageSize, opts.PageSize)
}

func TestParseQueryOptions_PageSizeExactlyAtMax(t *testing.T) {
	c := createTestContext("page[size]=100")
	opts := ParseQueryOptions(c)

	assert.Equal(t, 100, opts.PageSize)
}

func TestParseQueryOptions_InvalidPageNumberDefaultsTo1(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"negative page", "page[number]=-1"},
		{"zero page", "page[number]=0"},
		{"non-numeric page", "page[number]=abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createTestContext(tt.query)
			opts := ParseQueryOptions(c)
			assert.Equal(t, 1, opts.PageNumber)
		})
	}
}

func TestParseQueryOptions_InvalidPageSizeKeepsDefault(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"negative size", "page[size]=-5"},
		{"zero size", "page[size]=0"},
		{"non-numeric size", "page[size]=xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := createTestContext(tt.query)
			opts := ParseQueryOptions(c)
			assert.Equal(t, DefaultPageSize, opts.PageSize)
		})
	}
}

func TestParseQueryOptions_FilterParsing(t *testing.T) {
	c := createTestContext("filter[status]=active")
	opts := ParseQueryOptions(c)

	assert.Equal(t, "active", opts.Filters["status"])
}

func TestParseQueryOptions_MultipleFilters(t *testing.T) {
	c := createTestContext("filter[status]=active&filter[type]=workflow&filter[priority]=high")
	opts := ParseQueryOptions(c)

	assert.Equal(t, 3, len(opts.Filters))
	assert.Equal(t, "active", opts.Filters["status"])
	assert.Equal(t, "workflow", opts.Filters["type"])
	assert.Equal(t, "high", opts.Filters["priority"])
}

func TestParseQueryOptions_SortParsing(t *testing.T) {
	c := createTestContext("sort=-created_at,name")
	opts := ParseQueryOptions(c)

	assert.Equal(t, []string{"-created_at", "name"}, opts.Sort)
}

func TestParseQueryOptions_SortSingleField(t *testing.T) {
	c := createTestContext("sort=name")
	opts := ParseQueryOptions(c)

	assert.Equal(t, []string{"name"}, opts.Sort)
}

func TestParseQueryOptions_IncludeParsing(t *testing.T) {
	c := createTestContext("include=author,comments")
	opts := ParseQueryOptions(c)

	assert.Equal(t, []string{"author", "comments"}, opts.Include)
}

func TestParseQueryOptions_IncludeSingleResource(t *testing.T) {
	c := createTestContext("include=author")
	opts := ParseQueryOptions(c)

	assert.Equal(t, []string{"author"}, opts.Include)
}

func TestParseQueryOptions_FieldsParsing(t *testing.T) {
	c := createTestContext("fields[articles]=title,body")
	opts := ParseQueryOptions(c)

	assert.Equal(t, "title,body", opts.Fields["articles"])
}

func TestParseQueryOptions_MultipleFieldsets(t *testing.T) {
	c := createTestContext("fields[articles]=title,body&fields[people]=name")
	opts := ParseQueryOptions(c)

	assert.Equal(t, "title,body", opts.Fields["articles"])
	assert.Equal(t, "name", opts.Fields["people"])
}

func TestParseQueryOptions_AllParamsCombined(t *testing.T) {
	c := createTestContext("page[number]=2&page[size]=10&filter[status]=active&sort=-created_at&include=author&fields[articles]=title")
	opts := ParseQueryOptions(c)

	assert.Equal(t, 2, opts.PageNumber)
	assert.Equal(t, 10, opts.PageSize)
	assert.Equal(t, "active", opts.Filters["status"])
	assert.Equal(t, []string{"-created_at"}, opts.Sort)
	assert.Equal(t, []string{"author"}, opts.Include)
	assert.Equal(t, "title", opts.Fields["articles"])
}

func TestNewError(t *testing.T) {
	err := NewError(404, "NOT_FOUND", "Not Found", "Resource not found")

	assert.Equal(t, "404", err.Status)
	assert.Equal(t, "NOT_FOUND", err.Code)
	assert.Equal(t, "Not Found", err.Title)
	assert.Equal(t, "Resource not found", err.Detail)
	assert.Nil(t, err.Source)
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("email", "must be a valid email")

	assert.Equal(t, "400", err.Status)
	assert.Equal(t, "VALIDATION_ERROR", err.Code)
	assert.Equal(t, "Invalid Attribute", err.Title)
	assert.Equal(t, "must be a valid email", err.Detail)
	assert.NotNil(t, err.Source)
	assert.Equal(t, "/data/attributes/email", err.Source.Pointer)
}

func TestNewResource(t *testing.T) {
	attrs := map[string]interface{}{"title": "Hello", "body": "World"}
	res := NewResource("articles", "1", attrs)

	assert.Equal(t, "articles", res.Type)
	assert.Equal(t, "1", res.ID)
	assert.Equal(t, "Hello", res.Attributes["title"])
	assert.Equal(t, "World", res.Attributes["body"])
}
