package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ijaidev/boat-go/model"
)

// Delivery headers sent with every webhook attempt.
const (
	// HeaderEvent carries the event type, such as "sandbox.ready".
	HeaderEvent = "X-Ascii-Event"
	// HeaderDelivery carries the delivery id used as the caller's idempotency key.
	HeaderDelivery = "X-Ascii-Delivery"
	// HeaderTimestamp carries Unix seconds as ASCII digits.
	HeaderTimestamp = "X-Ascii-Timestamp"
	// HeaderSignature carries "v1=" followed by the lowercase hex HMAC-SHA256.
	HeaderSignature = "X-Ascii-Signature"
	// HeaderAttempt carries the 1-based delivery attempt number.
	HeaderAttempt = "X-Ascii-Attempt"
)

// DefaultTolerance is the maximum accepted clock skew of X-Ascii-Timestamp.
const DefaultTolerance = 5 * time.Minute

const (
	signaturePrefix = "v1="
	maxAttempt      = 8
)

// Verification failures. Use errors.Is to distinguish them.
var (
	// ErrMissingSecret means Verify was called with an empty signing secret.
	ErrMissingSecret = errors.New("webhooks: signing secret is required")
	// ErrMissingHeader means a required X-Ascii-* delivery header is absent.
	ErrMissingHeader = errors.New("webhooks: missing delivery header")
	// ErrInvalidHeader means a delivery header has the wrong shape, such as a non-numeric
	// timestamp or an out-of-range attempt.
	ErrInvalidHeader = errors.New("webhooks: invalid delivery header")
	// ErrInvalidSignature means the signature is malformed or does not match the raw body.
	ErrInvalidSignature = errors.New("webhooks: invalid signature")
	// ErrStaleTimestamp means the signed timestamp is outside VerifyOptions.Tolerance.
	ErrStaleTimestamp = errors.New("webhooks: timestamp outside tolerance")
	// ErrInvalidEvent means the event type is unknown or the header and body disagree.
	ErrInvalidEvent = errors.New("webhooks: invalid event")
	// ErrInvalidPayload means the verified body is not valid JSON for a webhook event.
	ErrInvalidPayload = errors.New("webhooks: invalid payload")
)

// VerifyOptions tunes signature verification.
//
// The zero value accepts timestamps within DefaultTolerance of now. A zero or negative Tolerance
// also uses DefaultTolerance. Now overrides the clock; leave it nil in production.
type VerifyOptions struct {
	// Tolerance is the maximum accepted difference between the signed timestamp and now. A zero
	// or negative value uses DefaultTolerance.
	Tolerance time.Duration
	// Now overrides the current time. Nil uses time.Now; tests can set a fixed clock.
	Now func() time.Time
}

// Event is a verified webhook delivery.
//
// The embedded model.WebhookEvent is the parsed JSON body. Its Type field is checked against the
// X-Ascii-Event header, so both values are equal after a successful verification.
//
// See Verify for the signature scheme and the caller-owned deduplication contract.
type Event struct {
	model.WebhookEvent

	// DeliveryID is the X-Ascii-Delivery header. It is stable across retries and is the caller's
	// idempotency key.
	DeliveryID string
	// Timestamp is X-Ascii-Timestamp parsed as Unix seconds and normalized to UTC.
	Timestamp time.Time
	// Attempt is X-Ascii-Attempt, the 1-based delivery attempt number.
	Attempt int
	// RawBody is the exact body that was signed. It aliases rawBody.
	RawBody []byte
}

// Verify checks a webhook delivery and returns its parsed event.
//
// The signature is HMAC-SHA256(secret, deliveryID + "." + timestamp + "." + rawBody), sent in
// X-Ascii-Signature as "v1=" followed by lowercase hex. Verification is constant-time and covers
// rawBody exactly: do not decode, reformat or re-serialize the body before calling Verify.
//
// Required headers are X-Ascii-Event, X-Ascii-Delivery, X-Ascii-Timestamp, X-Ascii-Signature and
// X-Ascii-Attempt. Missing or malformed headers are rejected, as are unknown event types and a
// header type that disagrees with the body. The timestamp must be within VerifyOptions.Tolerance
// (five minutes by default), which rejects replayed deliveries. JSON is parsed only after the
// signature and timestamp are valid.
//
// Verify keeps no state. Deliveries are at-least-once and may arrive out of order, so store
// Event.DeliveryID and ignore a delivery ID you have already processed. Return a 2xx promptly;
// Boat retries anything else with exponential backoff.
//
// The signing secret and the signature are never included in errors.
func Verify(rawBody []byte, headers http.Header, secret string, opts VerifyOptions) (Event, error) {
	if secret == "" {
		return Event{}, ErrMissingSecret
	}

	eventTypeRaw := headers.Get(HeaderEvent)
	if eventTypeRaw == "" {
		return Event{}, missingHeader(HeaderEvent)
	}
	deliveryID := headers.Get(HeaderDelivery)
	if deliveryID == "" {
		return Event{}, missingHeader(HeaderDelivery)
	}
	timestampRaw := headers.Get(HeaderTimestamp)
	if timestampRaw == "" {
		return Event{}, missingHeader(HeaderTimestamp)
	}
	signatureRaw := headers.Get(HeaderSignature)
	if signatureRaw == "" {
		return Event{}, missingHeader(HeaderSignature)
	}
	attemptRaw := headers.Get(HeaderAttempt)
	if attemptRaw == "" {
		return Event{}, missingHeader(HeaderAttempt)
	}

	eventType := model.WebhookEventType(eventTypeRaw)
	if !eventType.Valid() {
		return Event{}, fmt.Errorf("%w: unknown type %q", ErrInvalidEvent, eventTypeRaw)
	}
	if !validDeliveryID(deliveryID) {
		return Event{}, invalidHeader(HeaderDelivery)
	}
	if !isDigits(timestampRaw) {
		return Event{}, invalidHeader(HeaderTimestamp)
	}
	attempt, err := strconv.Atoi(attemptRaw)
	if !isDigits(attemptRaw) || err != nil || attempt < 1 || attempt > maxAttempt {
		return Event{}, invalidHeader(HeaderAttempt)
	}
	signature, ok := decodeSignature(signatureRaw)
	if !ok {
		return Event{}, ErrInvalidSignature
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(deliveryID))
	mac.Write([]byte("."))
	mac.Write([]byte(timestampRaw))
	mac.Write([]byte("."))
	mac.Write(rawBody)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return Event{}, ErrInvalidSignature
	}

	seconds, err := strconv.ParseInt(timestampRaw, 10, 64)
	if err != nil {
		return Event{}, invalidHeader(HeaderTimestamp)
	}
	timestamp := time.Unix(seconds, 0).UTC()
	tolerance := opts.Tolerance
	if tolerance <= 0 {
		tolerance = DefaultTolerance
	}
	now := time.Now()
	if opts.Now != nil {
		now = opts.Now()
	}
	skew := now.Sub(timestamp)
	if skew < 0 {
		skew = -skew
	}
	if skew > tolerance {
		return Event{}, fmt.Errorf("%w: %s is outside the %s window", ErrStaleTimestamp, timestampRaw, tolerance)
	}

	var event model.WebhookEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return Event{}, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	if !event.Type.Valid() {
		return Event{}, fmt.Errorf("%w: unknown body type %q", ErrInvalidEvent, event.Type)
	}
	if event.Type != eventType {
		return Event{}, fmt.Errorf("%w: header %q does not match body %q", ErrInvalidEvent, eventType, event.Type)
	}

	return Event{
		WebhookEvent: event,
		DeliveryID:   deliveryID,
		Timestamp:    timestamp,
		Attempt:      attempt,
		RawBody:      rawBody,
	}, nil
}

func missingHeader(name string) error {
	return fmt.Errorf("%w: %s", ErrMissingHeader, name)
}

func invalidHeader(name string) error {
	return fmt.Errorf("%w: %s", ErrInvalidHeader, name)
}

func validDeliveryID(id string) bool {
	rest, ok := strings.CutPrefix(id, "evt_")
	if !ok || len(rest) != 32 {
		return false
	}
	for i := 0; i < len(rest); i++ {
		if !isLowerHex(rest[i]) {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func isLowerHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
}

func decodeSignature(value string) ([]byte, bool) {
	hexPart, ok := strings.CutPrefix(value, signaturePrefix)
	if !ok || len(hexPart) != sha256.Size*2 {
		return nil, false
	}
	signature, err := hex.DecodeString(hexPart)
	if err != nil {
		return nil, false
	}
	return signature, true
}
