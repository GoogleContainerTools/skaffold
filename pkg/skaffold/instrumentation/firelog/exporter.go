/*
Copyright 2020 The Skaffold Authors

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

package firelog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/GoogleContainerTools/skaffold/v2/fs"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

var (
	APIKey  = ""
	POST    = http.Post
	Marshal = json.Marshal

	// TODO: Implement a persistent client installation ID
	GetClientInstallID = uuid.NewString
)

type Exporter struct {
}

func NewFireLogExporter() (metric.Exporter, error) {
	b, err := fs.AssetsFS.ReadFile("assets/firelog_generated/key.txt")
	if err == nil {
		APIKey = string(b)
		return &Exporter{}, nil
	}
	log.Entry(context.TODO()).Debugf("failed to create firelog exporter due to error: %v", err)

	// export metrics to std out if local env is set.
	if _, ok := os.LookupEnv("SKAFFOLD_EXPORT_TO_STDOUT"); ok {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		exporter, err := stdoutmetric.New(stdoutmetric.WithEncoder(enc))
		log.Entry(context.TODO()).Debugln("created a stdout exporter instead")
		return exporter, err
	}
	log.Entry(context.TODO()).Debugln("did not create any log exporter")
	return nil, nil
}

// Temporality returns the Temporality to use for an instrument kind.
func (e *Exporter) Temporality(ik metric.InstrumentKind) metricdata.Temporality {
	return metric.DefaultTemporalitySelector(ik)
}

// Aggregation returns the Aggregation to use for an instrument kind.
func (e *Exporter) Aggregation(ik metric.InstrumentKind) metric.Aggregation {
	return metric.DefaultAggregationSelector(ik)
}

func (e *Exporter) Export(ctx context.Context, md *metricdata.ResourceMetrics) error {
	for _, sm := range md.ScopeMetrics {
		for _, m := range sm.Metrics {
			if err := processMetrics(m); err != nil {
				return err
			}
		}
	}
	return nil
}

func processMetrics(m metricdata.Metrics) error {
	switch a := m.Data.(type) {
	case metricdata.Gauge[int64]:
		for _, pt := range a.DataPoints {
			if err := sendDataPoint(m.Name, DataPointInt64(pt)); err != nil {
				return err
			}
		}
	case metricdata.Gauge[float64]:
		for _, pt := range a.DataPoints {
			if err := sendDataPoint(m.Name, DataPointFloat64(pt)); err != nil {
				return err
			}
		}
	case metricdata.Sum[int64]:
		for _, pt := range a.DataPoints {
			if err := sendDataPoint(m.Name, DataPointInt64(pt)); err != nil {
				return err
			}
		}
	case metricdata.Sum[float64]:
		for _, pt := range a.DataPoints {
			if err := sendDataPoint(m.Name, DataPointFloat64(pt)); err != nil {
				return err
			}
		}
	case metricdata.Histogram[float64]:
		for _, pt := range a.DataPoints {
			if err := sendDataPoint(m.Name, DataPointHistogram(pt)); err != nil {
				return err
			}
		}
	}
	return nil
}

func sendDataPoint(name string, dp DataPoint) error {
	kvs := toEventMetadata(dp.attributes())
	kvs = append(kvs, KeyValue{Key: name, Value: dp.value()})
	str, err := buildProtoStr(name, kvs)
	if err != nil {
		return err
	}
	data := buildMetricData(str, dp.eventTime())

	resp, err := POST(fmt.Sprintf(`https://firebaselogging-pa.googleapis.com/v1/firelog/legacy/log?key=%s`, APIKey), "application/json", data.newReader())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("one platform returned an non-success response: %d", resp.StatusCode)
	}

	return err
}

func buildMetricData(proto string, startTimeMS int64) MetricData {
	return MetricData{
		ClientInfo: ClientInfo{ClientType: "DESKTOP"},
		LogSource:  "CONCORD",
		LogEvent: LogEvent{
			EventTimeMS:                  startTimeMS,
			SourceExtensionJSONProto3Str: proto,
		},
		RequestTimeMS: startTimeMS,
	}
}

func buildProtoStr(name string, kvs EventMetadata) (string, error) {
	proto3 := SourceExtensionJSONProto3{
		ConsoleType:     "SKAFFOLD",
		ClientInstallID: GetClientInstallID(),
		EventName:       name,
		EventMetadata:   kvs,
	}

	b, err := Marshal(proto3)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metricdata")
	}
	return string(b), nil
}

func toEventMetadata(attributes attribute.Set) EventMetadata {
	kvs := EventMetadata{}
	iterator := attributes.Iter()
	for iterator.Next() {
		attr := iterator.Attribute()
		kv := KeyValue{
			Key:   string(attr.Key),
			Value: attr.Value.Emit(),
		}
		kvs = append(kvs, kv)
	}
	return kvs
}

func (e *Exporter) ForceFlush(ctx context.Context) error {
	return ctx.Err()
}

func (e *Exporter) Shutdown(ctx context.Context) error {
	return ctx.Err()
}
