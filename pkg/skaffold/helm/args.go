/*
Copyright 2022 The Skaffold Authors

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
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/mitchellh/go-homedir"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/output/log"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	maps "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util/map"
)

// ConstructOverrideArgs creates the command line arguments for overrides
func ConstructOverrideArgs(r *latest.HelmRelease, builds []graph.Artifact, args []string, manifestOverrides map[string]string) ([]string, error) {
	for _, k := range maps.SortKeys(r.SetValues) {
		args = append(args, "--set", fmt.Sprintf("%s=%s", k, r.SetValues[k]))
	}

	for _, k := range maps.SortKeys(r.SetFiles) {
		exp, err := homedir.Expand(r.SetFiles[k])
		if err != nil {
			return nil, fmt.Errorf("unable to expand %q: %w", r.SetFiles[k], err)
		}
		exp = SanitizeFilePath(exp, runtime.GOOS == "windows")

		args = append(args, "--set-file", fmt.Sprintf("%s=%s", k, exp))
	}

	envMap := map[string]string{}
	for idx, b := range builds {
		idxSuffix := ""
		// replace commonly used image name chars that are illegal helm template chars "/" & "-" with "_"
		nameSuffix := util.SanitizeHelmTemplateValue(b.ImageName)
		if idx > 0 {
			idxSuffix = strconv.Itoa(idx + 1)
		}

		for k, v := range envVarForImage(b.ImageName, b.Tag) {
			envMap[k+idxSuffix] = v
			envMap[k+"_"+nameSuffix] = v
		}
	}
	log.Entry(context.TODO()).Debugf("EnvVarMap: %+v\n", envMap)

	for _, k := range maps.SortKeys(r.SetValueTemplates) {
		v, err := util.ExpandEnvTemplate(r.SetValueTemplates[k], envMap)
		if err != nil {
			return nil, err
		}
		expandedKey, err := util.ExpandEnvTemplate(k, envMap)
		if err != nil {
			return nil, err
		}

		// hack required when using new Skaffold v2.X.Y setValueTemplates w/ Skaffold v1.X.Y "imageStrategy: helm"
		// ex: image: "{{.Values.image.repository}}:{{.Values.image.tag}}"
		// when the helm template replacements are done with `dev` and `run` there
		// is an additional `@` suffix inserted that needs to be removed or else deploys will fail
		v = strings.TrimSuffix(v, "@")

		args = append(args, "--set", fmt.Sprintf("%s=%s", expandedKey, v))
	}

	for _, v := range r.ValuesFiles {
		exp, err := homedir.Expand(v)
		if err != nil {
			return nil, fmt.Errorf("unable to expand %q: %w", v, err)
		}

		exp, err = util.ExpandEnvTemplate(exp, envMap)
		if err != nil {
			return nil, err
		}

		args = append(args, "-f", exp)
	}

	for _, k := range maps.SortKeys(manifestOverrides) {
		args = append(args, "--set", fmt.Sprintf("%s=%s", k, manifestOverrides[k]))
	}

	return args, nil
}

// GetArgs calculates the correct arguments to "helm get"
func GetArgs(releaseName string, namespace string) []string {
	args := []string{"get", "all"}
	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}
	return append(args, releaseName)
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
		log.Entry(context.TODO()).Warnf("unable to extract values for %v, %v and %v from image %v due to error:\n%v", constants.ImageRef.Repo, constants.ImageRef.Tag, constants.ImageRef.Digest, digest, err)
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

	// IMAGE_DOMAIN and IMAGE_REPO_NO_DOMAIN added for v2beta* 'helm+explicitRegistry' -> v3alpha* and beyond compatibility
	customMap["IMAGE_DOMAIN"] = ref.Domain
	customMap["IMAGE_REPO_NO_DOMAIN"] = strings.TrimPrefix(ref.BaseName, ref.Domain+"/")
	customMap["IMAGE_FULLY_QUALIFIED"] = digest
	return customMap
}
