package jsonapi

// Reference: https://jsonapi.org/format/

// Response is the top-level JSON:API object.
type Response struct {
	Data     interface{} `json:"data,omitempty"`     // Can be *Resource, []*Resource, or null
	Errors   []*Error    `json:"errors,omitempty"`   // Array of error objects
	Meta     interface{} `json:"meta,omitempty"`     // Meta information (pagination, etc.)
	Jsonapi  *Jsonapi    `json:"jsonapi,omitempty"`  // JSON:API version
	Links    *Links      `json:"links,omitempty"`    // Top-level links
	Included []*Resource `json:"included,omitempty"` // Sideloaded resources
}

// Resource represents a single resource object.
type Resource struct {
	Type          string                 `json:"type"`
	ID            string                 `json:"id"`
	Attributes    map[string]interface{} `json:"attributes,omitempty"`
	Relationships map[string]Relation    `json:"relationships,omitempty"`
	Links         *Links                 `json:"links,omitempty"`
	Meta          interface{}            `json:"meta,omitempty"`
}

// Relation represents a relationship link.
type Relation struct {
	Data  interface{} `json:"data,omitempty"` // ResourceIdentifier or []ResourceIdentifier
	Links *Links      `json:"links,omitempty"`
	Meta  interface{} `json:"meta,omitempty"`
}

// ResourceIdentifier is the minimal representation for relationships.
type ResourceIdentifier struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Error represents an error object.
type Error struct {
	ID     string       `json:"id,omitempty"`
	Status string       `json:"status,omitempty"` // HTTP status code as string
	Code   string       `json:"code,omitempty"`   // App-specific error code
	Title  string       `json:"title,omitempty"`
	Detail string       `json:"detail,omitempty"`
	Source *ErrorSource `json:"source,omitempty"`
	Meta   interface{}  `json:"meta,omitempty"`
}

// ErrorSource indicates the source of the error.
type ErrorSource struct {
	Pointer   string `json:"pointer,omitempty"`   // JSON Pointer to the associated entity
	Parameter string `json:"parameter,omitempty"` // Query parameter causing the error
}

// Links represents HATEOAS links.
type Links struct {
	Self    string `json:"self,omitempty"`
	Related string `json:"related,omitempty"`
	First   string `json:"first,omitempty"`
	Prev    string `json:"prev,omitempty"`
	Next    string `json:"next,omitempty"`
	Last    string `json:"last,omitempty"`
}

// Jsonapi object describing the server's implementation.
type Jsonapi struct {
	Version string `json:"version,omitempty"`
}

// Meta is a free-form object.
type Meta map[string]interface{}
