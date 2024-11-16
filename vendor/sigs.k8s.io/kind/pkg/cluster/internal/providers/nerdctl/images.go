/*
Copyright 2019 The Kubernetes Authors.

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

package nerdctl

import (
	"fmt"
	"strings"
	"time"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
	"sigs.k8s.io/kind/pkg/log"

	"sigs.k8s.io/kind/pkg/cluster/internal/providers/common"
	"sigs.k8s.io/kind/pkg/internal/apis/config"
	"sigs.k8s.io/kind/pkg/internal/cli"
)

// ensureNodeImages ensures that the node images used by the create
// configuration are present
func ensureNodeImages(logger log.Logger, status *cli.Status, cfg *config.Cluster, binaryName string) error {
	// pull each required image
	for _, image := range common.RequiredNodeImages(cfg).List() {
		// prints user friendly message
		friendlyImageName, image := sanitizeImage(image)
		status.Start(fmt.Sprintf("Ensuring node image (%s) 🖼", friendlyImageName))
		if _, err := pullIfNotPresent(logger, image, 4, binaryName); err != nil {
			status.End(false)
			return err
		}
	}
	return nil
}

// pullIfNotPresent will pull an image if it is not present locally
// retrying up to retries times
// it returns true if it attempted to pull, and any errors from pulling
func pullIfNotPresent(logger log.Logger, image string, retries int, binaryName string) (pulled bool, err error) {
	// TODO(bentheelder): switch most (all) of the logging here to debug level
	// once we have configurable log levels
	// if this did not return an error, then the image exists locally
	cmd := exec.Command(binaryName, "inspect", "--type=image", image)
	if err := cmd.Run(); err == nil {
		logger.V(1).Infof("Image: %s present locally", image)
		return false, nil
	}
	// otherwise try to pull it
	return true, pull(logger, image, retries, binaryName)
}

// pull pulls an image, retrying up to retries times
func pull(logger log.Logger, image string, retries int, binaryName string) error {
	logger.V(1).Infof("Pulling image: %s ...", image)
	err := exec.Command(binaryName, "pull", image).Run()
	// retry pulling up to retries times if necessary
	if err != nil {
		for i := 0; i < retries; i++ {
			time.Sleep(time.Second * time.Duration(i+1))
			logger.V(1).Infof("Trying again to pull image: %q ... %v", image, err)
			// TODO(bentheelder): add some backoff / sleep?
			err = exec.Command(binaryName, "pull", image).Run()
			if err == nil {
				break
			}
		}
	}
	return errors.Wrapf(err, "failed to pull image %q", image)
}

// sanitizeImage is a helper to return human readable image name and
// the docker pullable image name from the provided image
func sanitizeImage(image string) (string, string) {
	if strings.Contains(image, "@sha256:") {
		return strings.Split(image, "@sha256:")[0], image
	}
	return image, image
}
