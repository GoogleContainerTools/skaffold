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

package encoding

import (
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"

	"sigs.k8s.io/kind/pkg/internal/apis/config"
)

// V1Alpha4ToInternal converts to the internal API version
func V1Alpha4ToInternal(cluster *v1alpha4.Cluster) *config.Cluster {
	v1alpha4.SetDefaultsCluster(cluster)
	return config.Convertv1alpha4(cluster)
}
