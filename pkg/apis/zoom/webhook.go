package zoom

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"
)

// Webhook (Event Subscription) request validation.
//
// Zoom authenticates webhook calls with an HMAC signature derived from the app's
// Secret Token — the token itself is never transmitted. Every event carries the
// headers x-zm-signature and x-zm-request-timestamp; the one-time
// endpoint.url_validation event instead carries a plainToken that must be echoed
// back HMAC'd to prove possession of the Secret Token.
//
// Reference: developers.zoom.us/docs/api/webhooks (request validation, url_validation).

// DefaultWebhookTolerance is the anti-replay window for x-zm-request-timestamp:
// a request whose timestamp is further than this from now is rejected.
const DefaultWebhookTolerance = 5 * time.Minute

var (
	// ErrWebhookMissingSecret means no Secret Token was configured; verification
	// fails closed rather than computing an HMAC under an empty (public) key.
	ErrWebhookMissingSecret = errors.New("zoom: webhook secret token not configured")
	// ErrWebhookMissingSignature means the signature or timestamp header was absent.
	ErrWebhookMissingSignature = errors.New("zoom: missing webhook signature or timestamp")
	// ErrWebhookBadTimestamp means x-zm-request-timestamp could not be parsed.
	ErrWebhookBadTimestamp = errors.New("zoom: malformed webhook timestamp")
	// ErrWebhookStaleTimestamp means the timestamp is outside the allowed tolerance (possible replay).
	ErrWebhookStaleTimestamp = errors.New("zoom: webhook timestamp outside tolerance")
	// ErrWebhookSignatureMismatch means the computed HMAC did not match x-zm-signature.
	ErrWebhookSignatureMismatch = errors.New("zoom: webhook signature mismatch")
)

// URLValidationResponse is the JSON body that answers an endpoint.url_validation event.
type URLValidationResponse struct {
	PlainToken     string `json:"plainToken"`
	EncryptedToken string `json:"encryptedToken"`
}

// AnswerURLValidation builds the challenge response for endpoint.url_validation:
//
//	encryptedToken = hex(HMAC_SHA256(secretToken, plainToken))
//
// The caller echoes both fields back in the HTTP 200 body to prove it knows the Secret Token.
func AnswerURLValidation(secretToken, plainToken string) URLValidationResponse {
	return URLValidationResponse{
		PlainToken:     plainToken,
		EncryptedToken: hexHMACSHA256(secretToken, plainToken),
	}
}

// VerifyWebhookSignature validates the x-zm-signature header against the raw request body:
//
//	signature = "v0=" + hex(HMAC_SHA256(secretToken, "v0:"+timestamp+":"+body))
//
// timestamp is the raw x-zm-request-timestamp header value and body is the raw,
// unparsed request body (re-serializing the JSON would change the bytes and break
// the HMAC). The comparison is constant-time. When tolerance > 0 the request
// timestamp must also be within tolerance of now (pass time.Now()) to mitigate
// replay; pass tolerance <= 0 to skip the freshness check. Returns nil when the
// request is authentic.
func VerifyWebhookSignature(secretToken, signature, timestamp string, body []byte, now time.Time, tolerance time.Duration) error {
	if secretToken == "" {
		return ErrWebhookMissingSecret
	}
	if signature == "" || timestamp == "" {
		return ErrWebhookMissingSignature
	}
	if tolerance > 0 {
		if err := verifyTimestampFresh(timestamp, now, tolerance); err != nil {
			return err
		}
	}
	expected := "v0=" + hexHMACSHA256(secretToken, "v0:"+timestamp+":"+string(body))
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrWebhookSignatureMismatch
	}
	return nil
}

// verifyTimestampFresh parses x-zm-request-timestamp and checks it is within tolerance of now.
// Zoom sends seconds since epoch; values that look like milliseconds (13+ digits) are normalized.
func verifyTimestampFresh(timestamp string, now time.Time, tolerance time.Duration) error {
	ts, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return ErrWebhookBadTimestamp
	}
	if ts > 1_000_000_000_000 { // milliseconds → seconds
		ts /= 1000
	}
	diff := now.Sub(time.Unix(ts, 0))
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		return ErrWebhookStaleTimestamp
	}
	return nil
}

func hexHMACSHA256(secret, msg string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}
