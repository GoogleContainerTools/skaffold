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
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/daemon"
	"github.com/google/ko/pkg/commands"
	"github.com/google/ko/pkg/commands/options"
	"github.com/google/ko/pkg/publish"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/version"
)

func (b *Builder) newKoPublisher(ref string) (publish.Interface, error) {
	po, err := publishOptions(ref, b.pushImages, b.localDocker.RawClient())
	if err != nil {
		return nil, err
	}
	return commands.NewPublisher(po)
}

func publishOptions(ref string, pushImages bool, dockerClient daemon.Client) (*options.PublishOptions, error) {
	imageRef, err := name.ParseReference(ref)
	if err != nil {
		return nil, err
	}
	imageNameWithoutTag := imageRef.Context().Name()
	return &options.PublishOptions{
		Bare:         true,
		DockerClient: dockerClient,
		DockerRepo:   imageNameWithoutTag,
		Local:        !pushImages,
		LocalDomain:  imageNameWithoutTag,
		Push:         pushImages,
		Tags:         []string{imageRef.Identifier()},
		UserAgent:    version.UserAgentWithClient(),
	}, nil
}
