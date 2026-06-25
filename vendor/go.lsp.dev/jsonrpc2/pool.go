// Copyright 2026 The Go Language Server Authors. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package jsonrpc2

import "sync"

// encodeBufInitCap is the initial capacity of a pooled encode buffer. It is
// sized to hold a typical small envelope without a reallocation.
const encodeBufInitCap = 256

// encodeBufMaxCap bounds the capacity of buffers returned to the pool, so that
// an occasional very large message does not keep a large buffer pinned for the
// lifetime of the process.
const encodeBufMaxCap = 1 << 16

// encodeBufPool recycles the byte buffers used to append message envelopes.
var encodeBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, encodeBufInitCap)
		return &b
	},
}

// getEncodeBuf returns a reset buffer from the pool.
func getEncodeBuf() *[]byte {
	bp := encodeBufPool.Get().(*[]byte)
	*bp = (*bp)[:0]
	return bp
}

// putEncodeBuf returns bp to the pool unless its backing array has grown beyond
// encodeBufMaxCap, in which case it is dropped so the oversized array can be
// collected.
func putEncodeBuf(bp *[]byte) {
	if cap(*bp) > encodeBufMaxCap {
		return
	}
	encodeBufPool.Put(bp)
}

// cloneBytes returns a copy of src in a freshly allocated, right-sized slice. A
// nil src yields a nil result so that "absent" stays distinguishable from
// "present but empty".
func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

// irPool recycles incomingRequest structs across the direct-return dispatch
// path. A pooled request is the per-request context, release token, replied
// flag, and request body in one allocation, so recycling it makes the
// synchronous dispatch path allocation-free.
var irPool = sync.Pool{
	New: func() any {
		return &incomingRequest{}
	},
}

// getIncomingRequest returns a reset request from the pool.
func getIncomingRequest() *incomingRequest {
	return irPool.Get().(*incomingRequest)
}

// putIncomingRequest resets every field of ir (mirroring putWaiter's
// discipline) and returns it to the pool. Once Put returns, ir belongs to the
// pool and must not be touched.
func putIncomingRequest(ir *incomingRequest) {
	resetIncomingRequest(ir)
	irPool.Put(ir)
}

// resetIncomingRequest zeroes every field of ir so a recycled request carries
// no trace of its previous life. Under the jsonrpc2poison build tag the
// request body is scribbled with loud sentinels instead of zeroed, so a
// handler that illegally retained the request observes the poison rather than
// silently reading a recycled request's data.
func resetIncomingRequest(ir *incomingRequest) {
	ir.parent = nil
	ir.realCtx = nil
	ir.realCancel = nil
	ir.request = Request{}
	ir.rel = releaser{}
	ir.id = ID{}
	ir.replied.done.Store(false)
	ir.isCall = false
	ir.canceled = false
	poisonRequest(&ir.request)
}
