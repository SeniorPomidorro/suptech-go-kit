package transport

import (
	"io"
	"net/url"
)

// ReadBodyLimited reads response body up to maxBytes.
func ReadBodyLimited(reader io.Reader, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return io.ReadAll(reader)
	}

	limited := &io.LimitedReader{R: reader, N: maxBytes}
	return io.ReadAll(limited)
}

// EncodeQuery converts URL values to encoded query string.
func EncodeQuery(values url.Values) string {
	if values == nil {
		return ""
	}
	return values.Encode()
}
