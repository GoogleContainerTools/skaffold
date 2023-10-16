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

package internal // import "go.opentelemetry.io/otel/sdk/metric/internal"

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// now is used to return the current local time while allowing tests to
// override the default time.Now function.
var now = time.Now

// Aggregator forms an aggregation from a collection of recorded measurements.
//
// Aggregators need to be comparable so they can be de-duplicated by the SDK
// when it creates them for multiple views.
type Aggregator[N int64 | float64] interface {
	// Aggregate records the measurement, scoped by attr, and aggregates it
	// into an aggregation.
	Aggregate(measurement N, attr attribute.Set)

	// Aggregation returns an Aggregation, for all the aggregated
	// measurements made and ends an aggregation cycle.
	Aggregation() metricdata.Aggregation
}

// precomputeAggregator is an Aggregator that receives values to aggregate that
// have been pre-computed by the caller.
type precomputeAggregator[N int64 | float64] interface {
	// The Aggregate method of the embedded Aggregator is used to record
	// pre-computed measurements, scoped by attributes that have not been
	// filtered by an attribute filter.
	Aggregator[N]

	// aggregateFiltered records measurements scoped by attributes that have
	// been filtered by an attribute filter.
	//
	// Pre-computed measurements of filtered attributes need to be recorded
	// separate from those that haven't been filtered so they can be added to
	// the non-filtered pre-computed measurements in a collection cycle and
	// then resets after the cycle (the non-filtered pre-computed measurements
	// are not reset).
	aggregateFiltered(N, attribute.Set)
}
