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
	"bytes"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type ClientInfo struct {
	ClientType string `json:"client_type"`
}

type MetricData struct {
	ClientInfo    ClientInfo `json:"client_info"`
	LogSource     string     `json:"log_source"`
	LogEvent      LogEvent   `json:"log_event"`
	RequestTimeMS int64      `json:"request_time_ms"`
}

type LogEvent struct {
	EventTimeMS                  int64  `json:"event_time_ms"`
	SourceExtensionJSONProto3Str string `json:"source_extension_json_proto3"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type EventMetadata []KeyValue

type SourceExtensionJSONProto3 struct {
	ConsoleType     string     `json:"console_type"`
	ClientInstallID string     `json:"client_install_id"`
	EventName       string     `json:"event_name"`
	EventMetadata   []KeyValue `json:"event_metadata"`
}

type Key string

func (k Key) Value(v string) KeyValue {
	return KeyValue{
		Key:   string(k),
		Value: v,
	}
}

func (md MetricData) newReader() *bytes.Reader {
	data, _ := json.Marshal(md)
	return bytes.NewReader(data)
}

type DataPoint interface {
	value() string
	attributes() attribute.Set
	eventTime() int64
}

type DataPointInt64 metricdata.DataPoint[int64]

func (d DataPointInt64) attributes() attribute.Set {
	return d.Attributes
}

func (d DataPointInt64) eventTime() int64 {
	return d.StartTime.UnixMilli()
}

type DataPointFloat64 metricdata.DataPoint[float64]

func (d DataPointFloat64) attributes() attribute.Set {
	return d.Attributes
}

func (d DataPointFloat64) eventTime() int64 {
	return d.StartTime.UnixMilli()
}

type DataPointHistogram metricdata.HistogramDataPoint[float64]

func (d DataPointHistogram) attributes() attribute.Set {
	return d.Attributes
}

func (d DataPointHistogram) eventTime() int64 {
	return d.StartTime.UnixMilli()
}

func (d DataPointInt64) value() string {
	return fmt.Sprintf("%d", d.Value)
}

func (d DataPointFloat64) value() string {
	return fmt.Sprintf("%f", d.Value)
}

func (d DataPointHistogram) value() string {
	return fmt.Sprintf("%f", d.Sum)
}
