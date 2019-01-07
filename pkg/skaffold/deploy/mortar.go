/*
Copyright 2018 The Skaffold Authors

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

package deploy

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// MortarDeployer deploys workflows using mortar.
type MortarDeployer struct {
	*latest.MortarDeploy
}

func NewMortarDeployer(cfg *latest.MortarDeploy) *MortarDeployer {
	return &MortarDeployer{
		MortarDeploy: cfg,
	}
}

// Labels returns the labels specific to mortar.
func (m *MortarDeployer) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Deployer: "mortar",
	}
}

// Deploy runs `mortar fire ...` on the configured manifests/templates.
func (m *MortarDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact) ([]Artifact, error) {
	logrus.Debugf("Mortar::Deploy: Cfg: %v\n", m.MortarDeploy)

	args := make([]string, 0)
	args = append(args, "fire")
	if m.MortarDeploy.Config != "" {
		args = append(args, "-c")
		args = append(args, m.MortarDeploy.Config)
	}
	args = append(args, m.MortarDeploy.Source)
	args = append(args, m.MortarDeploy.Name)

	logrus.Debugf("Firing with args: %v", args)
	cmd := exec.CommandContext(ctx, "mortar", args...)
	err := util.RunCmd(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "mortar fire")
	}
	return nil, nil
}

// Cleanup deletes what was deployed by calling Deploy.
func (m *MortarDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	cmd := exec.CommandContext(ctx, "mortar", "yank", "--force", m.MortarDeploy.Name)
	err := util.RunCmd(cmd)
	if err != nil {
		return errors.Wrap(err, "mortar yank")
	}
	return nil
}

// Dependencies lists all the files that can change what needs to be deployed.
func (m *MortarDeployer) Dependencies() ([]string, error) {
	var deps []string
	// Add all related manifest files
	list, err := findManifestFiles(m.MortarDeploy.Source)
	if err != nil {
		return nil, errors.Wrap(err, "expanding mortar manifest paths")
	}
	deps = append(deps, list...)

	// Look for config file and append if found
	switch {
	case m.MortarDeploy.Config != "":
		deps = append(deps, m.MortarDeploy.Config)
	case fileExists("shot.yml"):
		deps = append(deps, "shot.yml")
	case fileExists("shot.yaml"):
		deps = append(deps, "shot.yaml")
	}

	logrus.Debugf("Found dependencies for mortar: %v", deps)

	return deps, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var knownSuffixes = [...]string{"yaml", "yml", "yaml.erb", "yml.erb"}

func hasKnownSuffix(file string) bool {
	for _, suffix := range knownSuffixes {
		if strings.HasSuffix(file, suffix) {
			return true
		}
	}
	return false
}

func findManifestFiles(base string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if hasKnownSuffix(path) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		logrus.Warnf("Error reading manifest files: %s", err)
		return nil, err
	}
	return files, nil
}
