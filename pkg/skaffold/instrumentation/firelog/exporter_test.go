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
	"testing"

	"go.opentelemetry.io/otel/attribute"

	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

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
					EventUptimeMS:                200,
					SourceExtensionJSONProto3Str: "{foo:a}",
				},
				RequestTimeMS:   100,
				RequestUptimeMS: 200,
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(t *testutil.T) {
			actual := buildMetricData(test.proto, test.startTimeMS, test.upTimeMS)
			t.CheckDeepEqual(test.expected, actual)
		})
	}
}
