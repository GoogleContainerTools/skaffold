// SPDX-FileCopyrightText: 2021 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import (
	"fmt"

	"github.com/segmentio/encoding/json"
)

// CancelParams params of cancelRequest.
type CancelParams struct {
	// ID is the request id to cancel.
	ID interface{} `json:"id"` // int32 | string
}

// ProgressParams params of Progress netification.
//
// @since 3.15.0.
type ProgressParams struct {
	// Token is the progress token provided by the client or server.
	Token ProgressToken `json:"token"`

	// Value is the progress data.
	Value interface{} `json:"value"`
}

// ProgressToken is the progress token provided by the client or server.
//
// @since 3.15.0.
type ProgressToken struct {
	name   string
	number int32
}

// compile time check whether the ProgressToken implements a fmt.Formatter, fmt.Stringer, json.Marshaler and json.Unmarshaler interfaces.
var (
	_ fmt.Formatter    = (*ProgressToken)(nil)
	_ fmt.Stringer     = (*ProgressToken)(nil)
	_ json.Marshaler   = (*ProgressToken)(nil)
	_ json.Unmarshaler = (*ProgressToken)(nil)
)

// NewProgressToken returns a new ProgressToken.
func NewProgressToken(s string) *ProgressToken {
	return &ProgressToken{name: s}
}

// NewNumberProgressToken returns a new number ProgressToken.
func NewNumberProgressToken(n int32) *ProgressToken {
	return &ProgressToken{number: n}
}

// Format writes the ProgressToken to the formatter.
//
// If the rune is q the representation is non ambiguous,
// string forms are quoted.
func (v ProgressToken) Format(f fmt.State, r rune) {
	const numF = `%d`
	strF := `%s`
	if r == 'q' {
		strF = `%q`
	}

	switch {
	case v.name != "":
		fmt.Fprintf(f, strF, v.name)
	default:
		fmt.Fprintf(f, numF, v.number)
	}
}

// String returns a string representation of the ProgressToken.
func (v ProgressToken) String() string {
	return fmt.Sprint(v)
}

// MarshalJSON implements json.Marshaler.
func (v *ProgressToken) MarshalJSON() ([]byte, error) {
	if v.name != "" {
		return json.Marshal(v.name)
	}

	return json.Marshal(v.number)
}

// UnmarshalJSON implements json.Unmarshaler.
func (v *ProgressToken) UnmarshalJSON(data []byte) error {
	*v = ProgressToken{}
	if err := json.Unmarshal(data, &v.number); err == nil {
		return nil
	}

	return json.Unmarshal(data, &v.name)
}
