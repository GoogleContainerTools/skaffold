package rocsp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/core"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/ocsp"
)

var ErrRedisNotFound = errors.New("redis key not found")

// ROClient represents a read-only Redis client.
type ROClient struct {
	rdb        *redis.Ring
	timeout    time.Duration
	clk        clock.Clock
	getLatency *prometheus.HistogramVec
}

// NewReadingClient creates a read-only client. The timeout applies to all
// requests, though a shorter timeout can be applied on a per-request basis
// using context.Context. rdb must be non-nil and calls to rdb.Options().Addrs
// must return at least one entry.
func NewReadingClient(rdb *redis.Ring, timeout time.Duration, clk clock.Clock, stats prometheus.Registerer) *ROClient {
	if len(rdb.Options().Addrs) == 0 {
		return nil
	}
	var addrs []string
	for addr := range rdb.Options().Addrs {
		addrs = append(addrs, addr)
	}
	labels := prometheus.Labels{
		"addresses": strings.Join(addrs, ", "),
		"user":      rdb.Options().Username,
	}
	stats.MustRegister(newMetricsCollector(rdb, labels))
	getLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "rocsp_get_latency",
			Help: "Histogram of latencies of rocsp.GetResponse calls with result",
			// 8 buckets, ranging from 0.5ms to 2s
			Buckets: prometheus.ExponentialBucketsRange(0.0005, 2, 8),
		},
		[]string{"result"},
	)
	stats.MustRegister(getLatency)

	return &ROClient{
		rdb:        rdb,
		timeout:    timeout,
		clk:        clk,
		getLatency: getLatency,
	}
}

func (c *ROClient) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	return c.rdb.Ping(ctx).Err()
}

// RWClient represents a Redis client that can both read and write.
type RWClient struct {
	*ROClient
	storeResponseLatency *prometheus.HistogramVec
}

// NewWritingClient creates a RWClient.
func NewWritingClient(rdb *redis.Ring, timeout time.Duration, clk clock.Clock, stats prometheus.Registerer) *RWClient {
	storeResponseLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "rocsp_store_response_latency",
			Help: "Histogram of latencies of rocsp.StoreResponse calls with result labels",
		},
		[]string{"result"},
	)
	stats.MustRegister(storeResponseLatency)
	return &RWClient{NewReadingClient(rdb, timeout, clk, stats), storeResponseLatency}
}

// StoreResponse parses the given bytes as an OCSP response, and stores it
// into Redis. The expiration time (ttl) of the Redis key is set to OCSP
// response `NextUpdate`.
func (c *RWClient) StoreResponse(ctx context.Context, resp *ocsp.Response) error {
	start := c.clk.Now()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	serial := core.SerialToString(resp.SerialNumber)

	// Set the ttl duration to the response `NextUpdate - now()`
	ttl := time.Until(resp.NextUpdate)

	err := c.rdb.Set(ctx, serial, resp.Raw, ttl).Err()
	if err != nil {
		state := "failed"
		if errors.Is(err, context.DeadlineExceeded) {
			state = "deadlineExceeded"
		} else if errors.Is(err, context.Canceled) {
			state = "canceled"
		}
		c.storeResponseLatency.With(prometheus.Labels{"result": state}).Observe(time.Since(start).Seconds())
		return fmt.Errorf("setting response: %w", err)
	}

	c.storeResponseLatency.With(prometheus.Labels{"result": "success"}).Observe(time.Since(start).Seconds())
	return nil
}

// GetResponse fetches a response for the given serial number.
// Returns error if the OCSP response fails to parse.
func (c *ROClient) GetResponse(ctx context.Context, serial string) ([]byte, error) {
	start := c.clk.Now()
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.rdb.Get(ctx, serial).Result()
	if err != nil {
		// go-redis `Get` returns redis.Nil error when key does not exist. In
		// that case return a `ErrRedisNotFound` error.
		if errors.Is(err, redis.Nil) {
			c.getLatency.With(prometheus.Labels{"result": "notFound"}).Observe(time.Since(start).Seconds())
			return nil, ErrRedisNotFound
		}

		state := "failed"
		if errors.Is(err, context.DeadlineExceeded) {
			state = "deadlineExceeded"
		} else if errors.Is(err, context.Canceled) {
			state = "canceled"
		}
		c.getLatency.With(prometheus.Labels{"result": state}).Observe(time.Since(start).Seconds())
		return nil, fmt.Errorf("getting response: %w", err)
	}

	c.getLatency.With(prometheus.Labels{"result": "success"}).Observe(time.Since(start).Seconds())
	return []byte(resp), nil
}

// ScanResponsesResult represents a single OCSP response entry in redis.
// `Serial` is the stringified serial number of the response. `Body` is the
// DER bytes of the response. If this object represents an error, `Err` will
// be non-nil and the other entries will have their zero values.
type ScanResponsesResult struct {
	Serial string
	Body   []byte
	Err    error
}

// ScanResponses scans Redis for all OCSP responses where the serial number matches the provided pattern.
// It returns immediately and emits results and errors on `<-chan ScanResponsesResult`. It closes the
// channel when it is done or hits an error.
func (c *ROClient) ScanResponses(ctx context.Context, serialPattern string) <-chan ScanResponsesResult {
	pattern := fmt.Sprintf("r{%s}", serialPattern)
	results := make(chan ScanResponsesResult)
	go func() {
		defer close(results)
		err := c.rdb.ForEachShard(ctx, func(ctx context.Context, rdb *redis.Client) error {
			iter := rdb.Scan(ctx, 0, pattern, 0).Iterator()
			for iter.Next(ctx) {
				key := iter.Val()
				val, err := c.rdb.Get(ctx, key).Result()
				if err != nil {
					results <- ScanResponsesResult{Err: fmt.Errorf("getting response: %w", err)}
					continue
				}
				results <- ScanResponsesResult{Serial: key, Body: []byte(val)}
			}
			return iter.Err()
		})
		if err != nil {
			results <- ScanResponsesResult{Err: err}
			return
		}
	}()
	return results
}
