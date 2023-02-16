package instrumentation

import (
	"context"

	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type Exporter struct {
	APIKey string
}

func newFireLogExporter(APIKey string) Exporter {
	return Exporter{
		APIKey: APIKey,
	}
}

// Temporality returns the Temporality to use for an instrument kind.
func (e Exporter) Temporality(metric.InstrumentKind) metricdata.Temporality {

}

// Aggregation returns the Aggregation to use for an instrument kind.
func (e Exporter) Aggregation(metric.InstrumentKind) aggregation.Aggregation {

}

// Export serializes and transmits metric data to a receiver.
//
// This is called synchronously, there is no concurrency safety
// requirement. Because of this, it is critical that all timeouts and
// cancellations of the passed context be honored.
//
// All retry logic must be contained in this function. The SDK does not
// implement any retry logic. All errors returned by this function are
// considered unrecoverable and will be reported to a configured error
// Handler.
//
// The passed ResourceMetrics may be reused when the call completes. If an
// exporter needs to hold this data after it returns, it needs to make a
// copy.
func (e Exporter) Export(context.Context, metricdata.ResourceMetrics) error {

}

// ForceFlush flushes any metric data held by an exporter.
//
// The deadline or cancellation of the passed context must be honored. An
// appropriate error should be returned in these situations.
func (e Exporter) ForceFlush(context.Context) error {

}

// Shutdown flushes all metric data held by an exporter and releases any
// held computational resources.
//
// The deadline or cancellation of the passed context must be honored. An
// appropriate error should be returned in these situations.
//
// After Shutdown is called, calls to Export will perform no operation and
// instead will return an error indicating the shutdown state.
func (e Exporter) Shutdown(context.Context) error {

}
