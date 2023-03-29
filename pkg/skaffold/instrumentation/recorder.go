/*
Copyright 2022 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package instrumentation

import (
	"context"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

// a single Google Cloud Monitoring write request can accommodate a maximum of 200 time series
// so we write the first 200 records and ignore the remaining.
// see https://cloud.google.com/monitoring/quotas
const maxRecordCount = 200

var recordCount int32 = 0

type float64ValueRecorder struct {
	name     string
	recorder instrument.Float64Counter
}

func (c float64ValueRecorder) Record(ctx context.Context, value float64, labels ...attribute.KeyValue) {
	if atomic.AddInt32(&recordCount, 1) >= maxRecordCount {
		log.Entry(ctx).Debugf("skipping recording metric %q, maximum quota of %q exceeded", c.name, maxRecordCount)
		return
	}
	c.recorder.Add(ctx, value, labels...)
}

type int64ValueRecorder struct {
	name     string
	recorder instrument.Int64Counter
}

func (c int64ValueRecorder) Record(ctx context.Context, value int64, labels ...attribute.KeyValue) {
	if atomic.AddInt32(&recordCount, 1) >= maxRecordCount {
		log.Entry(ctx).Debugf("skipping recording metric %q, maximum quota of %d exceeded", c.name, maxRecordCount)
		return
	}
	c.recorder.Add(ctx, value, labels...)
}

func NewFloat64ValueRecorder(m metric.Meter, name string, mos ...instrument.Float64Option) float64ValueRecorder {
	recorder, _ := m.Float64Counter(name, mos...)
	return float64ValueRecorder{name: name, recorder: recorder}
}

func NewInt64ValueRecorder(m metric.Meter, name string, mos ...instrument.Int64Option) int64ValueRecorder {
	recorder, _ := m.Int64Counter(name, mos...)
	return int64ValueRecorder{name: name, recorder: recorder}
}
