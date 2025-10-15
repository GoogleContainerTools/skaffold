// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spanner

import (
	"context"
	"log"
	"sync"

	"cloud.google.com/go/spanner/internal"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

// OtInstrumentationScope is the instrumentation name that will be associated with the emitted telemetry.
const OtInstrumentationScope = "cloud.google.com/go"
const metricsPrefix = "spanner/"

var (
	attributeKeyClientID      = attribute.Key("client_id")
	attributeKeyDatabase      = attribute.Key("database")
	attributeKeyInstance      = attribute.Key("instance_id")
	attributeKeyLibVersion    = attribute.Key("library_version")
	attributeKeyType          = attribute.Key("type")
	attributeKeyMethod        = attribute.Key("grpc_client_method")
	attributeKeyIsMultiplexed = attribute.Key("is_multiplexed")

	attributeNumInUseSessions = attributeKeyType.String("num_in_use_sessions")
	attributeNumSessions      = attributeKeyType.String("num_sessions")
	// openTelemetryMetricsEnabled is used to track if OpenTelemetry Metrics need to be recorded
	openTelemetryMetricsEnabled = false
	// mutex to avoid data race in reading/writing the above flag
	otMu = sync.RWMutex{}
)

func createOpenTelemetryConfig(ctx context.Context, mp metric.MeterProvider, logger *log.Logger, sessionClientID string, db string) (*openTelemetryConfig, error) {
	// Important: snapshot the value of the global variable to ensure a
	// consistent value for the lifetime of this client.
	enabled := IsOpenTelemetryMetricsEnabled()
	_, instance, database, err := parseDatabaseName(db)
	if err != nil {
		return nil, err
	}
	config := &openTelemetryConfig{
		enabled:      enabled,
		attributeMap: []attribute.KeyValue{},
		commonTraceStartOptions: []trace.SpanStartOption{
			trace.WithAttributes(
				attribute.String("db.name", database),
				attribute.String("instance.name", instance),
				attribute.String("cloud.region", detectClientLocation(ctx)),
				attribute.String("gcp.client.version", internal.Version),
				attribute.String("gcp.client.repo", gcpClientRepo),
				attribute.String("gcp.client.artifact", gcpClientArtifact),
			),
		},
	}
	if !enabled {
		return config, nil
	}

	// Construct attributes for Metrics
	attributeMap := []attribute.KeyValue{
		attributeKeyClientID.String(sessionClientID),
		attributeKeyDatabase.String(database),
		attributeKeyInstance.String(instance),
		attributeKeyLibVersion.String(internal.Version),
	}
	config.attributeMap = append(config.attributeMap, attributeMap...)

	config.attributeMapWithMultiplexed = append(config.attributeMapWithMultiplexed, attributeMap...)
	config.attributeMapWithMultiplexed = append(config.attributeMapWithMultiplexed, attributeKeyIsMultiplexed.String("true"))

	config.attributeMapWithoutMultiplexed = append(config.attributeMapWithoutMultiplexed, attributeMap...)
	config.attributeMapWithoutMultiplexed = append(config.attributeMapWithoutMultiplexed, attributeKeyIsMultiplexed.String("false"))
	setOpenTelemetryMetricProvider(config, mp, logger)
	return config, nil
}

func setOpenTelemetryMetricProvider(config *openTelemetryConfig, mp metric.MeterProvider, logger *log.Logger) {
	// Fallback to global meter provider in OpenTelemetry
	if mp == nil {
		mp = otel.GetMeterProvider()
	}
	config.meterProvider = mp
	initializeMetricInstruments(config, logger)
}

func initializeMetricInstruments(config *openTelemetryConfig, logger *log.Logger) {
	if !config.enabled {
		return
	}
	meter := config.meterProvider.Meter(OtInstrumentationScope, metric.WithInstrumentationVersion(internal.Version))

	openSessionCountInstrument, err := meter.Int64ObservableGauge(
		metricsPrefix+"open_session_count",
		metric.WithDescription("Number of sessions currently opened"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logf(logger, "Error during registering instrument for metric spanner/open_session_count, error: %v", err)
	}
	config.openSessionCount = openSessionCountInstrument

	maxAllowedSessionsCountInstrument, err := meter.Int64ObservableGauge(
		metricsPrefix+"max_allowed_sessions",
		metric.WithDescription("The maximum number of sessions allowed. Configurable by the user."),
		metric.WithUnit("1"),
	)
	if err != nil {
		logf(logger, "Error during registering instrument for metric spanner/max_allowed_sessions, error: %v", err)
	}
	config.maxAllowedSessionsCount = maxAllowedSessionsCountInstrument

	sessionsCountInstrument, _ := meter.Int64ObservableGauge(
		metricsPrefix+"num_sessions_in_pool",
		metric.WithDescription("The number of sessions currently in use."),
		metric.WithUnit("1"),
	)
	if err != nil {
		logf(logger, "Error during registering instrument for metric spanner/num_sessions_in_pool, error: %v", err)
	}
	config.sessionsCount = sessionsCountInstrument

	maxInUseSessionsCountInstrument, err := meter.Int64ObservableGauge(
		metricsPrefix+"max_in_use_sessions",
		metric.WithDescription("The maximum number of sessions in use during the last 10 minute interval."),
		metric.WithUnit("1"),
	)
	if err != nil {
		logf(logger, "Error during registering instrument for metric spanner/max_in_use_sessions, error: %v", err)
	}
	config.maxInUseSessionsCount = maxInUseSessionsCountInstrument

	getSessionTimeoutsCountInstrument, err := meter.Int64Counter(
		metricsPrefix+"get_session_timeouts",
		metric.WithDescription("The number of get sessions timeouts due to pool exhaustion."),
		metric.WithUnit("1"),
	)
	if err != nil {
		logf(logger, "Error during registering instrument for metric spanner/get_session_timeouts, error: %v", err)
	}
	config.getSessionTimeoutsCount = getSessionTimeoutsCountInstrument

	acquiredSessionsCountInstrument, err := meter.Int64Counter(
		metricsPrefix+"num_acquired_sessions",
		metric.WithDescription("The number of sessions acquired from the session pool."),
		metric.WithUnit("1"),
	)
	if err != nil {
		logf(logger, "Error during registering instrument for metric spanner/num_acquired_sessions, error: %v", err)
	}
	config.acquiredSessionsCount = acquiredSessionsCountInstrument

	releasedSessionsCountInstrument, err := meter.Int64Counter(
		metricsPrefix+"num_released_sessions",
		metric.WithDescription("The number of sessions released by the user and pool maintainer."),
		metric.WithUnit("1"),
	)
	if err != nil {
		logf(logger, "Error during registering instrument for metric spanner/num_released_sessions, error: %v", err)
	}
	config.releasedSessionsCount = releasedSessionsCountInstrument

	gfeLatencyInstrument, err := meter.Int64Histogram(
		metricsPrefix+"gfe_latency",
		metric.WithDescription("Latency between Google's network receiving an RPC and reading back the first byte of the response"),
		metric.WithUnit("ms"),
		metric.WithExplicitBucketBoundaries(0.0, 0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 8.0, 10.0, 13.0,
			16.0, 20.0, 25.0, 30.0, 40.0, 50.0, 65.0, 80.0, 100.0, 130.0, 160.0, 200.0, 250.0,
			300.0, 400.0, 500.0, 650.0, 800.0, 1000.0, 2000.0, 5000.0, 10000.0, 20000.0, 50000.0,
			100000.0),
	)
	if err != nil {
		logf(logger, "Error during registering instrument for metric spanner/gfe_latency, error: %v", err)
	}
	config.gfeLatency = gfeLatencyInstrument

	gfeHeaderMissingCountInstrument, err := meter.Int64Counter(
		metricsPrefix+"gfe_header_missing_count",
		metric.WithDescription("Number of RPC responses received without the server-timing header, most likely means that the RPC never reached Google's network"),
		metric.WithUnit("1"),
	)
	if err != nil {
		logf(logger, "Error during registering instrument for metric spanner/gfe_header_missing_count, error: %v", err)
	}
	config.gfeHeaderMissingCount = gfeHeaderMissingCountInstrument
}

func registerSessionPoolOTMetrics(pool *sessionPool) error {
	otConfig := pool.otConfig
	if otConfig == nil || !otConfig.enabled {
		return nil
	}

	attributes := otConfig.attributeMap
	attributesInUseSessions := append(attributes, attributeNumInUseSessions)
	attributesAvailableSessions := append(attributes, attributeNumSessions)

	reg, err := otConfig.meterProvider.Meter(OtInstrumentationScope, metric.WithInstrumentationVersion(internal.Version)).RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			pool.mu.Lock()
			defer pool.mu.Unlock()
			if pool.multiplexedSession != nil {
				o.ObserveInt64(otConfig.openSessionCount, int64(1), metric.WithAttributes(otConfig.attributeMapWithMultiplexed...))
			}
			o.ObserveInt64(otConfig.openSessionCount, int64(pool.numOpened), metric.WithAttributes(attributes...))
			o.ObserveInt64(otConfig.maxAllowedSessionsCount, int64(pool.SessionPoolConfig.MaxOpened), metric.WithAttributes(attributes...))
			o.ObserveInt64(otConfig.sessionsCount, int64(pool.numInUse), metric.WithAttributes(append(attributesInUseSessions, attribute.Key("is_multiplexed").String("false"))...))
			o.ObserveInt64(otConfig.sessionsCount, int64(pool.numSessions), metric.WithAttributes(attributesAvailableSessions...))
			o.ObserveInt64(otConfig.maxInUseSessionsCount, int64(pool.maxNumInUse), metric.WithAttributes(append(attributes, attribute.Key("is_multiplexed").String("false"))...))
			return nil
		},
		otConfig.openSessionCount,
		otConfig.maxAllowedSessionsCount,
		otConfig.sessionsCount,
		otConfig.maxInUseSessionsCount,
	)
	pool.otConfig.otMetricRegistration = reg
	return err
}

// EnableOpenTelemetryMetrics enables OpenTelemetery metrics
func EnableOpenTelemetryMetrics() {
	setOpenTelemetryMetricsFlag(true)
}

// IsOpenTelemetryMetricsEnabled tells whether OpenTelemtery metrics is enabled or not.
func IsOpenTelemetryMetricsEnabled() bool {
	otMu.RLock()
	defer otMu.RUnlock()
	return openTelemetryMetricsEnabled
}

func setOpenTelemetryMetricsFlag(enable bool) {
	otMu.Lock()
	openTelemetryMetricsEnabled = enable
	otMu.Unlock()
}

func recordGFELatencyMetricsOT(ctx context.Context, md metadata.MD, keyMethod string, otConfig *openTelemetryConfig) error {
	if otConfig == nil || !otConfig.enabled || md == nil {
		return nil
	}
	attr := otConfig.attributeMap
	metrics := parseServerTimingHeader(md)
	if len(metrics) == 0 && otConfig.gfeHeaderMissingCount != nil {
		otConfig.gfeHeaderMissingCount.Add(ctx, 1, metric.WithAttributes(attr...))
		return nil
	}
	attr = append(attr, attributeKeyMethod.String(keyMethod))
	if otConfig.gfeLatency != nil {
		otConfig.gfeLatency.Record(ctx, metrics[gfeTimingHeader].Milliseconds(), metric.WithAttributes(attr...))
	}
	return nil
}
