// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

//go:build !jsonrpc2poison

package jsonrpc2

// poisonRequest is a no-op in normal builds. Under the jsonrpc2poison build
// tag it scribbles loud sentinel values into a pooled request so that illegal
// retention past handler return is observed as poison instead of silently
// reading another request's data.
func poisonRequest(*Request) {}
