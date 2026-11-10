package v22

import (
	"errors"
	"fmt"
)

var (
	// ErrCreateFailed indicates the API did not return a created campaign ID.
	ErrCreateFailed = errors.New("create campaign failed: empty id")
	// ErrUpdateFailed indicates the API did not confirm a successful update.
	ErrUpdateFailed = errors.New("update campaign failed: empty id")
)

// MissingFieldError indicates that a required field was missing on an entity.
type MissingFieldError struct {
	Entity string // e.g. "Campaign"
	Field  string // e.g. "id"
}

func (e *MissingFieldError) Error() string {
	return fmt.Sprintf("%s missing required field %q", e.Entity, e.Field)
}

// AlreadyExistsError indicates an attempt to create an entity that already exists
type AlreadyExistsError struct {
	Entity string
	ID     string
}

func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s id: %s already exist", e.Entity, e.ID)
}

// UpstreamError wraps transport/client errors (HTTP, JSON, network).
// The wrapped Err is accessible via Unwrap, enabling errors.Is/As.
type UpstreamError struct {
	Op    string // e.g. "Get", "Create", "Update", "List"
	Route string // final URL string
	Code  int    // HTTP/status if available
	Err   error  // wrapped root error from fb client
}

func (e *UpstreamError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("%s upstream failed (code=%d) route=%q: %v", e.Op, e.Code, e.Route, e.Err)
	}
	return fmt.Sprintf("%s upstream failed route=%q: %v", e.Op, e.Route, e.Err)
}

func (e *UpstreamError) Unwrap() error { return e.Err }

// RemoteAPIError represents a non-success response reported by the remote API.
type RemoteAPIError struct {
	Op     string
	Route  string
	Detail string // extracted from res.GetError()
}

func (e *RemoteAPIError) Error() string {
	return fmt.Sprintf("%s API call failed route=%q: %s", e.Op, e.Route, e.Detail)
}

// Compile-time assertions that types implement error.
var (
	_ error = (*MissingFieldError)(nil)
	_ error = (*AlreadyExistsError)(nil)
	_ error = (*UpstreamError)(nil)
	_ error = (*RemoteAPIError)(nil)
)
