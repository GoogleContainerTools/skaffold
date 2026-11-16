// Copyright 2024 The Go Language Server Authors
// SPDX-License-Identifier: BSD-3-Clause

package protocol

import "go.lsp.dev/uri"

// URI is a Uniform Resource Identifier as defined by RFC 3986 and used by the
// LSP base protocol. It is transported as a JSON string.
//
// Generated URI and URI fields use [uri.URI] directly. This local URI
// type is retained as a narrow compatibility and sealed-union bridge where Go
// requires a package-local receiver type for generated marker methods.
// Prefer [uri.URI] for ordinary fields and convert with URI(u) only when a
// generated union arm such as [RelativePatternBaseURI] needs the local bridge.
type URI uri.URI
