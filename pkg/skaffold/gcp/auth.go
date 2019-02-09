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

package gcp

import (
	"os/exec"
	"strings"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/sirupsen/logrus"
)

// AutoConfigureGCRCredentialHelper automatically adds the `gcloud` credential helper
// to docker's configuration.
// This doesn't modify the ~/.docker/config.json. It's only in-memory
func AutoConfigureGCRCredentialHelper(cf *configfile.ConfigFile, registry string) {
	if registry != "gcr.io" && !strings.HasSuffix(registry, ".gcr.io") {
		logrus.Debugln("Skipping credential configuration because registry is not gcr.")
		return
	}

	if cf.CredentialHelpers != nil && cf.CredentialHelpers[registry] != "" {
		logrus.Debugln("Skipping credential configuration because credentials are already configured.")
		return
	}

	if path, _ := exec.LookPath("docker-credential-gcloud"); path == "" {
		logrus.Debugln("Skipping credential configuration because docker-credential-gcloud is not on PATH.")
		return
	}

	logrus.Debugf("Adding `gcloud` as a docker credential helper for [%s]", registry)

	if cf.CredentialHelpers == nil {
		cf.CredentialHelpers = make(map[string]string)
	}
	cf.CredentialHelpers[registry] = "gcloud"
}
