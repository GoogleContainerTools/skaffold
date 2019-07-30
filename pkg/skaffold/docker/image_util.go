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
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func RetrieveWorkingDir(tagged string, insecureRegistries map[string]bool) (string, error) {
	var cf *v1.ConfigFile
	var err error

	if strings.ToLower(tagged) == "scratch" {
		return "/", nil
	}

	// TODO: use the proper RunContext
	localDocker, err := NewAPIClient(&runcontext.RunContext{})
	if err == nil {
		cf, err = localDocker.ConfigFile(context.Background(), tagged)
	}
	if err != nil {
		// No local Docker is available
		cf, err = RetrieveRemoteConfig(tagged, insecureRegistries)
	}
	if err != nil {
		return "", errors.Wrap(err, "retrieving image config")
	}

	if cf.Config.WorkingDir == "" {
		logrus.Debugf("Using default workdir '/' for %s", tagged)
		return "/", nil
	}
	return cf.Config.WorkingDir, nil
}
