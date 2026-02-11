package transport

import (
	"fmt"
	"net/http"
)

// APIError describes non-2xx responses.
type APIError struct {
	StatusCode int
	Status     string
	Body       string
	Headers    http.Header
	RequestID  string
}

func (e *APIError) Error() string {
	if e == nil {
		return "transport: api error"
	}
	if e.Body == "" {
		return fmt.Sprintf("transport: api error status=%d", e.StatusCode)
	}
	return fmt.Sprintf("transport: api error status=%d body=%q", e.StatusCode, e.Body)
}

// NewAPIError builds APIError from HTTP response and consumes response body.
func NewAPIError(resp *http.Response, maxBodyBytes int64) *APIError {
	if resp == nil {
		return &APIError{}
	}

	bodyBytes, _ := ReadBodyLimited(resp.Body, maxBodyBytes)
	reqID := resp.Header.Get("X-Request-Id")
	if reqID == "" {
		reqID = resp.Header.Get("X-Trace-Id")
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       string(bodyBytes),
		Headers:    resp.Header.Clone(),
		RequestID:  reqID,
	}
}
