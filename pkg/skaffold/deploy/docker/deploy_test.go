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
	"os"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"gopkg.in/yaml.v3"

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

// Tests for Docker Compose deployer

func TestReplaceComposeImages(t *testing.T) {
	tests := []struct {
		name           string
		composeConfig  map[string]interface{}
		artifact       graph.Artifact
		expectedConfig map[string]interface{}
		shouldErr      bool
	}{
		{
			name: "single service image replacement",
			composeConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"app": map[string]interface{}{
						"image": "myapp",
					},
				},
			},
			artifact: graph.Artifact{
				ImageName: "myapp",
				Tag:       "myapp:v1.2.3",
			},
			expectedConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"app": map[string]interface{}{
						"image": "myapp:v1.2.3",
					},
				},
			},
			shouldErr: false,
		},
		{
			name: "multiple services only matching one replaced",
			composeConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"frontend": map[string]interface{}{
						"image": "frontend-app",
					},
					"backend": map[string]interface{}{
						"image": "backend-app",
					},
					"database": map[string]interface{}{
						"image": "postgres:14",
					},
				},
			},
			artifact: graph.Artifact{
				ImageName: "frontend-app",
				Tag:       "frontend-app:latest",
			},
			expectedConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"frontend": map[string]interface{}{
						"image": "frontend-app:latest",
					},
					"backend": map[string]interface{}{
						"image": "backend-app",
					},
					"database": map[string]interface{}{
						"image": "postgres:14",
					},
				},
			},
			shouldErr: false,
		},
		{
			name: "no services section returns error",
			composeConfig: map[string]interface{}{
				"version": "3.8",
			},
			artifact: graph.Artifact{
				ImageName: "myapp",
				Tag:       "myapp:v1",
			},
			expectedConfig: nil,
			shouldErr:      true,
		},
		{
			name: "service without image field is skipped",
			composeConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"app": map[string]interface{}{
						"build": ".",
					},
				},
			},
			artifact: graph.Artifact{
				ImageName: "myapp",
				Tag:       "myapp:v1",
			},
			expectedConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"app": map[string]interface{}{
						"build": ".",
					},
				},
			},
			shouldErr: false,
		},
		{
			name: "partial image name match with contains",
			composeConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"app": map[string]interface{}{
						"image": "myapp",
					},
				},
			},
			artifact: graph.Artifact{
				ImageName: "gcr.io/project/myapp",
				Tag:       "gcr.io/project/myapp:sha256",
			},
			expectedConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"app": map[string]interface{}{
						"image": "gcr.io/project/myapp:sha256",
					},
				},
			},
			shouldErr: false,
		},
		{
			name: "reject false positive: 'app' should not match 'my-app'",
			composeConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"myapp": map[string]interface{}{
						"image": "app",
					},
				},
			},
			artifact: graph.Artifact{
				ImageName: "my-app",
				Tag:       "my-app:v1.0.0",
			},
			expectedConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"myapp": map[string]interface{}{
						"image": "app", // Should NOT be replaced
					},
				},
			},
			shouldErr: false,
		},
		{
			name: "match with registry prefix",
			composeConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"frontend": map[string]interface{}{
						"image": "frontend",
					},
				},
			},
			artifact: graph.Artifact{
				ImageName: "frontend",
				Tag:       "gcr.io/project/frontend:sha256-abc",
			},
			expectedConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"frontend": map[string]interface{}{
						"image": "gcr.io/project/frontend:sha256-abc",
					},
				},
			},
			shouldErr: false,
		},
		{
			name: "match with tag in compose file",
			composeConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"backend": map[string]interface{}{
						"image": "backend:latest",
					},
				},
			},
			artifact: graph.Artifact{
				ImageName: "backend",
				Tag:       "backend:v2.0.0",
			},
			expectedConfig: map[string]interface{}{
				"services": map[string]interface{}{
					"backend": map[string]interface{}{
						"image": "backend:v2.0.0",
					},
				},
			},
			shouldErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := &Deployer{}
			err := d.replaceComposeImages(test.composeConfig, test.artifact)

			if test.shouldErr && err == nil {
				t.Error("expected error but got none")
			}
			if !test.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !test.shouldErr && err == nil {
				testutil.CheckDeepEqual(t, test.expectedConfig, test.composeConfig)
			}
		})
	}
}

func TestGetComposeFilePath(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		setEnv       bool
		expectedPath string
	}{
		{
			name:         "default path when env not set",
			setEnv:       false,
			expectedPath: "docker-compose.yml",
		},
		{
			name:         "custom path from env variable",
			envValue:     "custom-compose.yml",
			setEnv:       true,
			expectedPath: "custom-compose.yml",
		},
		{
			name:         "custom path with directory",
			envValue:     "/path/to/my-compose.yml",
			setEnv:       true,
			expectedPath: "/path/to/my-compose.yml",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Save original env value
			originalEnv := os.Getenv("SKAFFOLD_COMPOSE_FILE")
			defer func() {
				if originalEnv != "" {
					os.Setenv("SKAFFOLD_COMPOSE_FILE", originalEnv)
				} else {
					os.Unsetenv("SKAFFOLD_COMPOSE_FILE")
				}
			}()

			// Set up test environment
			if test.setEnv {
				os.Setenv("SKAFFOLD_COMPOSE_FILE", test.envValue)
			} else {
				os.Unsetenv("SKAFFOLD_COMPOSE_FILE")
			}

			d := &Deployer{}
			path := d.getComposeFilePath()

			if path != test.expectedPath {
				t.Errorf("expected path %q but got %q", test.expectedPath, path)
			}
		})
	}
}

func TestDeployWithComposeFileOperations(t *testing.T) {
	tests := []struct {
		name        string
		composeFile string
		shouldErr   bool
	}{
		{
			name:        "valid compose file",
			composeFile: "testdata/docker-compose.yml",
			shouldErr:   false,
		},
		{
			name:        "valid compose file with multiple services",
			composeFile: "testdata/docker-compose-multi.yml",
			shouldErr:   false,
		},
		{
			name:        "invalid yaml returns error",
			composeFile: "testdata/docker-compose-invalid.yml",
			shouldErr:   true,
		},
		{
			name:        "non-existent file returns error",
			composeFile: "testdata/does-not-exist.yml",
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test file reading and parsing
			data, err := os.ReadFile(test.composeFile)
			if err != nil && !test.shouldErr {
				t.Fatalf("failed to read test file: %v", err)
			}
			if err != nil && test.shouldErr {
				// Expected error for non-existent file
				return
			}

			var composeConfig map[string]interface{}
			err = yaml.Unmarshal(data, &composeConfig)

			if test.shouldErr && err == nil {
				t.Error("expected error parsing yaml but got none")
			}
			if !test.shouldErr && err != nil {
				t.Errorf("unexpected error parsing yaml: %v", err)
			}
			if !test.shouldErr && err == nil {
				// Verify we can access services
				if services, ok := composeConfig["services"]; !ok {
					t.Error("expected services section in compose config")
				} else if services == nil {
					t.Error("services section is nil")
				}
			}
		})
	}
}
