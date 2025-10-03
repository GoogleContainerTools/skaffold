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
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	f_log "github.com/transparency-dev/formats/log"
	"github.com/transparency-dev/merkle/rfc6962"
	"github.com/transparency-dev/tessera/api/layout"
	"github.com/transparency-dev/tessera/internal/otel"
	"github.com/transparency-dev/tessera/internal/parse"
	"github.com/transparency-dev/tessera/internal/witness"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/mod/sumdb/note"
	"k8s.io/klog/v2"
)

const (
	// DefaultBatchMaxSize is used by storage implementations if no WithBatching option is provided when instantiating it.
	DefaultBatchMaxSize = 256
	// DefaultBatchMaxAge is used by storage implementations if no WithBatching option is provided when instantiating it.
	DefaultBatchMaxAge = 250 * time.Millisecond
	// DefaultCheckpointInterval is used by storage implementations if no WithCheckpointInterval option is provided when instantiating it.
	DefaultCheckpointInterval = 10 * time.Second
	// DefaultPushbackMaxOutstanding is used by storage implementations if no WithPushback option is provided when instantiating it.
	DefaultPushbackMaxOutstanding = 4096
	// DefaultGarbageCollectionInterval is the default value used if no WithGarbageCollectionInterval option is provided.
	DefaultGarbageCollectionInterval = time.Minute
	// DefaultAntispamInMemorySize is the recommended default limit on the number of entries in the in-memory antispam cache.
	// The amount of data stored for each entry is small (32 bytes of hash + 8 bytes of index), so in the general case it should be fine
	// to have a very large cache.
	DefaultAntispamInMemorySize = 256 << 10
)

var (
	appenderAddsTotal         metric.Int64Counter
	appenderAddHistogram      metric.Int64Histogram
	appenderHighestIndex      metric.Int64Gauge
	appenderIntegratedSize    metric.Int64Gauge
	appenderIntegrateLatency  metric.Int64Histogram
	appenderDeadlineRemaining metric.Int64Histogram
	appenderNextIndex         metric.Int64Gauge
	appenderSignedSize        metric.Int64Gauge
	appenderWitnessedSize     metric.Int64Gauge
	appenderWitnessRequests   metric.Int64Counter

	followerEntriesProcessed metric.Int64Gauge
	followerLag              metric.Int64Gauge

	// Custom histogram buckets as we're still interested in details in the 1-2s area.
	histogramBuckets = []float64{0, 10, 50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000, 1200, 1400, 1600, 1800, 2000, 2500, 3000, 4000, 5000, 6000, 8000, 10000}
)

func init() {
	var err error

	appenderAddsTotal, err = meter.Int64Counter(
		"tessera.appender.add.calls",
		metric.WithDescription("Number of calls to the appender lifecycle Add function"),
		metric.WithUnit("{call}"))
	if err != nil {
		klog.Exitf("Failed to create appenderAddsTotal metric: %v", err)
	}

	appenderAddHistogram, err = meter.Int64Histogram(
		"tessera.appender.add.duration",
		metric.WithDescription("Duration of calls to the appender lifecycle Add function"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(histogramBuckets...))
	if err != nil {
		klog.Exitf("Failed to create appenderAddDuration metric: %v", err)
	}

	appenderHighestIndex, err = meter.Int64Gauge(
		"tessera.appender.index",
		metric.WithDescription("Highest index assigned by appender lifecycle Add function"))
	if err != nil {
		klog.Exitf("Failed to create appenderHighestIndex metric: %v", err)
	}

	appenderIntegratedSize, err = meter.Int64Gauge(
		"tessera.appender.integrated.size",
		metric.WithDescription("Size of the integrated (but not necessarily published) tree"),
		metric.WithUnit("{entry}"))
	if err != nil {
		klog.Exitf("Failed to create appenderIntegratedSize metric: %v", err)
	}

	appenderIntegrateLatency, err = meter.Int64Histogram(
		"tessera.appender.integrate.latency",
		metric.WithDescription("Duration between an index being assigned by Add, and that index being integrated in the tree"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(histogramBuckets...))
	if err != nil {
		klog.Exitf("Failed to create appenderIntegrateLatency metric: %v", err)
	}

	appenderDeadlineRemaining, err = meter.Int64Histogram(
		"tessera.appender.deadline.remaining",
		metric.WithDescription("Duration remaining before context cancellation when appender is invoked (only set for contexts with deadline)"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(histogramBuckets...))
	if err != nil {
		klog.Exitf("Failed to create appenderDeadlineRemaining metric: %v", err)
	}

	appenderNextIndex, err = meter.Int64Gauge(
		"tessera.appender.next_index",
		metric.WithDescription("The next available index to be assigned to entries"))
	if err != nil {
		klog.Exitf("Failed to create appenderNextIndex metric: %v", err)
	}

	appenderSignedSize, err = meter.Int64Gauge(
		"tessera.appender.signed.size",
		metric.WithDescription("Size of the latest signed checkpoint"),
		metric.WithUnit("{entry}"))
	if err != nil {
		klog.Exitf("Failed to create appenderSignedSize metric: %v", err)
	}

	appenderWitnessedSize, err = meter.Int64Gauge(
		"tessera.appender.witnessed.size",
		metric.WithDescription("Size of the latest successfully witnessed checkpoint"),
		metric.WithUnit("{entry}"))
	if err != nil {
		klog.Exitf("Failed to create appenderWitnessedSize metric: %v", err)
	}

	followerEntriesProcessed, err = meter.Int64Gauge(
		"tessera.follower.processed",
		metric.WithDescription("Number of entries processed"),
		metric.WithUnit("{entry}"))
	if err != nil {
		klog.Exitf("Failed to create followerEntriesProcessed metric: %v", err)
	}

	followerLag, err = meter.Int64Gauge(
		"tessera.follower.lag",
		metric.WithDescription("Number of unprocessed entries in the current integrated tree"),
		metric.WithUnit("{entry}"))
	if err != nil {
		klog.Exitf("Failed to create followerLag metric: %v", err)
	}

	appenderWitnessRequests, err = meter.Int64Counter(
		"tessera.appender.witness.requests",
		metric.WithDescription("Number of attempts to witness a log checkpoint"),
		metric.WithUnit("{call}"))
	if err != nil {
		klog.Exitf("Failed to create appenderWitnessRequests metric: %v", err)
	}

}

// AddFn adds a new entry to be sequenced by the storage implementation.
//
// This method should quickly return an IndexFuture, which can be called to resolve to the
// index **durably** assigned to the new entry (or an error).
//
// Implementations MUST NOT allow the future to resolve to an index value unless/until it has
// been durably committed by the storage.
//
// Callers MUST NOT assume that an entry has been accepted or durably stored until they have
// successfully resolved the future.
//
// Once the future resolves and returns an index, the entry can be considered to have been
// durably sequenced and will be preserved even in the event that the process terminates.
//
// Once an entry is sequenced, the storage implementation MUST integrate it into the tree soon
// (how long this is expected to take is left unspecified, but as a guideline it should happen
// within single digit seconds). Until the entry is integrated and published, clients of the log
// will not be able to verifiably access this value.
//
// Personalities which require blocking until the entry is integrated (e.g. because they wish
// to return an inclusion proof) may use the PublicationAwaiter to wrap the call to this method.
type AddFn func(ctx context.Context, entry *Entry) IndexFuture

// IndexFuture is the signature of a function which can return an assigned index or error.
//
// Implementations of this func are likely to be "futures", or a promise to return this data at
// some point in the future, and as such will block when called if the data isn't yet available.
type IndexFuture func() (Index, error)

// Index represents a durably assigned index for some entry.
type Index struct {
	// Index is the location in the log to which a particular entry has been assigned.
	Index uint64
	// IsDup is true if Index represents a previously assigned index for an identical entry.
	IsDup bool
}

// Appender allows personalities access to the lifecycle methods associated with logs
// in sequencing mode. This only has a single method, but other methods are likely to be added
// such as a Shutdown method for #341.
type Appender struct {
	Add AddFn
}

// NewAppender returns an Appender, which allows a personality to incrementally append new
// leaves to the log and to read from it.
//
// The return values are the Appender for adding new entries, a shutdown function, a log reader,
// and an error if any of the objects couldn't be constructed.
//
// Shutdown ensures that all calls to Add that have returned a value will be resolved. Any
// futures returned by _this appender_ which resolve to an index will be integrated and have
// a checkpoint that commits to them published if this returns successfully. After this returns,
// any calls to Add will fail.
//
// The context passed into this function will be referenced by any background tasks that are started
// in the Appender. The correct process for shutting down an Appender cleanly is to first call the
// shutdown function that is returned, and then cancel the context. Cancelling the context without calling
// shutdown first may mean that some entries added by this appender aren't in the log when the process
// exits.
func NewAppender(ctx context.Context, d Driver, opts *AppendOptions) (*Appender, func(ctx context.Context) error, LogReader, error) {
	type appendLifecycle interface {
		Appender(context.Context, *AppendOptions) (*Appender, LogReader, error)
	}
	lc, ok := d.(appendLifecycle)
	if !ok {
		return nil, nil, nil, fmt.Errorf("driver %T does not implement Appender lifecycle", d)
	}
	if opts == nil {
		return nil, nil, nil, errors.New("opts cannot be nil")
	}
	if err := opts.valid(); err != nil {
		return nil, nil, nil, err
	}
	a, r, err := lc.Appender(ctx, opts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to init appender lifecycle: %v", err)
	}
	for i := len(opts.addDecorators) - 1; i >= 0; i-- {
		a.Add = opts.addDecorators[i](a.Add)
	}
	sd := &integrationStats{}
	a.Add = sd.statsDecorator(a.Add)
	for _, f := range opts.followers {
		go f.Follow(ctx, r)
		go followerStats(ctx, f, r.IntegratedSize)
	}
	go sd.updateStats(ctx, r)
	t := terminator{
		delegate:       a.Add,
		readCheckpoint: r.ReadCheckpoint,
	}
	// TODO(mhutchinson): move this into the decorators
	a.Add = func(ctx context.Context, entry *Entry) IndexFuture {
		if deadline, ok := ctx.Deadline(); ok {
			appenderDeadlineRemaining.Record(ctx, time.Until(deadline).Milliseconds())
		}
		ctx, span := tracer.Start(ctx, "tessera.Appender.Add")
		defer span.End()

		// NOTE: We memoize the returned value here so that repeated calls to the returned
		//		 future don't result in unexpected side-effects from inner AddFn functions
		//		 being called multiple times.
		//		 Currently this is the outermost wrapping of Add so we do the memoization
		//		 here, if this changes, ensure that we move the memoization call so that
		//		 this remains true.
		return memoizeFuture(t.Add(ctx, entry))
	}
	return a, t.Shutdown, r, nil
}

// memoizeFuture wraps an AddFn delegate with logic to ensure that the delegate is called at most
// once.
func memoizeFuture(delegate IndexFuture) IndexFuture {
	f := sync.OnceValues(func() (Index, error) {
		return delegate()
	})
	return f
}

func followerStats(ctx context.Context, f Follower, size func(context.Context) (uint64, error)) {
	name := f.Name()
	t := time.NewTicker(200 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}

		n, err := f.EntriesProcessed(ctx)
		if err != nil {
			klog.Errorf("followerStats: follower %q EntriesProcessed(): %v", name, err)
			continue
		}
		s, err := size(ctx)
		if err != nil {
			klog.Errorf("followerStats: follower %q size(): %v", name, err)
		}
		attrs := metric.WithAttributes(followerNameKey.String(name))
		followerEntriesProcessed.Record(ctx, otel.Clamp64(n), attrs)
		followerLag.Record(ctx, otel.Clamp64(s-n), attrs)
	}
}

// idxAt represents an index first seen at a particular time.
type idxAt struct {
	idx uint64
	at  time.Time
}

// integrationStats knows how to track and populate metrics related to integration performance.
//
// Currently, this tracks integration latency only.
// The integration latency tracking works via a "sample & consume" mechanism, whereby an Add decorator
// will record an assigned index along with the time it was assigned. An asynchronous process will
// periodically compare the sample with the current integrated tree size, and if the sampled index is
// found to be covered by the tree the elapsed period is recorded and the sample "consumed".
//
// Only one sample may be held at a time.
type integrationStats struct {
	// indexSample points to a sampled indexAt, or nil if there has been no sample made _or_ the sample was consumed.
	indexSample atomic.Pointer[idxAt]
}

// sample creates a new sample with the provided index if no sample is already held.
func (i *integrationStats) sample(idx uint64) {
	i.indexSample.CompareAndSwap(nil, &idxAt{idx: idx, at: time.Now()})
}

// latency will check whether the provided tree size is larger than the currently sampled index (if one exists),
// and, if so, "consume" the sample and return the elapsed interval since the sample was taken.
//
// The returned bool is true if a sample exists and whose index is lower than the provided tree size, and
// false otherwise.
func (i *integrationStats) latency(size uint64) (time.Duration, bool) {
	ia := i.indexSample.Load()
	// If there _is_ a sample...
	if ia != nil {
		// and the sampled index is lower than the tree size
		if ia.idx < size {
			// then reset the sample store here so that we're able to accept a future sample.
			i.indexSample.Store(nil)
		}
		return time.Since(ia.at), true
	}
	return 0, false
}

// updateStates periodically checks the current integrated tree size and attempts to
// consume any held sample, updating the metric if possible.
//
// This is a long running function, exitingly only when the provided context is done.
func (i *integrationStats) updateStats(ctx context.Context, r LogReader) {
	if r == nil {
		klog.Warning("updateStates: nil logreader provided, not updating stats")
		return
	}
	t := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		s, err := r.IntegratedSize(ctx)
		if err != nil {
			klog.Errorf("IntegratedSize: %v", err)
			continue
		}
		appenderIntegratedSize.Record(ctx, otel.Clamp64(s))
		if d, ok := i.latency(s); ok {
			appenderIntegrateLatency.Record(ctx, d.Milliseconds())
		}
		i, err := r.NextIndex(ctx)
		if err != nil {
			klog.Errorf("NextIndex: %v", err)
		}
		appenderNextIndex.Record(ctx, otel.Clamp64(i))
	}
}

// statsDecorator wraps a delegate AddFn with code to calculate/update
// metric stats.
func (i *integrationStats) statsDecorator(delegate AddFn) AddFn {
	return func(ctx context.Context, entry *Entry) IndexFuture {
		start := time.Now()
		f := delegate(ctx, entry)

		return func() (Index, error) {
			idx, err := f()
			attr := []attribute.KeyValue{}
			pushbackType := "" // This will be used for the pushback attribute below, empty string means no pushback

			if err != nil {
				if errors.Is(err, ErrPushback) {
					// record the the fact there was pushback, and use the error string as the type.
					pushbackType = err.Error()
				} else {
					// Just flag that it's an errored request to avoid high cardinality of attribute values.
					// TODO(al): We might want to bucket errors into OTel status codes in the future, though.
					attr = append(attr, attribute.String("tessera.error.type", "_OTHER"))
				}
			}

			attr = append(attr, attribute.String("tessera.pushback", strings.ReplaceAll(pushbackType, " ", "_")))
			attr = append(attr, attribute.Bool("tessera.duplicate", idx.IsDup))

			appenderAddsTotal.Add(ctx, 1, metric.WithAttributes(attr...))
			d := time.Since(start)
			appenderAddHistogram.Record(ctx, d.Milliseconds(), metric.WithAttributes(attr...))

			if !idx.IsDup {
				i.sample(idx.Index)
			}

			return idx, err
		}
	}
}

type terminator struct {
	delegate       AddFn
	readCheckpoint func(ctx context.Context) ([]byte, error)
	// This mutex guards the stopped state. We use this instead of an atomic.Boolean
	// to get the property that no readers of this state can have the lock when the
	// write gets it. This means that no in-flight Add operations will be occurring on
	// Shutdown.
	mu      sync.RWMutex
	stopped bool

	// largestIssued tracks the largest index allocated by this appender.
	largestIssued atomic.Uint64
}

func (t *terminator) Add(ctx context.Context, entry *Entry) IndexFuture {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.stopped {
		return func() (Index, error) {
			return Index{}, errors.New("appender has been shut down")
		}
	}
	res := t.delegate(ctx, entry)
	return func() (Index, error) {
		i, err := res()
		if err != nil {
			return i, err
		}

		// https://github.com/golang/go/issues/63999 - atomically set largest issued index
		old := t.largestIssued.Load()
		for old < i.Index && !t.largestIssued.CompareAndSwap(old, i.Index) {
			old = t.largestIssued.Load()
		}
		appenderHighestIndex.Record(ctx, otel.Clamp64(t.largestIssued.Load()))

		return i, err
	}
}

// Shutdown ensures that all calls to Add that have returned a value will be resolved. Any
// futures returned by _this appender_ which resolve to an index will be integrated and have
// a checkpoint that commits to them published if this returns successfully.
//
// After this returns, any calls to Add will fail.
func (t *terminator) Shutdown(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopped = true
	maxIndex := t.largestIssued.Load()
	if maxIndex == 0 {
		// special case no work done
		return nil
	}
	sleepTime := 0 * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(sleepTime)
		}
		sleepTime = 100 * time.Millisecond // after the first time, ensure we sleep in any other loops

		cp, err := t.readCheckpoint(ctx)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			continue
		}
		_, size, _, err := parse.CheckpointUnsafe(cp)
		if err != nil {
			return err
		}
		klog.V(1).Infof("Shutting down, waiting for checkpoint committing to size %d (current checkpoint is %d)", maxIndex, size)
		if size > maxIndex {
			return nil
		}
	}
}

// NewAppendOptions creates a new options struct for configuring appender lifecycle instances.
//
// These options are configured through the use of the various `With.*` function calls on the returned
// instance.
func NewAppendOptions() *AppendOptions {
	return &AppendOptions{
		batchMaxSize:              DefaultBatchMaxSize,
		batchMaxAge:               DefaultBatchMaxAge,
		entriesPath:               layout.EntriesPath,
		bundleIDHasher:            defaultIDHasher,
		checkpointInterval:        DefaultCheckpointInterval,
		addDecorators:             make([]func(AddFn) AddFn, 0),
		pushbackMaxOutstanding:    DefaultPushbackMaxOutstanding,
		garbageCollectionInterval: DefaultGarbageCollectionInterval,
	}
}

// AppendOptions holds settings for all storage implementations.
type AppendOptions struct {
	// newCP knows how to format and sign checkpoints.
	newCP func(ctx context.Context, size uint64, hash []byte) ([]byte, error)

	batchMaxAge  time.Duration
	batchMaxSize uint

	pushbackMaxOutstanding uint

	// EntriesPath knows how to format entry bundle paths.
	entriesPath func(n uint64, p uint8) string
	// bundleIDHasher knows how to create antispam leaf identities for entries in a serialised bundle.
	bundleIDHasher func([]byte) ([][]byte, error)

	checkpointInterval time.Duration
	witnesses          WitnessGroup
	witnessOpts        WitnessOptions

	addDecorators []func(AddFn) AddFn
	followers     []Follower

	// garbageCollectionInterval of zero should be interpreted as requesting garbage collection to be disabled.
	garbageCollectionInterval time.Duration
}

// valid returns an error if an invalid combination of options has been set, or nil otherwise.
func (o AppendOptions) valid() error {
	if o.newCP == nil {
		return errors.New("invalid AppendOptions: WithCheckpointSigner must be set")
	}
	return nil
}

// WithAntispam configures the appender to use the antispam mechanism to reduce the number of duplicates which
// can be added to the log.
//
// As a starting point, the minimum size of the of in-memory cache should be set to the configured PushbackThreshold
// of the provided antispam implementation, multiplied by the number of concurrent front-end instances which
// are accepting write-traffic. Data stored in the in-memory cache is relatively small (32 bytes hash, 8 bytes index),
// so we recommend erring on the larger side as there is little downside to over-sizing the cache; consider using
// the DefaultAntispamInMemorySize as the value here.
//
// For more details on how the antispam mechanism works, including tuning guidance, see docs/design/antispam.md.
func (o *AppendOptions) WithAntispam(inMemEntries uint, as Antispam) *AppendOptions {
	o.addDecorators = append(o.addDecorators, newInMemoryDedup(inMemEntries))
	if as != nil {
		o.addDecorators = append(o.addDecorators, as.Decorator())
		o.followers = append(o.followers, as.Follower(o.bundleIDHasher))
	}
	return o
}

// CheckpointPublisher returns a function which should be used to create, sign, and potentially witness a new checkpoint.
func (o AppendOptions) CheckpointPublisher(lr LogReader, httpClient *http.Client) func(context.Context, uint64, []byte) ([]byte, error) {
	wg := witness.NewWitnessGateway(o.witnesses, httpClient, lr.ReadTile)
	return func(ctx context.Context, size uint64, root []byte) ([]byte, error) {
		ctx, span := tracer.Start(ctx, "tessera.CheckpointPublisher")
		defer span.End()

		cp, err := o.newCP(ctx, size, root)
		if err != nil {
			return nil, fmt.Errorf("newCP: %v", err)
		}
		appenderSignedSize.Record(ctx, otel.Clamp64(size))

		witAttr := []attribute.KeyValue{}
		cp, err = wg.Witness(ctx, cp)
		if err != nil {
			if !o.witnessOpts.FailOpen {
				appenderWitnessRequests.Add(ctx, 1, metric.WithAttributes(attribute.String("error.type", "failed")))
				return nil, err
			}
			klog.Warningf("WitnessGateway: failing-open despite error: %v", err)
			witAttr = append(witAttr, attribute.String("error.type", "failed_open"))
		}

		appenderWitnessRequests.Add(ctx, 1, metric.WithAttributes(witAttr...))
		appenderWitnessedSize.Record(ctx, otel.Clamp64(size))

		return cp, nil
	}
}

func (o AppendOptions) BatchMaxAge() time.Duration {
	return o.batchMaxAge
}

func (o AppendOptions) BatchMaxSize() uint {
	return o.batchMaxSize
}

func (o AppendOptions) PushbackMaxOutstanding() uint {
	return o.pushbackMaxOutstanding
}

func (o AppendOptions) EntriesPath() func(uint64, uint8) string {
	return o.entriesPath
}

func (o AppendOptions) CheckpointInterval() time.Duration {
	return o.checkpointInterval
}

func (o AppendOptions) GarbageCollectionInterval() time.Duration {
	return o.garbageCollectionInterval
}

// WithCheckpointSigner is an option for setting the note signer and verifier to use when creating and parsing checkpoints.
// This option is mandatory for creating logs where the checkpoint is signed locally, e.g. in
// the Appender mode. This does not need to be provided where the storage will be used to mirror
// other logs.
//
// A primary signer must be provided:
// - the primary signer is the "canonical" signing identity which should be used when creating new checkpoints.
//
// Zero or more dditional signers may also be provided.
// This enables cases like:
//   - a rolling key rotation, where checkpoints are signed by both the old and new keys for some period of time,
//   - using different signature schemes for different audiences, etc.
//
// When providing additional signers, their names MUST be identical to the primary signer name, and this name will be used
// as the checkpoint Origin line.
//
// Checkpoints signed by these signer(s) will be standard checkpoints as defined by https://c2sp.org/tlog-checkpoint.
func (o *AppendOptions) WithCheckpointSigner(s note.Signer, additionalSigners ...note.Signer) *AppendOptions {
	origin := s.Name()
	for _, signer := range additionalSigners {
		if origin != signer.Name() {
			klog.Exitf("WithCheckpointSigner: additional signer name (%q) does not match primary signer name (%q)", signer.Name(), origin)
		}
	}
	o.newCP = func(ctx context.Context, size uint64, hash []byte) ([]byte, error) {
		_, span := tracer.Start(ctx, "tessera.SignCheckpoint")
		defer span.End()

		// If we're signing a zero-sized tree, the tlog-checkpoint spec says (via RFC6962) that
		// the root must be SHA256 of the empty string, so we'll enforce that here:
		if size == 0 {
			emptyRoot := rfc6962.DefaultHasher.EmptyRoot()
			hash = emptyRoot[:]
		}
		cpRaw := f_log.Checkpoint{
			Origin: origin,
			Size:   size,
			Hash:   hash,
		}.Marshal()

		n, err := note.Sign(&note.Note{Text: string(cpRaw)}, append([]note.Signer{s}, additionalSigners...)...)
		if err != nil {
			return nil, fmt.Errorf("note.Sign: %w", err)
		}

		return n, nil
	}
	return o
}

// WithBatching configures the batching behaviour of leaves being sequenced.
// A batch will be allowed to grow in memory until either:
//   - the number of entries in the batch reach maxSize
//   - the first entry in the batch has reached maxAge
//
// At this point the batch will be sent to the sequencer.
//
// Configuring these parameters allows the personality to tune to get the desired
// balance of sequencing latency with cost. In general, larger batches allow for
// lower cost of operation, where more frequent batches reduce the amount of time
// required for entries to be included in the log.
//
// If this option isn't provided, storage implementations with use the DefaultBatchMaxSize and DefaultBatchMaxAge consts above.
func (o *AppendOptions) WithBatching(maxSize uint, maxAge time.Duration) *AppendOptions {
	o.batchMaxSize = maxSize
	o.batchMaxAge = maxAge
	return o
}

// WithPushback allows configuration of when the storage should start pushing back on add requests.
//
// maxOutstanding is the number of "in-flight" add requests - i.e. the number of entries with sequence numbers
// assigned, but which are not yet integrated into the log.
func (o *AppendOptions) WithPushback(maxOutstanding uint) *AppendOptions {
	o.pushbackMaxOutstanding = maxOutstanding
	return o
}

// WithCheckpointInterval configures the frequency at which Tessera will attempt to create & publish
// a new checkpoint.
//
// Well behaved clients of the log will only "see" newly sequenced entries once a new checkpoint is published,
// so it's important to set that value such that it works well with your ecosystem.
//
// Regularly publishing new checkpoints:
//   - helps show that the log is "live", even if no entries are being added.
//   - enables clients of the log to reason about how frequently they need to have their
//     view of the log refreshed, which in turn helps reduce work/load across the ecosystem.
//
// Note that this option probably only makes sense for long-lived applications (e.g. HTTP servers).
//
// If this option isn't provided, storage implementations will use the DefaultCheckpointInterval const above.
func (o *AppendOptions) WithCheckpointInterval(interval time.Duration) *AppendOptions {
	o.checkpointInterval = interval
	return o
}

// WithWitnesses configures the set of witnesses that Tessera will contact in order to counter-sign
// a checkpoint before publishing it. A request will be sent to every witness referenced by the group
// using the URLs method. The checkpoint will be accepted for publishing when a sufficient number of
// witnesses to Satisfy the group have responded.
//
// If this method is not called, then the default empty WitnessGroup will be used, which contacts zero
// witnesses and requires zero witnesses in order to publish.
func (o *AppendOptions) WithWitnesses(witnesses WitnessGroup, opts *WitnessOptions) *AppendOptions {
	if opts == nil {
		opts = &WitnessOptions{}
	}

	o.witnesses = witnesses
	o.witnessOpts = *opts
	return o
}

// WitnessOptions contains extra optional configuration for how Tessera should use/interact with
// a user-provided WitnessGroup policy.
type WitnessOptions struct {
	// FailOpen controls whether a checkpoint, for which the witness policy was unable to be met,
	// should still be published.
	//
	// This setting is intended only for facilitating early "non-blocking" adoption of witnessing,
	// and will be disabled and/or removed in the future.
	FailOpen bool
}

// WithGarbageCollectionInterval allows the interval between scans to remove obsolete partial
// tiles and entry bundles.
//
// Setting to zero disables garbage collection.
func (o *AppendOptions) WithGarbageCollectionInterval(interval time.Duration) *AppendOptions {
	o.garbageCollectionInterval = interval
	return o
}
