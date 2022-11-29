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
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

type testAccessConfig struct {
	options  config.PortForwardOptions
	forwards []*latest.PortForwardResource
}

func (t *testAccessConfig) PortForwardOptions() config.PortForwardOptions {
	return t.options
}
func (t *testAccessConfig) Mode() config.RunMode {
	return config.RunModes.Run
}
func (t *testAccessConfig) Tail() bool {
	return true
}
func (t *testAccessConfig) PortForwardResources() []*latest.PortForwardResource {
	return t.forwards
}
func newTestConfig(forwardModes string) *testAccessConfig {
	options := config.PortForwardOptions{}
	options.Set(forwardModes)
	return &testAccessConfig{options: options}
}
func TestNewAccessor(t *testing.T) {
	tests := []struct {
		name          string
		forwardModes  string
		numForwarders int
	}{
		{
			name:          "forwards services has forwarder",
			forwardModes:  "services",
			numForwarders: 1,
		},
		{
			name:          "forwards user only has no forwarders",
			forwardModes:  "user",
			numForwarders: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := newTestConfig(test.forwardModes)
			accessor := NewAccessor(cfg, "")
			if len(accessor.forwarders) != test.numForwarders {
				t.Fatalf("expected %d forwarders, but got %v", test.numForwarders, accessor.forwarders)
			}
		})
	}
}
func TestResourcesAddedConfigurePorts(t *testing.T) {
	tests := []struct {
		name           string
		resources      []RunResourceName
		forwardConfigs []*latest.PortForwardResource
		outputs        []forwardedResource
	}{
		{
			name: "no forwards has no ports set",
			resources: []RunResourceName{
				{
					Project: "test-proj",
					Region:  "test-region",
					Service: "test-service",
				},
			},
			outputs: []forwardedResource{
				{
					name: RunResourceName{
						Project: "test-proj",
						Region:  "test-region",
						Service: "test-service",
					},
					port: 0,
				},
			},
		},
		{
			name: "forward has port set",
			resources: []RunResourceName{
				{
					Project: "test-proj",
					Region:  "test-region",
					Service: "test-service",
				},
			},
			forwardConfigs: []*latest.PortForwardResource{
				{
					Type:      "service",
					Name:      "test-service",
					LocalPort: 9000,
				},
			},
			outputs: []forwardedResource{
				{
					name: RunResourceName{
						Project: "test-proj",
						Region:  "test-region",
						Service: "test-service",
					},
					port: 9000,
				},
			},
		},
		{
			name: "name mismatch has no port set",
			resources: []RunResourceName{
				{
					Project: "test-proj",
					Region:  "test-region",
					Service: "test-service",
				},
			},
			forwardConfigs: []*latest.PortForwardResource{
				{
					Type:      "service",
					Name:      "test-service2",
					LocalPort: 9000,
				},
			},
			outputs: []forwardedResource{
				{
					name: RunResourceName{
						Project: "test-proj",
						Region:  "test-region",
						Service: "test-service",
					},
					port: 0,
				},
			},
		},
		{
			name: "resources added twice only have one port forward configured",
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
			forwardConfigs: []*latest.PortForwardResource{},
			outputs: []forwardedResource{
				{
					name: RunResourceName{
						Project: "test-proj",
						Region:  "test-region",
						Service: "test-service",
					},
					port: 0,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := newTestConfig("")
			cfg.forwards = test.forwardConfigs
			accessor := NewAccessor(cfg, "")
			for _, resource := range test.resources {
				accessor.AddResource(resource)
			}
			if len(test.outputs) != len(accessor.resources.resources) {
				t.Fatalf("Mismatch in expected outputs. Expected %v, got %v", test.outputs, accessor.resources.resources)
			}
			for _, output := range test.outputs {
				got, found := accessor.resources.resources[output.name]
				if !found {
					t.Fatalf("expected to find port forward for %v but got nothing", output.name)
				}
				if output.name != got.name || output.port != got.port {
					t.Fatalf("did not get expected port set. Expected %v, got %v", output, got)
				}
			}
		})
	}
}

func TestGcloudFoundServicesForwarder(t *testing.T) {
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
			expectStatus: proto.StatusCode_PORT_FORWARD_RUN_GCLOUD_NOT_FOUND,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			forwarder := runProxyForwarder{resources: &resourceTracker{}}
			gcloudInstalled = func() bool { return test.gcloudFound }
			ctx := context.Background()
			var b bytes.Buffer
			writer := bufio.NewWriter(&b)
			err := forwarder.Start(ctx, writer)
			writer.Flush()
			output := b.Bytes()
			if test.expectStatus == proto.StatusCode_OK {
				if err != nil {
					t.Fatalf("expected success, got error: %v with output %s", err, string(output))
				}
			} else {
				if err == nil {
					t.Fatalf("expected failure with code %v, got success with output: %s", test.expectStatus, string(output))
				}
				sErr := err.(*sErrors.ErrDef)
				if sErr.StatusCode() != test.expectStatus {
					t.Fatalf("expected failure with code %v, got code %v with output: %s", test.expectStatus, sErr.StatusCode(), string(output))
				}
			}
		})
	}
}
