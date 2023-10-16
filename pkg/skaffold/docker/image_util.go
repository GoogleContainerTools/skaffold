/*
Copyright 2019 The Skaffold Authors

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
	"fmt"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
)

func RetrieveConfigFile(ctx context.Context, tagged string, cfg Config) (*v1.ConfigFile, error) {
	if strings.ToLower(tagged) == "scratch" {
		return nil, nil
	}

	var cf *v1.ConfigFile
	var err error

	localDocker, err := NewAPIClient(ctx, cfg)
	if err == nil {
		cf, err = localDocker.ConfigFile(context.Background(), tagged)
	}
	if err != nil {
		// No local Docker is available
		cf, err = RetrieveRemoteConfig(tagged, cfg, v1.Platform{})
	}
	if err != nil {
		return nil, fmt.Errorf("retrieving image config: %w", err)
	}

	return cf, err
}

func RetrieveWorkingDir(ctx context.Context, tagged string, cfg Config) (string, error) {
	cf, err := RetrieveConfigFile(ctx, tagged, cfg)
	switch {
	case err != nil:
		return "", err
	case cf == nil:
		return "/", nil
	case cf.Config.WorkingDir == "":
		log.Entry(context.TODO()).Debugf("Using default workdir '/' for %s", tagged)
		return "/", nil
	default:
		return cf.Config.WorkingDir, nil
	}
}

func RetrieveLabels(ctx context.Context, tagged string, cfg Config) (map[string]string, error) {
	cf, err := RetrieveConfigFile(ctx, tagged, cfg)
	switch {
	case err != nil:
		return nil, err
	case cf == nil:
		return nil, nil
	default:
		return cf.Config.Labels, nil
	}
}
