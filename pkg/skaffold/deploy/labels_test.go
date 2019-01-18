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
	"encoding/json"
	"net/http/httptest"
	"testing"

	rbac_v1 "k8s.io/api/rbac/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	skaffold_kubernetes "github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestLabelDeployResults_clusterWideResource(t *testing.T) {

	// fake handler that does PATCH and saves merge result
	ph := &patchHandler{
		orig: clusterRole,
		t:    t,
	}

	// fake server that will accept PATCH
	srv := httptest.NewServer(ph)

	// mocks
	skaffold_kubernetes.Client = getFakeClientsetWithDiscovery([]*meta_v1.APIResourceList{
		{
			GroupVersion: rbac_v1.SchemeGroupVersion.String(),
			APIResources: []meta_v1.APIResource{
				{
					Name:       "clusterroles",
					Namespaced: false,
					Kind:       "ClusterRole",
				},
			},
		},
	})
	skaffold_kubernetes.DynamicClient = getFakeDynamicClientFunc(srv.URL)

	// Run test target

	labelDeployResults(
		fooBarLabeller{}.Labels(),
		deployArtifacts,
	)

	// asserts for original artifact

	labels := deployArtifacts[0].Obj.(*rbac_v1.ClusterRole).GetLabels()
	testutil.CheckDeepEqual(t, 1, len(labels))
	testutil.CheckDeepEqual(t, "web-server", labels["app"])

	// asserts for patched resource

	var patchedClusterRole rbac_v1.ClusterRole
	err := json.Unmarshal(ph.patchedData, &patchedClusterRole)
	testutil.CheckError(t, false, err)

	testutil.CheckDeepEqual(t, map[string]string{
		"deployed-with": "skaffold",
		"foo":           "bar",
		"app":           "web-server",
	}, patchedClusterRole.GetLabels())
}

var deployArtifacts = []Artifact{
	{
		Obj:       clusterRole,
		Namespace: "does-not-matter",
	},
}

var clusterRole = &rbac_v1.ClusterRole{
	TypeMeta: meta_v1.TypeMeta{
		Kind:       "ClusterRole",
		APIVersion: "rbac.authorization.k8s.io/v1",
	},
	ObjectMeta: meta_v1.ObjectMeta{
		Name:      "web-server",
		Namespace: "", // Cluster-wide object has empty namespace.
		Labels: map[string]string{
			"app": "web-server",
		},
	},
	Rules: []rbac_v1.PolicyRule{
		{
			Verbs:     []string{"get", "list", "watch"},
			APIGroups: []string{""},
			Resources: []string{"endpoints"},
		},
	},
}

type fooBarLabeller struct{}

func (fooBarLabeller) Labels() map[string]string {
	return map[string]string{"foo": "bar"}
}
