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

package integration

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/build/ko"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/platform"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
)

func TestBuildAndPushKoImageProgrammatically(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	// Start a local registry server.
	// This registry hosts the base image, and it is the target registry for the built image.
	baseimageNamespace := "baseimage"
	registryServer, err := registryServerWithImage(baseimageNamespace)
	if err != nil {
		t.Fatalf("could not create test registry server: %v", err)
	}
	defer registryServer.Close()
	registryAddr := registryServer.Listener.Addr().String()
	baseImage := fmt.Sprintf("%s/%s", registryAddr, baseimageNamespace)

	// Get the directory of the basic ko sample app from the `examples` directory.
	exampleAppDir, err := koExampleAppDir()
	if err != nil {
		t.Fatalf("could not get ko example app dir: %+v", err)
	}

	// Build the artifact
	b := ko.NewArtifactBuilder(nil, true, config.RunModes.Build, nil)
	artifact := &latest.Artifact{
		ArtifactType: latest.ArtifactType{
			KoArtifact: &latest.KoArtifact{
				BaseImage: baseImage,
			},
		},
		Workspace: exampleAppDir,
	}
	imageName := fmt.Sprintf("%s/%s", registryAddr, "skaffold-ko")
	_, err = b.Build(context.Background(), nil, artifact, imageName, platform.Matcher{})
	if err != nil {
		t.Fatalf("b.Build(): %+v", err)
	}
}

// registryServerWithImage starts a local registry and pushes a random image.
// Use this to speed up tests, by not having to reach out to a real registry.
// The registry uses a NOP logger to avoid spamming test logs.
// Remember to call `defer Close()` on the returned `httptest.Server`.
func registryServerWithImage(namespace string) (*httptest.Server, error) {
	nopLog := stdlog.New(io.Discard, "", 0)
	r := registry.New(registry.Logger(nopLog))
	s := httptest.NewServer(r)
	imageName := fmt.Sprintf("%s/%s", s.Listener.Addr().String(), namespace)
	image, err := random.Image(1024, 1)
	if err != nil {
		return nil, fmt.Errorf("random.Image(): %+v", err)
	}
	// ko 0.12 starts to validate image platform, the random generated image doesn't that info, we need to manually to set
	// image platform back to the image. image.ConfigFile() is a deepCopy() method, we need to use mutate.ConfigFile() to inject
	// the platform info.
	configFile, _ := image.ConfigFile()
	configFile.OS = runtime.GOOS
	configFile.Architecture = runtime.GOARCH
	configFile.OSVersion = runtime.Version()

	image, err = mutate.ConfigFile(image, configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to mutate image: %+v", err)
	}

	err = crane.Push(image, imageName)
	if err != nil {
		return nil, fmt.Errorf("crane.Push(): %+v", err)
	}
	return s, nil
}

// koExampleAppDir returns the directory path of the basic ko builder sample app.
func koExampleAppDir() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("could not get current filename")
	}
	basepath := filepath.Dir(filename)
	exampleDir, err := filepath.Abs(filepath.Join(basepath, "examples", "ko"))
	if err != nil {
		return "", fmt.Errorf("could not get absolute path of example from basepath %q: %w", basepath, err)
	}
	return exampleDir, nil
}
