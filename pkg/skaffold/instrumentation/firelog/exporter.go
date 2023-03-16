package firelog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

var APIKey = ""
var url = fmt.Sprintf(`https://firebaselogging-pa.googleapis.com/v1/firelog/legacy/log?key=%s`, APIKey)

type Exporter struct {
}

func NewFireLogExporter() (metric.Exporter, error) {

	if APIKey == "" {
		// export metrics to std out if local env is set.
		if _, ok := os.LookupEnv("SKAFFOLD_EXPORT_TO_STDOUT"); ok {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			exporter, err := stdoutmetric.New(stdoutmetric.WithEncoder(enc))
			return exporter, err
		}
		return nil, nil
	}
	return &Exporter{}, nil
}

// Temporality returns the Temporality to use for an instrument kind.
func (e *Exporter) Temporality(ik metric.InstrumentKind) metricdata.Temporality {
	return metric.DefaultTemporalitySelector(ik)

}

// Aggregation returns the Aggregation to use for an instrument kind.
func (e *Exporter) Aggregation(ik metric.InstrumentKind) aggregation.Aggregation {
	return metric.DefaultAggregationSelector(ik)
}

func (e *Exporter) Export(ctx context.Context, md metricdata.ResourceMetrics) error {

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
	case metricdata.Histogram:
		for _, pt := range a.DataPoints {
			if err := sendDataPoint(m.Name, DataPointHistogram(pt)); err != nil {
				return err
			}
		}
	}
	return nil
}

func sendDataPoint[T DataPoint](name string, dp T) error {
	kvs := toEventMetadata(dp.attributes())
	kvs = append(kvs, KeyValue{Key: name, Value: dp.value()})
	str, err := buildProtoStr(name, kvs)
	if err != nil {
		return err
	}
	data := buildMetricData(str, dp.eventTime(), dp.upTime())

	resp, err := http.Post(url, "application/json", data.newReader())
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("one platform returned an non-success response: %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	return err
}

func buildMetricData(proto string, startTimeMS int64, upTimeMS int64) MetricData {

	return MetricData{
		ClientInfo: ClientInfo{ClientType: "DESKTOP"},
		LogSource:  "CONCORD",
		LogEvent: LogEvent{
			EventTimeMS:                  startTimeMS,
			EventUptimeMS:                upTimeMS,
			SourceExtensionJSONProto3Str: proto,
		},
		RequestTimeMS:   startTimeMS,
		RequestUptimeMS: upTimeMS,
	}
}

func buildProtoStr(name string, kvs EventMetadata) (string, error) {
	proto3 := SourceExtensionJSONProto3{
		ProjectID:       "skaffold",
		ConsoleType:     "SKAFFOLD",
		ClientInstallID: "",
		EventName:       name,
		EventMetadata:   kvs,
	}

	b, err := json.Marshal(proto3)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metricdata")
	}
	return string(b), nil
}

func toEventMetadata(attributes attribute.Set) EventMetadata {
	var kvs EventMetadata
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
