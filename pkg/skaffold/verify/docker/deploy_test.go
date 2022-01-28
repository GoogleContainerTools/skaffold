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

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/debug/types"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker/debugger"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/testutil"
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
		name      string
		artifacts []debugArtifact
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
			d, _ := NewVerifier(context.TODO(), mockConfig{}, &label.DefaultLabeller{}, nil, nil, "")

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

				bindings, err := d.portManager.AllocatePorts(a.image, d.resources, &config, debugBindings)
				testutil.CheckErrorAndFailNow(t, false, err)

				// CheckDeepEqual unfortunately doesn't work when the map elements are slices
				for k, v := range a.expectedBindings {
					testutil.CheckDeepEqual(t, v, bindings[k])
				}
			}
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
