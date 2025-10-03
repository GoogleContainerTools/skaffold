// Copyright 2024 Google LLC. All Rights Reserved.
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

// Package storage provides implementations and shared components for tessera storage backends.
package storage

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/transparency-dev/tessera"
	"github.com/transparency-dev/tessera/internal/future"
)

// Queue knows how to queue up a number of entries in order.
//
// When the buffered queue grows past a defined size, or the age of the oldest entry in the
// queue reaches a defined threshold, the queue will call a provided FlushFunc with
// a slice containing all queued entries in the same order as they were added.
type Queue struct {
	maxSize uint
	maxAge  time.Duration

	timer *time.Timer
	work  chan []queueItem

	mu    sync.Mutex
	items []queueItem
}

// FlushFunc is the signature of a function which will receive the slice of queued entries.
// Normally, this function would be provided by storage implementations. It's important to note
// that the implementation MUST call each entry's MarshalBundleData function before attempting
// to integrate it into the tree.
// See the comment on Entry.MarshalBundleData for further info.
type FlushFunc func(ctx context.Context, entries []*tessera.Entry) error

// NewQueue creates a new queue with the specified maximum age and size.
//
// The provided FlushFunc will be called with a slice containing the contents of the queue, in
// the same order as they were added, when either the oldest entry in the queue has been there
// for maxAge, or the size of the queue reaches maxSize.
func NewQueue(ctx context.Context, maxAge time.Duration, maxSize uint, f FlushFunc) *Queue {
	q := &Queue{
		maxSize: maxSize,
		maxAge:  maxAge,
		work:    make(chan []queueItem, 1),
		items:   make([]queueItem, 0, maxSize),
	}

	// Spin off a worker thread to write the queue flushes to storage.
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case entries := <-q.work:
				q.doFlush(ctx, f, entries)
			}
		}
	}(ctx)
	return q
}

// Add places e into the queue, and returns a func which should be called to retrieve the assigned index.
func (q *Queue) Add(ctx context.Context, e *tessera.Entry) tessera.IndexFuture {
	qi := newEntry(e)

	q.mu.Lock()

	q.items = append(q.items, qi)

	// If this is the first item, start the timer.
	if len(q.items) == 1 {
		q.timer = time.AfterFunc(q.maxAge, q.flush)
	}

	// If we've reached max size, flush.
	var itemsToFlush []queueItem
	if len(q.items) >= int(q.maxSize) {
		itemsToFlush = q.flushLocked()
	}
	q.mu.Unlock()

	if itemsToFlush != nil {
		q.work <- itemsToFlush
	}

	return qi.f
}

// flush is called by the timer to flush the buffer.
func (q *Queue) flush() {
	q.mu.Lock()
	itemsToFlush := q.flushLocked()
	q.mu.Unlock()

	if itemsToFlush != nil {
		q.work <- itemsToFlush
	}
}

// flushLocked must be called with q.mu held.
// It prepares items for flushing and returns them.
func (q *Queue) flushLocked() []queueItem {
	if len(q.items) == 0 {
		return nil
	}

	if q.timer != nil {
		q.timer.Stop()
		q.timer = nil
	}

	itemsToFlush := q.items
	q.items = make([]queueItem, 0, q.maxSize)

	return itemsToFlush
}

// doFlush handles the queue flush, and sending notifications of assigned log indices.
func (q *Queue) doFlush(ctx context.Context, f FlushFunc, entries []queueItem) {
	ctx, span := tracer.Start(ctx, "tessera.storage.queue.doFlush")
	defer span.End()

	entriesData := make([]*tessera.Entry, 0, len(entries))
	for _, e := range entries {
		entriesData = append(entriesData, e.entry)
	}

	err := f(ctx, entriesData)

	// Send assigned indices to all the waiting Add() requests
	for _, e := range entries {
		e.notify(err)
	}
}

// queueItem represents an in-flight queueItem in the queue.
//
// The f field acts as a future for the queueItem's assigned index/error, and will
// hang until assign is called.
type queueItem struct {
	entry *tessera.Entry
	f     tessera.IndexFuture
	set   func(tessera.Index, error)
}

// newEntry creates a new entry for the provided data.
func newEntry(data *tessera.Entry) queueItem {
	f, set := future.NewFutureErr[tessera.Index]()
	e := queueItem{
		entry: data,
		f:     f.Get,
		set:   set,
	}
	return e
}

// notify sets the assigned log index (or an error) to the entry.
//
// This func must only be called once, and will cause any current or future callers of index()
// to be given the values provided here.
func (e *queueItem) notify(err error) {
	if e.entry.Index() == nil && err == nil {
		panic(errors.New("logic error: flush complete without error, but entry was not assigned an index - did storage fail to call entry.MarshalBundleData?"))
	}
	var idx uint64
	if e.entry.Index() != nil {
		idx = *e.entry.Index()
	}
	e.set(tessera.Index{Index: idx}, err)
}
