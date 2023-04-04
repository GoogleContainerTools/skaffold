package updater

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jmhodges/clock"
	"github.com/prometheus/client_golang/prometheus"

	capb "github.com/letsencrypt/boulder/ca/proto"
	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/db"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/sa"
)

// ocspDb is an interface collecting the methods that the read/write parts of
// OCSPUpdater rely on. This allows the tests to swap out the db implementation.
type ocspDb interface {
	db.MappedExecutor
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// failCounter provides a concurrent safe counter.
type failCounter struct {
	mu    sync.Mutex
	count int
}

func (c *failCounter) Add(i int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count += i
}

func (c *failCounter) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count = 0
}

func (c *failCounter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// OCSPUpdater contains the useful objects for the Updater
type OCSPUpdater struct {
	log blog.Logger
	clk clock.Clock

	db         ocspDb
	readOnlyDb db.MappedExecutor

	ogc capb.OCSPGeneratorClient

	batchSize int

	tickWindow    time.Duration
	maxBackoff    time.Duration
	backoffFactor float64
	readFailures  failCounter

	serialSuffixes []string
	queryBody      string

	// Used to calculate how far back stale OCSP responses should be looked for
	ocspMinTimeToExpiry time.Duration
	// Maximum number of individual OCSP updates to attempt in parallel. Making
	// these requests in parallel allows us to get higher total throughput.
	parallelGenerateOCSPRequests int

	tickHistogram        *prometheus.HistogramVec
	stalenessHistogram   prometheus.Histogram
	genStoreHistogram    prometheus.Histogram
	generatedCounter     *prometheus.CounterVec
	storedCounter        *prometheus.CounterVec
	markExpiredCounter   *prometheus.CounterVec
	findStaleOCSPCounter *prometheus.CounterVec
}

func New(
	stats prometheus.Registerer,
	clk clock.Clock,
	db *db.WrappedMap,
	readOnlyDb *db.WrappedMap,
	serialSuffixes []string,
	ogc capb.OCSPGeneratorClient,
	batchSize int,
	windowSize time.Duration,
	retryBackoffMax time.Duration,
	retryBackoffFactor float64,
	ocspMinTimeToExpiry time.Duration,
	parallelGenerateOCSPRequests int,
	log blog.Logger,
) (*OCSPUpdater, error) {
	if batchSize == 0 {
		return nil, errors.New("loop batch sizes must be non-zero")
	}
	if windowSize == 0 {
		return nil, errors.New("loop window sizes must be non-zero")
	}
	if parallelGenerateOCSPRequests == 0 {
		// Default to 1
		parallelGenerateOCSPRequests = 1
	}
	for _, s := range serialSuffixes {
		if len(s) != 1 || strings.ToLower(s) != s {
			return nil, fmt.Errorf("serial suffixes must all be one lowercase character, got %q, expected %q", s, strings.ToLower(s))
		}
		c := s[0]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return nil, errors.New("valid range for suffixes is [0-9a-f]")
		}
	}

	var queryBody strings.Builder
	queryBody.WriteString("WHERE ocspLastUpdated < ? AND NOT isExpired ")
	if len(serialSuffixes) > 0 {
		fmt.Fprintf(&queryBody, "AND RIGHT(serial, 1) IN ( %s ) ",
			getQuestionsForShardList(len(serialSuffixes)),
		)
	}
	queryBody.WriteString("ORDER BY ocspLastUpdated ASC LIMIT ?")

	genStoreHistogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "ocsp_updater_generate_and_store",
		Help: "A histogram of latencies of OCSP generation and storage latencies",
	})
	stats.MustRegister(genStoreHistogram)
	generatedCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ocsp_updater_generated",
		Help: "A counter of OCSP response generation calls labeled by result",
	}, []string{"result"})
	stats.MustRegister(generatedCounter)
	storedCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ocsp_updater_stored",
		Help: "A counter of OCSP response storage calls labeled by result",
	}, []string{"result"})
	stats.MustRegister(storedCounter)
	tickHistogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ocsp_updater_ticks",
		Help:    "A histogram of ocsp-updater tick latencies labelled by result and whether the tick was considered longer than expected",
		Buckets: []float64{0.01, 0.2, 0.5, 1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000},
	}, []string{"result", "long"})
	stats.MustRegister(tickHistogram)
	stalenessHistogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "ocsp_status_staleness",
		Help:    "How long past the refresh time a status is when we try to refresh it. Will always be > 0, but must stay well below 12 hours.",
		Buckets: []float64{10, 100, 1000, 10000, 21600, 32400, 36000, 39600, 43200, 54000, 64800, 75600, 86400, 108000, 129600, 172800},
	})
	stats.MustRegister(stalenessHistogram)
	markExpiredCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "mark_expired",
		Help: "A counter of mark expired calls labeled by result",
	}, []string{"result"})
	stats.MustRegister(markExpiredCounter)
	findStaleOCSPCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "find_stale_ocsp",
		Help: "A counter of query for stale OCSP responses labeled by result",
	}, []string{"result"})
	stats.MustRegister(findStaleOCSPCounter)

	updater := OCSPUpdater{
		log:                          log,
		clk:                          clk,
		db:                           db,
		readOnlyDb:                   readOnlyDb,
		ogc:                          ogc,
		batchSize:                    batchSize,
		tickWindow:                   windowSize,
		maxBackoff:                   retryBackoffMax,
		backoffFactor:                retryBackoffFactor,
		readFailures:                 failCounter{},
		serialSuffixes:               serialSuffixes,
		queryBody:                    queryBody.String(),
		ocspMinTimeToExpiry:          ocspMinTimeToExpiry,
		parallelGenerateOCSPRequests: parallelGenerateOCSPRequests,
		tickHistogram:                tickHistogram,
		stalenessHistogram:           stalenessHistogram,
		genStoreHistogram:            genStoreHistogram,
		generatedCounter:             generatedCounter,
		storedCounter:                storedCounter,
		markExpiredCounter:           markExpiredCounter,
		findStaleOCSPCounter:         findStaleOCSPCounter,
	}

	return &updater, nil
}

func getQuestionsForShardList(count int) string {
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

// findStaleOCSPResponses sends a goroutine to fetch rows of stale OCSP
// responses from the database and returns results on a channel.
func (updater *OCSPUpdater) findStaleOCSPResponses(ctx context.Context, oldestLastUpdatedTime time.Time, batchSize int) <-chan *sa.CertStatusMetadata {
	// staleStatusesOut channel contains all stale ocsp responses that need
	// updating.
	staleStatusesOut := make(chan *sa.CertStatusMetadata)

	args := make([]interface{}, 0)
	args = append(args, oldestLastUpdatedTime)

	// If serialSuffixes is unset, this will be deliberately a no-op.
	for _, c := range updater.serialSuffixes {
		args = append(args, c)
	}
	args = append(args, batchSize)

	go func() {
		defer close(staleStatusesOut)

		selector, err := db.NewMappedSelector[sa.CertStatusMetadata](updater.readOnlyDb)
		if err != nil {
			updater.log.AuditErrf("failed to initialize database map: %s", err)
			updater.findStaleOCSPCounter.WithLabelValues("failed").Inc()
			updater.readFailures.Add(1)
			return
		}

		rows, err := selector.Query(ctx, updater.queryBody, args...)

		// If error, log and increment retries for backoff. Else no
		// error, proceed to push statuses to channel.
		if err != nil {
			updater.log.AuditErrf("failed to find stale OCSP responses: %s", err)
			updater.findStaleOCSPCounter.WithLabelValues("failed").Inc()
			updater.readFailures.Add(1)
			return
		}
		defer func() {
			err := rows.Close()
			if err != nil {
				updater.log.AuditErrf("closing query rows: %s", err)
			}
		}()

		for rows.Next() {
			meta, err := rows.Get()
			if err != nil {
				updater.log.AuditErrf("failed to scan metadata status row: %s", err)
				updater.findStaleOCSPCounter.WithLabelValues("failed").Inc()
				updater.readFailures.Add(1)
				return
			}
			staleness := oldestLastUpdatedTime.Sub(meta.OCSPLastUpdated).Seconds()
			updater.stalenessHistogram.Observe(staleness)
			select {
			case <-ctx.Done():
				err := ctx.Err()
				if err != nil {
					updater.log.AuditErrf("context done reading rows: %s", err)
				}
				return
			case staleStatusesOut <- meta:
			}
		}

		// Ensure the query wasn't interrupted before it could complete.
		err = rows.Err()
		if err != nil {
			updater.log.AuditErrf("finishing row scan: %s", err)
			updater.findStaleOCSPCounter.WithLabelValues("failed").Inc()
			updater.readFailures.Add(1)
			return
		}

		updater.findStaleOCSPCounter.WithLabelValues("success").Inc()
		updater.readFailures.Reset()
	}()

	return staleStatusesOut
}

func statusFromMetaAndResp(meta *sa.CertStatusMetadata, resp []byte) *core.CertificateStatus {
	return &core.CertificateStatus{
		ID:                    meta.ID,
		Serial:                meta.Serial,
		Status:                meta.Status,
		OCSPLastUpdated:       meta.OCSPLastUpdated,
		RevokedDate:           meta.RevokedDate,
		RevokedReason:         meta.RevokedReason,
		LastExpirationNagSent: meta.LastExpirationNagSent,
		OCSPResponse:          resp,
		NotAfter:              meta.NotAfter,
		IsExpired:             meta.IsExpired,
		IssuerID:              meta.IssuerID,
	}
}

// generateResponse signs an new OCSP response for a given certStatus row.
func (updater *OCSPUpdater) generateResponse(ctx context.Context, meta *sa.CertStatusMetadata) (*core.CertificateStatus, error) {
	if meta.IssuerID == 0 {
		return nil, errors.New("cert status has 0 IssuerID")
	}
	ocspReq := capb.GenerateOCSPRequest{
		Serial:    meta.Serial,
		IssuerID:  meta.IssuerID,
		Status:    string(meta.Status),
		Reason:    int32(meta.RevokedReason),
		RevokedAt: meta.RevokedDate.UnixNano(),
	}

	ocspResponse, err := updater.ogc.GenerateOCSP(ctx, &ocspReq)
	if err != nil {
		return nil, err
	}

	meta.OCSPLastUpdated = updater.clk.Now()
	return statusFromMetaAndResp(meta, ocspResponse.Response), nil
}

// storeResponse stores a given CertificateStatus in the database.
func (updater *OCSPUpdater) storeResponse(ctx context.Context, status *core.CertificateStatus) error {
	// Update the certificateStatus table with the new OCSP response, the status
	// WHERE is used make sure we don't overwrite a revoked response with a one
	// containing a 'good' status.
	_, err := updater.db.WithContext(ctx).Exec(
		`UPDATE certificateStatus
		 SET ocspResponse=?,ocspLastUpdated=?
		 WHERE id=?
		 AND status=?`,
		status.OCSPResponse,
		status.OCSPLastUpdated,
		status.ID,
		string(status.Status),
	)

	return err
}

// markExpired updates a given CertificateStatus to have `isExpired` set.
func (updater *OCSPUpdater) markExpired(ctx context.Context, meta *sa.CertStatusMetadata) error {
	_, err := updater.db.WithContext(ctx).Exec(
		`UPDATE certificateStatus
 		SET isExpired = TRUE
 		WHERE id = ?`,
		meta.ID,
	)
	return err
}

// processExpired is a pipeline step to process a channel of
// `sa.CertStatusMetadata` and set `isExpired` in the database.
func (updater *OCSPUpdater) processExpired(ctx context.Context, staleStatusesIn <-chan *sa.CertStatusMetadata) <-chan *sa.CertStatusMetadata {
	tickStart := updater.clk.Now()
	staleStatusesOut := make(chan *sa.CertStatusMetadata)
	go func() {
		defer close(staleStatusesOut)
		for meta := range staleStatusesIn {
			if !meta.IsExpired && tickStart.After(meta.NotAfter) {
				err := updater.markExpired(ctx, meta)
				if err != nil {
					// Update error counters and log
					updater.log.AuditErrf("Failed to set certificate expired: %s", err)
					updater.markExpiredCounter.WithLabelValues("failed").Inc()
				} else {
					updater.markExpiredCounter.WithLabelValues("success").Inc()
				}
			}
			select {
			case <-ctx.Done():
				return
			case staleStatusesOut <- meta:
			}
		}
	}()

	return staleStatusesOut
}

// generateOCSPResponses is the final stage of a pipeline. It takes a
// channel of `core.CertificateStatus` and sends a goroutine for each to
// obtain a new OCSP response and update the status in the database.
func (updater *OCSPUpdater) generateOCSPResponses(ctx context.Context, staleStatusesIn <-chan *sa.CertStatusMetadata) {
	// Use the semaphore pattern from
	// https://github.com/golang/go/wiki/BoundingResourceUse to send a number of
	// GenerateOCSP / storeResponse requests in parallel, while limiting the total number of
	// outstanding requests. The number of outstanding requests equals the
	// capacity of the channel.
	sem := make(chan int, updater.parallelGenerateOCSPRequests)
	wait := func() {
		sem <- 1 // Block until there's capacity.
	}
	done := func(start time.Time) {
		<-sem // Indicate there's more capacity.
		updater.genStoreHistogram.Observe(time.Since(start).Seconds())
	}

	// Work runs as a goroutine per ocsp response to obtain a new ocsp
	// response and store it in the database.
	work := func(meta *sa.CertStatusMetadata) {
		defer done(updater.clk.Now())

		status, err := updater.generateResponse(ctx, meta)
		if err != nil {
			updater.log.AuditErrf("Failed to generate OCSP response: %s", err)
			updater.generatedCounter.WithLabelValues("failed").Inc()
			return
		}
		updater.generatedCounter.WithLabelValues("success").Inc()

		err = updater.storeResponse(ctx, status)
		if err != nil {
			updater.log.AuditErrf("Failed to store OCSP response: %s", err)
			updater.storedCounter.WithLabelValues("failed").Inc()
			return
		}
		updater.storedCounter.WithLabelValues("success").Inc()
	}

	// Consume the stale statuses channel and send off a sign/store request
	// for each stale response.
	for status := range staleStatusesIn {
		wait()
		go work(status)
	}

	// Block until the sem channel reaches its full capacity again,
	// indicating each goroutine has completed.
	for i := 0; i < updater.parallelGenerateOCSPRequests; i++ {
		wait()
	}
}

func (updater *OCSPUpdater) Tick() {
	start := updater.clk.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	oldestLastUpdatedTime := updater.clk.Now().Add(-updater.ocspMinTimeToExpiry)

	// Run pipeline
	updater.generateOCSPResponses(ctx, updater.processExpired(ctx, updater.findStaleOCSPResponses(ctx, oldestLastUpdatedTime, updater.batchSize)))

	end := updater.clk.Now()
	took := end.Sub(start)
	long, state := "false", "success"
	if took > updater.tickWindow {
		long = "true"
	}

	// Set sleep duration to the configured tickWindow.
	sleepDur := start.Add(updater.tickWindow).Sub(end)

	// Set sleep duration higher to backoff starting the next tick and
	// reading from the database if the last read failed.
	readFails := updater.readFailures.Value()
	if readFails > 0 {
		sleepDur = core.RetryBackoff(
			readFails,
			updater.tickWindow,
			updater.maxBackoff,
			updater.backoffFactor,
		)
	}
	updater.tickHistogram.WithLabelValues(state, long).Observe(took.Seconds())
	updater.clk.Sleep(sleepDur)
}
