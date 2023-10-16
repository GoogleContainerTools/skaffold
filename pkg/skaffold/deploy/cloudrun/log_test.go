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

package cloudrun

import (
	"bufio"
	"bytes"
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

type testConfig struct {
	tail bool
}

func (t *testConfig) Tail() bool {
	return t.tail
}

func (t *testConfig) Mode() config.RunMode {
	return config.RunModes.Run
}

func (t *testConfig) PortForwardOptions() config.PortForwardOptions {
	return config.PortForwardOptions{}
}

func (t *testConfig) PortForwardResources() []*latest.PortForwardResource {
	return nil
}
func newLogTestConfig(tail bool) *testConfig {
	return &testConfig{tail: tail}
}
func TestNewLoggerAggregator(t *testing.T) {
	tests := []struct {
		name       string
		tail       bool
		numTailers int
	}{
		{
			name:       "tailing is turned on",
			tail:       true,
			numTailers: 1,
		},
		{
			name:       "tailing is turned off",
			tail:       false,
			numTailers: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := newLogTestConfig(test.tail)
			logAggregator := NewLoggerAggregator(cfg, "test")
			if len(logAggregator.logTailers) != test.numTailers {
				t.Fatalf("expected %d forwarders, but got %v", test.numTailers, logAggregator.logTailers)
			}
		})
	}
}

func TestResourcesAddedLogTailers(t *testing.T) {
	tests := []struct {
		name      string
		resources []RunResourceName
		outputs   []logTailerResource
	}{
		{
			name: "one service with one color",
			resources: []RunResourceName{
				{
					Project: "test-proj",
					Region:  "test-region",
					Service: "test-service",
				},
			},
			outputs: []logTailerResource{
				{
					name: RunResourceName{
						Project: "test-proj",
						Region:  "test-region",
						Service: "test-service",
					},
					formatter: LogFormatter{
						prefix:      "test-service",
						outputColor: output.DefaultColorCodes[0],
					},
					isTailing: false,
				},
			},
		},
		{
			name: "two services with same name",
			resources: []RunResourceName{
				{
					Project: "test-proj",
					Region:  "test-region",
					Service: "test-service",
				},
				{
					Project: "test-proj",
					Region:  "test-region",
					Service: "test-service",
				},
			},
			outputs: []logTailerResource{
				{
					name: RunResourceName{
						Project: "test-proj",
						Region:  "test-region",
						Service: "test-service",
					},
					formatter: LogFormatter{
						prefix:      "test-service",
						outputColor: output.DefaultColorCodes[0],
					},
					isTailing: true,
				},
			},
		},
		{
			name: "two services with different names",
			resources: []RunResourceName{
				{
					Project: "test-proj",
					Region:  "test-region",
					Service: "test-service",
				},
				{
					Project: "test-proj",
					Region:  "test-region",
					Service: "test-service2",
				},
			},
			outputs: []logTailerResource{
				{
					name: RunResourceName{
						Project: "test-proj",
						Region:  "test-region",
						Service: "test-service",
					},
					formatter: LogFormatter{
						prefix:      "test-service",
						outputColor: output.DefaultColorCodes[0],
					},
					isTailing: false,
				},
				{
					name: RunResourceName{
						Project: "test-proj",
						Region:  "test-region",
						Service: "test-service2",
					},
					formatter: LogFormatter{
						prefix:      "test-service2",
						outputColor: output.DefaultColorCodes[1],
					},
					isTailing: false,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := newLogTestConfig(true)
			logAggregator := NewLoggerAggregator(cfg, "")
			for _, resource := range test.resources {
				logAggregator.AddResource(resource)
			}
			if len(test.outputs) != len(logAggregator.resources.resources) {
				t.Fatalf("Mismatch in expected outputs. Expected %v, got %v", test.outputs, logAggregator.resources.resources)
			}
			for _, result := range test.outputs {
				got, found := logAggregator.resources.resources[result.name]
				if !found {
					t.Fatalf("expected to find log resource %v but got nothing", result.name)
				}
				if result.name != got.name || result.formatter != got.formatter {
					t.Fatalf("did not get expected result. Expected %v, got %v", result, got)
				}
			}
		})
	}
}

func TestGcloudFoundLogTailing(t *testing.T) {
	tests := []struct {
		name         string
		gcloudFound  bool
		expectStatus proto.StatusCode
	}{
		{
			name:        "gcloud found",
			gcloudFound: true,
		},
		{
			name:         "gcloud not found",
			gcloudFound:  false,
			expectStatus: proto.StatusCode_LOG_STREAM_RUN_GCLOUD_NOT_FOUND,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logTailers := runLogTailer{resources: &loggerTracker{}}
			gcloudInstalled = func() bool { return test.gcloudFound }
			ctx := context.Background()
			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			err := logTailers.Start(ctx, writer)
			writer.Flush()
			result := b.Bytes()
			if test.expectStatus == proto.StatusCode_OK {
				if err != nil {
					t.Fatalf("expected success, got error: %v with output %s", err, string(result))
				}
			} else {
				if err == nil {
					t.Fatalf("expected failure with code %v, got success with output: %s", test.expectStatus, string(result))
				}
				sErr := err.(*sErrors.ErrDef)
				if sErr.StatusCode() != test.expectStatus {
					t.Fatalf("expected failure with code %v, got code %v with output: %s", test.expectStatus, sErr.StatusCode(), string(result))
				}
			}
		})
	}
}
