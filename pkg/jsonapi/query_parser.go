package jsonapi

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// QueryOptions represents standard JSON:API query parameters.
type QueryOptions struct {
	PageNumber int               // page[number]
	PageSize   int               // page[size]
	Filters    map[string]string // filter[field]=value
	Sort       []string          // sort=-created_at,name
	Include    []string          // include=author,comments
	Fields     map[string]string // fields[articles]=title,body
}

// DefaultPageSize is the default number of items per page.
const DefaultPageSize = 20

// MaxPageSize is the maximum allowed page size.
const MaxPageSize = 100

// ParseQueryOptions extracts JSON:API query parameters from the request context.
func ParseQueryOptions(c *gin.Context) QueryOptions {
	opts := QueryOptions{
		PageNumber: 1,
		PageSize:   DefaultPageSize,
		Filters:    make(map[string]string),
		Fields:     make(map[string]string),
	}

	// Parse Pagination
	if pageNumStr := c.Query("page[number]"); pageNumStr != "" {
		if num, err := strconv.Atoi(pageNumStr); err == nil && num > 0 {
			opts.PageNumber = num
		}
	}
	if pageSizeStr := c.Query("page[size]"); pageSizeStr != "" {
		if size, err := strconv.Atoi(pageSizeStr); err == nil && size > 0 {
			if size > MaxPageSize {
				opts.PageSize = MaxPageSize
			} else {
				opts.PageSize = size
			}
		}
	}

	// Parse Filters: filter[status]=active
	for key, values := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "filter[") && strings.HasSuffix(key, "]") {
			fieldName := strings.TrimSuffix(strings.TrimPrefix(key, "filter["), "]")
			if len(values) > 0 {
				opts.Filters[fieldName] = values[0]
			}
		}
	}

	// Parse Sorting: sort=-created_at,name
	if sortStr := c.Query("sort"); sortStr != "" {
		opts.Sort = strings.Split(sortStr, ",")
	}

	// Parse Includes: include=author,comments
	if includeStr := c.Query("include"); includeStr != "" {
		opts.Include = strings.Split(includeStr, ",")
	}

	// Parse Sparse Fieldsets: fields[articles]=title,body
	for key, values := range c.Request.URL.Query() {
		if strings.HasPrefix(key, "fields[") && strings.HasSuffix(key, "]") {
			resourceType := strings.TrimSuffix(strings.TrimPrefix(key, "fields["), "]")
			if len(values) > 0 {
				opts.Fields[resourceType] = values[0]
			}
		}
	}

	return opts
}
