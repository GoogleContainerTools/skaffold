/*
Copyright 2020 The Skaffold Authors

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

package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/blang/semver"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

func IsHelmChart(path string) bool {
	return filepath.Base(path) == "Chart.yaml"
}

// copy of cmd/skaffold/app/flags.BuildOutputs
type buildOutputs struct {
	Builds []build.Artifact `json:"builds"`
}

func writeBuildArtifacts(builds []build.Artifact) (string, func(), error) {
	buildOutput, err := json.Marshal(buildOutputs{builds})
	if err != nil {
		return "", nil, fmt.Errorf("cannot marshal build artifacts: %w", err)
	}

	f, err := ioutil.TempFile("", "builds*.yaml")
	if err != nil {
		return "", nil, fmt.Errorf("cannot create temp file: %w", err)
	}
	if _, err := f.Write(buildOutput); err != nil {
		return "", nil, fmt.Errorf("cannot write to temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", nil, fmt.Errorf("cannot close temp file: %w", err)
	}
	return f.Name(), func() { os.Remove(f.Name()) }, nil
}

// sortKeys returns the map keys in sorted order
func sortKeys(m map[string]string) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

// binVer returns the version of the helm binary found in PATH.
func binVer() (semver.Version, error) {
	cmd := exec.Command("helm", "version", "--client")
	b, err := util.RunCmdOut(cmd)
	if err != nil {
		return semver.Version{}, fmt.Errorf("helm version command failed %q: %w", string(b), err)
	}
	raw := string(b)
	matches := versionRegex.FindStringSubmatch(raw)
	if len(matches) == 0 {
		return semver.Version{}, fmt.Errorf("unable to parse output: %q", raw)
	}
	return semver.ParseTolerant(matches[1])
}

// imageSetFromConfig calculates the --set-string value from the helm config
func imageSetFromConfig(cfg *latest.HelmConventionConfig, valueName string, tag string) (string, error) {
	if cfg == nil {
		return fmt.Sprintf("%s=%s", valueName, tag), nil
	}

	ref, err := docker.ParseReference(tag)
	if err != nil {
		return "", fmt.Errorf("cannot parse the image reference %q: %w", tag, err)
	}

	var imageTag string
	if ref.Digest != "" {
		imageTag = fmt.Sprintf("%s@%s", ref.Tag, ref.Digest)
	} else {
		imageTag = ref.Tag
	}

	if cfg.ExplicitRegistry {
		if ref.Domain == "" {
			return "", fmt.Errorf("image reference %s has no domain", tag)
		}
		return fmt.Sprintf("%[1]s.registry=%[2]s,%[1]s.repository=%[3]s,%[1]s.tag=%[4]s", valueName, ref.Domain, ref.Path, imageTag), nil
	}

	return fmt.Sprintf("%[1]s.repository=%[2]s,%[1]s.tag=%[3]s", valueName, ref.BaseName, imageTag), nil
}

// pairParamsToArtifacts associates parameters to the build artifact it creates
func pairParamsToArtifacts(builds []build.Artifact, params map[string]string) (map[string]build.Artifact, error) {
	imageToBuildResult := map[string]build.Artifact{}
	for _, b := range builds {
		imageToBuildResult[b.ImageName] = b
	}

	paramToBuildResult := map[string]build.Artifact{}

	for param, imageName := range params {
		b, ok := imageToBuildResult[imageName]
		if !ok {
			return nil, noMatchingBuild(imageName)
		}

		paramToBuildResult[param] = b
	}

	return paramToBuildResult, nil
}

func (h *Deployer) generateSkaffoldDebugFilter(buildsFile string) []string {
	args := []string{"filter", "--debugging", "--kube-context", h.kubeContext}
	if len(buildsFile) > 0 {
		args = append(args, "--build-artifacts", buildsFile)
	}
	args = append(args, h.Flags.Global...)

	if h.kubeConfig != "" {
		args = append(args, "--kubeconfig", h.kubeConfig)
	}
	return args
}

func (h *Deployer) releaseNamespace(r latest.HelmRelease) (string, error) {
	if h.namespace != "" {
		return h.namespace, nil
	} else if r.Namespace != "" {
		namespace, err := util.ExpandEnvTemplateOrFail(r.Namespace, nil)
		if err != nil {
			return "", fmt.Errorf("cannot parse the release namespace template: %w", err)
		}
		return namespace, nil
	}
	return "", nil
}

// envVarForImage creates an environment map for an image and digest tag (fqn)
func envVarForImage(imageName string, digest string) map[string]string {
	customMap := map[string]string{
		"IMAGE_NAME": imageName,
		"DIGEST":     digest, // The `DIGEST` name is kept for compatibility reasons
	}

	// Standardize access to Image reference fields in templates
	ref, err := docker.ParseReference(digest)
	if err == nil {
		customMap[constants.ImageRef.Repo] = ref.BaseName
		customMap[constants.ImageRef.Tag] = ref.Tag
		customMap[constants.ImageRef.Digest] = ref.Digest
	} else {
		logrus.Warnf("unable to extract values for %v, %v and %v from image %v due to error:\n%v", constants.ImageRef.Repo, constants.ImageRef.Tag, constants.ImageRef.Digest, digest, err)
	}

	if digest == "" {
		return customMap
	}

	// DIGEST_ALGO and DIGEST_HEX are deprecated and will contain nonsense values
	names := strings.SplitN(digest, ":", 2)
	if len(names) >= 2 {
		customMap["DIGEST_ALGO"] = names[0]
		customMap["DIGEST_HEX"] = names[1]
	} else {
		customMap["DIGEST_HEX"] = digest
	}
	return customMap
}

// exec executes the helm command, writing combined stdout/stderr to the provided writer
func (h *Deployer) exec(ctx context.Context, out io.Writer, useSecrets bool, env []string, args ...string) error {
	args = append([]string{"--kube-context", h.kubeContext}, args...)
	args = append(args, h.Flags.Global...)

	if h.kubeConfig != "" {
		args = append(args, "--kubeconfig", h.kubeConfig)
	}

	if useSecrets {
		args = append([]string{"secrets"}, args...)
	}

	cmd := exec.CommandContext(ctx, "helm", args...)
	if len(env) > 0 {
		cmd.Env = env
	}
	cmd.Stdout = out
	cmd.Stderr = out

	return util.RunCmd(cmd)
}
