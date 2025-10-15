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

// Package gcp contains a GCP-based antispam implementation for Tessera.
//
// A Spanner database provides a mechanism for maintaining an index of
// hash --> log position for detecting duplicate submissions.
package gcp

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"sync/atomic"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"cloud.google.com/go/spanner/apiv1/spannerpb"

	"github.com/transparency-dev/tessera"
	"github.com/transparency-dev/tessera/client"
	"google.golang.org/grpc/codes"
	"k8s.io/klog/v2"
)

const (
	DefaultMaxBatchSize      = 1500
	DefaultPushbackThreshold = 2048
)

var errPushback = fmt.Errorf("antispam %w", tessera.ErrPushback)

// AntispamOpts allows configuration of some tunable options.
type AntispamOpts struct {
	// MaxBatchSize is the largest number of mutations permitted in a single BatchWrite operation when
	// updating the antispam index.
	//
	// Larger batches can enable (up to a point) higher throughput, but care should be taken not to
	// overload the Spanner instance.
	//
	// During testing, we've found that 1500 appears to offer maximum throughput when using Spanner instances
	// with 300 or more PU. Smaller deployments (e.g. 100 PU) will likely perform better with smaller batch
	// sizes of around 64.
	MaxBatchSize uint

	// PushbackThreshold allows configuration of when to start responding to Add requests with pushback due to
	// the antispam follower falling too far behind.
	//
	// When the antispam follower is at least this many entries behind the size of the locally integrated tree,
	// the antispam decorator will return a wrapped tessera.ErrPushback for every Add request.
	PushbackThreshold uint
}

// NewAntispam returns an antispam driver which uses Spanner to maintain a mapping of
// previously seen entries and their assigned indices.
//
// Note that the storage for this mapping is entirely separate and unconnected to the storage used for
// maintaining the Merkle tree.
//
// This functionality is experimental!
func NewAntispam(ctx context.Context, spannerDB string, opts AntispamOpts) (*AntispamStorage, error) {
	if opts.MaxBatchSize == 0 {
		opts.MaxBatchSize = DefaultMaxBatchSize
	}
	if opts.PushbackThreshold == 0 {
		opts.PushbackThreshold = DefaultPushbackThreshold
	}
	if err := createAndPrepareTables(
		ctx, spannerDB,
		[]string{
			"CREATE TABLE IF NOT EXISTS FollowCoord (id INT64 NOT NULL, nextIdx INT64 NOT NULL) PRIMARY KEY (id)",
			"CREATE TABLE IF NOT EXISTS IDSeq (h BYTES(32) NOT NULL, idx INT64 NOT NULL) PRIMARY KEY (h)",
		},
		[][]*spanner.Mutation{
			{spanner.Insert("FollowCoord", []string{"id", "nextIdx"}, []any{0, 0})},
		},
	); err != nil {
		return nil, fmt.Errorf("failed to create tables: %v", err)
	}

	db, err := spanner.NewClient(ctx, spannerDB)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Spanner: %v", err)
	}

	r := &AntispamStorage{
		opts:   opts,
		dbPool: db,
	}

	return r, nil
}

type AntispamStorage struct {
	opts AntispamOpts

	dbPool *spanner.Client

	// pushBack is used to prevent the follower from getting too far underwater.
	// Populate dynamically will set this to true/false based on how far behind the follower is from the
	// currently integrated tree size.
	// When pushBack is true, the decorator will start returning a wrapped ErrPushback to all calls.
	pushBack atomic.Bool

	numLookups atomic.Uint64
	numWrites  atomic.Uint64
	numHits    atomic.Uint64
}

// index returns the index (if any) previously associated with the provided hash
func (d *AntispamStorage) index(ctx context.Context, h []byte) (*uint64, error) {
	ctx, span := tracer.Start(ctx, "tessera.antispam.gcp.index")
	defer span.End()

	d.numLookups.Add(1)
	var idx int64
	if row, err := d.dbPool.Single().ReadRow(ctx, "IDSeq", spanner.Key{h}, []string{"idx"}); err != nil {
		if c := spanner.ErrCode(err); c == codes.NotFound {
			span.AddEvent("tessera.miss")
			return nil, nil
		}
		return nil, err
	} else {
		if err := row.Column(0, &idx); err != nil {
			return nil, fmt.Errorf("failed to read antispam index: %v", err)
		}
		idx := uint64(idx)
		span.AddEvent("tessera.hit")
		d.numHits.Add(1)
		return &idx, nil
	}
}

// Decorator returns a function which will wrap an underlying Add delegate with
// code to dedup against the stored data.
func (d *AntispamStorage) Decorator() func(f tessera.AddFn) tessera.AddFn {
	return func(delegate tessera.AddFn) tessera.AddFn {
		return func(ctx context.Context, e *tessera.Entry) tessera.IndexFuture {
			ctx, span := tracer.Start(ctx, "tessera.antispam.gcp.Add")
			defer span.End()

			if d.pushBack.Load() {
				span.AddEvent("tessera.pushback")
				// The follower is too far behind the currently integrated tree, so we're going to push back against
				// the incoming requests.
				// This should have two effects:
				//   1. The tree will cease growing, giving the follower a chance to catch up, and
				//   2. We'll stop doing lookups for each submission, freeing up Spanner CPU to focus on catching up.
				//
				// We may decide in the future that serving duplicate reads is more important than catching up as quickly
				// as possible, in which case we'd move this check down below the call to index.
				return func() (tessera.Index, error) { return tessera.Index{}, errPushback }
			}
			idx, err := d.index(ctx, e.Identity())
			if err != nil {
				return func() (tessera.Index, error) { return tessera.Index{}, err }
			}
			if idx != nil {
				return func() (tessera.Index, error) { return tessera.Index{Index: *idx, IsDup: true}, nil }
			}

			return delegate(ctx, e)
		}
	}
}

// Follower returns a follower which knows how to populate the antispam index.
//
// This implements tessera.Antispam.
func (d *AntispamStorage) Follower(b func([]byte) ([][]byte, error)) tessera.Follower {
	f := &follower{
		as:           d,
		bundleHasher: b,
	}
	// Use the "normal" BatchWrite mechanism to update the antispam index.
	// This will be overriden by the test to use an "inline" mechanism since spannertest
	// does not support BatchWrite :(
	f.updateIndex = f.batchUpdateIndex

	if r := os.Getenv("SPANNER_EMULATOR_HOST"); r != "" {
		const warn = `H4sIAAAAAAAAA83VwRGAIAwEwH+qoFwrsEAr8eEDPZO7gxkcGV6G7IAJ2tr8iDp07Fs6J7BnImcK5J3EmHVIT2Dvp2YTVJMu/y1+X+jiFQ84LtK9mLHr0aqh+K15PwkWRDaPrcbU5WdMKILtCDMF5hSgQEdJlw/36D7eRYqPfsVNVBcMsNH2QQKq/p957Yr8RfWIE22t7L7ABwAA`
		r, _ := base64.StdEncoding.DecodeString(warn)
		gzr, _ := gzip.NewReader(bytes.NewReader([]byte(r)))
		w, _ := io.ReadAll(gzr)
		klog.Warningf("%s\nWarning: you're running under the Spanner emulator - this is not a supported environment!\n\n", string(w))

		// Hack in a workaround for spannertest not supporting BatchWrites
		f.updateIndex = emulatorWorkaroundUpdateIndexTx
	}

	return f
}

// follower is a struct which knows how to populate the antispam storage with identity hashes
// for entries in a log.
type follower struct {
	as *AntispamStorage

	// updateIndex knows how to apply the provided slice of mutations to the underlying Spanner DB.
	//
	// In normal operation this simply points to the batchUpdateIndex func below, but spannertest
	// does not support either:
	//   - BatchWrite operations, or
	//   - nested transactions
	// so we use this member as a hook to fallback to
	// a regular transaction for tests.
	updateIndex func(context.Context, *spanner.ReadWriteTransaction, []*spanner.Mutation) error

	bundleHasher func([]byte) ([][]byte, error)
}

func (f *follower) Name() string {
	return "GCP antispam"
}

// Follow uses entry data from the log to populate the antispam storage.
func (f *follower) Follow(ctx context.Context, lr tessera.LogReader) {
	errOutOfSync := errors.New("out-of-sync")

	t := time.NewTicker(time.Second)
	var (
		next func() (client.Entry[[]byte], error, bool)
		stop func()

		curEntries [][]byte
		curIndex   uint64
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		// logSize is the latest known size of the log we're following.
		// This will get initialised below, inside the loop.
		var logSize uint64

		// Busy loop while there are entries to be consumed from the stream
		for streamDone := false; !streamDone; {
			_, err := f.as.dbPool.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				ctx, span := tracer.Start(ctx, "tessera.antispam.gcp.FollowTxn")
				defer span.End()

				// Figure out the last entry we used to populate our antispam storage.
				row, err := txn.ReadRowWithOptions(ctx, "FollowCoord", spanner.Key{0}, []string{"nextIdx"}, &spanner.ReadOptions{LockHint: spannerpb.ReadRequest_LOCK_HINT_EXCLUSIVE})
				if err != nil {
					return err
				}

				var nextIdx int64 // Spanner doesn't support uint64
				if err := row.Columns(&nextIdx); err != nil {
					return fmt.Errorf("failed to read follow coordination info: %v", err)
				}
				span.SetAttributes(followFromKey.Int64(nextIdx))

				followFrom := uint64(nextIdx)
				if followFrom >= logSize {
					// Our view of the log is out of date, update it
					logSize, err = lr.IntegratedSize(ctx)
					if err != nil {
						streamDone = true
						return fmt.Errorf("populate: IntegratedSize(): %v", err)
					}
					switch {
					case followFrom > logSize:
						streamDone = true
						return fmt.Errorf("followFrom %d > size %d", followFrom, logSize)
					case followFrom == logSize:
						// We're caught up, so unblock pushback and go back to sleep
						streamDone = true
						f.as.pushBack.Store(false)
						return nil
					default:
						// size > followFrom, so there's more work to be done!
					}
				}

				pushback := logSize-followFrom > uint64(f.as.opts.PushbackThreshold)
				span.SetAttributes(pushbackKey.Bool(pushback))
				f.as.pushBack.Store(pushback)

				// If this is the first time around the loop we need to start the stream of entries now that we know where we want to
				// start reading from:
				if next == nil {
					span.AddEvent("Start streaming entries")
					sizeFn := func(_ context.Context) (uint64, error) {
						return logSize, nil
					}
					numFetchers := uint(10)
					next, stop = iter.Pull2(client.Entries(client.EntryBundles(ctx, numFetchers, sizeFn, lr.ReadEntryBundle, followFrom, logSize-followFrom), f.bundleHasher))
				}

				if curIndex == followFrom && curEntries != nil {
					// Note that it's possible for Spanner to automatically retry transactions in some circumstances, when it does
					// it'll call this function again.
					// If the above condition holds, then we're in a retry situation and we must use the same data again rather
					// than continue reading entries which will take us out of sync.
				} else {
					bs := uint64(f.as.opts.MaxBatchSize)
					if r := logSize - followFrom; r < bs {
						bs = r
					}
					batch := make([][]byte, 0, bs)
					for i := range int(bs) {
						e, err, ok := next()
						if !ok {
							// The entry stream has ended so we'll need to start a new stream next time around the loop:
							stop()
							next = nil
							break
						}
						if err != nil {
							return fmt.Errorf("entryReader.next: %v", err)
						}
						if wantIdx := followFrom + uint64(i); e.Index != wantIdx {
							// We're out of sync
							return errOutOfSync
						}
						batch = append(batch, e.Entry)
					}
					curEntries = batch
					curIndex = followFrom
				}

				if len(curEntries) == 0 {
					return nil
				}

				// Now update the index.
				{
					ms := make([]*spanner.Mutation, 0, len(curEntries))
					for i, e := range curEntries {
						ms = append(ms, spanner.Insert("IDSeq", []string{"h", "idx"}, []any{e, int64(curIndex + uint64(i))}))
					}
					if err := f.updateIndex(ctx, txn, ms); err != nil {
						return err
					}
				}

				numAdded := uint64(len(curEntries))
				f.as.numWrites.Add(numAdded)

				// Insertion of dupe entries was successful, so update our follow coordination row:
				m := make([]*spanner.Mutation, 0)
				m = append(m, spanner.Update("FollowCoord", []string{"id", "nextIdx"}, []any{0, int64(followFrom + numAdded)}))

				return txn.BufferWrite(m)
			})
			if err != nil {
				if err != errOutOfSync {
					klog.Errorf("Failed to commit antispam population tx: %v", err)
				}
				stop()
				next = nil
				streamDone = true
				continue
			}
			curEntries = nil
		}
	}
}

// batchUpdateIndex applies the provided mutations using Spanner's BatchWrite support.
//
// Note that we _do not_ use the passed in txn here -  we're writing the antispam entries outside of the transaction.
// The reason is because we absolutely do not want the larger transaction to fail if there's already an entry for the
// same hash in the IDSeq table - this would cause us to get stuck retrying forever, so we use BatchWrite and ignore
// any AlreadyExists errors we encounter.
//
// It looks unusual, but is ok because:
//   - individual antispam entries failing to insert because there's already an entry for that hash is perfectly ok,
//   - we'll only continue on to update FollowCoord if no errors (other than AlreadyExists) occur while inserting entries,
//   - similarly, if we manage to insert antispam entries here, but then fail to update FollowCoord, we'll end up
//     retrying over the same set of log entries, and then ignoring the AlreadyExists which will occur.
//
// Alternative approaches are:
//   - Use InsertOrUpdate, but that will keep updating the index associated with the ID hash, and we'd rather keep serving
//     the earliest index known for that entry.
//   - Perform reads for each of the hashes we're about to write, and use that to filter writes.
//     This would work, but would also incur an extra round-trip of data which isn't really necessary but would
//     slow the process down considerably and add extra load to Spanner for no benefit.
func (f *follower) batchUpdateIndex(ctx context.Context, _ *spanner.ReadWriteTransaction, ms []*spanner.Mutation) error {
	ctx, span := tracer.Start(ctx, "tessera.antispam.gcp.batchUpdateIndex")
	defer span.End()

	mgs := make([]*spanner.MutationGroup, 0, len(ms))
	for _, m := range ms {
		mgs = append(mgs, &spanner.MutationGroup{
			Mutations: []*spanner.Mutation{m},
		})
	}

	i := f.as.dbPool.BatchWrite(ctx, mgs)
	return i.Do(func(r *spannerpb.BatchWriteResponse) error {
		s := r.GetStatus()
		if c := codes.Code(s.Code); c != codes.OK && c != codes.AlreadyExists {
			return fmt.Errorf("failed to write antispam record: %v (%v)", s.GetMessage(), c)
		}
		return nil
	})
}

// EntriesProcessed returns the total number of log entries processed.
func (f *follower) EntriesProcessed(ctx context.Context) (uint64, error) {
	row, err := f.as.dbPool.Single().ReadRow(ctx, "FollowCoord", spanner.Key{0}, []string{"nextIdx"})
	if err != nil {
		return 0, err
	}

	var nextIdx int64 // Spanner doesn't support uint64
	if err := row.Columns(&nextIdx); err != nil {
		return 0, fmt.Errorf("failed to read follow coordination info: %v", err)
	}
	return uint64(nextIdx), nil
}

// createAndPrepareTables applies the passed in list of DDL statements and groups of mutations.
//
// This is intended to be used to create and initialise Spanner instances on first use.
// DDL should likely be of the form "CREATE TABLE IF NOT EXISTS".
// Mutation groups should likey be one or more spanner.Insert operations - AlreadyExists errors will be silently ignored.
func createAndPrepareTables(ctx context.Context, spannerDB string, ddl []string, mutations [][]*spanner.Mutation) error {
	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := adminClient.Close(); err != nil {
			klog.Warningf("adminClient.Close(): %v", err)
		}
	}()

	op, err := adminClient.UpdateDatabaseDdl(ctx, &adminpb.UpdateDatabaseDdlRequest{
		Database:   spannerDB,
		Statements: ddl,
	})
	if err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}
	if err := op.Wait(ctx); err != nil {
		return err
	}

	dbPool, err := spanner.NewClient(ctx, spannerDB)
	if err != nil {
		return fmt.Errorf("failed to connect to Spanner: %v", err)
	}
	defer dbPool.Close()

	// Set default values for a newly initialised schema using passed in mutation groups.
	// Note that this will only succeed if no row exists, so there's no danger of "resetting" an existing log.
	for _, mg := range mutations {
		if _, err := dbPool.Apply(ctx, mg); err != nil && spanner.ErrCode(err) != codes.AlreadyExists {
			return err
		}
	}
	return nil
}

// emulatorWorkaroundUpdateIndexTx is a workaround for spannertest not supporting BatchWrites.
// We use this func as a replacement for follower's updateIndex hook, and simply commit the index
// updates inline with the larger transaction.
func emulatorWorkaroundUpdateIndexTx(_ context.Context, txn *spanner.ReadWriteTransaction, ms []*spanner.Mutation) error {
	return txn.BufferWrite(ms)
}
