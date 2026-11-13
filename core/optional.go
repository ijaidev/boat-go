package core

import "encoding/json"

// Optional is a JSON field with an explicit null state.
//
// Use it through a pointer field so that all three states are representable:
//
//	type Request struct {
//		TTLSeconds *core.Optional[int] `json:"ttlSeconds,omitempty"`
//	}
//
//	Request{}                          // field omitted
//	Request{TTLSeconds: core.Null[int]()}  // "ttlSeconds": null
//	Request{TTLSeconds: core.Some(3600)}   // "ttlSeconds": 3600
//
// This matters for Boat fields such as ttlSeconds, where omitting the field uses the server default
// but an explicit null disables auto-stop.
type Optional[T any] struct {
	Value T
	Valid bool
}

// Ptr returns a pointer to v. It is a shorthand for optional request fields such as *bool.
func Ptr[T any](v T) *T {
	return &v
}

// Some returns an Optional carrying v.
func Some[T any](v T) *Optional[T] {
	return &Optional[T]{Value: v, Valid: true}
}

// Null returns an Optional that marshals as JSON null.
func Null[T any]() *Optional[T] {
	return &Optional[T]{}
}

// Get returns the value and whether it is a value rather than null.
func (o Optional[T]) Get() (T, bool) {
	return o.Value, o.Valid
}

// MarshalJSON implements json.Marshaler.
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(o.Value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		o.Valid = false
		var zero T
		o.Value = zero
		return nil
	}
	if err := json.Unmarshal(data, &o.Value); err != nil {
		return err
	}
	o.Valid = true
	return nil
}
