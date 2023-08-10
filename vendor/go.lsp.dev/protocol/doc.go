// SPDX-FileCopyrightText: 2019 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

// Package protocol implements Language Server Protocol specification in Go.
//
// This package contains the structs that map directly to the wire format
// of the Language Server Protocol.
//
// It is a literal transcription, with unmodified comments, and only the changes
// required to make it Go code.
//
// - Names are uppercased to export them.
//
// - All fields have JSON tags added to correct the names.
//
// - Fields marked with a ? are also marked as "omitempty".
//
// - Fields that are "|| null" are made pointers.
//
// - Fields that are string or number are left as string.
//
// - Fields that are type "number" are made float64.
package protocol // import "go.lsp.dev/protocol"
