package zoom

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

// Zoom-documented error codes used in our flows. Reference: developers.zoom.us/docs/integrations/oauth-error-messages.
const (
	ErrCodeInvalidAccessToken = 124
	ErrCodeMissingScope       = 4711
)

// Error wraps transport.APIError with parsed Zoom error body.
type Error struct {
	*transport.APIError
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	if e == nil {
		return "zoom: api error"
	}
	if e.Code != 0 {
		return fmt.Sprintf("zoom: api error status=%d code=%d message=%q", e.StatusCode, e.Code, e.Message)
	}
	if e.APIError != nil {
		return e.APIError.Error()
	}
	return "zoom: api error"
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.APIError
}

// IsInvalidToken reports whether the error means the access token was rejected
// (expired/revoked/malformed). Caller can refresh and retry once.
func IsInvalidToken(err error) bool {
	var ze *Error
	if !errors.As(err, &ze) {
		return false
	}
	return ze.Code == ErrCodeInvalidAccessToken
}

// IsMissingScope reports whether the error means the app lacks a required OAuth scope.
// Not retryable — surface to caller.
func IsMissingScope(err error) bool {
	var ze *Error
	if !errors.As(err, &ze) {
		return false
	}
	return ze.Code == ErrCodeMissingScope || strings.Contains(ze.Message, "does not contain scopes")
}

// asZoomError parses transport.APIError body into Zoom Error if possible.
// Falls back to wrapping the transport error with empty Code/Message.
func asZoomError(apiErr *transport.APIError) *Error {
	out := &Error{APIError: apiErr}
	if apiErr == nil || apiErr.Body == "" {
		return out
	}
	_ = json.Unmarshal([]byte(apiErr.Body), out)
	return out
}
