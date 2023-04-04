package rocsp

import (
	"strings"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type mockPoolStatGetter struct{}

var _ poolStatGetter = mockPoolStatGetter{}

func (mockPoolStatGetter) PoolStats() *redis.PoolStats {
	return &redis.PoolStats{
		Hits:       13,
		Misses:     7,
		Timeouts:   4,
		TotalConns: 1000,
		IdleConns:  500,
		StaleConns: 10,
	}
}

func TestMetrics(t *testing.T) {
	mets := newMetricsCollector(mockPoolStatGetter{},
		prometheus.Labels{
			"foo": "bar",
		})
	// Check that it has the correct type to satisfy MustRegister
	metrics.NoopRegisterer.MustRegister(mets)

	expectedMetrics := 6
	outChan := make(chan prometheus.Metric, expectedMetrics)
	mets.Collect(outChan)

	results := make(map[string]bool)
	for i := 0; i < expectedMetrics; i++ {
		metric := <-outChan
		results[metric.Desc().String()] = true
	}

	expected := strings.Split(
		`Desc{fqName: "redis_connection_pool_lookups", help: "Number of lookups for a connection in the pool, labeled by hit/miss", constLabels: {foo="bar"}, variableLabels: [result]}
Desc{fqName: "redis_connection_pool_lookups", help: "Number of lookups for a connection in the pool, labeled by hit/miss", constLabels: {foo="bar"}, variableLabels: [result]}
Desc{fqName: "redis_connection_pool_lookups", help: "Number of lookups for a connection in the pool, labeled by hit/miss", constLabels: {foo="bar"}, variableLabels: [result]}
Desc{fqName: "redis_connection_pool_total_conns", help: "Number of total connections in the pool.", constLabels: {foo="bar"}, variableLabels: []}
Desc{fqName: "redis_connection_pool_idle_conns", help: "Number of idle connections in the pool.", constLabels: {foo="bar"}, variableLabels: []}
Desc{fqName: "redis_connection_pool_stale_conns", help: "Number of stale connections removed from the pool.", constLabels: {foo="bar"}, variableLabels: []}`,
		"\n")

	for _, e := range expected {
		if !results[e] {
			t.Errorf("expected metrics to contain %q, but they didn't", e)
		}
	}

	if len(results) > len(expected) {
		t.Errorf("expected metrics to contain %d entries, but they contained %d",
			len(expected), len(results))
	}
}
