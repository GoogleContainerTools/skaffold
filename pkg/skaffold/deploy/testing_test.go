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
	"io/ioutil"
	"net/http"
	"testing"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/discovery"
	fakedisco "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	clienttesting "k8s.io/client-go/testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

// getFakeClientsetWithDiscovery returns a function that creates Clientset with
// a discovery using provided resources.
func getFakeClientsetWithDiscovery(resources []*meta_v1.APIResourceList) func() (kubernetes.Interface, error) {
	fakeDiscoveryClient := &fakedisco.FakeDiscovery{Fake: &clienttesting.Fake{}}
	fakeDiscoveryClient.Resources = resources

	return func() (kubernetes.Interface, error) {
		return &fakeClientset{
			Interface: fakekube.NewSimpleClientset(),
			discovery: fakeDiscoveryClient,
		}, nil
	}
}

// fakeClientset is a fake Clientset that has configurable discovery.
type fakeClientset struct {
	kubernetes.Interface
	discovery *fakedisco.FakeDiscovery
}

func (fc *fakeClientset) Discovery() discovery.DiscoveryInterface {
	return fc.discovery
}

// getFakeDynamicClientFunc returns a function that creates dynamic client.
func getFakeDynamicClientFunc(hostURL string) func() (dynamic.Interface, error) {
	return func() (dynamic.Interface, error) {
		return dynamic.NewForConfig(&rest.Config{
			Host: hostURL,
		})
	}
}

// patchHandler is a http handler that processes PATCH request.
type patchHandler struct {
	t           *testing.T
	orig        runtime.Object
	patchedData []byte
}

func (h *patchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PATCH" {
		h.t.Errorf("Expected PATCH method, got %s", r.Method)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	content := r.Header.Get("Content-Type")
	if content != string(types.StrategicMergePatchType) {
		h.t.Errorf("Expected %s Content-Type, got %s", types.StrategicMergePatchType, content)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	patch, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.t.Errorf("Unexpected error reading body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	orig := h.orig.DeepCopyObject()
	origJSON, _ := json.Marshal(orig)
	merged, err := strategicpatch.StrategicMergePatch(origJSON, patch, orig)
	if err != nil {
		h.t.Errorf("Unexpected error merging patch: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// to assert further
	h.patchedData = merged

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(merged)
	testutil.CheckError(h.t, false, err)
}
