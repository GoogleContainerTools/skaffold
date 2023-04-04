package updater

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/jmhodges/clock"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/types/known/emptypb"

	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/crl"
	cspb "github.com/letsencrypt/boulder/crl/storer/proto"
	"github.com/letsencrypt/boulder/issuance"
	blog "github.com/letsencrypt/boulder/log"
	sapb "github.com/letsencrypt/boulder/sa/proto"
)

type crlUpdater struct {
	issuers        map[issuance.IssuerNameID]*issuance.Certificate
	numShards      int
	shardWidth     time.Duration
	lookbackPeriod time.Duration
	updatePeriod   time.Duration
	updateOffset   time.Duration
	maxParallelism int

	sa sapb.StorageAuthorityReadOnlyClient
	ca capb.CRLGeneratorClient
	cs cspb.CRLStorerClient

	tickHistogram  *prometheus.HistogramVec
	updatedCounter *prometheus.CounterVec

	log blog.Logger
	clk clock.Clock
}

func NewUpdater(
	issuers []*issuance.Certificate,
	numShards int,
	shardWidth time.Duration,
	lookbackPeriod time.Duration,
	updatePeriod time.Duration,
	updateOffset time.Duration,
	maxParallelism int,
	sa sapb.StorageAuthorityReadOnlyClient,
	ca capb.CRLGeneratorClient,
	cs cspb.CRLStorerClient,
	stats prometheus.Registerer,
	log blog.Logger,
	clk clock.Clock,
) (*crlUpdater, error) {
	issuersByNameID := make(map[issuance.IssuerNameID]*issuance.Certificate, len(issuers))
	for _, issuer := range issuers {
		issuersByNameID[issuer.NameID()] = issuer
	}

	if numShards < 1 {
		return nil, fmt.Errorf("must have positive number of shards, got: %d", numShards)
	}

	if updatePeriod >= 7*24*time.Hour {
		return nil, fmt.Errorf("must update CRLs at least every 7 days, got: %s", updatePeriod)
	}

	if updateOffset >= updatePeriod {
		return nil, fmt.Errorf("update offset must be less than period: %s !< %s", updateOffset, updatePeriod)
	}

	if lookbackPeriod < 2*updatePeriod {
		return nil, fmt.Errorf("lookbackPeriod must be at least 2x updatePeriod: %s !< 2 * %s", lookbackPeriod, updatePeriod)
	}

	if maxParallelism <= 0 {
		maxParallelism = 1
	}

	tickHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "crl_updater_ticks",
		Help:    "A histogram of crl-updater tick latencies labeled by issuer and result",
		Buckets: []float64{0.01, 0.2, 0.5, 1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000},
	}, []string{"issuer", "result"})
	stats.MustRegister(tickHistogram)

	updatedCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "crl_updater_generated",
		Help: "A counter of CRL generation calls labeled by result",
	}, []string{"issuer", "result"})
	stats.MustRegister(updatedCounter)

	return &crlUpdater{
		issuersByNameID,
		numShards,
		shardWidth,
		lookbackPeriod,
		updatePeriod,
		updateOffset,
		maxParallelism,
		sa,
		ca,
		cs,
		tickHistogram,
		updatedCounter,
		log,
		clk,
	}, nil
}

// Run causes the crlUpdater to enter its processing loop. It waits until the
// next scheduled run time based on the current time and the updateOffset, then
// begins running once every updatePeriod.
func (cu *crlUpdater) Run(ctx context.Context) error {
	// We don't want the times at which crlUpdater runs to be dependent on when
	// the process starts. So wait until the appropriate time before kicking off
	// the first run and the main ticker loop.
	currOffset := cu.clk.Now().UnixNano() % cu.updatePeriod.Nanoseconds()
	var waitNanos int64
	if currOffset <= cu.updateOffset.Nanoseconds() {
		waitNanos = cu.updateOffset.Nanoseconds() - currOffset
	} else {
		waitNanos = cu.updatePeriod.Nanoseconds() - currOffset + cu.updateOffset.Nanoseconds()
	}
	cu.log.Infof("Running, next tick in %ds", waitNanos*int64(time.Nanosecond)/int64(time.Second))
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(waitNanos)):
	}

	// Tick once immediately, but create the ticker first so that it starts
	// counting from the appropriate time.
	ticker := time.NewTicker(cu.updatePeriod)
	cu.Tick(ctx, cu.clk.Now())

	for {
		// If we have overrun *and* been canceled, both of the below cases could be
		// selectable at the same time, so check for context cancellation first.
		if ctx.Err() != nil {
			ticker.Stop()
			return ctx.Err()
		}
		select {
		case <-ticker.C:
			atTime := cu.clk.Now()
			err := cu.Tick(ctx, atTime)
			if err != nil {
				// We only log, rather than return, so that the long-lived process can
				// continue and try again at the next tick.
				cu.log.AuditErrf(
					"Generating CRLs failed: number=[%s] err=[%s]",
					(*big.Int)(crl.Number(atTime)), err)
			}
		case <-ctx.Done():
			ticker.Stop()
			return ctx.Err()
		}
	}
}

// Tick runs the entire update process once immediately. It processes each
// configured issuer serially, and processes all of them even if an early one
// encounters an error. All errors encountered are returned as a single combined
// error at the end.
func (cu *crlUpdater) Tick(ctx context.Context, atTime time.Time) (err error) {
	defer func() {
		// This func closes over the named return value `err`, so can reference it.
		result := "success"
		if err != nil {
			result = "failed"
		}
		cu.tickHistogram.WithLabelValues("all", result).Observe(cu.clk.Since(atTime).Seconds())
	}()
	cu.log.Debugf("Ticking at time %s", atTime)

	var errIssuers []string
	for id := range cu.issuers {
		// For now, process each issuer serially. This keeps the worker pool system
		// simple, and processing all of the issuers in parallel likely wouldn't
		// meaningfully speed up the overall process.
		err := cu.tickIssuer(ctx, atTime, id)
		if err != nil {
			cu.log.AuditErrf(
				"Generating CRLs for issuer failed: number=[%d] issuer=[%s] err=[%s]",
				(*big.Int)(crl.Number(atTime)), cu.issuers[id].Subject.CommonName, err)
			errIssuers = append(errIssuers, cu.issuers[id].Subject.CommonName)
		}
	}

	if len(errIssuers) != 0 {
		return fmt.Errorf("%d issuers failed: %v", len(errIssuers), strings.Join(errIssuers, ", "))
	}
	return nil
}

// tickIssuer performs the full CRL issuance cycle for a single issuer cert. It
// processes all of the shards of this issuer's CRL concurrently, and processes
// all of them even if an early one encounters an error. All errors encountered
// are returned as a single combined error at the end.
func (cu *crlUpdater) tickIssuer(ctx context.Context, atTime time.Time, issuerNameID issuance.IssuerNameID) (err error) {
	start := cu.clk.Now()
	defer func() {
		// This func closes over the named return value `err`, so can reference it.
		result := "success"
		if err != nil {
			result = "failed"
		}
		cu.tickHistogram.WithLabelValues(cu.issuers[issuerNameID].Subject.CommonName+" (Overall)", result).Observe(cu.clk.Since(start).Seconds())
	}()
	cu.log.Debugf("Ticking issuer %d at time %s", issuerNameID, atTime)

	shardMap, err := cu.getShardMappings(ctx, atTime)
	if err != nil {
		return fmt.Errorf("computing shardmap: %w", err)
	}

	type shardResult struct {
		shardIdx int
		err      error
	}

	shardWorker := func(in <-chan int, out chan<- shardResult) {
		for idx := range in {
			select {
			case <-ctx.Done():
				return
			default:
				out <- shardResult{
					shardIdx: idx,
					err:      cu.tickShard(ctx, atTime, issuerNameID, idx, shardMap[idx]),
				}
			}
		}
	}

	shardIdxs := make(chan int, cu.numShards)
	shardResults := make(chan shardResult, cu.numShards)
	for i := 0; i < cu.maxParallelism; i++ {
		go shardWorker(shardIdxs, shardResults)
	}

	for shardIdx := 0; shardIdx < cu.numShards; shardIdx++ {
		shardIdxs <- shardIdx
	}
	close(shardIdxs)

	var errShards []int
	for i := 0; i < cu.numShards; i++ {
		res := <-shardResults
		if res.err != nil {
			cu.log.AuditErrf(
				"Generating CRL failed: id=[%s] err=[%s]",
				crl.Id(issuerNameID, crl.Number(atTime), res.shardIdx), res.err)
			errShards = append(errShards, res.shardIdx)
		}
	}

	if len(errShards) != 0 {
		sort.Ints(errShards)
		return fmt.Errorf("%d shards failed: %v", len(errShards), errShards)
	}
	return nil
}

// tickShard processes a single shard. It computes the shard's boundaries, gets
// the list of revoked certs in that shard from the SA, gets the CA to sign the
// resulting CRL, and gets the crl-storer to upload it. It returns an error if
// any of these operations fail.
func (cu *crlUpdater) tickShard(ctx context.Context, atTime time.Time, issuerNameID issuance.IssuerNameID, shardIdx int, chunks []chunk) (err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	crlID := crl.Id(issuerNameID, crl.Number(atTime), shardIdx)

	start := cu.clk.Now()
	defer func() {
		// This func closes over the named return value `err`, so can reference it.
		result := "success"
		if err != nil {
			result = "failed"
		}
		cu.tickHistogram.WithLabelValues(cu.issuers[issuerNameID].Subject.CommonName, result).Observe(cu.clk.Since(start).Seconds())
		cu.updatedCounter.WithLabelValues(cu.issuers[issuerNameID].Subject.CommonName, result).Inc()
	}()

	cu.log.Infof(
		"Generating CRL shard: id=[%s] numChunks=[%d]", crlID, len(chunks))

	// Get the full list of CRL Entries for this shard from the SA.
	var crlEntries []*proto.CRLEntry
	for _, chunk := range chunks {
		saStream, err := cu.sa.GetRevokedCerts(ctx, &sapb.GetRevokedCertsRequest{
			IssuerNameID:  int64(issuerNameID),
			ExpiresAfter:  chunk.start.UnixNano(),
			ExpiresBefore: chunk.end.UnixNano(),
			RevokedBefore: atTime.UnixNano(),
		})
		if err != nil {
			return fmt.Errorf("connecting to SA: %w", err)
		}

		for {
			entry, err := saStream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("retrieving entry from SA: %w", err)
			}
			crlEntries = append(crlEntries, entry)
		}

		cu.log.Infof(
			"Queried SA for CRL shard: id=[%s] expiresAfter=[%s] expiresBefore=[%s] numEntries=[%d]",
			crlID, chunk.start, chunk.end, len(crlEntries))
	}

	// Send the full list of CRL Entries to the CA.
	caStream, err := cu.ca.GenerateCRL(ctx)
	if err != nil {
		return fmt.Errorf("connecting to CA: %w", err)
	}

	err = caStream.Send(&capb.GenerateCRLRequest{
		Payload: &capb.GenerateCRLRequest_Metadata{
			Metadata: &capb.CRLMetadata{
				IssuerNameID: int64(issuerNameID),
				ThisUpdate:   atTime.UnixNano(),
				ShardIdx:     int64(shardIdx),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("sending CA metadata: %w", err)
	}

	for _, entry := range crlEntries {
		err = caStream.Send(&capb.GenerateCRLRequest{
			Payload: &capb.GenerateCRLRequest_Entry{
				Entry: entry,
			},
		})
		if err != nil {
			return fmt.Errorf("sending entry to CA: %w", err)
		}
	}

	err = caStream.CloseSend()
	if err != nil {
		return fmt.Errorf("closing CA request stream: %w", err)
	}

	// Receive the full bytes of the signed CRL from the CA.
	crlLen := 0
	crlHash := sha256.New()
	var crlChunks [][]byte
	for {
		out, err := caStream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("receiving CRL bytes: %w", err)
		}

		crlLen += len(out.Chunk)
		crlHash.Write(out.Chunk)
		crlChunks = append(crlChunks, out.Chunk)
	}

	// Send the full bytes of the signed CRL to the Storer.
	csStream, err := cu.cs.UploadCRL(ctx)
	if err != nil {
		return fmt.Errorf("connecting to CRLStorer: %w", err)
	}

	err = csStream.Send(&cspb.UploadCRLRequest{
		Payload: &cspb.UploadCRLRequest_Metadata{
			Metadata: &cspb.CRLMetadata{
				IssuerNameID: int64(issuerNameID),
				Number:       atTime.UnixNano(),
				ShardIdx:     int64(shardIdx),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("sending CRLStorer metadata: %w", err)
	}

	for _, chunk := range crlChunks {
		err = csStream.Send(&cspb.UploadCRLRequest{
			Payload: &cspb.UploadCRLRequest_CrlChunk{
				CrlChunk: chunk,
			},
		})
		if err != nil {
			return fmt.Errorf("uploading CRL bytes: %w", err)
		}
	}

	_, err = csStream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("closing CRLStorer upload stream: %w", err)
	}

	cu.log.Infof(
		"Generated CRL shard: id=[%s] size=[%d] hash=[%x]",
		crlID, crlLen, crlHash.Sum(nil))
	return nil
}

// anchorTime is used as a universal starting point against which other times
// can be compared. This time must be less than 290 years (2^63-1 nanoseconds)
// in the past, to ensure that Go's time.Duration can represent that difference.
// The significance of 2015-06-04 11:04:38 UTC is left as an exercise to the
// reader.
func anchorTime() time.Time {
	return time.Date(2015, time.June, 04, 11, 04, 38, 0, time.UTC)
}

// chunk represents a fixed slice of time during which some certificates
// presumably expired or will expire. Its non-unique index indicates which shard
// it will be mapped to. The start boundary is inclusive, the end boundary is
// exclusive.
type chunk struct {
	start time.Time
	end   time.Time
	idx   int
}

// shardMap is a mapping of shard indices to the set of chunks which should be
// included in that shard. Under most circumstances there is a one-to-one
// mapping, but certain configuration (such as having very narrow shards, or
// having a very long lookback period) can result in more than one chunk being
// mapped to a single shard.
type shardMap [][]chunk

// getShardMappings determines which chunks are currently relevant, based on
// the current time, the configured lookbackPeriod, and the farthest-future
// certificate expiration in the database. It then maps all of those chunks to
// their corresponding shards, and returns that mapping.
//
// The idea here is that shards should be stable. Picture a timeline, divided
// into chunks. Number those chunks from 0 (starting at the anchor time) up to
// numShards, then repeat the cycle when you run out of numbers:
//
//	chunk:  0     1     2     3     4     0     1     2     3     4     0
//	     |-----|-----|-----|-----|-----|-----|-----|-----|-----|-----|-----...
//	     ^-anchorTime
//
// The total time window we care about goes from atTime-lookbackPeriod, forward
// through the time of the farthest-future notAfter date found in the database.
// The lookbackPeriod must be larger than the updatePeriod, to ensure that any
// certificates which were both revoked *and* expired since the last time we
// issued CRLs get included in this generation. Because these times are likely
// to fall in the middle of chunks, we include the whole chunks surrounding
// those times in our output CRLs:
//
//	included chunk:     4     0     1     2     3     4     0     1
//	      ...--|-----|-----|-----|-----|-----|-----|-----|-----|-----|-----...
//	atTime-lookbackPeriod-^   ^-atTime                lastExpiry-^
//
// Because this total period of time may include multiple chunks with the same
// number, we then coalesce these chunks into a single shard. Ideally, this
// will never happen: it should only happen if the lookbackPeriod is very
// large, or if the shardWidth is small compared to the lastExpiry (such that
// numShards * shardWidth is less than lastExpiry - atTime). In this example,
// shards 0, 1, and 4 all get the contents of two chunks mapped to them, while
// shards 2 and 3 get only one chunk each.
//
//	included chunk:     4     0     1     2     3     4     0     1
//	      ...--|-----|-----|-----|-----|-----|-----|-----|-----|-----|-----...
//	                    │     │     │     │     │     │     │     │
//	shard 0: <────────────────┘─────────────────────────────┘     │
//	shard 1: <──────────────────────┘─────────────────────────────┘
//	shard 2: <────────────────────────────┘     │     │
//	shard 3: <──────────────────────────────────┘     │
//	shard 4: <──────────┘─────────────────────────────┘
//
// Under this scheme, the shard to which any given certificate will be mapped is
// a function of only three things: that certificate's notAfter timestamp, the
// chunk width, and the number of shards.
func (cu *crlUpdater) getShardMappings(ctx context.Context, atTime time.Time) (shardMap, error) {
	res := make(shardMap, cu.numShards)

	// Get the farthest-future expiration timestamp to ensure we cover everything.
	lastExpiry, err := cu.sa.GetMaxExpiration(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	// Find the id number and boundaries of the earliest chunk we care about.
	first := atTime.Add(-cu.lookbackPeriod)
	c, err := cu.getChunkAtTime(first)
	if err != nil {
		return nil, err
	}

	// Iterate over chunks until we get completely beyond the farthest-future
	// expiration.
	for c.start.Before(lastExpiry.AsTime()) {
		res[c.idx] = append(res[c.idx], c)
		c = chunk{
			start: c.end,
			end:   c.end.Add(cu.shardWidth),
			idx:   (c.idx + 1) % cu.numShards,
		}
	}

	return res, nil
}

// getChunkAtTime returns the chunk whose boundaries contain the given time.
// It is broken out solely for the purpose of unit testing.
func (cu *crlUpdater) getChunkAtTime(atTime time.Time) (chunk, error) {
	// Compute the amount of time between the current time and the anchor time.
	timeSinceAnchor := atTime.Sub(anchorTime())
	if timeSinceAnchor == time.Duration(math.MaxInt64) || timeSinceAnchor < 0 {
		return chunk{}, errors.New("shard boundary math broken: anchor time too far away")
	}

	// Determine how many full chunks fit within that time, and from that the
	// index number of the desired chunk.
	chunksSinceAnchor := timeSinceAnchor.Nanoseconds() / cu.shardWidth.Nanoseconds()
	chunkIdx := int(chunksSinceAnchor) % cu.numShards

	// Determine the boundaries of the chunk.
	timeSinceChunk := time.Duration(timeSinceAnchor.Nanoseconds() % cu.shardWidth.Nanoseconds())
	left := atTime.Add(-timeSinceChunk)
	right := left.Add(cu.shardWidth)

	return chunk{left, right, chunkIdx}, nil
}
