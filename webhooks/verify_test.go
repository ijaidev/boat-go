package webhooks_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ijaidev/boat-go/model"
	"github.com/ijaidev/boat-go/webhooks"
)

const (
	knownSecret    = "whsec_test_0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	knownDelivery  = "evt_0123456789abcdef0123456789abcdef"
	knownTimestamp = "1794571200"
	knownBody      = `{"id":"evt_0123456789abcdef0123456789abcdef","type":"sandbox.ready","createdAt":"2026-11-13T12:00:00Z","data":{"sandbox":{"id":"bx_example","name":"Example"},"previousState":"provisioning","state":"ready"}}`
	knownSignature = "ad0354b08492b8dc91e8f4f715e62135abecf5d675bd848d5e0137d4f52a0161"
)

func knownHeaders() http.Header {
	return http.Header{
		webhooks.HeaderEvent:     {string(model.SandboxReady)},
		webhooks.HeaderDelivery:  {knownDelivery},
		webhooks.HeaderTimestamp: {knownTimestamp},
		webhooks.HeaderSignature: {"v1=" + knownSignature},
		webhooks.HeaderAttempt:   {"1"},
	}
}

func knownNow() time.Time {
	return time.Unix(1794571200, 0)
}

func sign(t *testing.T, secret, delivery, timestamp, body string) string {
	t.Helper()

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(delivery + "." + timestamp + "." + body))
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerify_KnownVector(t *testing.T) {
	t.Parallel()

	event, err := webhooks.Verify([]byte(knownBody), knownHeaders(), knownSecret, webhooks.VerifyOptions{Now: knownNow})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if event.DeliveryID != knownDelivery {
		t.Fatalf("DeliveryID = %q, want %q", event.DeliveryID, knownDelivery)
	}
	if event.ID != knownDelivery {
		t.Fatalf("ID = %q, want %q", event.ID, knownDelivery)
	}
	if event.Type != model.SandboxReady {
		t.Fatalf("Type = %q, want %q", event.Type, model.SandboxReady)
	}
	if event.Timestamp.Unix() != 1794571200 {
		t.Fatalf("Timestamp = %v", event.Timestamp)
	}
	if event.Attempt != 1 {
		t.Fatalf("Attempt = %d, want 1", event.Attempt)
	}
	if string(event.RawBody) != knownBody {
		t.Fatalf("RawBody changed")
	}
	if event.Data.Sandbox.ID != "bx_example" || event.Data.Sandbox.Name != "Example" {
		t.Fatalf("Sandbox = %#v", event.Data.Sandbox)
	}
	if event.Data.State == nil || *event.Data.State != "ready" {
		t.Fatalf("State = %v", event.Data.State)
	}
	if event.Data.PreviousState == nil || *event.Data.PreviousState != "provisioning" {
		t.Fatalf("PreviousState = %v", event.Data.PreviousState)
	}
}

func TestVerify_RequiresExactRawBody(t *testing.T) {
	t.Parallel()

	headers := knownHeaders()

	t.Run("extra whitespace", func(t *testing.T) {
		t.Parallel()

		mutated := strings.Replace(knownBody, `"id":`, `"id": `, 1)
		if mutated == knownBody {
			t.Fatal("body was not mutated")
		}
		if _, err := webhooks.Verify([]byte(mutated), headers, knownSecret, webhooks.VerifyOptions{Now: knownNow}); !errors.Is(err, webhooks.ErrInvalidSignature) {
			t.Fatalf("err = %v, want ErrInvalidSignature", err)
		}
	})

	t.Run("re-serialized", func(t *testing.T) {
		t.Parallel()

		var parsed map[string]any
		if err := json.Unmarshal([]byte(knownBody), &parsed); err != nil {
			t.Fatalf("unmarshal fixture: %v", err)
		}
		reserialized, err := json.Marshal(parsed)
		if err != nil {
			t.Fatalf("marshal fixture: %v", err)
		}
		if string(reserialized) == knownBody {
			t.Fatal("fixture did not change on re-serialization")
		}
		if _, err := webhooks.Verify(reserialized, headers, knownSecret, webhooks.VerifyOptions{Now: knownNow}); !errors.Is(err, webhooks.ErrInvalidSignature) {
			t.Fatalf("err = %v, want ErrInvalidSignature", err)
		}
	})
}

func TestVerify_InvalidSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		secret    string
		signature string
	}{
		{name: "wrong secret", secret: "whsec_test_wrong", signature: "v1=" + knownSignature},
		{name: "wrong digest", secret: knownSecret, signature: "v1=" + strings.Repeat("0", 64)},
		{name: "missing version", secret: knownSecret, signature: knownSignature},
		{name: "wrong version", secret: knownSecret, signature: "v2=" + knownSignature},
		{name: "malformed hex", secret: knownSecret, signature: "v1=" + strings.Repeat("z", 64)},
		{name: "short digest", secret: knownSecret, signature: "v1=" + knownSignature[:62]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := knownHeaders()
			headers.Set(webhooks.HeaderSignature, tt.signature)
			if _, err := webhooks.Verify([]byte(knownBody), headers, tt.secret, webhooks.VerifyOptions{Now: knownNow}); !errors.Is(err, webhooks.ErrInvalidSignature) {
				t.Fatalf("err = %v, want ErrInvalidSignature", err)
			}
		})
	}
}

func TestVerify_RejectsStaleTimestamp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		now  time.Time
		opts webhooks.VerifyOptions
		want error
	}{
		{name: "too old", now: knownNow().Add(5*time.Minute + time.Second), want: webhooks.ErrStaleTimestamp},
		{name: "too far in the future", now: knownNow().Add(-5*time.Minute - time.Second), want: webhooks.ErrStaleTimestamp},
		{name: "just inside", now: knownNow().Add(5 * time.Minute)},
		{name: "custom tolerance", now: knownNow().Add(6 * time.Minute), opts: webhooks.VerifyOptions{Tolerance: 10 * time.Minute}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			now := tt.now
			tt.opts.Now = func() time.Time { return now }
			_, err := webhooks.Verify([]byte(knownBody), knownHeaders(), knownSecret, tt.opts)
			if tt.want != nil {
				if !errors.Is(err, tt.want) {
					t.Fatalf("err = %v, want %v", err, tt.want)
				}
				return
			}
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}
		})
	}
}

func TestVerify_MissingHeaders(t *testing.T) {
	t.Parallel()

	t.Run("nil header map", func(t *testing.T) {
		t.Parallel()

		if _, err := webhooks.Verify([]byte(knownBody), nil, knownSecret, webhooks.VerifyOptions{Now: knownNow}); !errors.Is(err, webhooks.ErrMissingHeader) {
			t.Fatalf("err = %v, want ErrMissingHeader", err)
		}
	})

	for _, name := range []string{
		webhooks.HeaderEvent,
		webhooks.HeaderDelivery,
		webhooks.HeaderTimestamp,
		webhooks.HeaderSignature,
		webhooks.HeaderAttempt,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			headers := knownHeaders()
			headers.Del(name)
			if _, err := webhooks.Verify([]byte(knownBody), headers, knownSecret, webhooks.VerifyOptions{Now: knownNow}); !errors.Is(err, webhooks.ErrMissingHeader) {
				t.Fatalf("err = %v, want ErrMissingHeader", err)
			}
		})
	}
}

func TestVerify_MissingSecret(t *testing.T) {
	t.Parallel()

	if _, err := webhooks.Verify([]byte(knownBody), knownHeaders(), "", webhooks.VerifyOptions{Now: knownNow}); !errors.Is(err, webhooks.ErrMissingSecret) {
		t.Fatalf("err = %v, want ErrMissingSecret", err)
	}
}

func TestVerify_InvalidHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "non-numeric timestamp", key: webhooks.HeaderTimestamp, value: "not-a-number"},
		{name: "negative timestamp", key: webhooks.HeaderTimestamp, value: "-1"},
		{name: "malformed delivery", key: webhooks.HeaderDelivery, value: "evt_NOT-HEX"},
		{name: "short delivery", key: webhooks.HeaderDelivery, value: "evt_0123"},
		{name: "attempt zero", key: webhooks.HeaderAttempt, value: "0"},
		{name: "attempt too high", key: webhooks.HeaderAttempt, value: "9"},
		{name: "non-numeric attempt", key: webhooks.HeaderAttempt, value: "first"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := knownHeaders()
			headers.Set(tt.key, tt.value)
			if _, err := webhooks.Verify([]byte(knownBody), headers, knownSecret, webhooks.VerifyOptions{Now: knownNow}); !errors.Is(err, webhooks.ErrInvalidHeader) {
				t.Fatalf("err = %v, want ErrInvalidHeader", err)
			}
		})
	}
}

func TestVerify_RejectsWrongEventType(t *testing.T) {
	t.Parallel()

	t.Run("header mismatch", func(t *testing.T) {
		t.Parallel()

		headers := knownHeaders()
		headers.Set(webhooks.HeaderEvent, string(model.SandboxError))
		if _, err := webhooks.Verify([]byte(knownBody), headers, knownSecret, webhooks.VerifyOptions{Now: knownNow}); !errors.Is(err, webhooks.ErrInvalidEvent) {
			t.Fatalf("err = %v, want ErrInvalidEvent", err)
		}
	})

	t.Run("unknown header type", func(t *testing.T) {
		t.Parallel()

		headers := knownHeaders()
		headers.Set(webhooks.HeaderEvent, "sandbox.unknown")
		if _, err := webhooks.Verify([]byte(knownBody), headers, knownSecret, webhooks.VerifyOptions{Now: knownNow}); !errors.Is(err, webhooks.ErrInvalidEvent) {
			t.Fatalf("err = %v, want ErrInvalidEvent", err)
		}
	})

	t.Run("unknown body type", func(t *testing.T) {
		t.Parallel()

		body := strings.Replace(knownBody, string(model.SandboxReady), "sandbox.unknown", 1)
		headers := knownHeaders()
		headers.Set(webhooks.HeaderEvent, "sandbox.unknown")
		headers.Set(webhooks.HeaderSignature, sign(t, knownSecret, knownDelivery, knownTimestamp, body))
		if _, err := webhooks.Verify([]byte(body), headers, knownSecret, webhooks.VerifyOptions{Now: knownNow}); !errors.Is(err, webhooks.ErrInvalidEvent) {
			t.Fatalf("err = %v, want ErrInvalidEvent", err)
		}
	})
}

func TestVerify_RejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	body := "not json"
	headers := knownHeaders()
	headers.Set(webhooks.HeaderSignature, sign(t, knownSecret, knownDelivery, knownTimestamp, body))
	if _, err := webhooks.Verify([]byte(body), headers, knownSecret, webhooks.VerifyOptions{Now: knownNow}); !errors.Is(err, webhooks.ErrInvalidPayload) {
		t.Fatalf("err = %v, want ErrInvalidPayload", err)
	}
}

func TestVerify_DoesNotDeduplicate(t *testing.T) {
	t.Parallel()

	opts := webhooks.VerifyOptions{Now: knownNow}
	first, err := webhooks.Verify([]byte(knownBody), knownHeaders(), knownSecret, opts)
	if err != nil {
		t.Fatalf("first Verify: %v", err)
	}
	second, err := webhooks.Verify([]byte(knownBody), knownHeaders(), knownSecret, opts)
	if err != nil {
		t.Fatalf("second Verify: %v", err)
	}
	if second.DeliveryID != first.DeliveryID {
		t.Fatalf("DeliveryID changed: %q then %q", first.DeliveryID, second.DeliveryID)
	}
}

func TestVerify_AcceptsLastAttempt(t *testing.T) {
	t.Parallel()

	headers := knownHeaders()
	headers.Set(webhooks.HeaderAttempt, "8")
	event, err := webhooks.Verify([]byte(knownBody), headers, knownSecret, webhooks.VerifyOptions{Now: knownNow})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if event.Attempt != 8 {
		t.Fatalf("Attempt = %d, want 8", event.Attempt)
	}
}

func TestVerify_ErrorsDoNotLeakSecrets(t *testing.T) {
	t.Parallel()

	headers := knownHeaders()
	headers.Set(webhooks.HeaderSignature, "v1="+strings.Repeat("0", 64))
	_, err := webhooks.Verify([]byte(knownBody), headers, knownSecret, webhooks.VerifyOptions{Now: knownNow})
	if err == nil {
		t.Fatal("Verify = nil, want error")
	}
	if strings.Contains(err.Error(), knownSecret) {
		t.Fatalf("error contains the signing secret: %v", err)
	}
	if strings.Contains(err.Error(), strings.Repeat("0", 64)) {
		t.Fatalf("error contains the signature: %v", err)
	}
}
