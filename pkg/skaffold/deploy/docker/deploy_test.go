/*
Copyright 2021 The Skaffold Authors

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

package docker

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker/debugger"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/util"
	"github.com/GoogleContainerTools/skaffold/v2/testutil"
)

type debugArtifact struct {
	image            string
	debug            bool
	expectedBindings nat.PortMap
}

func TestDebugBindings(t *testing.T) {
	// essentially, emulates pairing an image name with a detected runtime (and a default debugger port)
	testDebugPorts := map[string]uint32{
		"go":     uint32(56268),
		"nodejs": uint32(9229),
	}

	tests := []struct {
		name                 string
		artifacts            []debugArtifact
		portForwardResources []*latest.PortForwardResource
	}{
		{
			name: "one artifact one binding",
			artifacts: []debugArtifact{
				{
					image: "go",
					debug: true,
					expectedBindings: nat.PortMap{
						"56268/tcp": {{HostIP: "127.0.0.1", HostPort: "56268"}},
					},
				},
			},
			portForwardResources: nil,
		},
		{
			name: "two artifacts two bindings",
			artifacts: []debugArtifact{
				{
					image: "go",
					debug: true,
					expectedBindings: nat.PortMap{
						"56268/tcp": {{HostIP: "127.0.0.1", HostPort: "56268"}},
					},
				},
				{
					image: "nodejs",
					debug: true,
					expectedBindings: nat.PortMap{
						"9229/tcp": {{HostIP: "127.0.0.1", HostPort: "9229"}},
					},
				},
			},
			portForwardResources: nil,
		},
		{
			name: "two artifacts but one not configured for debugging",
			artifacts: []debugArtifact{
				{
					image: "go",
					debug: true,
					expectedBindings: nat.PortMap{
						"56268/tcp": {{HostIP: "127.0.0.1", HostPort: "56268"}},
					},
				},
				{
					image: "nodejs",
					debug: false,
				},
			},
			portForwardResources: nil,
		},
		{
			name: "two artifacts with same runtime - port collision",
			artifacts: []debugArtifact{
				{
					image: "go",
					debug: true,
					expectedBindings: nat.PortMap{
						"56268/tcp": {{HostIP: "127.0.0.1", HostPort: "56268"}},
					},
				},
				{
					image: "go",
					debug: true,
					expectedBindings: nat.PortMap{
						"56268/tcp": {{HostIP: "127.0.0.1", HostPort: "56269"}},
					},
				},
			},
			portForwardResources: nil,
		},
		{
			name: "two artifacts two bindings and two port forward resources",
			artifacts: []debugArtifact{
				{
					image: "go",
					debug: true,
					expectedBindings: nat.PortMap{
						"56268/tcp": {{HostIP: "127.0.0.1", HostPort: "56268"}},
						"9000/tcp":  nil, // Allow any mapping
					},
				},
				{
					image: "nodejs",
					debug: true,
					expectedBindings: nat.PortMap{
						"9229/tcp": {{HostIP: "127.0.0.1", HostPort: "9229"}},
						"9090/tcp": nil, // Allow any mapping
					},
				},
			},
			portForwardResources: []*latest.PortForwardResource{
				{
					Name: "go",
					Type: "container",
					Port: util.IntOrString{
						IntVal: 9000,
					},
				},
				{
					Name: "nodejs",
					Type: "container",
					Port: util.IntOrString{
						IntVal: 9090,
					},
				},
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.name, func(tt *testutil.T) {
			// this override ensures that the returned debug configurations are set on the DebugManager
			tt.Override(&debugger.TransformImage, func(ctx context.Context, artifact graph.Artifact, cfg *container.Config, insecureRegistries map[string]bool, debugHelpersRegistry string) (map[string]types.ContainerDebugConfiguration, []*container.Config, error) {
				configs := make(map[string]types.ContainerDebugConfiguration)
				ports := make(map[string]uint32)
				// tie the provided debug port to the artifact's image name to emulate this image being configured for debugging
				// the debug runtime is not relevant for this test, and neither is the port key.
				ports["ignored"] = testDebugPorts[artifact.ImageName]
				configs[artifact.ImageName] = types.ContainerDebugConfiguration{
					Ports: ports,
				}
				return configs, nil, nil
			})

			d, _ := NewDeployer(context.TODO(), mockConfig{}, &label.DefaultLabeller{}, nil, test.portForwardResources, "default")

			for _, a := range test.artifacts {
				config := container.Config{
					Image: a.image,
				}
				var (
					debugBindings nat.PortMap
					err           error
				)

				if a.debug {
					debugBindings, err = d.setupDebugging(context.TODO(), nil, graph.Artifact{ImageName: a.image}, &config)
				}
				testutil.CheckErrorAndFailNow(t, false, err)

				filteredPFResources := d.filterPortForwardingResources(a.image)

				bindings, err := d.portManager.AllocatePorts(a.image, filteredPFResources, &config, debugBindings)
				testutil.CheckErrorAndFailNow(t, false, err)

				// CheckDeepEqual unfortunately doesn't work when the map elements are slices
				for k, v := range a.expectedBindings {
					// TODO: If given nil, assume that any value works.
					// Otherwise, perform equality check.
					if v == nil {
						continue
					}
					testutil.CheckDeepEqual(t, v, bindings[k])
				}
				if len(a.expectedBindings) != len(bindings) {
					t.Errorf("mismatch number of bindings. Expected number of bindings: %d. Actual number of bindings: %d\n", len(a.expectedBindings), len(bindings))
				}
			}
		})
	}
}

func TestFilterPortForwardingResources(t *testing.T) {
	resources := []*latest.PortForwardResource{
		{
			Name: "image1",
			Port: util.FromInt(8000),
		},
		{
			Name: "image2",
			Port: util.FromInt(8001),
		},
		{
			Name: "image2",
			Port: util.FromInt(8002),
		},
	}
	tests := []struct {
		name                  string
		imageName             string
		expectedPortResources []*latest.PortForwardResource
	}{
		{
			name:                  "image name not in list",
			imageName:             "image3",
			expectedPortResources: []*latest.PortForwardResource{},
		},
		{
			name:      "image in list. return one",
			imageName: "image1",
			expectedPortResources: []*latest.PortForwardResource{
				{
					Name: "image1",
					Port: util.FromInt(8000),
				},
			},
		},
		{
			name:      "image in list. return multiple",
			imageName: "image2",
			expectedPortResources: []*latest.PortForwardResource{
				{
					Name: "image2",
					Port: util.FromInt(8001),
				},
				{
					Name: "image2",
					Port: util.FromInt(8002),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := Deployer{resources: resources}
			pfResources := d.filterPortForwardingResources(test.imageName)
			testutil.CheckDeepEqual(t, test.expectedPortResources, pfResources)
		})
	}
}

type mockConfig struct{}

func (m mockConfig) ContainerDebugging() bool               { return true }
func (m mockConfig) GetInsecureRegistries() map[string]bool { return nil }
func (m mockConfig) GetKubeContext() string                 { return "" }
func (m mockConfig) GlobalConfig() string                   { return "" }
func (m mockConfig) MinikubeProfile() string                { return "" }
func (m mockConfig) Mode() config.RunMode                   { return "" }
func (m mockConfig) Prune() bool                            { return false }
