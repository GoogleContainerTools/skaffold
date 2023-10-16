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

package ko

import (
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/ko/pkg/commands"
	"github.com/google/ko/pkg/commands/options"
	"github.com/google/ko/pkg/publish"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/version"
)

func (b *Builder) newKoPublisher(ref string) (publish.Interface, error) {
	var dockerClient daemon.Client
	if b.localDocker != nil {
		dockerClient = b.localDocker.RawClient()
	}
	po, err := publishOptions(ref, b.pushImages, dockerClient, b.insecureRegistries)
	if err != nil {
		return nil, err
	}
	return commands.NewPublisher(po)
}

func publishOptions(ref string, pushImages bool, dockerClient daemon.Client, insecureRegistries map[string]bool) (*options.PublishOptions, error) {
	imageRef, err := name.ParseReference(ref)
	if err != nil {
		return nil, err
	}
	imageNameWithoutTag := imageRef.Context().Name()
	localDomain := ""
	if !pushImages {
		localDomain = imageNameWithoutTag
	}

	return &options.PublishOptions{
		Bare:             true,
		DockerClient:     dockerClient,
		DockerRepo:       imageNameWithoutTag,
		InsecureRegistry: useInsecureRegistry(imageNameWithoutTag, insecureRegistries),
		Local:            !pushImages,
		LocalDomain:      localDomain,
		Push:             pushImages,
		Tags:             []string{imageRef.Identifier()},
		UserAgent:        version.UserAgentWithClient(),
	}, nil
}

func useInsecureRegistry(imageName string, insecureRegistries map[string]bool) bool {
	for registry := range insecureRegistries {
		if strings.HasPrefix(imageName, registry) {
			return true
		}
	}
	return false
}
