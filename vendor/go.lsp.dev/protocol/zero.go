// Copyright 2026 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import "reflect"

// isZeroOmitValue reports whether v is the Go zero value for generated
// omitzero checks that cannot use a cheaper type-specific guard.
func isZeroOmitValue(v any) bool {
	if v == nil {
		return true
	}
	if z, ok := v.(interface{ IsZero() bool }); ok {
		return z.IsZero()
	}
	return reflect.ValueOf(v).IsZero()
}
