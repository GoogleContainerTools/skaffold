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
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"

	"github.com/GoogleContainerTools/skaffold/v2/fs"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

func TestNewFireLogExporter(t *testing.T) {
	var tests = []struct {
		name       string
		expected   metric.Exporter
		fileSystem *testutil.FakeFileSystem
		wantErr    bool
	}{
		{
			name: "no api key",
			fileSystem: &testutil.FakeFileSystem{
				Files: map[string][]byte{},
			},
			expected: nil,
		},
		{
			name:     "has api key",
			expected: &Exporter{},
			fileSystem: &testutil.FakeFileSystem{
				Files: map[string][]byte{"assets/firelog_generated/key.txt": []byte("test key")},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			fs.AssetsFS = test.fileSystem
			out, err := NewFireLogExporter()
			t.CheckError(test.wantErr, err)
			t.CheckDeepEqual(test.expected, out)
		})
	}
}
func TestToEventMetadata(t *testing.T) {
	tests := []struct {
		name       string
		attributes attribute.Set
		expected   EventMetadata
	}{
		{
			name:       "no attributes",
			attributes: attribute.Set{},
			expected:   EventMetadata{},
		},
		{
			name:       "one attribute",
			attributes: attribute.NewSet(attribute.String("key1", "value1")),
			expected: EventMetadata{
				KeyValue{
					Key:   "key1",
					Value: "value1",
				},
			},
		},
		{
			name: "two attributes",
			attributes: attribute.NewSet(attribute.String("key1", "value1"),
				attribute.String("key2", "value2")),
			expected: EventMetadata{
				KeyValue{
					Key:   "key1",
					Value: "value1",
				},
				KeyValue{
					Key:   "key2",
					Value: "value2",
				},
			},
		},
		{
			name: "two attributes mixed types",
			attributes: attribute.NewSet(attribute.String("key1", "value1"),
				attribute.Int("key2", 50)),
			expected: EventMetadata{
				KeyValue{
					Key:   "key1",
					Value: "value1",
				},
				KeyValue{
					Key:   "key2",
					Value: "50",
				},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			actual := toEventMetadata(test.attributes)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestBuildMetricData(t *testing.T) {
	tests := []struct {
		name        string
		proto       string
		startTimeMS int64
		upTimeMS    int64
		expected    MetricData
	}{
		{
			name:        "build metrics data",
			proto:       "{foo:a}",
			startTimeMS: 100,
			upTimeMS:    200,
			expected: MetricData{
				ClientInfo: ClientInfo{ClientType: "DESKTOP"},
				LogSource:  "CONCORD",
				LogEvent: LogEvent{
					EventTimeMS:                  100,
					SourceExtensionJSONProto3Str: "{foo:a}",
				},
				RequestTimeMS: 100,
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			actual := buildMetricData(test.proto, test.startTimeMS)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}

func TestBuildProtoStr(t *testing.T) {
	tests := []struct {
		name        string
		eventName   string
		kvs         EventMetadata
		expected    string
		wantErr     bool
		marshalFunc func(v any) ([]byte, error)
	}{
		{
			name:      "no kvs",
			eventName: "no kvs",
			kvs:       EventMetadata{},
			expected:  `{"console_type":"SKAFFOLD","client_install_id":"00000000-0000-0000-0000-000000000000","event_name":"no kvs","event_metadata":[]}`,
			wantErr:   false,
		},
		{
			name:      "with kvs",
			eventName: "with kvs",
			kvs: EventMetadata{
				{"key1", "value1"},
				{"key2", "value2"},
			},
			expected: `{"console_type":"SKAFFOLD","client_install_id":"00000000-0000-0000-0000-000000000000","event_name":"with kvs","event_metadata":[{"key":"key1","value":"value1"},{"key":"key2","value":"value2"}]}`,
			wantErr:  false,
		},
		{
			name: "fail to marshal",
			kvs: EventMetadata{
				{"key1", "value1"},
				{"key2", "value2"},
			},
			expected: "",
			marshalFunc: func(v any) ([]byte, error) {
				return nil, fmt.Errorf("failed to marshal")
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Override(&GetClientInstallID, func() string { return "00000000-0000-0000-0000-000000000000" })
			if test.marshalFunc != nil {
				t.Override(&Marshal, test.marshalFunc)
			}
			out, err := buildProtoStr(test.eventName, test.kvs)
			t.CheckErrorAndDeepEqual(test.wantErr, err, test.expected, out)
		})
	}
}

func TestSendDataPoint(t *testing.T) {
	tests := []struct {
		name      string
		dp        DataPoint
		POSTFunc  func(url, contentType string, body io.Reader) (resp *http.Response, err error)
		shouldErr bool
	}{
		{
			name: "no attributes",
			dp: DataPointInt64{
				Value:      10,
				Attributes: attribute.NewSet(),
				StartTime:  time.UnixMilli(123),
				Time:       time.UnixMilli(123),
			},
			POSTFunc: func(url, contentType string, body io.Reader) (resp *http.Response, err error) {
				responseBody := io.NopCloser(bytes.NewReader([]byte(`{"value":"fixed"}`)))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       responseBody,
				}, nil
			},
			shouldErr: false,
		},
		{
			name: "with attributes",
			dp: DataPointInt64{
				Value:      10,
				Attributes: attribute.NewSet(attribute.Int("iteration", 10)),
				StartTime:  time.UnixMilli(123),
				Time:       time.UnixMilli(123),
			},
			POSTFunc: func(url, contentType string, body io.Reader) (resp *http.Response, err error) {
				responseBody := io.NopCloser(bytes.NewReader([]byte(`{"value":"fixed"}`)))
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       responseBody,
				}, nil
			},
			shouldErr: false,
		},
		{
			name: "http non-200",
			dp: DataPointInt64{
				Value:      10,
				Attributes: attribute.NewSet(attribute.Int("iteration", 10)),
				StartTime:  time.UnixMilli(123),
				Time:       time.UnixMilli(123),
			},
			POSTFunc: func(url, contentType string, body io.Reader) (resp *http.Response, err error) {
				responseBody := io.NopCloser(bytes.NewReader([]byte(`{"value":"fixed"}`)))
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       responseBody,
				}, nil
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			t.Override(&POST, test.POSTFunc)
			err := sendDataPoint(test.name, test.dp)
			t.CheckError(test.shouldErr, err)
		})
	}
}
