# jsonrpc2

[![test][test-badge]][test]
[![pkg.go.dev][pkg.go.dev-badge]][pkg.go.dev]
[![Go module][module-badge]][module]
[![codecov.io][codecov-badge]][codecov]

Package `jsonrpc2` is a fast, allocation-conscious implementation of the
[JSON-RPC 2.0](https://www.jsonrpc.org/specification) wire protocol for Go,
designed for Language Server Protocol (LSP) and Model Context Protocol (MCP)
style transports.

## Overview

The library is built around a reflection-free wire core plus pluggable framing,
a swappable payload codec, and a bidirectional connection state machine:

- **Reflection-free envelope.** Message envelopes are encoded by appending
  directly into a pooled byte buffer and decoded with a single-pass span
  scanner, so the hot path performs no reflection and copies each payload at most
  once. Reflection is confined to the user payload (`params` / `result`) and only
  through a swappable codec.
- **Symmetric peer connection.** A `Conn` is a bidirectional peer that can both
  issue and answer calls, notifications, responses, and errors — the shape LSP
  requires.
- **Batch, cancellation, graceful shutdown.** JSON-RPC 2.0 batch arrays,
  per-request cancellation, and idle-detecting graceful close are all supported.
- **Two wire framings and a pluggable codec.** Newline-delimited JSON (MCP
  stdio) and LSP `Content-Length` header framing; the payload codec defaults to
  `encoding/json/v2` and can be swapped for opt-in `sonic` or `goccy` codecs.
- **Explicit fast-path modes.** `Conn`/`Peer` stay bidirectional; `SingleClient`
  is the serialized caller-owned-read-loop path; `PipelineClient` is a
  concurrent client-only mode; `BatchClient` exposes raw frame batch I/O.

## Install

```sh
go get go.lsp.dev/jsonrpc2@latest
```

The module requires Go 1.26 or later. The importable core depends only on
[`github.com/go-json-experiment/json`](https://github.com/go-json-experiment/json)
(encoding/json/v2); no assembly, JIT, or heavy transitive dependencies enter the
core module graph.

## Runtime modes and borrowed views

Choose the smallest mode that matches the workload; the fast paths are not all
interchangeable:

- `NewConn` / `NewPeer`: bidirectional JSON-RPC peer mode. Use it for LSP-style
  connections where either side may send calls, notifications, responses, and
  server-initiated requests.
- `NewSingleClient` / `NewSyncClient`: serialized single-flight client mode. The
  caller owns the read loop for each `Call`, so it avoids the background-reader
  hand-off, but only one call may be outstanding and server-initiated requests
  are not dispatched.
- `NewPipelineClient`: concurrent client-only mode. It uses generated numeric
  IDs, dense wait slots, pooled waiters, and a canonical success-response scanner
  for the common response shape. Start its response reader with `Go` once; use
  `Conn`/`Peer` instead when the remote can initiate requests.
- `NewBatchClient`: raw-frame batch mode for callers that already build a JSON
  batch array and want to write/read one frame without routing each member
  through `Conn.Call`.

For parser-only experimental fast paths, `ScanMessageView`, `ScanFrameView`, and
`AppendRequestViews` return borrowed spans over caller-owned frame bytes. Those
views are valid only while the source frame remains valid and unmodified; call
`Clone`/`Owned` before retaining data beyond the callback or read iteration. They
are not a default replacement for `DecodeMessage`/`ParseRequests`: the current
corpus proves zero allocations and targeted wins, but not a universal ns/op win
across invalid and small single-message inputs.

## Quickstart

### Client: issue a call

```go
package main

import (
	"context"
	"log"
	"net"

	"go.lsp.dev/jsonrpc2"
)

func main() {
	ctx := context.Background()

	nc, err := net.Dial("tcp", "127.0.0.1:4389")
	if err != nil {
		log.Fatal(err)
	}

	// NewStream uses the LSP Content-Length header framing by default.
	conn := jsonrpc2.NewConn(jsonrpc2.NewStream(nc))
	conn.Go(ctx, jsonrpc2.MethodNotFoundHandler)
	defer conn.Close()

	type hoverParams struct {
		URI  string `json:"uri"`
		Line int    `json:"line"`
	}
	type hoverResult struct {
		Contents string `json:"contents"`
	}

	var res hoverResult
	if _, err := conn.Call(ctx, "textDocument/hover",
		hoverParams{URI: "file:///a.go", Line: 12}, &res); err != nil {
		log.Fatal(err)
	}
	log.Printf("hover: %s", res.Contents)

	// Notifications are fire-and-forget: no id, no response.
	if err := conn.Notify(ctx, "textDocument/didSave", hoverParams{URI: "file:///a.go"}); err != nil {
		log.Fatal(err)
	}
}
```

### Server: HandlerServer + Serve

A `Handler` answers each incoming request by calling `reply` exactly once for a
call. `HandlerServer` adapts a `Handler` into a `StreamServer`, and `Serve`
accepts connections from a `net.Listener`, driving each on its own goroutine.

```go
package main

import (
	"context"
	"log"
	"net"

	"go.lsp.dev/jsonrpc2"
)

func handler(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
	switch req.Method() {
	case "textDocument/hover":
		// Decode params, do work, reply with a typed result.
		return reply(ctx, map[string]string{"contents": "hello"}, nil)
	default:
		// Answer unknown calls with the standard error.
		return reply(ctx, nil, jsonrpc2.ErrMethodNotFound)
	}
}

func main() {
	ctx := context.Background()

	ln, err := net.Listen("tcp", "127.0.0.1:4389")
	if err != nil {
		log.Fatal(err)
	}

	server := jsonrpc2.HandlerServer(handler)

	// idleTimeout = 0 means "serve until ctx is canceled or accept fails".
	if err := jsonrpc2.Serve(ctx, ln, server, 0); err != nil {
		log.Fatal(err)
	}
}
```

`ListenAndServe(ctx, network, addr, server, idleTimeout)` is a convenience that
creates the listener for you (and removes the socket file for a `unix` network).

A request handler runs inline on the read goroutine by default, so handlers
observe requests in wire order. A handler that needs to overlap later requests
(a long-running call, or a server that calls back into the same connection) must
release itself with `jsonrpc2.Async(ctx)` or be wrapped with
`jsonrpc2.AsyncHandler`. `jsonrpc2.CancelHandler` adds cancellation by request
id.

## Framing options

A `Stream` adapts a byte transport (`io.ReadWriteCloser`) to message reads and
writes. Two framings are provided:

| Constructor | Framing | Compatible with |
|-------------|---------|-----------------|
| `NewStream` / `NewHeaderStream` | `Content-Length` header block then body | LSP, gopls |
| `NewNDJSONStream` / `NewRawStream` | one JSON value per line (`\n`-delimited) | MCP stdio transport |

```go
// LSP header framing (the gopls-compatible default).
conn := jsonrpc2.NewConn(jsonrpc2.NewStream(rwc))

// Newline-delimited JSON framing (MCP-compatible).
conn := jsonrpc2.NewConn(jsonrpc2.NewNDJSONStream(rwc))
```

Both framings write each message with a single `Write` (header and body, or
payload and its newline, are composed into one pooled buffer), avoiding the
two-syscall-per-message pattern.

## Pluggable codec

The envelope is never routed through a codec; only the user payload (`params`
and `result`) is. The payload `Codec` is swappable per connection and defaults to
`encoding/json/v2`:

```go
// Default: encoding/json/v2 (pure Go, all platforms, no JIT/asm).
conn := jsonrpc2.NewConn(stream)

// Opt-in faster codecs live in separate modules so their dependencies never
// enter the core module graph:
import sonic "go.lsp.dev/jsonrpc2/codec/sonic"
conn := jsonrpc2.NewConn(stream, jsonrpc2.WithCodec(sonic.Codec{}))

import goccy "go.lsp.dev/jsonrpc2/codec/goccy"
conn := jsonrpc2.NewConn(stream, jsonrpc2.WithCodec(goccy.Codec{}))
```

A `RawMessage` passed as params/result, or decoded into a `*RawMessage`, bypasses
the codec entirely and is carried verbatim.

## Performance

`jsonrpc2` is benchmarked head-to-head against
[`github.com/creachadair/jrpc2`](https://github.com/creachadair/jrpc2) and the
[`github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2`](https://github.com/modelcontextprotocol/go-sdk/tree/main/internal/jsonrpc2), using the harness in
[`internal/benchmark`](./internal/benchmark). The harness holds the workload
constant, reports two transport families, and records raw `benchstat` artifacts
for each optimization round.

Lower is better. Current Round 5 final artifacts:

- linux/amd64: `internal/benchmark/artifacts/20260610T113339Z-g011-linux-amd64-final`
  (`go1.26.4`, Debian 13, Intel Xeon Platinum 8481C, `-count=10`).
- darwin/arm64: `internal/benchmark/artifacts/20260610T111717Z-combined-regression-gate`
  (`go1.26.4`, Apple M3 Max, `-count=10`).

### Current linux/amd64 headline rows

| Workload | jsonrpc2 | jrpc2 | go-sdk | Winner |
|---|---:|---:|---:|---|
| Void round trip, native | **3.37 us / 657 B / 11 allocs** | 13.10 us / 4478 B / 100 | 34.08 us / 67980 B / 46 | jsonrpc2 |
| Void round trip, common | **3.33 us / 657 B / 11 allocs** | 18.87 us / 4572 B / 102 | 33.99 us / 67973 B / 46 | jsonrpc2 |
| Parallel void P4, native | **3.87 us / 658 B / 11 allocs** | 13.22 us / 4486 B / 100 | 31.24 us / 67896 B / 44 | jsonrpc2 |
| Parallel void P12, native | **3.91 us / 658 B / 11 allocs** | 13.83 us / 4488 B / 100 | 29.93 us / 67835 B / 44 | jsonrpc2 |
| Params small, native | **3.55 us / 809 B / 13 allocs** | 14.29 us / 4883 B / 108 | 33.90 us / 68279 B / 50 | jsonrpc2 |
| Params medium, native | **3.98 us / 1210 B / 13 allocs** | 18.89 us / 5678 B / 109 | 36.56 us / 69484 B / 50 | jsonrpc2 |
| Params large, native | **12.95 us / 9608 B / 13 allocs** | 96.63 us / 22530 B / 109 | 78.73 us / 97636 B / 53 | jsonrpc2 |
| Notify, native | **1.00 us / 276 B / 4 allocs** | 4.26 us / 2072 B / 42 | 12.22 us / 33831 B / 20 | jsonrpc2 |

The same linux/amd64 artifact records every affected `common` row as a jsonrpc2
win as well. The cross-library check in `internal-affected-fastest-check.txt`
shows jsonrpc2 fastest by sec/op mean on every affected apples-to-apples
native/common row.

### Current darwin/arm64 headline rows

| Workload | jsonrpc2 | jrpc2 | go-sdk | Winner |
|---|---:|---:|---:|---|
| Void round trip, native | **1.77 us / 656 B / 11 allocs** | 7.73 us / 4469 B / 100 | 17.26 us / 67820 B / 46 | jsonrpc2 |
| Void round trip, common | **1.78 us / 656 B / 11 allocs** | 9.84 us / 4565 B / 102 | 18.81 us / 67819 B / 46 | jsonrpc2 |
| Params large, native | **6.93 us / 9602 B / 13 allocs** | 63.748 us / 22600 B / 109 | 46.583 us / 98149 B / 54 | jsonrpc2 |
| Notify, native | **0.61 us / 276 B / 4 allocs** | 2.61 us / 2068 B / 42 | 7.11 us / 33801 B / 20 | jsonrpc2 |

Root repository benchmarks use the ordinary `Conn` over stream transports and
show the server/client hot path allocation floor independently of the comparative
harness. On darwin/arm64, root `BenchmarkVoidRoundTrip` is **2.885 us / 324 B /
4 allocs**, down from 6 allocs before the Round 5 direct-unmarshal survivor; the
high-inflight rows were statistically neutral in the combined gate. On
linux/amd64, the reverse-order root rerun is neutral or improved on sec/op and
reduces every root affected row from 6 to **4 allocs/op**.

### Transport-family disclosure

The `native` family is each implementation's fastest in-memory transport. Round
5 adds `jsonrpc2.NewChannelStreamPair`, so `jsonrpc2/native` now uses a bounded
in-memory encoded-frame channel stream. It still performs JSON encode/decode and
frame scanning; `jrpc2/native` remains `server.NewLocal` / `channel.Direct`,
which passes message buffers in memory with no framing. The `common` family keeps
all three libraries on the same `net.Pipe` + NDJSON-style framing path.

The channel stream copies queued frames to preserve single-owner buffer semantics.
That is why small/void/notify harness rows allocate more than the old net.Pipe
native rows (`RoundTripVoid` moved from 8 to 11 harness allocs/op), even though
wall-clock time improves sharply. Root `Conn` allocation counts improve because
the direct-unmarshal survivor removes response allocation on the ordinary stream
path.

### Caveats

- Keep claims artifact-scoped: quote the raw artifact path, host, Go version,
  GOOS/GOARCH, command, and transport family with any number.
- Batch rows are excluded from the lowest-cost-on-every-workload claim because
  the three libraries expose different batch mechanics.
- Standalone `DecodeMessage` and `ParseRequests` intentionally own their returned
  bytes. Their allocation floor is documented in
  [`internal/benchmark/RESULTS.md`](./internal/benchmark/RESULTS.md); connection
  round trips use a separate fast path.

The full methodology, per-workload tables, keep/kill ledger, and reproduction
commands live in [`internal/benchmark/RESULTS.md`](./internal/benchmark/RESULTS.md).

## License

BSD-3-Clause. See [LICENSE](./LICENSE).


<!-- badge links -->
[test]: https://github.com/go-language-server/jsonrpc2/actions/workflows/test.yaml
[pkg.go.dev]: https://pkg.go.dev/go.lsp.dev/jsonrpc2
[module]: https://github.com/go-language-server/jsonrpc2/releases/latest
[codecov]: https://app.codecov.io/gh/go-language-server/jsonrpc2

[test-badge]: https://img.shields.io/github/actions/workflow/status/go-language-server/jsonrpc2/test.yaml?branch=main&style=for-the-badge&label=TEST&logo=github
[pkg.go.dev-badge]: https://img.shields.io/badge/pkg.go.dev-doc-00add8?style=for-the-badge&logo=go
[module-badge]: https://img.shields.io/github/release/go-language-server/jsonrpc2.svg?color=00add8&label=MODULE&style=for-the-badge&logo=go
[codecov-badge]: https://img.shields.io/codecov/c/github/go-language-server/jsonrpc2/main?logo=codecov&style=for-the-badge
