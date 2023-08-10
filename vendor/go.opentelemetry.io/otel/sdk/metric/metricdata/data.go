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

package metricdata // import "go.opentelemetry.io/otel/sdk/metric/metricdata"

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
)

// ResourceMetrics is a collection of ScopeMetrics and the associated Resource
// that created them.
type ResourceMetrics struct {
	// Resource represents the entity that collected the metrics.
	Resource *resource.Resource
	// ScopeMetrics are the collection of metrics with unique Scopes.
	ScopeMetrics []ScopeMetrics
}

// ScopeMetrics is a collection of Metrics Produces by a Meter.
type ScopeMetrics struct {
	// Scope is the Scope that the Meter was created with.
	Scope instrumentation.Scope
	// Metrics are a list of aggregations created by the Meter.
	Metrics []Metrics
}

// Metrics is a collection of one or more aggregated timeseries from an Instrument.
type Metrics struct {
	// Name is the name of the Instrument that created this data.
	Name string
	// Description is the description of the Instrument, which can be used in documentation.
	Description string
	// Unit is the unit in which the Instrument reports.
	Unit string
	// Data is the aggregated data from an Instrument.
	Data Aggregation
}

// Aggregation is the store of data reported by an Instrument.
// It will be one of: Gauge, Sum, Histogram.
type Aggregation interface {
	privateAggregation()
}

// Gauge represents a measurement of the current value of an instrument.
type Gauge[N int64 | float64] struct {
	// DataPoints reprents individual aggregated measurements with unique Attributes.
	DataPoints []DataPoint[N]
}

func (Gauge[N]) privateAggregation() {}

// Sum represents the sum of all measurements of values from an instrument.
type Sum[N int64 | float64] struct {
	// DataPoints reprents individual aggregated measurements with unique Attributes.
	DataPoints []DataPoint[N]
	// Temporality describes if the aggregation is reported as the change from the
	// last report time, or the cumulative changes since a fixed start time.
	Temporality Temporality
	// IsMonotonic represents if this aggregation only increases or decreases.
	IsMonotonic bool
}

func (Sum[N]) privateAggregation() {}

// DataPoint is a single data point in a timeseries.
type DataPoint[N int64 | float64] struct {
	// Attributes is the set of key value pairs that uniquely identify the
	// timeseries.
	Attributes attribute.Set
	// StartTime is when the timeseries was started. (optional)
	StartTime time.Time `json:",omitempty"`
	// Time is the time when the timeseries was recorded. (optional)
	Time time.Time `json:",omitempty"`
	// Value is the value of this data point.
	Value N
}

// Histogram represents the histogram of all measurements of values from an instrument.
type Histogram struct {
	// DataPoints reprents individual aggregated measurements with unique Attributes.
	DataPoints []HistogramDataPoint
	// Temporality describes if the aggregation is reported as the change from the
	// last report time, or the cumulative changes since a fixed start time.
	Temporality Temporality
}

func (Histogram) privateAggregation() {}

// HistogramDataPoint is a single histogram data point in a timeseries.
type HistogramDataPoint struct {
	// Attributes is the set of key value pairs that uniquely identify the
	// timeseries.
	Attributes attribute.Set
	// StartTime is when the timeseries was started.
	StartTime time.Time
	// Time is the time when the timeseries was recorded.
	Time time.Time

	// Count is the number of updates this histogram has been calculated with.
	Count uint64
	// Bounds are the upper bounds of the buckets of the histogram. Because the
	// last boundary is +infinity this one is implied.
	Bounds []float64
	// BucketCounts is the count of each of the buckets.
	BucketCounts []uint64

	// Min is the minimum value recorded. (optional)
	Min Extrema
	// Max is the maximum value recorded. (optional)
	Max Extrema
	// Sum is the sum of the values recorded.
	Sum float64
}

// Extrema is the minimum or maximum value of a dataset.
type Extrema struct {
	value float64
	valid bool
}

// NewExtrema returns an Extrema set to v.
func NewExtrema(v float64) Extrema {
	return Extrema{value: v, valid: true}
}

// Value returns the Extrema value and true if the Extrema is defined.
// Otherwise, if the Extrema is its zero-value, defined will be false.
func (e Extrema) Value() (v float64, defined bool) {
	return e.value, e.valid
}
