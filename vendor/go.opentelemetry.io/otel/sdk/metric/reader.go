// Copyright The OpenTelemetry Authors
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

package metric // import "go.opentelemetry.io/otel/sdk/metric"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// errDuplicateRegister is logged by a Reader when an attempt to registered it
// more than once occurs.
var errDuplicateRegister = fmt.Errorf("duplicate reader registration")

// ErrReaderNotRegistered is returned if Collect or Shutdown are called before
// the reader is registered with a MeterProvider.
var ErrReaderNotRegistered = fmt.Errorf("reader is not registered")

// ErrReaderShutdown is returned if Collect or Shutdown are called after a
// reader has been Shutdown once.
var ErrReaderShutdown = fmt.Errorf("reader is shutdown")

// errNonPositiveDuration is logged when an environmental variable
// has non-positive value.
var errNonPositiveDuration = fmt.Errorf("non-positive duration")

// Reader is the interface used between the SDK and an
// exporter.  Control flow is bi-directional through the
// Reader, since the SDK initiates ForceFlush and Shutdown
// while the initiates collection.  The Register() method here
// informs the Reader that it can begin reading, signaling the
// start of bi-directional control flow.
//
// Typically, push-based exporters that are periodic will
// implement PeroidicExporter themselves and construct a
// PeriodicReader to satisfy this interface.
//
// Pull-based exporters will typically implement Register
// themselves, since they read on demand.
type Reader interface {
	// register registers a Reader with a MeterProvider.
	// The producer argument allows the Reader to signal the sdk to collect
	// and send aggregated metric measurements.
	register(sdkProducer)

	// RegisterProducer registers a an external Producer with this Reader.
	// The Producer is used as a source of aggregated metric data which is
	// incorporated into metrics collected from the SDK.
	RegisterProducer(Producer)

	// temporality reports the Temporality for the instrument kind provided.
	temporality(InstrumentKind) metricdata.Temporality

	// aggregation returns what Aggregation to use for an instrument kind.
	aggregation(InstrumentKind) aggregation.Aggregation // nolint:revive  // import-shadow for method scoped by type.

	// Collect gathers and returns all metric data related to the Reader from
	// the SDK and stores it in out. An error is returned if this is called
	// after Shutdown or if out is nil.
	Collect(ctx context.Context, rm *metricdata.ResourceMetrics) error

	// ForceFlush flushes all metric measurements held in an export pipeline.
	//
	// This deadline or cancellation of the passed context are honored. An appropriate
	// error will be returned in these situations. There is no guaranteed that all
	// telemetry be flushed or all resources have been released in these
	// situations.
	ForceFlush(context.Context) error

	// Shutdown flushes all metric measurements held in an export pipeline and releases any
	// held computational resources.
	//
	// This deadline or cancellation of the passed context are honored. An appropriate
	// error will be returned in these situations. There is no guaranteed that all
	// telemetry be flushed or all resources have been released in these
	// situations.
	//
	// After Shutdown is called, calls to Collect will perform no operation and instead will return
	// an error indicating the shutdown state.
	Shutdown(context.Context) error
}

// sdkProducer produces metrics for a Reader.
type sdkProducer interface {
	// produce returns aggregated metrics from a single collection.
	//
	// This method is safe to call concurrently.
	produce(context.Context) (metricdata.ResourceMetrics, error)
}

// Producer produces metrics for a Reader from an external source.
type Producer interface {
	// Produce returns aggregated metrics from an external source.
	//
	// This method should be safe to call concurrently.
	Produce(context.Context) ([]metricdata.ScopeMetrics, error)
}

// produceHolder is used as an atomic.Value to wrap the non-concrete producer
// type.
type produceHolder struct {
	produce func(context.Context) (metricdata.ResourceMetrics, error)
}

// shutdownProducer produces an ErrReaderShutdown error always.
type shutdownProducer struct{}

// produce returns an ErrReaderShutdown error.
func (p shutdownProducer) produce(context.Context) (metricdata.ResourceMetrics, error) {
	return metricdata.ResourceMetrics{}, ErrReaderShutdown
}

// TemporalitySelector selects the temporality to use based on the InstrumentKind.
type TemporalitySelector func(InstrumentKind) metricdata.Temporality

// DefaultTemporalitySelector is the default TemporalitySelector used if
// WithTemporalitySelector is not provided. CumulativeTemporality will be used
// for all instrument kinds if this TemporalitySelector is used.
func DefaultTemporalitySelector(InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

// AggregationSelector selects the aggregation and the parameters to use for
// that aggregation based on the InstrumentKind.
type AggregationSelector func(InstrumentKind) aggregation.Aggregation

// DefaultAggregationSelector returns the default aggregation and parameters
// that will be used to summarize measurement made from an instrument of
// InstrumentKind. This AggregationSelector using the following selection
// mapping: Counter ⇨ Sum, Observable Counter ⇨ Sum, UpDownCounter ⇨ Sum,
// Observable UpDownCounter ⇨ Sum, Observable Gauge ⇨ LastValue,
// Histogram ⇨ ExplicitBucketHistogram.
func DefaultAggregationSelector(ik InstrumentKind) aggregation.Aggregation {
	switch ik {
	case InstrumentKindCounter, InstrumentKindUpDownCounter, InstrumentKindObservableCounter, InstrumentKindObservableUpDownCounter:
		return aggregation.Sum{}
	case InstrumentKindObservableGauge:
		return aggregation.LastValue{}
	case InstrumentKindHistogram:
		return aggregation.ExplicitBucketHistogram{
			Boundaries: []float64{0, 5, 10, 25, 50, 75, 100, 250, 500, 750, 1000, 2500, 5000, 7500, 10000},
			NoMinMax:   false,
		}
	}
	panic("unknown instrument kind")
}
