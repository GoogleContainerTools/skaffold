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

package inspect

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/kubectl/pkg/scheme"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/inspect"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/webhook/constants"
)

type resourceToInfoContainer struct {
	ResourceToInfoMap map[string][]resourceInfo `json:"resourceToInfoMap"`
}

type resourceInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func PrintNamespacesList(ctx context.Context, out io.Writer, manifestFile string, opts inspect.Options) error {
	// do some additional processing here
	b, err := ioutil.ReadFile(manifestFile)
	if err != nil {
		return err
	}

	// Create a runtime.Decoder from the Codecs field within
	// k8s.io/client-go that's pre-loaded with the schemas for all
	// the standard Kubernetes resource types.
	decoder := scheme.Codecs.UniversalDeserializer()

	resourceToInfoMap := map[string][]resourceInfo{}
	for _, resourceYAML := range strings.Split(string(b), "---") {
		// skip empty documents, `Decode` will fail on them
		if len(resourceYAML) == 0 {
			continue
		}
		// - obj is the API object (e.g., Deployment)
		// - groupVersionKind is a generic object that allows
		//   detecting the API type we are dealing with, for
		//   accurate type casting later.
		obj, groupVersionKind, err := decoder.Decode(
			[]byte(resourceYAML),
			nil,
			nil)
		if err != nil {
			log.Print(err)
			continue
		}
		// Only process Deployments for now
		if groupVersionKind.Group == "apps" && groupVersionKind.Version == "v1" && groupVersionKind.Kind == "Deployment" {
			deployment := obj.(*appsv1.Deployment)

			if _, ok := resourceToInfoMap[groupVersionKind.String()]; !ok {
				resourceToInfoMap[groupVersionKind.String()] = []resourceInfo{}
			}
			resourceToInfoMap[groupVersionKind.String()] = append(resourceToInfoMap[groupVersionKind.String()], resourceInfo{
				Name:      deployment.ObjectMeta.Name,
				Namespace: deployment.ObjectMeta.Namespace,
			})
		}
	}

	formatter := inspect.OutputFormatter(out, opts.OutFormat)
	cfgs, err := inspect.GetConfigSet(ctx, config.SkaffoldOptions{
		ConfigurationFile:   opts.Filename,
		ConfigurationFilter: opts.Modules,
		RepoCacheDir:        opts.RepoCacheDir,
		Profiles:            opts.Profiles,
		PropagateProfiles:   opts.PropagateProfiles,
	})
	if err != nil {
		formatter.WriteErr(err)
		return err
	}

	defaultNamespace := constants.Namespace
	flagNamespace := ""
	for _, c := range cfgs {
		if c.Deploy.KubectlDeploy != nil {
			if c.Deploy.KubectlDeploy.DefaultNamespace != nil && *c.Deploy.KubectlDeploy.DefaultNamespace != "" {
				defaultNamespace = *c.Deploy.KubectlDeploy.DefaultNamespace
			}
			if namespaceVal := util.ParseNamespaceFromFlags(c.Deploy.KubectlDeploy.Flags.Global); namespaceVal != "" {
				flagNamespace = namespaceVal
			}
			if namespaceVal := util.ParseNamespaceFromFlags(c.Deploy.KubectlDeploy.Flags.Apply); namespaceVal != "" {
				flagNamespace = namespaceVal
			}
			// NOTE: Cloud Deploy uses `skaffold apply` which always uses kubectl deployer.  As such other
			// namespace config should be ignored - eg: .Deploy.LegacyHelmDeploy.Releases[i].Namespace
		}
	}

	for gvk, ris := range resourceToInfoMap {
		for i := range ris {
			if ris[i].Namespace == "" {
				if flagNamespace != "" {
					resourceToInfoMap[gvk][i].Namespace = flagNamespace
					continue
				}
				resourceToInfoMap[gvk][i].Namespace = defaultNamespace
			}
		}
	}
	l := &resourceToInfoContainer{ResourceToInfoMap: resourceToInfoMap}

	return formatter.Write(l)
}
