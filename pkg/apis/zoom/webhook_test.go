package zoom

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"testing"
	"time"
)

// signWebhook reproduces Zoom's signature independently of the code under test.
func signWebhook(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("v0:" + timestamp + ":" + string(body)))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}

func TestAnswerURLValidation(t *testing.T) {
	t.Parallel()
	const (
		secret = "secret-token"
		plain  = "abc123plain"
	)

	got := AnswerURLValidation(secret, plain)

	if got.PlainToken != plain {
		t.Fatalf("PlainToken: want %q, got %q", plain, got.PlainToken)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(plain))
	want := hex.EncodeToString(mac.Sum(nil))
	if got.EncryptedToken != want {
		t.Fatalf("EncryptedToken: want %q, got %q", want, got.EncryptedToken)
	}
}

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	t.Parallel()
	const secret = "secret-token"
	now := time.Unix(1_700_000_000, 0)
	ts := strconv.FormatInt(now.Unix(), 10)
	body := []byte(`{"event":"meeting.rtms_started"}`)
	sig := signWebhook(secret, ts, body)

	if err := VerifyWebhookSignature(secret, sig, ts, body, now, DefaultWebhookTolerance); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestVerifyWebhookSignature_Rejections(t *testing.T) {
	t.Parallel()
	const secret = "secret-token"
	now := time.Unix(1_700_000_000, 0)
	ts := strconv.FormatInt(now.Unix(), 10)
	body := []byte(`{"event":"x"}`)
	validSig := signWebhook(secret, ts, body)

	cases := []struct {
		name      string
		secret    string
		signature string
		timestamp string
		body      []byte
		want      error
	}{
		{"missing signature", secret, "", ts, body, ErrWebhookMissingSignature},
		{"missing timestamp", secret, validSig, "", body, ErrWebhookMissingSignature},
		{"bad timestamp", secret, validSig, "not-a-number", body, ErrWebhookBadTimestamp},
		{"tampered body", secret, validSig, ts, []byte(`{"event":"y"}`), ErrWebhookSignatureMismatch},
		{"wrong secret", "other-secret", validSig, ts, body, ErrWebhookSignatureMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := VerifyWebhookSignature(tc.secret, tc.signature, tc.timestamp, tc.body, now, DefaultWebhookTolerance)
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
}

func TestVerifyWebhookSignature_EmptySecretFailsClosed(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_700_000_000, 0)
	ts := strconv.FormatInt(now.Unix(), 10)
	body := []byte(`{"event":"x"}`)
	// An attacker who knows body+timestamp can forge a valid HMAC under the empty
	// key, so an empty secret must be rejected outright, never verified.
	forged := signWebhook("", ts, body)
	if err := VerifyWebhookSignature("", forged, ts, body, now, DefaultWebhookTolerance); !errors.Is(err, ErrWebhookMissingSecret) {
		t.Fatalf("want ErrWebhookMissingSecret, got %v", err)
	}
}

func TestVerifyWebhookSignature_StaleTimestamp(t *testing.T) {
	t.Parallel()
	const secret = "secret-token"
	signedAt := time.Unix(1_700_000_000, 0)
	ts := strconv.FormatInt(signedAt.Unix(), 10)
	body := []byte(`{"event":"x"}`)
	sig := signWebhook(secret, ts, body)

	// now is 10 minutes after the signed timestamp — outside the 5m window.
	now := signedAt.Add(10 * time.Minute)
	if err := VerifyWebhookSignature(secret, sig, ts, body, now, DefaultWebhookTolerance); !errors.Is(err, ErrWebhookStaleTimestamp) {
		t.Fatalf("want ErrWebhookStaleTimestamp, got %v", err)
	}
}

func TestVerifyWebhookSignature_SkipFreshnessWhenToleranceZero(t *testing.T) {
	t.Parallel()
	const secret = "secret-token"
	signedAt := time.Unix(1_700_000_000, 0)
	ts := strconv.FormatInt(signedAt.Unix(), 10)
	body := []byte(`{"event":"x"}`)
	sig := signWebhook(secret, ts, body)

	now := signedAt.Add(24 * time.Hour) // ancient, but freshness disabled
	if err := VerifyWebhookSignature(secret, sig, ts, body, now, 0); err != nil {
		t.Fatalf("want nil with tolerance=0, got %v", err)
	}
}

func TestVerifyWebhookSignature_MillisecondTimestamp(t *testing.T) {
	t.Parallel()
	const secret = "secret-token"
	now := time.Unix(1_700_000_000, 0)
	// Zoom usually sends seconds; ensure a millisecond value still validates (signature
	// uses the raw string, freshness normalizes it).
	tsMillis := strconv.FormatInt(now.UnixMilli(), 10)
	body := []byte(`{"event":"x"}`)
	sig := signWebhook(secret, tsMillis, body)

	if err := VerifyWebhookSignature(secret, sig, tsMillis, body, now, DefaultWebhookTolerance); err != nil {
		t.Fatalf("want nil for ms timestamp, got %v", err)
	}
}
