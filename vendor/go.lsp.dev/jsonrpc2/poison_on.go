// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

//go:build jsonrpc2poison

package jsonrpc2

// requestPoisonMethod is the sentinel a retained request's Method reports
// after the request has been returned to the pool under the poison build.
const requestPoisonMethod = "jsonrpc2: POISONED: request retained after handler return"

// requestPoisonParams is the sentinel a retained request's Params reports
// after the request has been returned to the pool under the poison build.
var requestPoisonParams = RawMessage(`"jsonrpc2: POISONED: request retained after handler return"`)

// poisonRequest scribbles loud sentinels into a pooled request body. A handler
// that illegally retained the request past its return observes these values,
// turning a silent read of recycled data into a loud, attributable failure.
func poisonRequest(r *Request) {
	r.method = requestPoisonMethod
	r.params = requestPoisonParams
}
