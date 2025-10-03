// Copyright 2024 The Tessera authors. All Rights Reserved.
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

package tessera

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/transparency-dev/tessera/internal/parse"
	"k8s.io/klog/v2"
)

// NewPublicationAwaiter provides an PublicationAwaiter that can be cancelled
// using the provided context. The PublicationAwaiter will poll every `pollPeriod`
// to fetch checkpoints using the `readCheckpoint` function.
func NewPublicationAwaiter(ctx context.Context, readCheckpoint func(ctx context.Context) ([]byte, error), pollPeriod time.Duration) *PublicationAwaiter {
	a := &PublicationAwaiter{
		c: sync.NewCond(&sync.Mutex{}),
	}
	go a.pollLoop(ctx, readCheckpoint, pollPeriod)
	return a
}

// PublicationAwaiter allows client threads to block until a leaf is published.
// This means it has a sequence number, and been integrated into the tree, and
// a checkpoint has been published for it.
// A single long-lived PublicationAwaiter instance
// should be reused for all requests in the application code as there is some
// overhead to each one; the core of an PublicationAwaiter is a poll loop that
// will fetch checkpoints whenever it has clients waiting.
//
// The expected call pattern is:
//
// i, cp, err := awaiter.Await(ctx, storage.Add(myLeaf))
//
// When used this way, it requires very little code at the point of use to
// block until the new leaf is integrated into the tree.
type PublicationAwaiter struct {
	c *sync.Cond

	// size, checkpoint, and err keep track of the latest size and checkpoint
	// (or error) seen by the poller.
	size       uint64
	checkpoint []byte
	err        error
}

// Await blocks until the IndexFuture is resolved, and this new index has been
// integrated into the log, i.e. the log has made a checkpoint available that
// commits to this new index. When this happens, Await returns the index at
// which the leaf has been added, and a checkpoint that commits to this index.
//
// This operation can be aborted early by cancelling the context. In this event,
// or in the event that there is an error getting a valid checkpoint, an error
// will be returned from this method.
func (a *PublicationAwaiter) Await(ctx context.Context, future IndexFuture) (Index, []byte, error) {
	_, span := tracer.Start(ctx, "tessera.Await")
	defer span.End()

	i, err := future()
	if err != nil {
		return i, nil, err
	}

	a.c.L.Lock()
	defer a.c.L.Unlock()
	for (a.size <= i.Index && a.err == nil) && ctx.Err() == nil {
		a.c.Wait()
	}
	// Ensure we propogate context done error, if any.
	if err := ctx.Err(); err != nil {
		a.err = err
	}
	return i, a.checkpoint, a.err
}

// pollLoop MUST be called in a goroutine when constructing an PublicationAwaiter
// and will run continually until its context is cancelled. It wakes up every
// `pollPeriod` to check if there are clients blocking. If there are, it requests
// the latest checkpoint from the log, parses the tree size, and releases all clients
// that were blocked on an index smaller than this tree size.
func (a *PublicationAwaiter) pollLoop(ctx context.Context, readCheckpoint func(ctx context.Context) ([]byte, error), pollPeriod time.Duration) {
	var (
		cp     []byte
		cpErr  error
		cpSize uint64
	)
	for done := false; !done; {
		select {
		case <-ctx.Done():
			klog.Info("PublicationAwaiter exiting due to context completion")
			cp, cpSize, cpErr = nil, 0, ctx.Err()
			done = true
		case <-time.After(pollPeriod):
			cp, cpErr = readCheckpoint(ctx)
			switch {
			case errors.Is(cpErr, os.ErrNotExist):
				continue
			case cpErr != nil:
				cpSize = 0
			default:
				_, cpSize, _, cpErr = parse.CheckpointUnsafe(cp)
			}
		}

		a.c.L.Lock()
		// Note that for now, this releases all clients in the event of a single failure.
		// If this causes problems, this could be changed to attempt retries.
		a.checkpoint = cp
		a.size = cpSize
		a.err = cpErr
		a.c.Broadcast()
		a.c.L.Unlock()
	}
}
