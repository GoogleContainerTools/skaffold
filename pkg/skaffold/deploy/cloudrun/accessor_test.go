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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

type testAccessConfig struct {
	options config.PortForwardOptions
}

func (t *testAccessConfig) PortForwardOptions() config.PortForwardOptions {
	return t.options
}
func (t *testAccessConfig) Mode() config.RunMode {
	return config.RunModes.Run
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
