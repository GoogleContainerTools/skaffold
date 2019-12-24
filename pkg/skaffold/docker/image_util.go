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

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
)

func RetrieveConfigFile(tagged string, runCtx *runcontext.RunContext) (*v1.ConfigFile, error) {
	if strings.ToLower(tagged) == "scratch" {
		return nil, nil
	}

	var cf *v1.ConfigFile
	var err error

	localDocker, err := NewAPIClient(runCtx)
	if err == nil {
		cf, err = localDocker.ConfigFile(context.Background(), tagged)
	}
	if err != nil {
		// No local Docker is available
		cf, err = RetrieveRemoteConfig(tagged, runCtx.InsecureRegistries)
	}
	if err != nil {
		return nil, errors.Wrap(err, "retrieving image config")
	}

	return cf, err
}

func RetrieveWorkingDir(tagged string, runCtx *runcontext.RunContext) (string, error) {
	cf, err := RetrieveConfigFile(tagged, runCtx)
	switch {
	case err != nil:
		return "", err
	case cf == nil:
		return "/", nil
	case cf.Config.WorkingDir == "":
		logrus.Debugf("Using default workdir '/' for %s", tagged)
		return "/", nil
	default:
		return cf.Config.WorkingDir, nil
	}
}
