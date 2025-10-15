// Copyright 2025 The Tessera Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package stream provides support for streaming contiguous entries from logs.
package client

import (
	"context"
	"fmt"
	"iter"

	"github.com/transparency-dev/tessera/api/layout"
	"k8s.io/klog/v2"
)

// TreeSizeFunc is a function which knows how to return the current tree size of a log.
type TreeSizeFunc func(ctx context.Context) (uint64, error)

// Bundle represents an entry bundle in a log, along with some metadata about which parts of the bundle
// are relevent.
type Bundle struct {
	// RangeInfo decribes which of the entries in this bundle are relevent.
	RangeInfo layout.RangeInfo
	// Data is the raw serialised bundle, as fetched from the log.
	//
	// For a tlog-tiles compliant log, this can be unmarshaled using api.EntryBundle.
	Data []byte
}

// EntryBundles produces an iterator which returns a stream of Bundle structs which cover the requested range of entries in their natural order in the log.
//
// If the adaptor encounters an error while reading an entry bundle, the encountered error will be returned via the iterator.
//
// This adaptor is optimised for the case where calling getBundle has some appreciable latency, and works
// around that by maintaining a read-ahead cache of subsequent bundles which is populated a number of parallel
// requests to getBundle. The request parallelism is set by the value of the numWorkers paramemter, which can be tuned
// to balance throughput against consumption of resources, but such balancing needs to be mindful of the nature of the
// source infrastructure, and how concurrent requests affect performance (e.g. GCS buckets vs. files on a single disk).
func EntryBundles(ctx context.Context, numWorkers uint, getSize TreeSizeFunc, getBundle EntryBundleFetcherFunc, fromEntry uint64, N uint64) iter.Seq2[Bundle, error] {
	ctx, span := tracer.Start(ctx, "tessera.storage.StreamAdaptor")
	defer span.End()

	// bundleOrErr represents a fetched entry bundle and its params, or an error if we couldn't fetch it for
	// some reason.
	type bundleOrErr struct {
		b   Bundle
		err error
	}

	// bundles will be filled with futures for in-order entry bundles by the worker
	// go routines below.
	// This channel will be drained by the loop at the bottom of this func which
	// yields the bundles to the caller.
	bundles := make(chan func() bundleOrErr, numWorkers)
	exit := make(chan struct{})

	// Fetch entry bundle resources in parallel.
	// We use a limited number of tokens here to prevent this from
	// consuming an unbounded amount of resources.
	go func() {
		ctx, span := tracer.Start(ctx, "tessera.storage.StreamAdaptorWorker")
		defer span.End()

		defer close(bundles)

		treeSize, err := getSize(ctx)
		if err != nil {
			bundles <- func() bundleOrErr { return bundleOrErr{err: err} }
			return
		}

		// We'll limit ourselves to numWorkers worth of on-going work using these tokens:
		tokens := make(chan struct{}, numWorkers)
		for range numWorkers {
			tokens <- struct{}{}
		}

		klog.V(1).Infof("stream.EntryBundles: streaming [%d, %d)", fromEntry, fromEntry+N)

		// For each bundle, pop a future into the bundles channel and kick off an async request
		// to resolve it.
		for ri := range layout.Range(fromEntry, fromEntry+N, treeSize) {
			select {
			case <-exit:
				return
			case <-tokens:
				// We'll return a token below, once the bundle is fetched _and_ is being yielded.
			}

			c := make(chan bundleOrErr, 1)
			go func(ri layout.RangeInfo) {
				b, err := getBundle(ctx, ri.Index, ri.Partial)
				c <- bundleOrErr{b: Bundle{RangeInfo: ri, Data: b}, err: err}
			}(ri)

			f := func() bundleOrErr {
				b := <-c
				// We're about to yield a value, so we can now return the token and unblock another fetch.
				tokens <- struct{}{}
				return b
			}

			bundles <- f
		}

		klog.V(1).Infof("stream.EntryBundles: exiting")
	}()

	return func(yield func(Bundle, error) bool) {
		defer close(exit)

		for f := range bundles {
			b := f()
			if !yield(b.b, b.err) {
				return
			}
			// For now, force the iterator to stop if we've just returned an error.
			// If there's a good reason to allow it to continue we can change this.
			if b.err != nil {
				return
			}
		}
		klog.V(1).Infof("stream.EntryBundles: iter done")
	}
}

// Entry represents a single leaf in a log.
type Entry[T any] struct {
	// Index is the index of the entry in the log.
	Index uint64
	// Entry is the entry from the log.
	Entry T
}

// Entries consumes an iterator of Bundle structs and transforms it using the provided unbundle function, and returns an iterator over the transformed data.
//
// Different unbundle implementations can be provided to return raw entry bytes, parsed entry structs, or derivations of entries (e.g. hashes) as needed.
func Entries[T any](bundles iter.Seq2[Bundle, error], unbundle func([]byte) ([]T, error)) iter.Seq2[Entry[T], error] {
	return func(yield func(Entry[T], error) bool) {
		for b, err := range bundles {
			if err != nil {
				yield(Entry[T]{}, err)
				return
			}
			es, err := unbundle(b.Data)
			if err != nil {
				yield(Entry[T]{}, err)
				return
			}
			if len(es) <= int(b.RangeInfo.First) {
				yield(Entry[T]{}, fmt.Errorf("logic error: First is %d but only %d entries", b.RangeInfo.First, len(es)))
				return
			}
			es = es[b.RangeInfo.First:]
			if len(es) > int(b.RangeInfo.N) {
				es = es[:b.RangeInfo.N]
			}

			rIdx := b.RangeInfo.Index*layout.EntryBundleWidth + uint64(b.RangeInfo.First)
			for i, e := range es {
				if !yield(Entry[T]{Index: rIdx + uint64(i), Entry: e}, nil) {
					return
				}
			}
		}
	}
}
