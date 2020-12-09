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
package util

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// UnmarshalClusterVolumes provides a helper function to
// for a custom unmarshler to deal with
// https://github.com/GoogleContainerTools/skaffold/issues/4175
func UnmarshalClusterVolumes(value *yaml.Node) (volumes []v1.Volume, remaining []byte, result error) {
	clusterMap := make(map[string]interface{})

	value.Decode(clusterMap)

	if vMap, hasVolumes := clusterMap["volumes"]; hasVolumes {
		volumes = []v1.Volume{}
		volumesBuff, err := json.Marshal(vMap)

		if err != nil {
			result = err
			return
		}

		if err := json.Unmarshal(volumesBuff, &volumes); err != nil {
			result = err
			return
		}

		delete(clusterMap, "volumes")
	}

	// Remarshal the remaining values
	remaining, result = yaml.Marshal(clusterMap)

	return
}

// UnmarshalKanikoArtifact provides a helper function to
// for a custom unmarshaller to deal with
// https://github.com/GoogleContainerTools/skaffold/issues/4175
func UnmarshalKanikoArtifact(value *yaml.Node) (mounts []v1.VolumeMount, remaining []byte, result error) {
	kaMap := make(map[string]interface{})

	value.Decode(kaMap)

	if vMap, hasVolumes := kaMap["volumeMounts"]; hasVolumes {
		mounts = []v1.VolumeMount{}
		volumesBuff, err := json.Marshal(vMap)

		if err != nil {
			result = err
			return
		}

		if err := json.Unmarshal(volumesBuff, &mounts); err != nil {
			result = err
			return
		}

		delete(kaMap, "volumeMounts")
	}

	// Remarshal the remaining values
	remaining, result = yaml.Marshal(kaMap)
	return
}
