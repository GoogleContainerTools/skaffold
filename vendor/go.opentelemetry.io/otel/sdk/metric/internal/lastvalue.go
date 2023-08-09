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
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// datapoint is timestamped measurement data.
type datapoint[N int64 | float64] struct {
	timestamp time.Time
	value     N
}

// lastValue summarizes a set of measurements as the last one made.
type lastValue[N int64 | float64] struct {
	sync.Mutex

	values map[attribute.Set]datapoint[N]
}

// NewLastValue returns an Aggregator that summarizes a set of measurements as
// the last one made.
func NewLastValue[N int64 | float64]() Aggregator[N] {
	return &lastValue[N]{values: make(map[attribute.Set]datapoint[N])}
}

func (s *lastValue[N]) Aggregate(value N, attr attribute.Set) {
	d := datapoint[N]{timestamp: now(), value: value}
	s.Lock()
	s.values[attr] = d
	s.Unlock()
}

func (s *lastValue[N]) Aggregation() metricdata.Aggregation {
	s.Lock()
	defer s.Unlock()

	if len(s.values) == 0 {
		return nil
	}

	gauge := metricdata.Gauge[N]{
		DataPoints: make([]metricdata.DataPoint[N], 0, len(s.values)),
	}
	for a, v := range s.values {
		gauge.DataPoints = append(gauge.DataPoints, metricdata.DataPoint[N]{
			Attributes: a,
			// The event time is the only meaningful timestamp, StartTime is
			// ignored.
			Time:  v.timestamp,
			Value: v.value,
		})
		// Do not report stale values.
		delete(s.values, a)
	}
	return gauge
}
