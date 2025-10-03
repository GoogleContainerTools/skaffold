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

// Package gcp contains a GCP-based storage implementation for Tessera.
//
// TODO: decide whether to rename this package.
//
// This storage implementation uses GCS for long-term storage and serving of
// entry bundles and log tiles, and Spanner for coordinating updates to GCS
// when multiple instances of a personality binary are running.
//
// A single GCS bucket is used to hold entry bundles and log internal tiles.
// The object keys for the bucket are selected so as to conform to the
// expected layout of a tile-based log.
//
// A Spanner database provides a transactional mechanism to allow multiple
// frontends to safely update the contents of the log.
package gcp

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	adminpb "cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"cloud.google.com/go/spanner/apiv1/spannerpb"

	gcs "cloud.google.com/go/storage"
	"github.com/google/go-cmp/cmp"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/tessera"
	"github.com/transparency-dev/tessera/api"
	"github.com/transparency-dev/tessera/api/layout"
	"github.com/transparency-dev/tessera/internal/fetcher"
	"github.com/transparency-dev/tessera/internal/migrate"
	"github.com/transparency-dev/tessera/internal/otel"
	"github.com/transparency-dev/tessera/internal/parse"
	storage "github.com/transparency-dev/tessera/storage/internal"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

const (
	// minCheckpointInterval is the shortest permitted interval between updating published checkpoints.
	// GCS has a rate limit 1 update per second for individual objects, but we've observed that attempting
	// to update at exactly that rate still results in the occasional refusal, so bake in a little wiggle
	// room.
	minCheckpointInterval = 1200 * time.Millisecond

	logContType      = "application/octet-stream"
	ckptContType     = "text/plain; charset=utf-8"
	logCacheControl  = "max-age=604800,immutable"
	ckptCacheControl = "no-cache"

	DefaultIntegrationSizeLimit = 5 * 4096

	// SchemaCompatibilityVersion represents the expected version (e.g. layout & serialisation) of stored data.
	//
	// A binary built with a given version of the Tessera library is compatible with stored data created by a different version
	// of the library if and only if this value is the same as the compatibilityVersion stored in the Tessera table.
	//
	// NOTE: if changing this version, you need to consider whether end-users are going to update their schema instances to be
	// compatible with the new format, and provide a means to do it if so.
	SchemaCompatibilityVersion = 1
)

// Storage is a GCP based storage implementation for Tessera.
type Storage struct {
	cfg Config
}

// sequencer describes a type which knows how to sequence entries.
//
// TODO(al): rename this as it's really more of a coordination for the log.
type sequencer interface {
	// assignEntries should durably allocate contiguous index numbers to the provided entries.
	assignEntries(ctx context.Context, entries []*tessera.Entry) error
	// consumeEntries should call the provided function with up to limit previously sequenced entries.
	// If the call to consumeFunc returns no error, the entries should be considered to have been consumed.
	// If any entries were successfully consumed, the implementation should also return true; this
	// serves as a weak hint that there may be more entries to be consumed.
	// If forceUpdate is true, then the consumeFunc should be called, with an empty slice of entries if
	// necessary. This allows the log self-initialise in a transactionally safe manner.
	consumeEntries(ctx context.Context, limit uint64, f consumeFunc, forceUpdate bool) (bool, error)
	// currentTree returns the tree state of the currently integrated tree according to the IntCoord table.
	currentTree(ctx context.Context) (uint64, []byte, error)
	// nextIndex returns the next available index in the log.
	nextIndex(ctx context.Context) (uint64, error)
	// publishCheckpoint coordinates the publication of new checkpoints based on the current integrated tree.
	publishCheckpoint(ctx context.Context, minAge time.Duration, f func(ctx context.Context, size uint64, root []byte) error) error
	// garbageCollect coordinates the removal of unneeded partial tiles/entry bundles for the provided tree size, up to a maximum number of deletes per invocation.
	garbageCollect(ctx context.Context, treeSize uint64, maxDeletes uint, removePrefix func(ctx context.Context, prefix string) error) error
}

// consumeFunc is the signature of a function which can consume entries from the sequencer and integrate
// them into the log.
// Returns the new rootHash once all passed entries have been integrated.
type consumeFunc func(ctx context.Context, from uint64, entries []storage.SequencedEntry) ([]byte, error)

// Config holds GCP project and resource configuration for a storage instance.
type Config struct {
	// GCSClient will be  used to interact with GCS. If unset, Tessera will create one.
	GCSClient *gcs.Client
	// SpannerClient will be used to interact with Spanner. If unset, Tessera will create one.
	SpannerClient *spanner.Client
	// HTTPClient will be used for other HTTP requests. If unset, Tessera will use the net/http DefaultClient.
	HTTPClient *http.Client

	// Bucket is the name of the GCS bucket to use for storing log state.
	Bucket string
	// BucketPrefix is an optional prefix to prepend to all log resource paths.
	// This can be used e.g. to store multiple logs in the same bucket.
	BucketPrefix string
	// Spanner is the GCP resource URI of the spanner database instance to use.
	Spanner string
}

// New creates a new instance of the GCP based Storage.
func New(ctx context.Context, cfg Config) (tessera.Driver, error) {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	return &Storage{
		cfg: cfg,
	}, nil
}

type LogReader struct {
	lrs            logResourceStore
	integratedSize func(context.Context) (uint64, error)
	nextIndex      func(context.Context) (uint64, error)
}

func (lr *LogReader) ReadCheckpoint(ctx context.Context) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.ReadCheckpoint")
	defer span.End()

	r, err := lr.lrs.getCheckpoint(ctx)
	if err != nil {
		if errors.Is(err, gcs.ErrObjectNotExist) {
			return r, os.ErrNotExist
		}
	}
	return r, err
}

func (lr *LogReader) ReadTile(ctx context.Context, l, i uint64, p uint8) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.ReadTile")
	defer span.End()

	return fetcher.PartialOrFullResource(ctx, p, func(ctx context.Context, p uint8) ([]byte, error) {
		return lr.lrs.getTile(ctx, l, i, p)
	})
}

func (lr *LogReader) ReadEntryBundle(ctx context.Context, i uint64, p uint8) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.ReadEntryBundle")
	defer span.End()

	return fetcher.PartialOrFullResource(ctx, p, func(ctx context.Context, p uint8) ([]byte, error) {
		return lr.lrs.getEntryBundle(ctx, i, p)
	})
}

func (lr *LogReader) IntegratedSize(ctx context.Context) (uint64, error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.IntegratedSize")
	defer span.End()

	return lr.integratedSize(ctx)
}

func (lr *LogReader) NextIndex(ctx context.Context) (uint64, error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.NextIndex")
	defer span.End()

	return lr.nextIndex(ctx)
}

// Appender creates a new tessera.Appender lifecycle object.
func (s *Storage) Appender(ctx context.Context, opts *tessera.AppendOptions) (*tessera.Appender, tessera.LogReader, error) {
	if s.cfg.GCSClient == nil {
		var err error
		s.cfg.GCSClient, err = gcs.NewClient(ctx, gcs.WithJSONReads())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create GCS client: %v", err)
		}
	}
	gs := &gcsStorage{
		gcsClient:    s.cfg.GCSClient,
		bucket:       s.cfg.Bucket,
		bucketPrefix: s.cfg.BucketPrefix,
	}

	var err error
	if s.cfg.SpannerClient == nil {
		s.cfg.SpannerClient, err = spanner.NewClient(ctx, s.cfg.Spanner)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect to Spanner: %v", err)
		}
	}
	if err := initDB(ctx, s.cfg.Spanner); err != nil {
		return nil, nil, fmt.Errorf("failed to verify/init Spanner schema: %v", err)
	}

	seq, err := newSpannerCoordinator(ctx, s.cfg.SpannerClient, uint64(opts.PushbackMaxOutstanding()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Spanner coordinator: %v", err)
	}

	a, lr, err := s.newAppender(ctx, gs, seq, opts)
	if err != nil {
		return nil, nil, err
	}
	return &tessera.Appender{
		Add: a.Add,
	}, lr, nil
}

// newAppender creates and initialises a tessera.Appender struct with the provided underlying storage implementations.
func (s *Storage) newAppender(ctx context.Context, o objStore, seq *spannerCoordinator, opts *tessera.AppendOptions) (*Appender, tessera.LogReader, error) {
	if opts.CheckpointInterval() < minCheckpointInterval {
		return nil, nil, fmt.Errorf("requested CheckpointInterval (%v) is less than minimum permitted %v", opts.CheckpointInterval(), minCheckpointInterval)
	}

	a := &Appender{
		logStore: &logResourceStore{
			objStore:    o,
			entriesPath: opts.EntriesPath(),
		},
		sequencer: seq,
		cpUpdated: make(chan struct{}),
	}
	a.queue = storage.NewQueue(ctx, opts.BatchMaxAge(), opts.BatchMaxSize(), a.sequencer.assignEntries)

	reader := &LogReader{
		lrs: *a.logStore,
		integratedSize: func(context.Context) (uint64, error) {
			s, _, err := a.sequencer.currentTree(ctx)
			return s, err
		},
		nextIndex: func(context.Context) (uint64, error) {
			return a.sequencer.nextIndex(ctx)
		},
	}
	a.newCP = opts.CheckpointPublisher(reader, s.cfg.HTTPClient)

	if err := a.init(ctx); err != nil {
		return nil, nil, fmt.Errorf("failed to initialise log storage: %v", err)
	}

	go a.integrateEntriesJob(ctx)
	go a.publishCheckpointJob(ctx, opts.CheckpointInterval())
	if i := opts.GarbageCollectionInterval(); i > 0 {
		go a.garbageCollectorJob(ctx, i)
	}

	return a, reader, nil
}

// Appender is an implementation of the Tessera appender lifecycle contract.
type Appender struct {
	newCP func(context.Context, uint64, []byte) ([]byte, error)

	sequencer sequencer
	logStore  *logResourceStore

	queue *storage.Queue

	cpUpdated chan struct{}
}

// Add is the entrypoint for adding entries to a sequencing log.
func (a *Appender) Add(ctx context.Context, e *tessera.Entry) tessera.IndexFuture {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.Add")
	defer span.End()

	return a.queue.Add(ctx, e)
}

// integrateEntriesJob periodically append newly sequenced entries.
//
// Blocks until ctx is done.
func (a *Appender) integrateEntriesJob(ctx context.Context) {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		func() {
			ctx, span := tracer.Start(ctx, "tessera.storage.gcp.integrateEntriesJob")
			defer span.End()

			// Don't quickloop for now, it causes issues updating checkpoint too frequently.
			cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if _, err := a.sequencer.consumeEntries(cctx, DefaultIntegrationSizeLimit, a.integrateEntries, false); err != nil {
				klog.Errorf("integrateEntriesJob: %v", err)
				return
			}
			select {
			case a.cpUpdated <- struct{}{}:
			default:
			}
		}()
	}
}

// publishCheckpointJob periodically attempts to publish a new checkpoint representing the current state
// of the tree, once per interval.
//
// Blocks until ctx is done.
func (a *Appender) publishCheckpointJob(ctx context.Context, i time.Duration) {
	t := time.NewTicker(i)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.cpUpdated:
		case <-t.C:
		}
		func() {
			ctx, span := tracer.Start(ctx, "tessera.storage.gcp.publishCheckpointJob")
			defer span.End()
			if err := a.sequencer.publishCheckpoint(ctx, i, a.publishCheckpoint); err != nil {
				klog.Warningf("publishCheckpoint failed: %v", err)
			}
		}()
	}
}

// garbageCollectorJob is a long-running function which handles the removal of obsolete partial tiles
// and entry bundles.
// Blocks until ctx is done.
func (a *Appender) garbageCollectorJob(ctx context.Context, i time.Duration) {
	t := time.NewTicker(i)
	defer t.Stop()

	// Entirely arbitrary number.
	maxBundlesPerRun := uint(100)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		func() {
			ctx, span := tracer.Start(ctx, "tessera.storage.gcp.garbageCollectTask")
			defer span.End()

			// Figure out the size of the latest published checkpoint - we can't be removing partial tiles implied by
			// that checkpoint just because we've done an integration and know about a larger (but as yet unpublished)
			// checkpoint!
			cp, err := a.logStore.getCheckpoint(ctx)
			if err != nil {
				klog.Warningf("Failed to get published checkpoint: %v", err)
				return
			}
			_, pubSize, _, err := parse.CheckpointUnsafe(cp)
			if err != nil {
				klog.Warningf("Failed to parse published checkpoint: %v", err)
				return
			}

			if err := a.sequencer.garbageCollect(ctx, pubSize, maxBundlesPerRun, a.logStore.objStore.deleteObjectsWithPrefix); err != nil {
				klog.Warningf("GarbageCollect failed: %v", err)
				return
			}
		}()
	}

}

// init ensures that the storage represents a log in a valid state.
func (a *Appender) init(ctx context.Context) error {
	if _, err := a.logStore.getCheckpoint(ctx); err != nil {
		if errors.Is(err, gcs.ErrObjectNotExist) {
			// No checkpoint exists, do a forced (possibly empty) integration to create one in a safe
			// way (setting the checkpoint directly here would not be safe as it's outside the transactional
			// framework which prevents the tree from rolling backwards or otherwise forking).
			cctx, c := context.WithTimeout(ctx, 10*time.Second)
			defer c()
			if _, err := a.sequencer.consumeEntries(cctx, DefaultIntegrationSizeLimit, a.integrateEntries, true); err != nil {
				return fmt.Errorf("forced integrate: %v", err)
			}
			select {
			case a.cpUpdated <- struct{}{}:
			default:
			}
			return nil
		}
		return fmt.Errorf("failed to read checkpoint: %v", err)
	}

	return nil
}

func (a *Appender) publishCheckpoint(ctx context.Context, size uint64, root []byte) error {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.publishCheckpoint")
	defer span.End()
	span.SetAttributes(treeSizeKey.Int64(otel.Clamp64(size)))

	cpRaw, err := a.newCP(ctx, size, root)
	if err != nil {
		return fmt.Errorf("newCP: %v", err)
	}

	if err := a.logStore.setCheckpoint(ctx, cpRaw); err != nil {
		return fmt.Errorf("writeCheckpoint: %v", err)
	}

	klog.V(2).Infof("Published latest checkpoint: %d, %x", size, root)

	return nil

}

// objStore describes a type which can store and retrieve objects.
type objStore interface {
	getObject(ctx context.Context, obj string) ([]byte, int64, error)
	setObject(ctx context.Context, obj string, data []byte, cond *gcs.Conditions, contType string, cacheCtl string) error
	deleteObjectsWithPrefix(ctx context.Context, prefix string) error
}

// logResourceStore knows how to read and write entries which represent a tiles log inside an objStore.
type logResourceStore struct {
	objStore    objStore
	entriesPath func(uint64, uint8) string
}

func (lrs *logResourceStore) setCheckpoint(ctx context.Context, cpRaw []byte) error {
	return lrs.objStore.setObject(ctx, layout.CheckpointPath, cpRaw, nil, ckptContType, ckptCacheControl)
}

func (lrs *logResourceStore) getCheckpoint(ctx context.Context) ([]byte, error) {
	r, _, err := lrs.objStore.getObject(ctx, layout.CheckpointPath)
	return r, err
}

// setTile idempotently stores the provided tile at the location implied by the given level, index, and treeSize.
//
// The location to which the tile is written is defined by the tile layout spec.
func (s *logResourceStore) setTile(ctx context.Context, level, index uint64, partial uint8, data []byte) error {
	tPath := layout.TilePath(level, index, partial)
	return s.objStore.setObject(ctx, tPath, data, &gcs.Conditions{DoesNotExist: true}, logContType, logCacheControl)
}

// getTile retrieves the raw tile from the provided location.
//
// The location to which the tile is written is defined by the tile layout spec.
func (s *logResourceStore) getTile(ctx context.Context, level, index uint64, partial uint8) ([]byte, error) {
	tPath := layout.TilePath(level, index, partial)
	d, _, err := s.objStore.getObject(ctx, tPath)
	return d, err
}

// getTiles returns the tiles with the given tile-coords for the specified log size.
//
// Tiles are returned in the same order as they're requested, nils represent tiles which were not found.
func (s *logResourceStore) getTiles(ctx context.Context, tileIDs []storage.TileID, logSize uint64) ([]*api.HashTile, error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.getTiles")
	defer span.End()

	r := make([]*api.HashTile, len(tileIDs))
	errG := errgroup.Group{}
	for i, id := range tileIDs {
		i := i
		id := id
		errG.Go(func() error {
			objName := layout.TilePath(id.Level, id.Index, layout.PartialTileSize(id.Level, id.Index, logSize))
			data, _, err := s.objStore.getObject(ctx, objName)
			if err != nil {
				if errors.Is(err, gcs.ErrObjectNotExist) {
					// Depending on context, this may be ok.
					// We'll signal to higher levels that it wasn't found by retuning a nil for this tile.
					return nil
				}
				return err
			}
			t := &api.HashTile{}
			if err := t.UnmarshalText(data); err != nil {
				return fmt.Errorf("unmarshal(%q): %v", objName, err)
			}
			r[i] = t
			return nil
		})
	}
	if err := errG.Wait(); err != nil {
		return nil, err
	}
	return r, nil
}

// getEntryBundle returns the serialised entry bundle at the location described by the given index and partial size.
// A partial size of zero implies a full tile.
//
// Returns a wrapped os.ErrNotExist if the bundle does not exist.
func (s *logResourceStore) getEntryBundle(ctx context.Context, bundleIndex uint64, p uint8) ([]byte, error) {
	objName := s.entriesPath(bundleIndex, p)
	data, _, err := s.objStore.getObject(ctx, objName)
	if err != nil {
		if errors.Is(err, gcs.ErrObjectNotExist) {
			// Return the generic NotExist error so that higher levels can differentiate
			// between this and other errors.
			return nil, fmt.Errorf("%v: %w", objName, os.ErrNotExist)
		}
		return nil, err
	}

	return data, nil
}

// setEntryBundle idempotently stores the serialised entry bundle at the location implied by the bundleIndex and treeSize.
func (s *logResourceStore) setEntryBundle(ctx context.Context, bundleIndex uint64, p uint8, bundleRaw []byte) error {
	objName := s.entriesPath(bundleIndex, p)
	// Note that setObject does an idempotent interpretation of DoesNotExist - it only
	// returns an error if the named object exists _and_ contains different data to what's
	// passed in here.
	if err := s.objStore.setObject(ctx, objName, bundleRaw, &gcs.Conditions{DoesNotExist: true}, logContType, logCacheControl); err != nil {
		return fmt.Errorf("setObject(%q): %v", objName, err)

	}
	return nil
}

// integrateEntries appends the provided entries into the log starting at fromSeq.
//
// Returns the new root hash of the log with the entries added.
func (a *Appender) integrateEntries(ctx context.Context, fromSeq uint64, entries []storage.SequencedEntry) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.integrateEntries")
	defer span.End()

	var newRoot []byte

	errG := errgroup.Group{}

	errG.Go(func() error {
		if err := a.updateEntryBundles(ctx, fromSeq, entries); err != nil {
			return fmt.Errorf("updateEntryBundles: %v", err)
		}
		return nil
	})

	errG.Go(func() error {
		lh := make([][]byte, len(entries))
		for i, e := range entries {
			lh[i] = e.LeafHash
		}
		r, err := integrate(ctx, fromSeq, lh, a.logStore)
		if err != nil {
			return fmt.Errorf("integrate: %v", err)
		}
		newRoot = r
		return nil
	})
	if err := errG.Wait(); err != nil {
		return nil, err
	}
	return newRoot, nil
}

// integrate adds the provided leaf hashes to the merkle tree, starting at the provided location.
func integrate(ctx context.Context, fromSeq uint64, lh [][]byte, logStore *logResourceStore) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.integrate")
	defer span.End()

	span.SetAttributes(fromSizeKey.Int64(otel.Clamp64(fromSeq)), numEntriesKey.Int(len(lh)))

	errG := errgroup.Group{}
	getTiles := func(ctx context.Context, tileIDs []storage.TileID, treeSize uint64) ([]*api.HashTile, error) {
		n, err := logStore.getTiles(ctx, tileIDs, treeSize)
		if err != nil {
			return nil, fmt.Errorf("getTiles: %w", err)
		}
		return n, nil
	}

	newSize, newRoot, tiles, err := storage.Integrate(ctx, getTiles, fromSeq, lh)
	if err != nil {
		return nil, fmt.Errorf("storage.Integrate: %v", err)
	}
	for k, v := range tiles {
		func(ctx context.Context, k storage.TileID, v *api.HashTile) {
			errG.Go(func() error {
				data, err := v.MarshalText()
				if err != nil {
					return err
				}
				return logStore.setTile(ctx, k.Level, k.Index, layout.PartialTileSize(k.Level, k.Index, newSize), data)
			})
		}(ctx, k, v)
	}
	if err := errG.Wait(); err != nil {
		return nil, err
	}
	klog.V(1).Infof("New tree: %d, %x", newSize, newRoot)

	return newRoot, nil
}

// updateEntryBundles adds the entries being integrated into the entry bundles.
//
// The right-most bundle will be grown, if it's partial, and/or new bundles will be created as required.
func (a *Appender) updateEntryBundles(ctx context.Context, fromSeq uint64, entries []storage.SequencedEntry) error {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.updateEntryBundles")
	defer span.End()

	if len(entries) == 0 {
		return nil
	}

	numAdded := uint64(0)
	bundleIndex, entriesInBundle := fromSeq/layout.EntryBundleWidth, fromSeq%layout.EntryBundleWidth
	bundleWriter := &bytes.Buffer{}
	if entriesInBundle > 0 {
		// If the latest bundle is partial, we need to read the data it contains in for our newer, larger, bundle.
		part, err := a.logStore.getEntryBundle(ctx, uint64(bundleIndex), uint8(entriesInBundle))
		if err != nil {
			return err
		}

		if _, err := bundleWriter.Write(part); err != nil {
			return fmt.Errorf("bundleWriter: %v", err)
		}
	}

	seqErr := errgroup.Group{}

	// goSetEntryBundle is a function which uses seqErr to spin off a go-routine to write out an entry bundle.
	// It's used in the for loop below.
	goSetEntryBundle := func(ctx context.Context, bundleIndex uint64, p uint8, bundleRaw []byte) {
		seqErr.Go(func() error {
			if err := a.logStore.setEntryBundle(ctx, bundleIndex, p, bundleRaw); err != nil {
				return err
			}
			return nil
		})
	}

	// Add new entries to the bundle
	for _, e := range entries {
		if _, err := bundleWriter.Write(e.BundleData); err != nil {
			return fmt.Errorf("bundleWriter.Write: %v", err)
		}
		entriesInBundle++
		fromSeq++
		numAdded++
		if entriesInBundle == layout.EntryBundleWidth {
			//  This bundle is full, so we need to write it out...
			klog.V(1).Infof("In-memory bundle idx %d is full, attempting write to GCS", bundleIndex)
			goSetEntryBundle(ctx, bundleIndex, 0, bundleWriter.Bytes())
			// ... and prepare the next entry bundle for any remaining entries in the batch
			bundleIndex++
			entriesInBundle = 0
			// Don't use Reset/Truncate here - the backing []bytes is still being used by goSetEntryBundle above.
			bundleWriter = &bytes.Buffer{}
			klog.V(1).Infof("Starting to fill in-memory bundle idx %d", bundleIndex)
		}
	}
	// If we have a partial bundle remaining once we've added all the entries from the batch,
	// this needs writing out too.
	if entriesInBundle > 0 {
		klog.V(1).Infof("Attempting to write in-memory partial bundle idx %d.%d to GCS", bundleIndex, entriesInBundle)
		goSetEntryBundle(ctx, bundleIndex, uint8(entriesInBundle), bundleWriter.Bytes())
	}
	return seqErr.Wait()
}

// spannerCoordinator uses Cloud Spanner to provide
// a durable and thread/multi-process safe sequencer.
type spannerCoordinator struct {
	dbPool         *spanner.Client
	maxOutstanding uint64
}

// newSpannerCoordinator returns a new spannerSequencer struct which uses the provided
// spanner resource name for its spanner connection.
func newSpannerCoordinator(ctx context.Context, dbPool *spanner.Client, maxOutstanding uint64) (*spannerCoordinator, error) {
	r := &spannerCoordinator{
		dbPool:         dbPool,
		maxOutstanding: maxOutstanding,
	}
	if err := r.checkDataCompatibility(ctx); err != nil {
		return nil, fmt.Errorf("schema is not compatible with this version of the Tessera library: %v", err)
	}
	return r, nil
}

// initDB ensures that the coordination DB is initialised correctly.
//
// The database schema consists of 3 tables:
//   - SeqCoord
//     This table only ever contains a single row which tracks the next available
//     sequence number.
//   - Seq
//     This table holds sequenced "batches" of entries. The batches are keyed
//     by the sequence number assigned to the first entry in the batch, and
//     each subsequent entry in the batch takes the numerically next sequence number.
//   - IntCoord
//     This table coordinates integration of the batches of entries stored in
//     Seq into the committed tree state.
func initDB(ctx context.Context, spannerDB string) error {
	return createAndPrepareTables(
		ctx, spannerDB,
		[]string{
			"CREATE TABLE IF NOT EXISTS Tessera (id INT64 NOT NULL, compatibilityVersion INT64 NOT NULL) PRIMARY KEY (id)",
			"CREATE TABLE IF NOT EXISTS SeqCoord (id INT64 NOT NULL, next INT64 NOT NULL,) PRIMARY KEY (id)",
			"CREATE TABLE IF NOT EXISTS Seq (id INT64 NOT NULL, seq INT64 NOT NULL, v BYTES(MAX),) PRIMARY KEY (id, seq)",
			"CREATE TABLE IF NOT EXISTS IntCoord (id INT64 NOT NULL, seq INT64 NOT NULL, rootHash BYTES(32)) PRIMARY KEY (id)",
			"CREATE TABLE IF NOT EXISTS PubCoord (id INT64 NOT NULL, publishedAt TIMESTAMP NOT NULL) PRIMARY KEY (id)",
			"CREATE TABLE IF NOT EXISTS GCCoord (id INT64 NOT NULL, fromSize INT64 NOT NULL) PRIMARY KEY (id)",
		},
		[][]*spanner.Mutation{
			{spanner.Insert("Tessera", []string{"id", "compatibilityVersion"}, []any{0, SchemaCompatibilityVersion})},
			{spanner.Insert("SeqCoord", []string{"id", "next"}, []any{0, 0})},
			{spanner.Insert("IntCoord", []string{"id", "seq", "rootHash"}, []any{0, 0, rfc6962.DefaultHasher.EmptyRoot()})},
			{spanner.Insert("PubCoord", []string{"id", "publishedAt"}, []any{0, time.Unix(0, 0)})},
			{spanner.Insert("GCCoord", []string{"id", "fromSize"}, []any{0, 0})},
		},
	)
}

// checkDataCompatibility compares the Tessera library SchemaCompatibilityVersion with the one stored in the
// database, and returns an error if they are not identical.
func (s *spannerCoordinator) checkDataCompatibility(ctx context.Context) error {
	row, err := s.dbPool.Single().ReadRow(ctx, "Tessera", spanner.Key{0}, []string{"compatibilityVersion"})
	if err != nil {
		return fmt.Errorf("failed to read schema compatibilityVersion: %v", err)
	}
	var compat int64
	if err := row.Columns(&compat); err != nil {
		return fmt.Errorf("failed to scan schema compatibilityVersion: %v", err)
	}

	if compat != SchemaCompatibilityVersion {
		return fmt.Errorf("schema compatibilityVersion (%d) != library compatibilityVersion (%d)", compat, SchemaCompatibilityVersion)
	}
	return nil
}

// assignEntries durably assigns each of the passed-in entries an index in the log.
//
// Entries are allocated contiguous indices, in the order in which they appear in the entries parameter.
// This is achieved by storing the passed-in entries in the Seq table in Spanner, keyed by the
// index assigned to the first entry in the batch.
func (s *spannerCoordinator) assignEntries(ctx context.Context, entries []*tessera.Entry) error {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.assignEntries")
	defer span.End()

	span.SetAttributes(numEntriesKey.Int(len(entries)))

	// First grab the treeSize in a non-locking read-only fashion (we don't want to block/collide with integration).
	// We'll use this value to determine whether we need to apply back-pressure.
	var treeSize int64
	if row, err := s.dbPool.Single().ReadRow(ctx, "IntCoord", spanner.Key{0}, []string{"seq"}); err != nil {
		return err
	} else {
		if err := row.Column(0, &treeSize); err != nil {
			return fmt.Errorf("failed to read integration coordination info: %v", err)
		}
	}
	span.SetAttributes(treeSizeKey.Int64(treeSize))

	var next int64 // Unfortunately, Spanner doesn't support uint64 so we'll have to cast around a bit.

	_, err := s.dbPool.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// First we need to grab the next available sequence number from the SeqCoord table.
		row, err := txn.ReadRowWithOptions(ctx, "SeqCoord", spanner.Key{0}, []string{"id", "next"}, &spanner.ReadOptions{LockHint: spannerpb.ReadRequest_LOCK_HINT_EXCLUSIVE})
		if err != nil {
			return fmt.Errorf("failed to read SeqCoord: %w", err)
		}
		var id int64
		if err := row.Columns(&id, &next); err != nil {
			return fmt.Errorf("failed to parse id column: %v", err)
		}

		// Check whether there are too many outstanding entries and we should apply
		// back-pressure.
		if outstanding := next - treeSize; outstanding > int64(s.maxOutstanding) {
			return tessera.ErrPushback
		}

		next := uint64(next) // Shadow next with a uint64 version of the same value to save on casts.
		sequencedEntries := make([]storage.SequencedEntry, len(entries))
		// Assign provisional sequence numbers to entries.
		// We need to do this here in order to support serialisations which include the log position.
		for i, e := range entries {
			sequencedEntries[i] = storage.SequencedEntry{
				BundleData: e.MarshalBundleData(next + uint64(i)),
				LeafHash:   e.LeafHash(),
			}
		}

		// Flatten the entries into a single slice of bytes which we can store in the Seq.v column.
		b := &bytes.Buffer{}
		e := gob.NewEncoder(b)
		if err := e.Encode(sequencedEntries); err != nil {
			return fmt.Errorf("failed to serialise batch: %v", err)
		}
		data := b.Bytes()
		num := len(entries)

		// TODO(al): think about whether aligning bundles to tile boundaries would be a good idea or not.
		m := []*spanner.Mutation{
			// Insert our newly sequenced batch of entries into Seq,
			spanner.Insert("Seq", []string{"id", "seq", "v"}, []any{0, int64(next), data}),
			// and update the next-available sequence number row in SeqCoord.
			spanner.Update("SeqCoord", []string{"id", "next"}, []any{0, int64(next) + int64(num)}),
		}
		if err := txn.BufferWrite(m); err != nil {
			return fmt.Errorf("failed to apply TX: %v", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to flush batch: %w", err)
	}

	return nil
}

// consumeEntries calls f with previously sequenced entries.
//
// Once f returns without error, the entries it was called with are considered to have been consumed and are
// removed from the Seq table.
//
// Returns true if some entries were consumed as a weak signal that there may be further entries waiting to be consumed.
func (s *spannerCoordinator) consumeEntries(ctx context.Context, limit uint64, f consumeFunc, forceUpdate bool) (bool, error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.consumeEntries")
	defer span.End()

	didWork := false
	_, err := s.dbPool.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Figure out which is the starting index of sequenced entries to start consuming from.
		row, err := txn.ReadRowWithOptions(ctx, "IntCoord", spanner.Key{0}, []string{"seq", "rootHash"}, &spanner.ReadOptions{LockHint: spannerpb.ReadRequest_LOCK_HINT_EXCLUSIVE})
		if err != nil {
			return err
		}
		var fromSeq int64 // Spanner doesn't support uint64
		var rootHash []byte
		if err := row.Columns(&fromSeq, &rootHash); err != nil {
			return fmt.Errorf("failed to read integration coordination info: %v", err)
		}

		// See how much potential work there is to do and trim our limit accordingly.
		row, err = txn.ReadRow(ctx, "SeqCoord", spanner.Key{0}, []string{"next"})
		if err != nil {
			return err
		}
		var endSeq int64 // Spanner doesn't support uint64
		if err := row.Columns(&endSeq); err != nil {
			return fmt.Errorf("failed to read sequence coordination info: %v", err)
		}
		if endSeq == fromSeq {
			return nil
		}
		if l := fromSeq + int64(limit); l < endSeq {
			endSeq = l
		}

		klog.V(1).Infof("Consuming bundles start from %d to at most %d", fromSeq, endSeq)

		// Now read the sequenced starting at the index we got above.
		rows := txn.ReadWithOptions(ctx, "Seq",
			spanner.KeyRange{Start: spanner.Key{0, fromSeq}, End: spanner.Key{0, endSeq}},
			[]string{"seq", "v"},
			&spanner.ReadOptions{LockHint: spannerpb.ReadRequest_LOCK_HINT_EXCLUSIVE})
		defer rows.Stop()

		seqsConsumed := []int64{}
		entries := make([]storage.SequencedEntry, 0, endSeq-fromSeq)
		orderCheck := fromSeq
		for {
			row, err := rows.Next()
			if row == nil || err == iterator.Done {
				break
			}

			var vGob []byte
			var seq int64 // spanner doesn't have uint64
			if err := row.Columns(&seq, &vGob); err != nil {
				return fmt.Errorf("failed to scan seq row: %v", err)
			}

			if orderCheck != seq {
				return fmt.Errorf("integrity fail - expected seq %d, but found %d", orderCheck, seq)
			}

			g := gob.NewDecoder(bytes.NewReader(vGob))
			b := []storage.SequencedEntry{}
			if err := g.Decode(&b); err != nil {
				return fmt.Errorf("failed to deserialise v: %v", err)
			}
			entries = append(entries, b...)
			seqsConsumed = append(seqsConsumed, seq)
			orderCheck += int64(len(b))
		}
		if len(seqsConsumed) == 0 && !forceUpdate {
			klog.V(1).Info("Found no rows to sequence")
			return nil
		}

		// Call consumeFunc with the entries we've found
		newRoot, err := f(ctx, uint64(fromSeq), entries)
		if err != nil {
			return err
		}

		// consumeFunc was successful, so we can update our coordination row, and delete the row(s) for
		// the then consumed entries.
		m := make([]*spanner.Mutation, 0)
		m = append(m, spanner.Update("IntCoord", []string{"id", "seq", "rootHash"}, []any{0, int64(orderCheck), newRoot}))
		for _, c := range seqsConsumed {
			m = append(m, spanner.Delete("Seq", spanner.Key{0, c}))
		}
		if len(m) > 0 {
			if err := txn.BufferWrite(m); err != nil {
				return err
			}
		}

		didWork = true
		return nil
	})
	if err != nil {
		return false, err
	}

	return didWork, nil
}

// currentTree returns the size and root hash of the currently integrated tree.
func (s *spannerCoordinator) currentTree(ctx context.Context) (uint64, []byte, error) {
	row, err := s.dbPool.Single().ReadRow(ctx, "IntCoord", spanner.Key{0}, []string{"seq", "rootHash"})
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read IntCoord: %v", err)
	}
	var fromSeq int64 // Spanner doesn't support uint64
	var rootHash []byte
	if err := row.Columns(&fromSeq, &rootHash); err != nil {
		return 0, nil, fmt.Errorf("failed to read integration coordination info: %v", err)
	}

	return uint64(fromSeq), rootHash, nil
}

// nextIndex returns the next available index in the log.
func (s *spannerCoordinator) nextIndex(ctx context.Context) (uint64, error) {
	txn := s.dbPool.ReadOnlyTransaction()
	defer txn.Close()

	var nextSeq int64 // Spanner doesn't support uint64
	row, err := txn.ReadRow(ctx, "SeqCoord", spanner.Key{0}, []string{"next"})
	if err != nil {
		return 0, fmt.Errorf("failed to read sequence coordination row: %v", err)
	}
	if err := row.Columns(&nextSeq); err != nil {
		return 0, fmt.Errorf("failed to read sequence coordination info: %v", err)
	}

	return uint64(nextSeq), nil
}

// publishCheckpoint checks when the last checkpoint was published, and if it was more than minAge ago, calls the provided
// function to publish a new one.
//
// This function uses PubCoord with an exclusive lock to guarantee that only one tessera instance can attempt to publish
// a checkpoint at any given time.
func (s *spannerCoordinator) publishCheckpoint(ctx context.Context, minAge time.Duration, f func(context.Context, uint64, []byte) error) error {
	if _, err := s.dbPool.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		pRow, err := txn.ReadRowWithOptions(ctx, "PubCoord", spanner.Key{0}, []string{"publishedAt"}, &spanner.ReadOptions{LockHint: spannerpb.ReadRequest_LOCK_HINT_EXCLUSIVE})
		if err != nil {
			return fmt.Errorf("failed to read PubCoord: %w", err)
		}
		var pubAt time.Time
		if err := pRow.Column(0, &pubAt); err != nil {
			return fmt.Errorf("failed to parse publishedAt: %v", err)
		}

		cpAge := time.Since(pubAt)
		if cpAge < minAge {
			klog.V(1).Infof("publishCheckpoint: last checkpoint published %s ago (< required %s), not publishing new checkpoint", cpAge, minAge)
			return nil
		}

		klog.V(1).Infof("publishCheckpoint: updating checkpoint (replacing %s old checkpoint)", cpAge)

		// Can't just use currentTree() here as the spanner emulator doesn't do nested transactions, so do it manually:
		row, err := txn.ReadRow(ctx, "IntCoord", spanner.Key{0}, []string{"seq", "rootHash"})
		if err != nil {
			return fmt.Errorf("failed to read IntCoord: %w", err)
		}
		var fromSeq int64 // Spanner doesn't support uint64
		var rootHash []byte
		if err := row.Columns(&fromSeq, &rootHash); err != nil {
			return fmt.Errorf("failed to parse integration coordination info: %v", err)
		}
		if err := f(ctx, uint64(fromSeq), rootHash); err != nil {
			return err
		}
		if err := txn.BufferWrite([]*spanner.Mutation{spanner.Update("PubCoord", []string{"id", "publishedAt"}, []any{0, time.Now()})}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}
	return nil
}

// garbageCollect will identify up to maxBundles unneeded partial entry bundles (and any unneeded partial tiles which sit above them in the tree) and
// call the provided function to remove them.
//
// Uses the `GCCoord` table to ensure that only one binary is actively garbage collecting at any given time, and to track progress so that we don't
// needlessly attempt to GC over regions which have already been cleaned.
func (s *spannerCoordinator) garbageCollect(ctx context.Context, treeSize uint64, maxBundles uint, deleteWithPrefix func(ctx context.Context, prefix string) error) error {
	_, err := s.dbPool.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRowWithOptions(ctx, "GCCoord", spanner.Key{0}, []string{"fromSize"}, &spanner.ReadOptions{LockHint: spannerpb.ReadRequest_LOCK_HINT_EXCLUSIVE})
		if err != nil {
			return fmt.Errorf("failed to read GCCoord: %w", err)
		}
		var fs int64
		if err := row.Columns(&fs); err != nil {
			return fmt.Errorf("failed to parse row contents: %v", err)
		}
		fromSize := uint64(fs)

		if fromSize == treeSize {
			return nil
		}

		d := uint(0)
		eg := errgroup.Group{}
		// GC the tree in "vertical" chunks defined by entry bundles.
		for ri := range layout.Range(fromSize, treeSize-fromSize, treeSize) {
			// Only known-full bundles are in-scope for for GC, so exit if the current bundle is partial or
			// we've reached our limit of chunks.
			if ri.Partial > 0 || d > maxBundles {
				break
			}

			// GC any partial versions of the entry bundle itself and the tile which sits immediately above it.
			eg.Go(func() error { return deleteWithPrefix(ctx, layout.EntriesPath(ri.Index, 0)+".p/") })
			eg.Go(func() error { return deleteWithPrefix(ctx, layout.TilePath(0, ri.Index, 0)+".p/") })
			fromSize += uint64(ri.N)
			d++

			// Now consider (only) the part of the tree which sits above the bundle.
			// We'll walk up the parent tiles for as a long as we're tracing the right-hand
			// edge of a perfect subtree.
			// This gives the property we'll only visit each parent tile once, rather than up to 256 times.
			pL, pIdx := uint64(0), ri.Index
			for isLastLeafInParent(pIdx) {
				// Move our coordinates up to the parent
				pL, pIdx = pL+1, pIdx>>layout.TileHeight
				// GC any partial versions of the parent tile.
				eg.Go(func() error { return deleteWithPrefix(ctx, layout.TilePath(pL, pIdx, 0)+".p/") })

			}
		}
		if err := eg.Wait(); err != nil {
			return fmt.Errorf("failed to delete one or more objects: %v", err)
		}

		if err := txn.BufferWrite([]*spanner.Mutation{spanner.Update("GCCoord", []string{"id", "fromSize"}, []any{0, int64(fromSize)})}); err != nil {
			return err
		}

		return nil
	})
	return err
}

// isLastLeafInParent returns true if a tile with the provided index is the final child node of a
// (hypothetical) full parent tile.
func isLastLeafInParent(i uint64) bool {
	return i%layout.TileWidth == layout.TileWidth-1
}

// gcsStorage knows how to store and retrieve objects from GCS.
type gcsStorage struct {
	bucket       string
	bucketPrefix string
	gcsClient    *gcs.Client
}

// getObject returns the data and generation of the specified object, or an error.
func (s *gcsStorage) getObject(ctx context.Context, obj string) ([]byte, int64, error) {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.getObject")
	defer span.End()

	if s.bucketPrefix != "" {
		obj = filepath.Join(s.bucketPrefix, obj)
	}

	span.SetAttributes(objectPathKey.String(obj))

	r, err := s.gcsClient.Bucket(s.bucket).Object(obj).NewReader(ctx)
	if err != nil {
		return nil, -1, fmt.Errorf("getObject: failed to create reader for object %q in bucket %q: %w", obj, s.bucket, err)
	}

	d, err := io.ReadAll(r)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to read %q: %v", obj, err)
	}
	return d, r.Attrs.Generation, r.Close()
}

// setObject stores the provided data in the specified object, optionally gated by a condition.
//
// cond can be used to specify preconditions for the write (e.g. write iff not exists, write iff
// current generation is X, etc.), or nil can be passed if no preconditions are desired.
//
// Note that when preconditions are specified and are not met, an error will be returned *unless*
// the currently stored data is bit-for-bit identical to the data to-be-written.
// This is intended to provide idempotentency for writes.
func (s *gcsStorage) setObject(ctx context.Context, objName string, data []byte, cond *gcs.Conditions, contType string, cacheCtl string) error {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.setObject")
	defer span.End()

	if s.bucketPrefix != "" {
		objName = filepath.Join(s.bucketPrefix, objName)
	}

	span.SetAttributes(objectPathKey.String(objName))

	bkt := s.gcsClient.Bucket(s.bucket)
	obj := bkt.Object(objName)

	var w *gcs.Writer
	if cond == nil {
		w = obj.NewWriter(ctx)

	} else {
		w = obj.If(*cond).NewWriter(ctx)
	}
	w.ContentType = contType
	w.CacheControl = cacheCtl
	// Limit the amount of memory used for buffers, see https://pkg.go.dev/cloud.google.com/go/storage#Writer
	w.ChunkSize = len(data) + 1024
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write object %q to bucket %q: %w", objName, s.bucket, err)
	}

	if err := w.Close(); err != nil {
		// If we run into a precondition failure error, check that the object
		// which exists contains the same content that we want to write.
		// If so, we can consider this write to be idempotently successful.
		preconditionFailed := false

		// Helpfully, the mechanism for detecting a failed precodition differs depending
		// on whether you're using the HTTP or gRPC GCS client, so test both.
		if ee, ok := err.(*googleapi.Error); ok && ee.Code == http.StatusPreconditionFailed {
			preconditionFailed = true
		} else if st, ok := status.FromError(err); ok && st.Code() == codes.FailedPrecondition {
			preconditionFailed = true
		}
		if preconditionFailed {
			existing, existingGen, err := s.getObject(ctx, objName)
			if err != nil {
				return fmt.Errorf("failed to fetch existing content for %q (@%d): %v", objName, existingGen, err)
			}
			if !bytes.Equal(existing, data) {
				span.AddEvent("Non-idempotent write")
				klog.Errorf("Resource %q non-idempotent write:\n%s", objName, cmp.Diff(existing, data))
				return fmt.Errorf("precondition failed: resource content for %q differs from data to-be-written", objName)
			}

			span.AddEvent("Idempotent write")
			klog.V(2).Infof("setObject: identical resource already exists for %q, continuing", objName)
			return nil
		}

		return fmt.Errorf("failed to close write on %q: %v", objName, err)
	}
	return nil
}

// deleteObjectsWithPrefix removes any objects with the provided prefix from GCS.
func (s *gcsStorage) deleteObjectsWithPrefix(ctx context.Context, objPrefix string) error {
	ctx, span := tracer.Start(ctx, "tessera.storage.gcp.deleteObject")
	defer span.End()

	if s.bucketPrefix != "" {
		objPrefix = filepath.Join(s.bucketPrefix, objPrefix)
	}
	span.SetAttributes(objectPathKey.String(objPrefix))

	bkt := s.gcsClient.Bucket(s.bucket)

	errs := []error(nil)
	it := bkt.Objects(ctx, &gcs.Query{Prefix: objPrefix})
	for {
		attr, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return err
		}
		klog.V(2).Infof("Deleting object %s", attr.Name)
		if err := bkt.Object(attr.Name).Delete(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

// MigrationWriter creates a new GCP storage for the MigrationTarget lifecycle mode.
func (s *Storage) MigrationWriter(ctx context.Context, opts *tessera.MigrationOptions) (migrate.MigrationWriter, tessera.LogReader, error) {
	var err error
	if s.cfg.GCSClient == nil {
		s.cfg.GCSClient, err = gcs.NewClient(ctx, gcs.WithJSONReads())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create GCS client: %v", err)
		}
	}

	if s.cfg.SpannerClient == nil {
		s.cfg.SpannerClient, err = spanner.NewClient(ctx, s.cfg.Spanner)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect to Spanner: %v", err)
		}
	}
	if err := initDB(ctx, s.cfg.Spanner); err != nil {
		return nil, nil, fmt.Errorf("failed to verify/init Spanner schema: %v", err)
	}

	seq, err := newSpannerCoordinator(ctx, s.cfg.SpannerClient, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Spanner sequencer: %v", err)
	}
	m := &MigrationStorage{
		s:            s,
		dbPool:       seq.dbPool,
		bundleHasher: opts.LeafHasher(),
		sequencer:    seq,
		logStore: &logResourceStore{
			objStore: &gcsStorage{
				gcsClient:    s.cfg.GCSClient,
				bucket:       s.cfg.Bucket,
				bucketPrefix: s.cfg.BucketPrefix,
			},
			entriesPath: opts.EntriesPath(),
		},
	}

	r := &LogReader{
		lrs: *m.logStore,
		integratedSize: func(context.Context) (uint64, error) {
			s, _, err := m.sequencer.currentTree(ctx)
			return s, err
		},
		nextIndex: func(context.Context) (uint64, error) {
			return 0, nil
		},
	}
	return m, r, nil
}

// MigrationStorgage implements the tessera.MigrationTarget lifecycle contract.
type MigrationStorage struct {
	s            *Storage
	dbPool       *spanner.Client
	bundleHasher func([]byte) ([][]byte, error)
	sequencer    sequencer
	logStore     *logResourceStore
}

var _ migrate.MigrationWriter = &MigrationStorage{}

func (m *MigrationStorage) AwaitIntegration(ctx context.Context, sourceSize uint64) ([]byte, error) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-t.C:
			from, _, err := m.sequencer.currentTree(ctx)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				klog.Warningf("readTreeState: %v", err)
				continue
			}
			klog.Infof("Integrate from %d (Target %d)", from, sourceSize)
			newSize, newRoot, err := m.buildTree(ctx, sourceSize)
			if err != nil {
				klog.Warningf("integrate: %v", err)
			}
			if newSize == sourceSize {
				klog.Infof("Integrated to %d with roothash %x", newSize, newRoot)
				return newRoot, nil
			}
		}
	}
}

func (m *MigrationStorage) SetEntryBundle(ctx context.Context, index uint64, partial uint8, bundle []byte) error {
	return m.logStore.setEntryBundle(ctx, index, partial, bundle)
}

func (m *MigrationStorage) IntegratedSize(ctx context.Context) (uint64, error) {
	sz, _, err := m.sequencer.currentTree(ctx)
	return sz, err
}

func (m *MigrationStorage) fetchLeafHashes(ctx context.Context, from, to, sourceSize uint64) ([][]byte, error) {
	// TODO(al): Make this configurable.
	const maxBundles = 300

	toBeAdded := sync.Map{}
	eg := errgroup.Group{}
	n := 0
	for ri := range layout.Range(from, to, sourceSize) {
		eg.Go(func() error {
			b, err := m.logStore.getEntryBundle(ctx, ri.Index, ri.Partial)
			if err != nil {
				return fmt.Errorf("getEntryBundle(%d.%d): %v", ri.Index, ri.Partial, err)
			}

			bh, err := m.bundleHasher(b)
			if err != nil {
				return fmt.Errorf("bundleHasherFunc for bundle index %d: %v", ri.Index, err)
			}
			toBeAdded.Store(ri.Index, bh[ri.First:ri.First+ri.N])
			return nil
		})
		n++
		if n >= maxBundles {
			break
		}
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	lh := make([][]byte, 0, maxBundles)
	for i := from / layout.EntryBundleWidth; ; i++ {
		v, ok := toBeAdded.LoadAndDelete(i)
		if !ok {
			break
		}
		bh := v.([][]byte)
		lh = append(lh, bh...)
	}

	return lh, nil
}

func (m *MigrationStorage) buildTree(ctx context.Context, sourceSize uint64) (uint64, []byte, error) {
	var newSize uint64
	var newRoot []byte

	_, err := m.dbPool.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Figure out which is the starting index of sequenced entries to start consuming from.
		row, err := txn.ReadRowWithOptions(ctx, "IntCoord", spanner.Key{0}, []string{"seq", "rootHash"}, &spanner.ReadOptions{LockHint: spannerpb.ReadRequest_LOCK_HINT_EXCLUSIVE})
		if err != nil {
			return err
		}
		var fromSeq int64 // Spanner doesn't support uint64
		var rootHash []byte
		if err := row.Columns(&fromSeq, &rootHash); err != nil {
			return fmt.Errorf("failed to read integration coordination info: %v", err)
		}

		from := uint64(fromSeq)
		klog.V(1).Infof("Integrating from %d", from)
		lh, err := m.fetchLeafHashes(ctx, from, sourceSize, sourceSize)
		if err != nil {
			return fmt.Errorf("fetchLeafHashes(%d, %d, %d): %v", from, sourceSize, sourceSize, err)
		}

		if len(lh) == 0 {
			klog.Infof("Integrate: nothing to do, nothing done")
			// Set these to the current state of the tree so we reflect that in buildTree's return values.
			newSize, newRoot = from, rootHash
			return nil
		}

		added := uint64(len(lh))
		klog.Infof("Integrate: adding %d entries to existing tree size %d", len(lh), from)
		newRoot, err = integrate(ctx, from, lh, m.logStore)
		if err != nil {
			klog.Warningf("integrate failed: %v", err)
			return fmt.Errorf("integrate failed: %v", err)
		}
		newSize = from + added
		klog.Infof("Integrate: added %d entries", added)

		// integration was successful, so we can update our coordination row
		m := make([]*spanner.Mutation, 0)
		m = append(m, spanner.Update("IntCoord", []string{"id", "seq", "rootHash"}, []any{0, int64(from + added), newRoot}))
		return txn.BufferWrite(m)
	})

	if err != nil {
		return 0, nil, err
	}
	return newSize, newRoot, nil
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
