package shared

// ListQuery represents pagination parameters for list operations.
type ListQuery struct {
	Offset int
	Limit  int
}

// NewListQuery creates a ListQuery from page number and size.
// Page numbers are 1-based. Returns a query with safe defaults if inputs are invalid.
func NewListQuery(pageNumber, pageSize int) ListQuery {
	if pageNumber < 1 {
		pageNumber = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return ListQuery{
		Offset: (pageNumber - 1) * pageSize,
		Limit:  pageSize,
	}
}
